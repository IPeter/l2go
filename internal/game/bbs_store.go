package game

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/IPeter/l2go/internal/bbs"
	"github.com/IPeter/l2go/internal/storage"
)

// The community board's storage (Plan 14 M11o, decision X4): the mails and
// favorites are loaded at start (Mobius MailBoard loads every mail too) and
// every change is written in the background by boardWriter, one write at a
// time in the order the loop made them, so a mail is stored before it is
// marked read or moved.

// boardLoadTimeout bounds reading the mails and favorites at start;
// boardWriteTimeout one write.
const (
	boardLoadTimeout  = 30 * time.Second
	boardWriteTimeout = 10 * time.Second
)

// loadBoard reads the stored mails and favorites into the feature.
func (s *Server) loadBoard() error {
	f := &s.board
	f.mail, f.favs = bbs.NewMailboxes(nil), bbs.NewFavorites(nil)
	if f.store == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), boardLoadTimeout)
	defer cancel()
	rows, err := f.store.LoadBBSMails(ctx)
	if err != nil {
		return err
	}
	mails := make([]bbs.Mail, 0, len(rows))
	for _, r := range rows {
		t, ok := bbs.ParseMailType(r.Location)
		if !ok {
			s.log.Warn("game: a bbs mail with an unknown location is left out", "id", r.ID, "location", r.Location)
			continue
		}
		mails = append(mails, bbs.Mail{ID: r.ID, Receiver: r.ReceiverID, Sender: r.SenderID, SenderName: r.SenderName, Type: t,
			Recipients: r.Recipients, Subject: r.Subject, Message: r.Message, Sent: time.UnixMilli(r.SentDate), Unread: r.Unread})
	}
	favRows, err := f.store.LoadBBSFavorites(ctx)
	if err != nil {
		return err
	}
	favs := make([]bbs.Favorite, 0, len(favRows))
	for _, r := range favRows {
		favs = append(favs, bbs.Favorite{ID: r.ID, Player: r.PlayerID, Title: r.Title, Bypass: r.Bypass, Added: time.UnixMilli(r.AddDate)})
	}
	f.mail, f.favs = bbs.NewMailboxes(mails), bbs.NewFavorites(favs)
	return nil
}

// mailRow is a mail as it is stored.
func mailRow(m *bbs.Mail) storage.BBSMail {
	return storage.BBSMail{ID: m.ID, ReceiverID: m.Receiver, SenderID: m.Sender, SenderName: m.SenderName, Location: m.Type.Location(),
		Recipients: m.Recipients, Subject: m.Subject, Message: m.Message, SentDate: m.Sent.UnixMilli(), Unread: m.Unread}
}

// saveMails writes mails in one write, in the background.
func (s *Server) saveMails(mails ...*bbs.Mail) {
	store := s.board.store
	if store == nil || len(mails) == 0 {
		return
	}
	rows := make([]storage.BBSMail, len(mails))
	for i, m := range mails {
		rows[i] = mailRow(m)
	}
	s.board.writer.do("save mails", func(ctx context.Context) error { return store.SaveBBSMails(ctx, rows) })
}

// deleteMailRow deletes a stored mail in the background.
func (s *Server) deleteMailRow(id int32) {
	if store := s.board.store; store != nil {
		s.board.writer.do("delete a mail", func(ctx context.Context) error { return store.DeleteBBSMail(ctx, id) })
	}
}

// saveFavorite writes a favorite in the background.
func (s *Server) saveFavorite(fav bbs.Favorite) {
	if store := s.board.store; store != nil {
		row := storage.BBSFavorite{ID: fav.ID, PlayerID: fav.Player, Title: fav.Title, Bypass: fav.Bypass, AddDate: fav.Added.UnixMilli()}
		s.board.writer.do("save a favorite", func(ctx context.Context) error { return store.SaveBBSFavorite(ctx, row) })
	}
}

// deleteFavoriteRow deletes a stored favorite in the background.
func (s *Server) deleteFavoriteRow(player, id int32) {
	if store := s.board.store; store != nil {
		s.board.writer.do("delete a favorite", func(ctx context.Context) error { return store.DeleteBBSFavorite(ctx, player, id) })
	}
}

// waitBoardWrites waits for the board's pending writes when the server
// stops.
func (s *Server) waitBoardWrites() {
	ctx, cancel := context.WithTimeout(s.background(), shutdownTimeout)
	defer cancel()
	if err := s.board.writer.wait(ctx); err != nil {
		s.log.Error("game: community board writes still pending at shutdown", "err", err)
	}
}

// boardWriter runs the board's writes one at a time, in order, on its own
// goroutine; do never blocks. A failed write is logged and dropped, as
// Mobius logs a failed statement. Safe for concurrent use.
type boardWriter struct {
	log *slog.Logger
	bg  func() context.Context

	mu      sync.Mutex
	queue   []boardWrite
	running bool
	idle    chan struct{} // closed when the queue empties; nil while idle
}

type boardWrite struct {
	what string
	fn   func(ctx context.Context) error
}

func newBoardWriter(log *slog.Logger, bg func() context.Context) *boardWriter {
	return &boardWriter{log: log, bg: bg}
}

// do queues a write.
func (w *boardWriter) do(what string, fn func(ctx context.Context) error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.queue = append(w.queue, boardWrite{what: what, fn: fn})
	if w.running {
		return
	}
	w.running = true
	w.idle = make(chan struct{})
	go w.run()
}

func (w *boardWriter) run() {
	for {
		w.mu.Lock()
		if len(w.queue) == 0 {
			w.running = false
			close(w.idle)
			w.idle = nil
			w.mu.Unlock()
			return
		}
		next := w.queue[0]
		w.queue = w.queue[1:]
		w.mu.Unlock()

		ctx, cancel := context.WithTimeout(w.bg(), boardWriteTimeout)
		if err := next.fn(ctx); err != nil {
			w.log.Error("game: community board write failed", "what", next.what, "err", err)
		}
		cancel()
	}
}

// wait waits until every queued write is done, or ctx ends.
func (w *boardWriter) wait(ctx context.Context) error {
	w.mu.Lock()
	idle := w.idle
	w.mu.Unlock()
	if idle == nil {
		return nil
	}
	select {
	case <-idle:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

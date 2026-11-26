package storage

import "context"

// The community board (Plan 14 M11o, decision X4): the mail board's mails
// (Mobius bbs_mail) and the bookmarked board pages (Mobius bbs_favorites).
// The game loads both at start and writes every change in the background,
// in order, never from the world loop.

// BBSMail is one copy of a mail: the recipient's (in its inbox, or where it
// moved it) or the sender's (in its sent box). A mail to three players is
// four rows, as in Mobius.
type BBSMail struct {
	ID         int32 // the key
	ReceiverID int32 // the owner of the copy
	SenderID   int32
	// SenderName is the sender's name when it sent the mail (not in Mobius,
	// which looks the name up at every display).
	SenderName string
	// Location is the folder: "inbox", "sentbox", "archive" or
	// "temparchive".
	Location   string
	Recipients string // the names the sender typed, joined with ";"
	Subject    string
	Message    string
	SentDate   int64 // Unix milliseconds
	Unread     bool
}

// BBSFavorite is a bookmarked board page.
type BBSFavorite struct {
	ID       int32 // the key, from the game's counter
	PlayerID int32
	Title    string
	Bypass   string
	AddDate  int64 // Unix milliseconds
}

// BBSStore keeps the community board's mails and favorites.
type BBSStore interface {
	// LoadBBSMails returns every stored mail sorted by ID, nil for none.
	LoadBBSMails(ctx context.Context) ([]BBSMail, error)
	// SaveBBSMails stores the mails, each replacing the one with its ID, in
	// one write.
	SaveBBSMails(ctx context.Context, mails []BBSMail) error
	// DeleteBBSMail deletes the mail with the ID; a missing one is no error.
	DeleteBBSMail(ctx context.Context, id int32) error

	// LoadBBSFavorites returns every stored favorite sorted by ID, nil for
	// none.
	LoadBBSFavorites(ctx context.Context) ([]BBSFavorite, error)
	// SaveBBSFavorite stores a favorite, replacing the one with its ID.
	SaveBBSFavorite(ctx context.Context, f BBSFavorite) error
	// DeleteBBSFavorite deletes the player's favorite with the ID; a missing
	// one, or one of another player, is no error and stays.
	DeleteBBSFavorite(ctx context.Context, playerID, id int32) error
}

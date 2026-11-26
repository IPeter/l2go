package game

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/quest"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The quest feature (Plan 11, phase Q4): each player's quest states
// (internal/quest) and player variables, saved with the character and its
// items; QuestList, the quest log's abort and the four tutorial packets;
// the global variables of the scripts. The quest window is in
// dialog_script.go, the world events in quest_events.go, the scripts' world
// (scripting.QuestWorld) in quest_world.go and the class change in
// quest_class.go.
func init() { registerFeature("quest", setupQuests) }

// tutorialQuest is the newbie tutorial (Mobius Q00255_Tutorial): the game
// sends it the "UC", "CE" and "QM" events and the tutorial links.
const tutorialQuest = "Q00255_Tutorial"

// questMaxActive is Mobius's quest limit: a new quest is refused when more
// quests than this are active (so 26 is the real maximum).
const questMaxActive = 25

// System messages of the quest feature.
const (
	// "Progress in a quest is possible only when your inventory's weight
	// and volume are less than 80 percent of capacity."
	sysMsgQuestInventoryFull serverpackets.SystemMessageID = 1118
)

// QuestSettings are the quest rules of a server (Mobius Rates.ini,
// Player.ini and General.ini). The zero value of a field means Mobius's
// default.
type QuestSettings struct {
	// SaveDelay is how long after a quest change the character is saved
	// (quests.saveDelay, 2 s).
	SaveDelay time.Duration
	// KillEventDelay is how long after a death the kill handlers run
	// (Mobius _onKillDelay, 2.5 s).
	KillEventDelay time.Duration
	// DisableTutorial turns the tutorial events off (DisableTutorial).
	DisableTutorial bool
	// NewCharAlwaysNewbie makes every character a newbie
	// (AltNewCharAlwaysIsNewbie).
	NewCharAlwaysNewbie bool
	// AutoDeleteInvalidQuestData deletes the stored states of quests that
	// are not loaded (AutoDeleteInvalidQuestData).
	AutoDeleteInvalidQuestData bool
	// The quest reward rates (RateQuestReward*); zero means 1.
	RewardRate, RewardAdenaRate, RewardXPRate, RewardSPRate                  float64
	RewardPotionRate, RewardScrollRate, RewardRecipeRate, RewardMaterialRate float64
	// UseRewardMultipliers applies the rates by item type
	// (UseQuestRewardMultipliers).
	UseRewardMultipliers bool
	// ItemDropAmount multiplies the quest items given (and those of
	// giveItemRandomly, on the script side) (QuestItemDropAmountMultiplier);
	// zero means 1.
	ItemDropAmount float64
}

// Mobius defaults of QuestSettings.
const (
	defaultQuestSaveDelay = 2 * time.Second
	defaultQuestKillDelay = 2500 * time.Millisecond
)

func (q QuestSettings) withDefaults() QuestSettings {
	if q.SaveDelay <= 0 {
		q.SaveDelay = defaultQuestSaveDelay
	}
	if q.KillEventDelay <= 0 {
		q.KillEventDelay = defaultQuestKillDelay
	}
	for _, r := range []*float64{
		&q.RewardRate, &q.RewardAdenaRate, &q.RewardXPRate, &q.RewardSPRate,
		&q.RewardPotionRate, &q.RewardScrollRate, &q.RewardRecipeRate, &q.RewardMaterialRate,
		&q.ItemDropAmount,
	} {
		if *r <= 0 {
			*r = 1
		}
	}
	return q
}

// questFeature is the quest feature's state on the Server (Server.quest).
// Loop-only, except settings and store, which New sets.
type questFeature struct {
	settings QuestSettings
	store    storage.GlobalVariableStore // nil keeps the globals in memory

	globals     *quest.Vars   // the global variables
	globalsSave world.TimerID // the pending save of the globals, or 0
	// scriptValues are the NPCs' script values (Mobius Npc.getScriptValue);
	// a spawn resets them.
	scriptValues map[*Npc]int
	// started is set once the scripts' on_start handlers ran.
	started bool
}

// playerQuests is a player's quest side (Player.quests): its quest states
// and variables, made when the player enters the world. Loop-only.
type playerQuests struct {
	book *quest.Book
	vars *quest.Vars
	// unknown are the stored states of quests that are not loaded: kept
	// as they are (decision Q4).
	unknown []storage.Quest
	// deathNotify are the quests told of the player's death (Mobius
	// addNotifyQuestOfDeath), in order.
	deathNotify []string
	// saveTimer is the pending quest save, or 0.
	saveTimer world.TimerID
}

// setupQuests registers the quest packets and hooks and loads the global
// variables. The scripts' on_start handlers run once the loop runs.
func setupQuests(s *Server) error {
	f := &s.quest
	f.scriptValues = make(map[*Npc]int)
	f.globals = quest.NewVars(nil)
	if f.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), saveTimeout)
		defer cancel()
		m, err := f.store.LoadGlobalVariables(ctx)
		if err != nil {
			return err
		}
		f.globals = quest.NewVars(m)
	}
	s.OnPlayerEnter(s.questsEnter)
	s.OnPlayerLeave(s.questsLeave)
	s.OnDie(s.questDied)
	s.OnHit(s.questHit)
	if err := errors.Join(
		s.Register(clientpackets.OpRequestQuestAbort, s.handleRequestQuestAbort),
		s.Register(clientpackets.OpRequestTutorialLinkHtml, s.handleTutorialLinkHtml),
		s.Register(clientpackets.OpRequestTutorialPassCmdToServer, s.handleTutorialPassCmd),
		s.Register(clientpackets.OpRequestTutorialQuestionMark, s.handleTutorialQuestionMark),
		s.Register(clientpackets.OpRequestTutorialClientEvent, s.handleTutorialClientEvent),
		s.registerQuestGMCommands(),
	); err != nil {
		return err
	}
	s.loop.Post(s.startScripts)
	return nil
}

// startScripts runs the scripts' on_start handlers once the world is ready
// (spawned), then their spawn handlers for the NPCs already in the world
// (Mobius loads the scripts before the spawns).
func (s *Server) startScripts() {
	if s.scripts == nil || s.quest.started {
		return
	}
	s.quest.started = true
	ctx := s.background()
	if err := s.scripts.Start(ctx); err != nil {
		s.log.Warn("game: a script's on_start failed", "err", err)
	}
	for _, sp := range s.npc.spawns {
		if n := sp.npc; n != nil && n.InWorld() {
			_ = s.scripts.Spawn(ctx, n.ScriptRef()) // failures are logged
		}
	}
}

// questLookup is the quest model's view of the loaded scripts.
func (s *Server) questLookup(name string) (quest.Info, bool) {
	if s.scripts == nil {
		return quest.Info{}, false
	}
	q, ok := s.scripts.Quest(name)
	if !ok {
		return quest.Info{}, false
	}
	return quest.Info{ID: int32(q.ID), Name: q.Name, Custom: q.Custom}, true //nolint:gosec // quest IDs are small
}

// questsOf returns p's quest side, loading the stored quests on first use
// (Mobius Quest.playerEnter). Loop-only.
func (s *Server) questsOf(p *Player) *playerQuests {
	q := &p.quests
	if q.book != nil {
		return q
	}
	q.book = quest.NewBook(s.questLookup, questListener{s: s, p: p}, s.Now)
	q.vars = quest.NewVars(p.char.Vars)
	if unknown := q.book.Load(p.char.Quests); len(unknown) > 0 {
		if s.quest.settings.AutoDeleteInvalidQuestData {
			q.book.DeleteStored(unknown...)
		} else {
			for _, d := range p.char.Quests {
				for _, name := range unknown {
					if d.Name == name {
						q.unknown = append(q.unknown, d)
					}
				}
			}
		}
		s.log.Info("game: stored quests that are not loaded", append(p.logArgs(), "quests", unknown, "deleted", s.quest.settings.AutoDeleteInvalidQuestData)...)
	}
	return q
}

// QuestState returns p's state of a quest, or nil (Mobius
// getQuestState). Loop-only.
func (s *Server) QuestState(p *Player, name string) *quest.State {
	return s.questsOf(p).book.Get(name)
}

// questListPacket builds p's QuestList (Mobius QuestList): the started
// quests with IDs 1 to 19999, by name.
func (s *Server) questListPacket(p *Player) []byte {
	active := s.questsOf(p).book.Active()
	entries := make([]serverpackets.QuestListEntry, len(active))
	for i, e := range active {
		entries[i] = serverpackets.QuestListEntry{ID: e.ID, Cond: e.Status}
	}
	return serverpackets.NewQuestListPacket(entries)
}

// questsEnterWorld runs Mobius EnterWorld's quest part, where the empty
// QuestList used to go: the stored quests load, the tutorial gets "UC",
// and QuestList is sent.
func (s *Server) questsEnterWorld(p *Player) {
	s.questsOf(p)
	s.tutorialEvent(p, "UC")
	p.Send(s.questListPacket(p))
}

// questsEnter is the OnPlayerEnter hook: the scripts that asked for it hear
// of the player (Mobius onPlayerEnter's login listeners).
func (s *Server) questsEnter(p *Player) {
	if s.scripts != nil {
		_ = s.scripts.EnterWorld(s.background(), p) // failures are logged
	}
}

// questsLeave is the OnPlayerLeave hook: the pending quest save goes with
// the leave's save, and the player's quest timers stop.
func (s *Server) questsLeave(p *Player) {
	if p.quests.saveTimer != 0 {
		s.loop.Cancel(p.quests.saveTimer)
		p.quests.saveTimer = 0
	}
	if s.scripts != nil {
		s.scripts.PlayerLeft(p.ObjectID())
	}
}

// scheduleQuestSave saves p quests.saveDelay from now, unless a save is
// already scheduled.
func (s *Server) scheduleQuestSave(p *Player) {
	if p.state != stateInGame || p.quests.saveTimer != 0 {
		return
	}
	p.quests.saveTimer = s.loop.After(s.quest.settings.SaveDelay, func() {
		p.quests.saveTimer = 0
		if p.state == stateInGame {
			s.Save(p, nil)
		}
	})
}

// takeSaveChanges returns what changed since p's last save: its items and
// its quests (written in one transaction, decision Q4).
func (p *Player) takeSaveChanges() storage.ItemChanges {
	ch := p.takeItemChanges()
	if p.quests.book != nil {
		ch.Quests = p.quests.book.TakeChanges()
	}
	return ch
}

// questSaveDone wraps done so that quest changes whose write failed for
// good are marked dirty again on the loop.
func (s *Server) questSaveDone(p *Player, quests storage.QuestChanges, done func(error)) func(error) {
	if quests.Empty() {
		return done
	}
	return func(err error) {
		if err != nil {
			s.loop.Post(func() {
				if p.state != stateLeft && p.quests.book != nil {
					p.quests.book.SaveFailed(quests)
				}
			})
		}
		if done != nil {
			done(err)
		}
	}
}

// questDocuments returns p's quest documents for a full save: the states
// of the book and the stored ones of quests that are not loaded.
func (p *Player) questDocuments() []storage.Quest {
	if p.quests.book == nil {
		return p.char.Quests
	}
	return append(p.quests.book.Documents(), p.quests.unknown...)
}

// questListener is the quest model's Listener for one player.
type questListener struct {
	s *Server
	p *Player
}

var _ quest.Listener = questListener{}

func (l questListener) QuestList() { l.p.Send(l.s.questListPacket(l.p)) }

func (l questListener) QuestMark(id int32) { l.p.Send(serverpackets.NewExShowQuestMarkPacket(id)) }

func (l questListener) PlaySound(sound string) { l.p.Send(serverpackets.NewQuestSoundPacket(sound)) }

func (l questListener) RemoveDeathNotify(st *quest.State) {
	q := &l.p.quests
	for i, name := range q.deathNotify {
		if name == st.Name() {
			q.deathNotify = append(q.deathNotify[:i], q.deathNotify[i+1:]...)
			return
		}
	}
}

// RemoveQuestItems takes the quest's registered items (Mobius
// removeRegisteredQuestItems).
func (l questListener) RemoveQuestItems(st *quest.State) {
	if l.s.scripts == nil {
		return
	}
	info, ok := l.s.scripts.Quest(st.Name())
	if !ok {
		return
	}
	for _, id := range info.QuestItems {
		l.s.takeQuestItems(l.p, id, -1)
	}
}

func (l questListener) Changed() { l.s.scheduleQuestSave(l.p) }

// AddDeathNotify makes the quest hear of p's death (Mobius
// addNotifyQuestOfDeath). Loop-only.
func (s *Server) AddDeathNotify(p *Player, questName string) {
	q := s.questsOf(p)
	for _, name := range q.deathNotify {
		if name == questName {
			return
		}
	}
	q.deathNotify = append(q.deathNotify, questName)
}

// handleRequestQuestList answers RequestQuestList (0x63) with the quest list
// (Mobius RequestQuestList).
func (s *Server) handleRequestQuestList(p *Player, _ []byte) { p.Send(s.questListPacket(p)) }

// handleRequestQuestAbort handles RequestQuestAbort (0x64), the abort of
// the quest log (Mobius RequestQuestAbort): a quest that is not completed
// ends as exit_quest(true) and QuestList follows. Q503's clan branch waits
// for clans.
func (s *Server) handleRequestQuestAbort(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestQuestAbort(body)
	if err != nil || s.scripts == nil {
		return
	}
	info, ok := s.scripts.QuestByID(int(req.QuestID))
	if !ok {
		return
	}
	st := s.QuestState(p, info.Name)
	if st == nil || st.IsCompleted() {
		return
	}
	st.ExitQuest(true, false)
	p.Send(s.questListPacket(p))
}

// tutorialEvent sends event to the tutorial (Mobius notifyEvent on
// Q00255_Tutorial) when the player has its state and the tutorial is on.
func (s *Server) tutorialEvent(p *Player, event string) {
	if s.scripts == nil || s.quest.settings.DisableTutorial || s.QuestState(p, tutorialQuest) == nil {
		return
	}
	s.scriptEventReply(p, nil, tutorialQuest, event)
}

// scriptEventReply runs a script's on_event and shows its reply (Mobius
// notifyEvent: showResult, or showError to GMs). n may be nil.
func (s *Server) scriptEventReply(p *Player, n *Npc, script, event string) {
	var ref scripting.NpcRef
	if n != nil {
		ref = n.ScriptRef()
	}
	reply, ok := s.scripts.Event(s.background(), script, event, p, ref)
	switch {
	case ok:
		s.showScriptReply(p, n, reply)
	case reply.Err != nil:
		s.showScriptError(p, reply.Err)
	}
}

// handleTutorialLinkHtml handles RequestTutorialLinkHtml (0x7B), a link of
// a tutorial page: "close" closes it; the link goes to the tutorial.
func (s *Server) handleTutorialLinkHtml(p *Player, body []byte) {
	req, _ := clientpackets.ParseRequestTutorialLinkHtml(body)
	if strings.EqualFold(req.Link, "close") {
		p.Send(serverpackets.NewTutorialCloseHtmlPacket())
	}
	s.tutorialEvent(p, req.Link)
}

// handleTutorialPassCmd handles RequestTutorialPassCmdToServer (0x7C).
func (s *Server) handleTutorialPassCmd(p *Player, body []byte) {
	req, _ := clientpackets.ParseRequestTutorialPassCmdToServer(body)
	s.tutorialEvent(p, req.Command)
}

// handleTutorialQuestionMark handles RequestTutorialQuestionMark (0x7D), a
// click on a question mark: "QM<number>".
func (s *Server) handleTutorialQuestionMark(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestTutorialQuestionMark(body)
	if err != nil {
		return
	}
	s.tutorialEvent(p, "QM"+strconv.Itoa(int(req.Number)))
}

// handleTutorialClientEvent handles RequestTutorialClientEvent (0x7E):
// "CE<event>".
func (s *Server) handleTutorialClientEvent(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestTutorialClientEvent(body)
	if err != nil {
		return
	}
	s.tutorialEvent(p, "CE"+strconv.Itoa(int(req.EventID)))
}

// NewCharacterQuests returns the quest documents a new character starts
// with (Mobius CharacterCreate.startTutorialQuest): the tutorial, started,
// unless it is turned off. Safe on any goroutine.
func (s *Server) NewCharacterQuests() []storage.Quest {
	if s.quest.settings.DisableTutorial {
		return nil
	}
	return []storage.Quest{{Name: tutorialQuest, Vars: map[string]string{quest.VarState: quest.Started.Name()}}}
}

// ScriptConstants returns the constant tables of the scripts
// (scripting.Options.Constants): the generated tables of c (may be nil),
// the classes of the player class data, and the Config values the quests
// read (ALT_PARTY_RANGE, DISABLE_TUTORIAL).
func ScriptConstants(c *data.Constants, partyRange int, q QuestSettings) scripting.Constants {
	out := scripting.Constants{Tables: make(map[string]map[string]any)}
	if c != nil {
		for _, t := range c.Tables {
			m := make(map[string]any, t.Len())
			for k, v := range t.Ints {
				m[k] = v
			}
			for k, v := range t.Strings {
				m[k] = v
			}
			out.Tables[t.Name] = m
		}
	}
	cfg := map[string]any{"DISABLE_TUTORIAL": q.DisableTutorial}
	if partyRange > 0 {
		cfg["ALT_PARTY_RANGE"] = partyRange
	}
	out.Tables["Config"] = cfg
	abnormal := make(map[string]any, data.NumAbnormalTypes)
	for i := range data.NumAbnormalTypes {
		name := data.AbnormalType(i).String()
		abnormal[name] = name
	}
	out.Tables["AbnormalType"] = abnormal // player:buff_info_by_abnormal_type
	for _, pc := range data.PlayerClasses() {
		parent := -1
		if p, ok := pc.Parent(); ok {
			parent = int(p)
		}
		out.Classes = append(out.Classes, scripting.ClassInfo{
			ID: int(pc), Name: pc.String(), Level: pc.Level(), Race: int(pc.Race()), Parent: parent, Mage: pc.IsMage(),
		})
	}
	return out
}

// ----- global variables -----

// setGlobalVar sets a global variable and schedules its save.
func (s *Server) setGlobalVar(name, value string) {
	s.quest.globals.Set(name, value)
	if s.quest.store == nil || s.quest.globalsSave != 0 {
		return
	}
	s.quest.globalsSave = s.loop.After(s.quest.settings.SaveDelay, func() {
		s.quest.globalsSave = 0
		s.saveGlobals(false)
	})
}

// saveGlobals writes the changed global variables, in the background, or
// at once with wait (shutdown). A failed write marks them changed again.
func (s *Server) saveGlobals(wait bool) {
	f := &s.quest
	if f.store == nil || !f.globals.Dirty() {
		return
	}
	ch := f.globals.TakeChanges()
	write := func() {
		ctx, cancel := context.WithTimeout(s.background(), saveTimeout)
		defer cancel()
		if err := f.store.SaveGlobalVariables(ctx, ch); err != nil {
			s.log.Error("game: cannot save the global variables", "err", err)
			s.loop.Post(func() { f.globals.SaveFailed(ch) })
		}
	}
	if wait {
		write()
		return
	}
	go write()
}

package game

import (
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/quest"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The scripts' world (scripting.QuestWorld and scripting.QuestActions):
// ScriptOutput implements both, so the engine gets it before the server
// exists (scripting.Options.World), and New connects it. Until then, and
// for refs that are not the game's, the calls do nothing. Everything here
// runs on the loop, while a handler runs or a quest timer fires.

var (
	_ scripting.QuestWorld   = (*ScriptOutput)(nil)
	_ scripting.QuestActions = (*ScriptOutput)(nil)

	_ scripting.PlayerDetails = (*Player)(nil)
	_ scripting.PlayerVars    = (*Player)(nil)
	_ scripting.PlayerClasses = (*Player)(nil)
	_ scripting.PlayerItems   = (*Player)(nil)
	_ scripting.Located       = (*Player)(nil)
	_ scripting.Collider      = (*Player)(nil)
	_ scripting.NpcDetails    = npcRef{}
	_ scripting.Located       = npcRef{}
	_ scripting.Collider      = npcRef{}
	_ scripting.QuestState    = questStateRef{}
)

// questStateRef is a quest.State as scripts see it.
type questStateRef struct{ st *quest.State }

func (r questStateRef) Quest() string                  { return r.st.Name() }
func (r questStateRef) State() int                     { return int(r.st.State()) }
func (r questStateRef) SetState(state int)             { r.st.SetState(quest.StateID(state)) } //nolint:gosec // 0-2, checked by the engine
func (r questStateRef) Get(name string) (string, bool) { return r.st.Get(name) }
func (r questStateRef) Set(name, value string)         { r.st.Set(name, value) }
func (r questStateRef) Unset(name string)              { r.st.Unset(name) }
func (r questStateRef) Cond() int                      { return int(r.st.Step()) }
func (r questStateRef) Exit(repeatable bool)           { r.st.ExitQuest(repeatable, false) }

// stateRef returns st for the engine, nil for none.
func stateRef(st *quest.State) scripting.QuestState {
	if st == nil {
		return nil
	}
	return questStateRef{st}
}

// playerOfRef returns the game's player behind a ref, or nil.
func playerOfRef(p scripting.PlayerRef) *Player {
	pl, _ := p.(*Player)
	return pl
}

// npcOfRef returns the game's NPC behind a ref, or nil.
func npcOfRef(n scripting.NpcRef) *Npc {
	if r, ok := n.(npcRef); ok {
		return r.n
	}
	return nil
}

// player returns the connected server and the game's player of p; ok is
// false when either is missing.
func (o *ScriptOutput) player(p scripting.PlayerRef) (*Server, *Player, bool) {
	s := o.srv.Load()
	pl := playerOfRef(p)
	return s, pl, s != nil && pl != nil
}

// ----- quest states -----

// QuestState returns the player's state of the quest, or nil.
func (o *ScriptOutput) QuestState(p scripting.PlayerRef, name string) scripting.QuestState {
	s, pl, ok := o.player(p)
	if !ok {
		return nil
	}
	return stateRef(s.QuestState(pl, name))
}

// NewQuestState creates a CREATED state (Mobius newQuestState).
func (o *ScriptOutput) NewQuestState(p scripting.PlayerRef, name string) scripting.QuestState {
	s, pl, ok := o.player(p)
	if !ok {
		return nil
	}
	return stateRef(s.questsOf(pl).book.New(name))
}

// ----- clock and timers -----

// Now is the loop clock.
func (o *ScriptOutput) Now() time.Time {
	if s := o.srv.Load(); s != nil {
		return s.Now()
	}
	return time.Now()
}

// After runs fn on the loop after d.
func (o *ScriptOutput) After(d time.Duration, fn func()) (stop func()) {
	s := o.srv.Load()
	if s == nil {
		return func() {}
	}
	id := s.loop.After(d, fn)
	return func() { s.loop.Cancel(id) }
}

// Every runs fn on the loop every d.
func (o *ScriptOutput) Every(d time.Duration, fn func()) (stop func()) {
	s := o.srv.Load()
	if s == nil {
		return func() {}
	}
	id := s.loop.Every(max(d, time.Millisecond), fn)
	return func() { s.loop.Cancel(id) }
}

// ----- items -----

// ItemCount is the count of the item, equipped ones included.
func (o *ScriptOutput) ItemCount(p scripting.PlayerRef, itemID int) int64 {
	if _, pl, ok := o.player(p); ok && pl.inv.Inventory != nil {
		return pl.inv.Count(int32(itemID), -1, true) //nolint:gosec // item IDs fit in int32
	}
	return 0
}

// EnchantLevel is the enchant level of the first item with this ID.
func (o *ScriptOutput) EnchantLevel(p scripting.PlayerRef, itemID int) int {
	if _, pl, ok := o.player(p); ok && pl.inv.Inventory != nil {
		if it := pl.inv.ByItemID(int32(itemID)); it != nil { //nolint:gosec // item IDs fit in int32
			return int(it.Enchant())
		}
	}
	return 0
}

// EquippedItemID is the item ID in a paperdoll slot, 0 if empty.
func (o *ScriptOutput) EquippedItemID(p scripting.PlayerRef, slot int) int {
	_, pl, ok := o.player(p)
	if !ok || pl.inv.Inventory == nil || slot < 0 || slot >= storage.PaperdollTotalSlots {
		return 0
	}
	if it := pl.inv.Paperdoll(storage.PaperdollSlot(slot)); it != nil {
		return int(it.ItemID())
	}
	return 0
}

// GiveItems is Mobius giveItems.
func (o *ScriptOutput) GiveItems(p scripting.PlayerRef, itemID int, count int64, enchant int) {
	if s, pl, ok := o.player(p); ok {
		s.giveQuestItems(pl, itemID, count, enchant, false)
	}
}

// RewardItems is Mobius rewardItems.
func (o *ScriptOutput) RewardItems(p scripting.PlayerRef, itemID int, count int64) {
	if s, pl, ok := o.player(p); ok {
		s.giveQuestItems(pl, itemID, count, 0, true)
	}
}

// AddItem is Mobius player.addItem(QUEST, id, count, npc, true).
func (o *ScriptOutput) AddItem(p scripting.PlayerRef, itemID int, count int64, _ scripting.NpcRef) bool {
	s, pl, ok := o.player(p)
	if !ok || count <= 0 {
		return false
	}
	if itemID == adenaID {
		return s.AddAdena(pl, count, true) > 0
	}
	it, err := s.AddItem(pl, int32(itemID), count, -1, true, false) //nolint:gosec // item IDs fit in int32
	return err == nil && it != nil
}

// CanAddItem is Mobius validateCapacityByItemId.
func (o *ScriptOutput) CanAddItem(p scripting.PlayerRef, itemID int) bool {
	_, pl, ok := o.player(p)
	return ok && pl.inv.Inventory != nil && pl.inv.ValidateCapacityByItemID(int32(itemID)) //nolint:gosec // item IDs fit in int32
}

// TakeItems is Mobius takeItems.
func (o *ScriptOutput) TakeItems(p scripting.PlayerRef, itemID int, count int64) {
	if s, pl, ok := o.player(p); ok {
		s.takeQuestItems(pl, itemID, count)
	}
}

// AddExpAndSp is Mobius addExpAndSp with the quest rates.
func (o *ScriptOutput) AddExpAndSp(p scripting.PlayerRef, exp, sp int64) {
	if s, pl, ok := o.player(p); ok {
		s.addQuestExpAndSP(pl, exp, sp)
	}
}

// ----- packets and pages -----

// PlaySound sends PlaySound with a quest sound.
func (o *ScriptOutput) PlaySound(p scripting.PlayerRef, sound string) {
	if _, pl, ok := o.player(p); ok {
		pl.Send(serverpackets.NewQuestSoundPacket(sound))
	}
}

// ShowReply shows a reply of a quest timer or q:show_html_file.
func (o *ScriptOutput) ShowReply(p scripting.PlayerRef, npc scripting.NpcRef, r scripting.Reply) {
	s, pl, ok := o.player(p)
	if !ok || pl.state != stateInGame {
		return
	}
	if r.Err != nil {
		s.showScriptError(pl, r.Err)
		return
	}
	s.showScriptReply(pl, npcOfRef(npc), r)
}

// HTML returns a page of the HTML folder.
func (o *ScriptOutput) HTML(p scripting.PlayerRef, path string) (string, bool) {
	s := o.srv.Load()
	if s == nil {
		return "", false
	}
	return s.htmlFile(playerOfRef(p), path)
}

// FindObject returns the player or NPC with this object ID.
func (o *ScriptOutput) FindObject(objectID int32) any {
	s := o.srv.Load()
	if s == nil {
		return nil
	}
	obj, ok := s.world.Find(world.ObjectID(objectID))
	if !ok {
		return nil
	}
	switch x := obj.(type) {
	case *Player:
		return x
	case *Npc:
		return x.ScriptRef()
	}
	return nil
}

// GlobalVar returns a global variable.
func (o *ScriptOutput) GlobalVar(name string) (string, bool) {
	if s := o.srv.Load(); s != nil && s.quest.globals != nil {
		return s.quest.globals.Get(name)
	}
	return "", false
}

// SetGlobalVar sets a global variable; it is saved soon after.
func (o *ScriptOutput) SetGlobalVar(name, value string) {
	if s := o.srv.Load(); s != nil && s.quest.globals != nil {
		s.setGlobalVar(name, value)
	}
}

// ----- actions -----

// SocialAction broadcasts SocialAction for a player or an NPC.
func (o *ScriptOutput) SocialAction(obj any, action int) {
	s := o.srv.Load()
	if s == nil {
		return
	}
	var wo world.Object
	switch x := obj.(type) {
	case *Player:
		wo = x
	case scripting.NpcRef:
		if n := npcOfRef(x); n != nil {
			wo = n
		}
	}
	if wo == nil {
		return
	}
	if _, in := s.world.Find(wo.ID()); !in {
		return
	}
	s.world.Broadcast(wo, serverpackets.NewSocialActionPacket(serverpackets.SocialAction{
		ObjectID: int32(wo.ID()), //nolint:gosec // object IDs fit in int32
		ActionID: int32(action),  //nolint:gosec // small
	}), true)
}

// Teleport moves a player.
func (o *ScriptOutput) Teleport(p scripting.PlayerRef, loc scripting.Location) {
	if s, pl, ok := o.player(p); ok {
		pos := world.Position{X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z), Heading: int32(loc.Heading)} //nolint:gosec // coordinates fit in int32
		if err := s.Teleport(pl, pos); err != nil {
			s.log.Debug("game: script teleport failed", append(pl.logArgs(), "err", err)...)
		}
	}
}

// SendMessage sends a text message.
func (o *ScriptOutput) SendMessage(p scripting.PlayerRef, text string) {
	if _, pl, ok := o.player(p); ok {
		pl.SendMessage(text)
	}
}

// SendSystemMessage sends a system message with string parameters.
func (o *ScriptOutput) SendSystemMessage(p scripting.PlayerRef, id int, params []string) {
	if _, pl, ok := o.player(p); ok {
		ps := make([]serverpackets.SystemMessageParam, len(params))
		for i, v := range params {
			ps[i] = serverpackets.TextParam(v)
		}
		pl.SendSystemMessage(serverpackets.SystemMessageID(id), ps...) //nolint:gosec // message IDs fit
	}
}

// ShowOnScreenMsg sends ExShowScreenMessage.
func (o *ScriptOutput) ShowOnScreenMsg(p scripting.PlayerRef, text string, ms int) {
	if _, pl, ok := o.player(p); ok {
		pl.Send(serverpackets.NewExShowScreenMessagePacket(serverpackets.ScreenText(text, int32(ms)))) //nolint:gosec // milliseconds fit
	}
}

// Radar sends RadarControl (Mobius addRadar, removeRadar, clearRadar).
func (o *ScriptOutput) Radar(p scripting.PlayerRef, op int, x, y, z int) {
	_, pl, ok := o.player(p)
	if !ok {
		return
	}
	r := serverpackets.RadarControl{Action: int32(op), Type: serverpackets.RadarTypeMarker, X: int32(x), Y: int32(y), Z: int32(z)} //nolint:gosec // small values
	if int32(op) == serverpackets.RadarClear {                                                                                     //nolint:gosec // small
		r = serverpackets.RadarControl{Action: serverpackets.RadarClear, Type: serverpackets.RadarTypeClear}
	}
	pl.Send(serverpackets.NewRadarControlPacket(r))
}

// Tutorial sends a tutorial packet.
func (o *ScriptOutput) Tutorial(p scripting.PlayerRef, op scripting.TutorialOp, arg string) {
	s, pl, ok := o.player(p)
	if !ok {
		return
	}
	switch op {
	case scripting.TutorialQuestionMark:
		pl.Send(serverpackets.NewTutorialShowQuestionMarkPacket(atoi32(arg)))
	case scripting.TutorialVoice:
		pos := pl.Position()
		pl.Send(serverpackets.NewPlaySoundPacket(serverpackets.PlaySound{Type: 2, Sound: arg, X: pos.X, Y: pos.Y, Z: pos.Z}))
	case scripting.TutorialShowHTML:
		pl.Send(serverpackets.NewTutorialShowHtmlPacket(s.tutorialPage(pl, arg)))
	case scripting.TutorialCloseHTML:
		pl.Send(serverpackets.NewTutorialCloseHtmlPacket())
	case scripting.TutorialClientEvent:
		pl.Send(serverpackets.NewTutorialEnableClientEventPacket(atoi32(arg)))
	}
}

// NpcBroadcastSay is Mobius npc.broadcastSay(chatType, text).
func (o *ScriptOutput) NpcBroadcastSay(npc scripting.NpcRef, chatType int, text string) {
	s := o.srv.Load()
	n := npcOfRef(npc)
	if s == nil || n == nil || !n.InWorld() {
		return
	}
	s.world.Broadcast(n, serverpackets.NewNpcSayPacket(serverpackets.NpcSay{
		ObjectID: int32(n.ID()),                    //nolint:gosec // object IDs fit in int32
		Type:     serverpackets.ChatType(chatType), //nolint:gosec // small
		NpcID:    int32(n.Template.DisplayID),      //nolint:gosec // NPC IDs fit in int32
		Text:     text,
	}), false)
}

// ShowChatWindow shows the NPC's own dialog.
func (o *ScriptOutput) ShowChatWindow(npc scripting.NpcRef, p scripting.PlayerRef) {
	s, pl, ok := o.player(p)
	if n := npcOfRef(npc); ok && n != nil {
		s.showFirstPage(pl, n)
	}
}

// DeleteNpc removes an NPC from the world (Mobius deleteMe).
func (o *ScriptOutput) DeleteNpc(npc scripting.NpcRef) {
	s := o.srv.Load()
	if n := npcOfRef(npc); s != nil && n != nil && n.InWorld() {
		s.DespawnNpc(n)
	}
}

// AddSpawn is Mobius addSpawn: a random offset of 50 to 100 each way, a
// random heading for -1, and a despawn after despawn when it is positive.
func (o *ScriptOutput) AddSpawn(npcID int, loc scripting.Location, randomOffset bool, despawn time.Duration, _ bool) scripting.NpcRef {
	s := o.srv.Load()
	if s == nil {
		return nil
	}
	if randomOffset {
		off := func() int {
			d := 50 + s.rnd.Get(51)
			if s.rnd.Get(2) == 1 {
				d = -d
			}
			return d
		}
		loc.X += off()
		loc.Y += off()
	}
	if loc.Heading == -1 { // Mobius Spawn.initializeNpc: a random heading
		loc.Heading = s.rnd.Get(spawnRandomHeading)
	}
	pos := world.Position{X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z), Heading: int32(loc.Heading)} //nolint:gosec // coordinates fit in int32
	n, err := s.SpawnNpc(npcID, pos)
	if err != nil {
		s.log.Warn("game: a script could not spawn an NPC", "npc", npcID, "err", err)
		return nil
	}
	if despawn > 0 {
		s.loop.After(despawn, func() {
			if n.InWorld() {
				s.DespawnNpc(n)
			}
		})
	}
	return n.ScriptRef()
}

// tutorialPage returns a tutorial page (Mobius showTutorialHTML): next to
// the tutorial script, then data/html/scripts/quests/Q00255_Tutorial/, or
// Mobius's text for a missing page.
func (s *Server) tutorialPage(p *Player, page string) string {
	dir, rel := "", "quests/"+tutorialQuest
	if s.scripts != nil {
		if q, ok := s.scripts.Quest(tutorialQuest); ok {
			dir, rel = q.Dir, q.RelDir
		}
	}
	if content, ok := s.readScriptPage(p, dir, rel, page); ok {
		return content
	}
	return "<html><body>File data/scripts/quests/Q00255_Tutorial/" + page + " not found or file is empty.</body></html>"
}

// ----- the player and NPC refs -----

// Sex is 0 male, 1 female (scripting.PlayerDetails).
func (p *Player) Sex() int { return int(p.char.Sex) }

// IsMageClass is Mobius isMageClass.
func (p *Player) IsMageClass() bool { return data.PlayerClass(p.ClassID()).IsMage() } //nolint:gosec // class IDs are small

// IsNewbie is Mobius isNewbie: the account's first character (the stored
// newbie flag), or every character with AltNewCharAlwaysIsNewbie.
func (p *Player) IsNewbie() bool { return p.srv.quest.settings.NewCharAlwaysNewbie || p.char.Newbie }

// InCategory is Mobius isInCategory with a Category value.
func (p *Player) InCategory(category int) bool {
	c := p.srv.items.categories
	return c != nil && c.InCategory(data.CategoryType(category), p.ClassID()) //nolint:gosec // categories are small
}

// Location is the player's position (scripting.Located).
func (p *Player) Location() scripting.Location {
	pos := p.Position()
	return scripting.Location{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z), Heading: int(pos.Heading)}
}

// CollisionRadius is the player's collision radius (scripting.Collider).
func (p *Player) CollisionRadius() float64 { return float64(attackCollision(p)) }

// Var returns a player variable (Mobius PlayerVariables).
func (p *Player) Var(name string) (string, bool) {
	return p.srv.questsOf(p).vars.Get(name)
}

// SetVar sets a player variable, saved with the character.
func (p *Player) SetVar(name, value string) {
	p.srv.questsOf(p).vars.Set(name, value)
	p.srv.scheduleQuestSave(p)
}

// SetClassID changes the class (Mobius setPlayerClass, quest_class.go).
func (p *Player) SetClassID(id int) { p.srv.ChangeClass(p, id) }

// SetBaseClassID sets the base class (Mobius setBaseClass).
func (p *Player) SetBaseClassID(id int) {
	p.char.BaseClassID = int32(id) //nolint:gosec // class IDs are small
	p.srv.scheduleQuestSave(p)
}

// IsDead reports whether the NPC is dead (scripting.NpcDetails).
func (r npcRef) IsDead() bool { return r.n.Dead }

// ScriptValue is Mobius getScriptValue.
func (r npcRef) ScriptValue() int { return r.n.srv.quest.scriptValues[r.n] }

// SetScriptValue is Mobius setScriptValue.
func (r npcRef) SetScriptValue(v int) {
	if v == 0 {
		delete(r.n.srv.quest.scriptValues, r.n)
		return
	}
	r.n.srv.quest.scriptValues[r.n] = v
}

// IsType reports whether the NPC is of the Mobius NPC class.
func (r npcRef) IsType(class string) bool { return r.n.srv.npcIs(r.n, class) }

// Location is the NPC's position.
func (r npcRef) Location() scripting.Location {
	pos := r.n.Position()
	return scripting.Location{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z), Heading: int(pos.Heading)}
}

// CollisionRadius is the NPC's collision radius.
func (r npcRef) CollisionRadius() float64 { return float64(attackCollision(r.n)) }

// atoi32 parses a number argument of the tutorial packets; 0 if it is not
// one.
func atoi32(s string) int32 {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(n)
}

// statsReward is a plain quest reward of exp and SP.
func statsReward(exp, sp int64) stats.Reward {
	return stats.Reward{Exp: exp, SP: sp, ShownExp: exp, ShownSP: sp}
}

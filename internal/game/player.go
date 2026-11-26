package game

import (
	"fmt"
	"math"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// playerState is where a player is between character selection and leaving.
type playerState uint8

const (
	// stateSelected: CharSelected was sent; the client loads and sends
	// EnterWorld.
	stateSelected playerState = iota
	// stateInGame: EnterWorld ran. The player is online (World.AddPlayer)
	// and in the world, except while it teleports.
	stateInGame
	// stateLeft: the player left (restart, logout, disconnection, kick). It
	// is no longer registered anywhere; timers that still hold it do
	// nothing.
	stateLeft
)

// Mobius appearance defaults (PlayerAppearance).
const (
	defaultNameColor  = 0xFFFFFF
	defaultTitleColor = 0xECF9A2
	// maxEnchantEffect caps the weapon glow level (Player.getEnchantEffect).
	maxEnchantEffect = 127
)

// Player is the character of a connected client, from character selection
// until it leaves the world. Like all world state it belongs to the loop:
// read and change it only in packet handlers, timers and hooks.
//
// It embeds world.Base (ID, position, region) and implements world.Viewer
// and scripting.PlayerRef. The exported fields are plain state that the
// game features (movement, targeting, dialogs, zones) own and change; the
// core only reads them where noted.
type Player struct {
	world.Base

	// Target is the object the player has selected, or nil. Targeting sets
	// it; when the target leaves the player's sight the OnForget hooks are
	// told (Mobius clears the target then).
	Target world.Object
	// Move is the movement in progress, or nil while the player stands
	// (decision W7). The movement feature sets and clears it; the core reads
	// Destination to show a new viewer where the player walks, and clears
	// Move when the player teleports or leaves.
	Move *MoveState
	// Zones are the zones the player is inside. The zone feature keeps them
	// up to date; the core reads ZoneWater, ZonePeace, ZoneNoPvP, ZonePvP,
	// ZoneSiege and ZoneDangerArea for the info packets.
	Zones Zones
	// HTML is the dialog link cache: the actions the last pages offered.
	HTML HTMLCache
	// LastNpc is the NPC the player talked to last (Mobius lastFolkNpc), or
	// 0.
	LastNpc world.ObjectID
	// Sitting is set while the player sits. Mobius loads players standing.
	Sitting bool
	// CurHP, CurMP and CurCP are the current HP, MP and CP. They are loaded
	// from storage, kept within their maximum and saved.
	CurHP, CurMP, CurCP float64
	// PvPFlag is the PvP flag (Mobius pvpFlag), 0 until combat exists.
	PvPFlag int32
	// SpawnProtectedUntil ends the spawn protection (Mobius
	// spawnProtectEndTime); zero when the player has none.
	SpawnProtectedUntil time.Time

	srv     *Server
	session Session
	char    storage.Character // as loaded; Items live in inv and items
	class   *data.ClassTemplate
	stats   *stats.Player
	inv     Inventory
	items   playerItems // the warehouse and the item feature's state (items.go)
	state   playerState
	leaving bool // leave is running

	lastView  stats.PlayerView // the numbers of the last UserInfo sent
	relations map[world.ObjectID]relation
	autosave  world.TimerID
	timers    []world.TimerID // pending timers of the enter-world sequence
	teleport  *teleportState

	// The status (status_player.go).
	dead         bool
	regen        regenState
	statusQueued bool // a StatusUpdate is due at the end of the tick

	skill *skillPlayer // skills and effects (skill_creature.go)

	quests playerQuests // quest states and player variables (quest.go)

	itemSkill playerItemSkill // shortcuts, shots, item skills, penalties (item_skill.go)

	admin playerAdmin // the GM toggles: invul, diet, silence, ... (admin_gm.go)

	store playerStore // the private store (store.go)

	social playerSocial // the friend and block lists (friends.go)

	party playerParty // the party, the request partner and summon requests (party.go)

	clan playerClan // the clan membership and privileges (clan.go)

	vehicle playerVehicle // the boat the player is on (vehicle.go)
}

var (
	_ world.Viewer        = (*Player)(nil)
	_ scripting.PlayerRef = (*Player)(nil)
	_ InfoProvider        = (*Player)(nil)
)

// newPlayer builds the player of a loaded character for sess. It touches no
// world state, so it may run off the loop.
func (s *Server) newPlayer(sess Session, c storage.Character) (*Player, error) {
	class, ok := s.data.ClassTemplate(int(c.ClassID))
	if !ok {
		return nil, fmt.Errorf("game: character %d (%s) has the unknown class %d", c.ObjectID, c.Name, c.ClassID)
	}
	tmpl := s.classes[int(c.ClassID)]
	if tmpl == nil {
		return nil, fmt.Errorf("game: class %d has no stat template", c.ClassID)
	}
	st, err := stats.NewPlayer(s.rules, tmpl, int(c.Level))
	if err != nil {
		return nil, fmt.Errorf("game: character %d (%s): %w", c.ObjectID, c.Name, err)
	}
	st.SetFemale(c.Sex == 1)
	st.SetGM(c.AccessLevel >= storage.AccessLevelAdmin)
	// Mobius GameClient.load: "running is default", "standing is default".
	st.SetMotion(stats.Motion{Running: true})

	p := &Player{
		Base:      world.NewBase(world.ObjectID(c.ObjectID), world.Position{X: c.X, Y: c.Y, Z: c.Z, Heading: c.Heading}),
		srv:       s,
		session:   sess,
		class:     class,
		stats:     st,
		relations: make(map[world.ObjectID]relation),
	}
	p.char = c
	p.char.Items = nil
	p.skill = s.newSkillPlayer(p) // the skills come on the loop (loadSkills)
	s.loadItems(p, c.Items)
	s.applyDeathPenalty(p) // Mobius restoreDeathPenaltyBuffLevel
	v := st.View()
	p.lastView = v
	// PlayerStatus.setCurrentHp and friends keep the values within the
	// maximum.
	p.CurHP = clampVital(c.CurHP, v.MaxHP)
	p.CurMP = clampVital(c.CurMP, v.MaxMP)
	p.CurCP = clampVital(c.CurCP, v.MaxCP)
	return p, nil
}

func clampVital(v float64, maxValue int32) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return min(max(v, 0), float64(maxValue))
}

// logArgs returns the slog attributes that name the player.
func (p *Player) logArgs() []any {
	return []any{"player", p.char.Name, "objectId", p.char.ObjectID, "account", p.char.Account}
}

// Sink returns the client's session, or nil once the player has none
// (world.Viewer).
func (p *Player) Sink() world.Sink {
	if p.session == nil {
		return nil
	}
	return p.session
}

// SeesInvisible reports whether the player sees invisible objects: GMs do
// (world.Viewer).
func (p *Player) SeesInvisible() bool { return p.IsGM() }

// ObjectID returns the object ID (scripting.PlayerRef).
func (p *Player) ObjectID() int32 { return int32(p.ID()) }

// Name returns the character name (scripting.PlayerRef).
func (p *Player) Name() string { return p.char.Name }

// Level returns the level (scripting.PlayerRef).
func (p *Player) Level() int { return p.stats.Level() }

// ClassID returns the current class (scripting.PlayerRef).
func (p *Player) ClassID() int { return int(p.char.ClassID) }

// Race returns the race: 0 human, 1 elf, 2 dark elf, 3 orc, 4 dwarf
// (scripting.PlayerRef).
func (p *Player) Race() int { return int(p.char.Race) }

// BaseClassID returns the base class (the class a subclass was added to).
func (p *Player) BaseClassID() int {
	if p.char.BaseClassID == 0 && p.char.ClassID > 0 {
		return int(p.char.ClassID)
	}
	return int(p.char.BaseClassID)
}

// Female reports whether the character is female.
func (p *Player) Female() bool { return p.char.Sex == 1 }

// Title returns the title shown under the name (//settitle sets it).
func (p *Player) Title() string { return p.char.Title }

// Karma returns the karma.
func (p *Player) Karma() int32 { return p.char.Karma }

// Account returns the account the character belongs to.
func (p *Player) Account() string { return p.char.Account }

// Session returns the client's session, or nil after the player left.
func (p *Player) Session() Session { return p.session }

// AccessLevel returns the access level of the character.
func (p *Player) AccessLevel() storage.AccessLevel { return p.char.AccessLevel }

// IsGM reports whether the player is a GM: access level 1 or more.
func (p *Player) IsGM() bool { return p.char.AccessLevel >= storage.AccessLevelAdmin }

// Class returns the class template.
func (p *Player) Class() *data.ClassTemplate { return p.class }

// Stats returns the stat set. After changing it (equipment, level, motion,
// //gmspeed), call RefreshStats so that the client sees the new numbers.
func (p *Player) Stats() *stats.Player { return p.stats }

// Inventory returns the items.
func (p *Player) Inventory() *Inventory { return &p.inv }

// Running reports whether the player runs rather than walks.
func (p *Player) Running() bool { return p.stats.Motion().Running }

// SetRunning switches between running and walking: it changes the speeds of
// the stat set. It sends nothing; the caller broadcasts ChangeMoveType and
// calls RefreshStats, as Mobius does.
func (p *Player) SetRunning(running bool) {
	m := p.stats.Motion()
	m.Running = running
	p.stats.SetMotion(m)
}

// InGame reports whether the player entered the world and has not left. A
// teleporting player is in the game but not in the world (InWorld).
func (p *Player) InGame() bool { return p.state == stateInGame }

// Teleporting reports whether the player waits for Appearing after a
// teleport. It is out of the world meanwhile.
func (p *Player) Teleporting() bool { return p.teleport != nil }

// Server returns the game server the player is on.
func (p *Player) Server() *Server { return p.srv }

// Send queues an encoded packet body for the player's client. Like every
// body handed to a sink, it must not be modified afterwards.
func (p *Player) Send(body []byte) { world.Send(p, body) }

// SendMessage sends free text to the player as a system message (Mobius
// Player.sendMessage).
func (p *Player) SendMessage(text string) { p.Send(serverpackets.NewSystemMessageTextPacket(text)) }

// SendSystemMessage sends a system message of the client's table.
func (p *Player) SendSystemMessage(id serverpackets.SystemMessageID, params ...serverpackets.SystemMessageParam) {
	p.Send(serverpackets.NewSystemMessagePacket(serverpackets.SystemMessage{ID: id, Params: params}))
}

// SendActionFailed sends ActionFailed, which unlocks the client after a
// refused or finished action.
func (p *Player) SendActionFailed() { p.Send(actionFailedPacket) }

// UpdateUserInfo sends the player its UserInfo at the end of the tick
// (Mobius updateUserInfo, coalesced).
func (p *Player) UpdateUserInfo() { p.srv.world.MarkUpdate(p, world.UpdateUserInfo) }

// BroadcastUserInfo sends the player its UserInfo and the players around it
// its CharInfo at the end of the tick (Mobius broadcastUserInfo).
func (p *Player) BroadcastUserInfo() {
	p.srv.world.MarkUpdate(p, world.UpdateUserInfo|world.UpdateCharInfo)
}

// RefreshStats sends UserInfo and CharInfo at the end of the tick when the
// numbers they show changed since the last UserInfo. Call it after changing
// the stat set; it costs one stat evaluation when nothing changed. Before
// EnterWorld it does nothing.
func (p *Player) RefreshStats() {
	if p.state == stateInGame && !p.stats.View().Equal(p.lastView) {
		p.BroadcastUserInfo()
	}
}

// relation is what a viewer was last told about a player's relation to it
// (Mobius RelationCache).
type relation struct {
	bits           int32
	autoAttackable bool
}

// relationTo returns the relation bits of p as seen by v (Mobius
// Player.getRelation, without sieges, which do not exist yet): the clan
// leader and war bits come from clanRelationBits (clan.go).
func (p *Player) relationTo(v *Player) int32 {
	var r int32
	if p.PvPFlag != 0 {
		r |= serverpackets.RelationPvPFlag
	}
	if p.char.Karma > 0 {
		r |= serverpackets.RelationHasKarma
	}
	return r | p.srv.clanRelationBits(p, v)
}

// autoAttackableBy reports whether v attacks p with a plain click (Mobius
// Player.isAutoAttackable for a player attacker, without duels, sieges and
// events, which do not exist yet).
func (p *Player) autoAttackableBy(v *Player) bool {
	switch {
	case p.Invisible() && p.IsGM():
		return false
	case v == p:
		return false
	case p.party.party != nil && p.party.party == v.party.party:
		// Party members never attack each other with a click (Mobius
		// Player.java 8383).
		return false
	case p.Zones.Inside(ZonePeace) || p.Zones.Inside(ZoneNoPvP):
		return false
	case warAutoAttackable(p, v): // clanwar.go
		return true
	case p.Zones.Inside(ZonePvP) && v.Zones.Inside(ZonePvP) && (!p.Zones.Inside(ZoneSiege) || !v.Zones.Inside(ZoneSiege)):
		return true
	case sameClanOrAlly(p, v): // alliance.go
		return false
	}
	return p.char.Karma > 0 || p.PvPFlag > 0
}

// recommendations returns the recommendations the player can still give and
// has received. Those received are stored (//rec sets them); those to give
// are not yet: Mobius restartRecom gives a new day 3, 6 or 9 to give from
// level 10, 20 and 40.
func (p *Player) recommendations() (left, have int16) {
	have = int16(min(max(p.char.RecomHave, 0), 255)) //nolint:gosec // clamped as Mobius setRecomHave
	switch level := p.Level(); {
	case level < 10:
		return 0, have
	case level < 20:
		return 3, have
	case level < 40:
		return 6, have
	default:
		return 9, have
	}
}

// enchantEffect is the weapon glow level (Player.getEnchantEffect).
func (p *Player) enchantEffect() uint8 {
	w := p.inv.Paperdoll(storage.PaperdollRHand)
	if w == nil {
		return 0
	}
	return uint8(min(max(w.Enchant(), 0), maxEnchantEffect)) //nolint:gosec // clamped to 0-127
}

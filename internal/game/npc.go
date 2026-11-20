package game

import (
	"errors"
	"fmt"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/world"
)

// The NPC feature: NPCs in the world (this file), their stat templates
// (npc_stats.go), the spawn lists, respawns and day/night spawns (spawn.go),
// targeting and talking (target.go), social actions (social.go) and the GM
// commands spawn, delete, respawn and npcinfo (npc_gm.go).
func init() { registerFeature("npc", setupNpcs) }

// setupNpcs registers the NPC feature's handlers and hooks, then spawns the
// spawn lists of the world data. It sets the fields of s.npc one by one:
// features set up earlier may have registered interaction hooks there.
func setupNpcs(s *Server) error {
	f := &s.npc
	f.rnd = &npcRand{s.rnd}
	f.ensureMaps()
	if err := errors.Join(s.setupTargeting(), s.setupSocial(), s.setupNpcGM()); err != nil {
		return err
	}
	s.spawnWorld()
	return nil
}

// Npc is a non-player character in the world: a town NPC or a monster. It
// lives on the world loop.
//
// The NPC feature owns this file and extends Npc and npcFeature. Other
// features rely on: Template, Stats, the Server.Npc lookup, ScriptRef and
// the OnNpcInteract hook.
type Npc struct {
	world.Base

	// Template is the NPC's data template.
	Template *data.NpcTemplate
	// Stats computes the NPC's stats from its template.
	Stats *stats.Npc

	// CurHP and CurMP are the current HP and MP: full when the NPC spawns.
	// Change CurHP with SetCurrentHP, so that the HP bars of the players
	// targeting the NPC follow.
	CurHP, CurMP float64

	// Title is the NPC's own title (Mobius Creature.getTitle), which
	// NpcInfo shows when the template does not use a server-side title.
	// Scripts may set it; it is empty by default.
	Title string
	// RHand and LHand are the item IDs of the weapons the NPC shows, and
	// Enchant their glow; they start as the template's equipment.
	RHand, LHand, Enchant int32
	// Move is the NPC's walk in progress, or nil while it stands
	// (npc_move.go). A new viewer gets a MoveToLocation with NpcInfo while
	// it is set.
	Move *MoveState
	// Dead is set while the NPC is dead (from Milestone 6).
	Dead bool
	// Summoned shows the summon animation when the NPC appears (Mobius
	// isShowSummonAnimation).
	Summoned bool

	srv   *Server
	class *npcClass
	// spawn is the spawn slot the NPC belongs to, or nil for an NPC that
	// no spawn manages.
	spawn *npcSpawn

	// cache holds the stat numbers of the info packets, computed again when
	// the stat set's version changes.
	cache   npcStatCache
	cacheOK bool

	hpBar           npcHPBarCheck
	regen           regenState // status_npc.go
	ai              npcAI      // the monster AI, hate and movement (ai.go)
	randomAnimation bool       // Mobius isRandomAnimationEnabled
	lastSocial      time.Time  // when the NPC last played a random animation
	// priceConfig is a merchant's tax region, set when it spawns (Mobius
	// Merchant.onSpawn, merchant_price.go); nil for other NPCs.
	priceConfig *data.PriceConfig
}

var (
	_ InfoProvider  = (*Npc)(nil)
	_ UpdateFlusher = (*Npc)(nil)
)

// NpcInteraction runs when a player interacts with an NPC: the second
// click on a targeted NPC once the player is within interaction range
// (Mobius NpcClick). The dialog feature registers one to open the NPC's
// chat window.
type NpcInteraction func(p *Player, n *Npc)

// npcFeature is the NPC feature's state on the Server (Server.npc).
// Loop-only.
type npcFeature struct {
	interact []NpcInteraction

	// rnd draws spawn points, headings, respawn delays and animations from
	// the server's random source (Server.Rand).
	rnd *npcRand
	// classes holds what the NPC code knows about each template, worked out
	// once per NPC ID and shared by all its NPCs.
	classes map[int]*npcClass

	// spawns holds every live spawn slot; npcSpawn.index is its place.
	spawns []*npcSpawn
	// byTerritory holds the slots of each spawn territory, for //delete.
	byTerritory map[string][]*npcSpawn
	// dayOnly and nightOnly hold the slots that exist only by day or by
	// night (Mobius DayNightSpawnManager); night is the period in force.
	dayOnly, nightOnly []*npcSpawn
	night              bool
	// pending holds the slots waiting for their respawn timer.
	pending map[*npcSpawn]struct{}

	// watchers holds, per targeted object, the players targeting it: they
	// get its HP bar (Mobius status listeners).
	watchers map[world.ObjectID][]*Player
	// approaches holds the players walking up to an NPC to talk to it.
	approaches map[*Player]*npcApproach
}

// ensureMaps makes the maps of the feature.
func (f *npcFeature) ensureMaps() {
	if f.classes == nil {
		f.classes = make(map[int]*npcClass)
	}
	if f.byTerritory == nil {
		f.byTerritory = make(map[string][]*npcSpawn)
	}
	if f.pending == nil {
		f.pending = make(map[*npcSpawn]struct{})
	}
	if f.watchers == nil {
		f.watchers = make(map[world.ObjectID][]*Player)
	}
	if f.approaches == nil {
		f.approaches = make(map[*Player]*npcApproach)
	}
}

// OnNpcInteract registers fn to run when a player interacts with an NPC.
// Call it from a feature's setup function.
//
// The hooks run on the loop when a player clicks the NPC it already
// targets, the NPC is not attackable and the player is within
// Settings.NPCs.InteractionRange (3D) and not sitting; or, when the player
// was too far, once it has walked up to the NPC. Before them a walking
// player stopped (StopMove), the player got MoveToPawn (turning it to the
// NPC) and Player.LastNpc is the NPC; right after them the player gets
// ActionFailed.
func (s *Server) OnNpcInteract(fn NpcInteraction) {
	s.npc.interact = append(s.npc.interact, fn)
}

// runNpcInteract runs the interaction hooks for p talking to n.
func (s *Server) runNpcInteract(p *Player, n *Npc) {
	for _, fn := range s.npc.interact {
		fn(p, n)
	}
}

// Npc returns the NPC with the given object ID in the world, or nil.
// Loop-only.
func (s *Server) Npc(id world.ObjectID) *Npc {
	o, ok := s.World().Find(id)
	if !ok {
		return nil
	}
	n, _ := o.(*Npc)
	return n
}

// ScriptRef returns the NPC as the scripting engine sees it.
// scripting.NpcRef.ID is the template ID, which clashes with
// world.Object.ID, so scripts get this small wrapper instead of the Npc.
func (n *Npc) ScriptRef() scripting.NpcRef { return npcRef{n: n} }

type npcRef struct{ n *Npc }

func (r npcRef) ObjectID() int32 { return int32(r.n.ID()) }
func (r npcRef) ID() int         { return r.n.Template.ID }
func (r npcRef) Name() string    { return r.n.Template.Name }

// DisplayID is the ID the client knows the NPC by (npcDisplayID).
func (r npcRef) DisplayID() int { return r.n.Template.DisplayID }

// Name returns the template's name (Mobius Npc.getName).
func (n *Npc) Name() string { return n.Template.Name }

// Level returns the template's level (Mobius Npc.getLevel).
func (n *Npc) Level() int { return n.Template.Level }

// Type returns the Mobius class of the NPC (Folk, Monster, Teleporter, ...).
func (n *Npc) Type() string { return n.Template.Type }

// IsMonster reports whether the NPC is a Monster or one of its subclasses
// (Mobius isMonster).
func (n *Npc) IsMonster() bool { return n.class != nil && n.class.monster }

// MaxHP returns the maximum HP.
func (n *Npc) MaxHP() int32 { return n.statView().MaxHP }

// MaxMP returns the maximum MP.
func (n *Npc) MaxMP() int32 {
	n.statView()
	return n.cache.maxMP
}

// SetCurrentHP sets the current HP, kept within 0 and the maximum, starts
// or stops regeneration, and sends the new HP bar to the players targeting
// the NPC when it changed by a pixel of the bar (Mobius setCurrentHp and
// broadcastStatusUpdate). A dead NPC keeps its HP. Loop-only.
func (n *Npc) SetCurrentHP(hp float64) { n.setCurrentHP(hp, true) }

// BroadcastInfo sends the NpcInfo again to the players that see the NPC at
// the end of the tick (Mobius broadcastInfo), after a change they must see.
// Loop-only.
func (n *Npc) BroadcastInfo() {
	if n.srv != nil {
		n.srv.world.MarkUpdate(n, world.UpdateCharInfo)
	}
}

// FlushUpdate sends the coalesced refreshes of the NPC (UpdateFlusher):
// its info to every player that sees it.
func (n *Npc) FlushUpdate(u world.Update) {
	if u&world.UpdateCharInfo == 0 || !n.InWorld() || n.srv == nil {
		return
	}
	var shared [2][]byte // by the attackable flag
	for v := range n.srv.world.VisiblePlayers(n) {
		vp, ok := v.(*Player)
		if !ok || !n.srv.world.CanSee(v, n) {
			continue
		}
		i := 0
		if n.autoAttackableBy(vp) {
			i = 1
		}
		if shared[i] == nil {
			shared[i] = n.infoPacket(i == 1)
		}
		vp.Send(shared[i])
	}
}

// InfoFor returns what a player that starts to see n gets (Mobius
// Npc.sendInfo): NpcInfo, or ServerObjectInfo for an NPC whose run speed is
// 0, and MoveToLocation while it walks (InfoProvider).
func (n *Npc) InfoFor(viewer *Player) [][]byte {
	body := n.infoPacket(n.autoAttackableBy(viewer))
	if m := n.Move; m != nil {
		return [][]byte{body, moveToLocationPacket(n, m.Destination)}
	}
	return [][]byte{body}
}

// infoPacket builds the NPC's info packet with the given attackable flag,
// following Mobius AbstractNpcInfo.NpcInfo and ServerObjectInfo.
func (n *Npc) infoPacket(attackable bool) []byte {
	v := n.statView()
	pos := n.Position()
	if n.cache.runSpeed == 0 {
		return serverpackets.NewServerObjectInfoPacket(serverpackets.ServerObjectInfo{
			ObjectID:        int32(n.ID()),
			DisplayID:       int32(n.Template.DisplayID), //nolint:gosec // NPC IDs are small
			Name:            n.serverName(),
			Attackable:      attackable,
			X:               pos.X,
			Y:               pos.Y,
			Z:               pos.Z,
			Heading:         pos.Heading,
			CollisionRadius: v.CollisionRadius,
			CollisionHeight: v.CollisionHeight,
			CurHP:           n.CurHP,
			MaxHP:           v.MaxHP,
		})
	}
	motion := n.Stats.Motion()
	return serverpackets.NewNpcInfoPacket(serverpackets.NpcInfo{
		ObjectID:              int32(n.ID()),
		DisplayID:             int32(n.Template.DisplayID), //nolint:gosec // NPC IDs are small
		Attackable:            attackable,
		X:                     pos.X,
		Y:                     pos.Y,
		Z:                     pos.Z,
		Heading:               pos.Heading,
		MAtkSpd:               v.MAtkSpd,
		PAtkSpd:               v.PAtkSpd,
		RunSpd:                v.RunSpd,
		WalkSpd:               v.WalkSpd,
		SwimRunSpd:            v.SwimRunSpd,
		SwimWalkSpd:           v.SwimWalkSpd,
		FlyRunSpd:             v.FlyRunSpd,
		FlyWalkSpd:            v.FlyWalkSpd,
		MoveMultiplier:        v.MoveMultiplier,
		AttackSpeedMultiplier: v.AttackSpeedMultiplier,
		CollisionRadius:       float32(v.CollisionRadius),
		CollisionHeight:       float32(v.CollisionHeight),
		RHand:                 n.RHand,
		LHand:                 n.LHand,
		Running:               motion.Running,
		InCombat:              n.srv != nil && n.srv.InCombat(n),
		Dead:                  n.Dead,
		Summoned:              n.Summoned,
		Name:                  n.serverName(),
		Title:                 n.serverTitle(),
		Invisible:             n.Invisible(),
		InWater:               motion.InWater,
		Flying:                motion.Flying,
		EnchantEffect:         n.Enchant,
	})
}

// serverName is the name NpcInfo sends: the template's for templates with
// usingServerSideName, else "" (the client shows its own).
func (n *Npc) serverName() string {
	if n.Template.UsingServerSideName {
		return n.Template.Name
	}
	return ""
}

// serverTitle is the title NpcInfo sends: the template's for templates with
// usingServerSideTitle, else the NPC's own.
func (n *Npc) serverTitle() string {
	if n.Template.UsingServerSideTitle {
		return n.Template.Title
	}
	return n.Title
}

// statView returns the stat numbers of the info packets, computed again
// only when the stat set changed.
func (n *Npc) statView() stats.NpcView {
	if !n.cacheOK || n.cache.version != n.Stats.Version() {
		n.cache = npcStatCache{
			view:     n.Stats.View(),
			runSpeed: n.Stats.RunSpeed(),
			maxMP:    int32(n.Stats.MaxMP()), //nolint:gosec // MP fits
			version:  n.Stats.Version(),
		}
		n.cacheOK = true
	}
	return n.cache.view
}

// autoAttackableBy reports whether p attacks n with a plain click (Mobius
// isAutoAttackable with a player attacker): monsters are, a FriendlyMob is
// for players with karma, every other NPC is not (guards and quest guards
// only for monsters, towers and defenders only in sieges, which do not
// exist yet).
func (n *Npc) autoAttackableBy(p *Player) bool {
	switch {
	case n.class == nil:
		return false
	case n.class.friendly:
		return p != nil && p.Karma() > 0
	case n.class.monster:
		return true
	}
	return false
}

// npcHPBarPixels is the width of the client's HP bar (Mobius
// MAX_HP_BAR_PX): the HP bar is refreshed only when the HP crosses a pixel.
const npcHPBarPixels = 352.0

// npcHPBarCheck is Mobius's HP bar refresh check (Creature.needHpUpdate).
type npcHPBarCheck struct{ inc, dec, interval float64 }

// reset starts the check from a full bar (initCharStatusUpdateValues).
func (c *npcHPBarCheck) reset(maxHP float64) {
	c.inc = maxHP
	c.interval = maxHP / npcHPBarPixels
	c.dec = c.inc - c.interval
}

// need reports whether the bar must be refreshed for the HP cur.
func (c *npcHPBarCheck) need(cur, maxHP float64) bool {
	if cur <= 1 || maxHP < npcHPBarPixels {
		return true
	}
	if cur > c.dec && cur < c.inc {
		return false
	}
	if cur == maxHP {
		c.inc = cur + 1
		c.dec = cur - c.interval
	} else {
		multi := cur / c.interval
		whole := float64(int64(multi))
		if multi < whole {
			whole--
		}
		c.dec = c.interval * whole
		c.inc = c.dec + c.interval
	}
	return true
}

// newNpc makes an NPC of template t with a new runtime ID, out of the world
// (Mobius new Npc): stats from the shared template, the template's
// equipment, full HP and MP.
func (s *Server) newNpc(t *data.NpcTemplate) (*Npc, error) {
	c := s.npcClassOf(t)
	st, err := stats.NewNpc(s.rules, c.stats)
	if err != nil {
		return nil, fmt.Errorf("game: NPC %d: %w", t.ID, err)
	}
	if t.Status.Flying {
		st.SetMotion(stats.Motion{Flying: true})
	}
	id, err := s.world.IDs().Next()
	if err != nil {
		return nil, fmt.Errorf("game: NPC %d: %w", t.ID, err)
	}
	n := &Npc{
		Base:     world.NewBase(id, world.Position{}),
		Template: t,
		Stats:    st,
		RHand:    int32(t.Equipment.RHand),         //nolint:gosec // item IDs are small
		LHand:    int32(t.Equipment.LHand),         //nolint:gosec // item IDs are small
		Enchant:  int32(t.Equipment.WeaponEnchant), //nolint:gosec // small
		srv:      s,
		class:    c,
	}
	// Every new NPC of a template has the same stat set: the numbers of the
	// first one serve them all.
	if c.firstOK && c.first.version == st.Version() {
		n.cache, n.cacheOK = c.first, true
	} else {
		n.statView()
		c.first, c.firstOK = n.cache, true
	}
	n.CurHP, n.CurMP = float64(n.cache.view.MaxHP), float64(n.cache.maxMP)
	n.hpBar.reset(float64(n.cache.view.MaxHP))
	return n, nil
}

// releaseNpcID gives the runtime ID of an NPC that is gone for good back to
// the allocator.
func (s *Server) releaseNpcID(n *Npc) {
	if err := s.world.IDs().Release(n.ID()); err != nil {
		s.log.Warn("game: cannot release an NPC object ID", "npc", n.Template.ID, "objectId", n.ID(), "err", err)
	}
}

// placeNpc puts n into the world at pos, alive and with full HP and MP
// (Mobius initializeNpc and spawnMe). aiData is the AI data of the NPC's
// spawn block (Mobius NpcPersonalAIData), or nil.
func (s *Server) placeNpc(n *Npc, pos world.Position, aiData map[string]int) error {
	if n.InWorld() {
		return world.ErrInWorld
	}
	n.Dead = false
	n.Move = nil
	n.CurHP, n.CurMP = float64(n.MaxHP()), float64(n.MaxMP())
	n.randomAnimation = n.class.randomAnim
	if aiData != nil {
		if off, ok := aiData["disableRandomAnimation"]; ok {
			n.randomAnimation = off == 0 && n.class.randomAnim
		}
	}
	if err := n.SetPosition(pos); err != nil {
		return err
	}
	if err := s.AddObject(n); err != nil {
		return err
	}
	s.npcSpawned(n, aiData, &pos) // ai.go: Mobius Attackable.onSpawn
	s.merchantSpawned(n)          // merchant_price.go: Mobius Merchant.onSpawn
	return nil
}

// SpawnNpc puts a new NPC of template npcID into the world at pos, managed
// by no spawn: it never respawns, and DespawnNpc ends it (Mobius addSpawn
// without a respawn, for scripts and events). Loop-only.
func (s *Server) SpawnNpc(npcID int, pos world.Position) (*Npc, error) {
	t, err := s.spawnableTemplate(npcID)
	if err != nil {
		return nil, err
	}
	n, err := s.newNpc(t)
	if err != nil {
		return nil, err
	}
	if err := s.placeNpc(n, pos, nil); err != nil {
		s.releaseNpcID(n)
		return nil, fmt.Errorf("game: spawn NPC %d at %v: %w", npcID, pos, err)
	}
	return n, nil
}

// DespawnNpc takes n out of the world the way Mobius deleteMe does (also
// what a decaying corpse will do): the players around get DeleteObject and
// drop it as their target, and its spawn, if it respawns, schedules the
// respawn. An NPC without a spawn is gone for good. Loop-only.
func (s *Server) DespawnNpc(n *Npc) {
	wasIn := n.InWorld()
	if wasIn {
		s.removeNpc(n)
	}
	switch sp := n.spawn; {
	case sp != nil:
		s.scheduleRespawn(sp)
	case wasIn:
		s.releaseNpcID(n)
	}
}

// errNoNpcTemplate means an NPC ID has no template that can be spawned.
var errNoNpcTemplate = errors.New("game: no NPC template that can be spawned")

// spawnableTemplate returns the template of npcID if NPCs of it can be put
// into the world: Mobius instantiates Npc subclasses only, and never spawns
// a Trap from a spawn.
func (s *Server) spawnableTemplate(npcID int) (*data.NpcTemplate, error) {
	if s.worldData == nil {
		return nil, fmt.Errorf("%w: %d (no world data)", errNoNpcTemplate, npcID)
	}
	t, ok := s.worldData.Npc(npcID)
	if !ok {
		return nil, fmt.Errorf("%w: %d does not exist", errNoNpcTemplate, npcID)
	}
	if c := s.npcClassOf(t); !c.isNpc || c.trap {
		return nil, fmt.Errorf("%w: %d is a %s", errNoNpcTemplate, npcID, t.Type)
	}
	return t, nil
}

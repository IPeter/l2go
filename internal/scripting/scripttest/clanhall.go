package scripttest

import (
	"slices"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
)

// ClanHall is a fake clan hall, what clanhall.get(id) and
// clanhall.of_npc(npc) of the NPCs whose Hall it is give. Change its fields
// between calls; the scripts' changes land in them.
type ClanHall struct {
	ID       int
	Name     string
	Owner    int32
	Siegable bool
	// SiegeDate is the next siege; Status "registering", "waiting_battle"
	// or "running".
	SiegeDate time.Time
	Status    string
	Attackers []int32
	Vars      map[string]string
	// ZoneSpawns are the hall zone's spawns by kind.
	ZoneSpawns map[string][]scripting.Location
	// Doors are the hall's doors by ID: true open.
	Doors map[int]bool
	// ZoneActive is the siege zone's state.
	ZoneActive bool

	// What the scripts did: Steps lists the engine calls in order
	// ("free", "banish_foreigners", "close_registration", "start_siege",
	// "cancel_siege", "end_siege", "end_siege accomplished",
	// "schedule <step> <ms>", "cancel_siege_task", "update_next_siege",
	// "banish_non_participants", "check_teleport_zone <i>");
	// Spawned the NPCs hall:spawn placed (Despawned the object IDs taken
	// away), Teleported the clan or player teleports, SiegeInfo the players
	// shown the siege window.
	Steps      []string
	Spawned    []*Npc
	Despawned  []int32
	Teleported []Action
	SiegeInfo  []int32
}

var (
	_ scripting.ClanHallWorld = (*World)(nil)
	_ scripting.ClanHallRef   = hallRef{}
)

// ScriptClanHall implements scripting.ClanHallWorld from Halls.
func (w *World) ScriptClanHall(id int) scripting.ClanHallRef {
	if h := w.Halls[id]; h != nil {
		return hallRef{w: w, h: h}
	}
	return nil
}

// NpcClanHall implements scripting.ClanHallWorld: the NPC's Hall.
func (w *World) NpcClanHall(npc scripting.NpcRef) scripting.ClanHallRef {
	n := npcOf(npc)
	if n == nil || n.Hall == nil {
		return nil
	}
	return hallRef{w: w, h: n.Hall}
}

// ClanInfo implements scripting.ClanHallWorld from HallClans.
func (w *World) ClanInfo(clanID int32) (scripting.ClanInfo, bool) {
	c, ok := w.HallClans[clanID]
	return c, ok
}

type hallRef struct {
	w *World
	h *ClanHall
}

func (r hallRef) step(s string) { r.h.Steps = append(r.h.Steps, s) }

func (r hallRef) HallID() int      { return r.h.ID }
func (r hallRef) HallName() string { return r.h.Name }
func (r hallRef) OwnerID() int32   { return r.h.Owner }
func (r hallRef) IsSiegable() bool { return r.h.Siegable }

func (r hallRef) SiegeDate() int64 {
	if r.h.SiegeDate.IsZero() {
		return 0
	}
	return r.h.SiegeDate.Unix()
}

func (r hallRef) SiegeStatus() string {
	if r.h.Status == "" {
		return "registering"
	}
	return r.h.Status
}

func (r hallRef) SetSiegeStatus(s string) { r.h.Status = s }
func (r hallRef) Attackers() []int32      { return slices.Clone(r.h.Attackers) }
func (r hallRef) IsAttacker(id int32) bool {
	return slices.Contains(r.h.Attackers, id)
}

func (r hallRef) AddAttacker(id int32) {
	if !slices.Contains(r.h.Attackers, id) {
		r.h.Attackers = append(r.h.Attackers, id)
	}
}

func (r hallRef) RemoveAttacker(id int32) {
	r.h.Attackers = slices.DeleteFunc(r.h.Attackers, func(x int32) bool { return x == id })
}

func (r hallRef) ClearAttackers() { r.h.Attackers = nil }

func (r hallRef) Free() {
	r.h.Owner = 0
	r.step("free")
}

func (r hallRef) BanishForeigners()      { r.step("banish_foreigners") }
func (r hallRef) BanishNonParticipants() { r.step("banish_non_participants") }

func (r hallRef) OpenCloseDoor(doorID int, open bool) {
	if r.h.Doors == nil {
		r.h.Doors = make(map[int]bool)
	}
	r.h.Doors[doorID] = open
}

func (r hallRef) SetZoneActive(active bool) { r.h.ZoneActive = active }
func (r hallRef) UpdateNextSiege()          { r.step("update_next_siege") }

func (r hallRef) CloseRegistration() {
	r.h.Status = "waiting_battle"
	r.step("close_registration")
}

func (r hallRef) Schedule(step string, ms int64) {
	r.step("schedule " + step + " " + strconv.FormatInt(ms, 10))
}

func (r hallRef) CancelSiegeTask() { r.step("cancel_siege_task") }

func (r hallRef) StartSiegeDefault() {
	r.h.Status = "running"
	r.h.ZoneActive = true
	r.step("start_siege")
}

func (r hallRef) CancelSiege() {
	r.h.Attackers = nil
	r.h.Status = "registering"
	r.step("cancel_siege")
}

func (r hallRef) EndSiege(accomplished bool) {
	s := "end_siege"
	if accomplished {
		s += " accomplished"
	}
	r.h.Status = "registering"
	r.h.ZoneActive = false
	r.step(s)
}

func (r hallRef) Var(name string) (string, bool) {
	v, ok := r.h.Vars[name]
	return v, ok
}

func (r hallRef) SetVar(name, value string) {
	if value == "" {
		delete(r.h.Vars, name)
		return
	}
	if r.h.Vars == nil {
		r.h.Vars = make(map[string]string)
	}
	r.h.Vars[name] = value
}

func (r hallRef) VarNames() []string {
	var out []string
	for k := range r.h.Vars {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func (r hallRef) Spawn(npcID int, loc scripting.Location, _ int64) scripting.NpcRef {
	ref := r.w.AddSpawn(npcID, loc, false, 0, false)
	if n := npcOf(ref); n != nil {
		r.h.Spawned = append(r.h.Spawned, n)
	}
	return ref
}

func (r hallRef) Despawn(objectID int32) { r.h.Despawned = append(r.h.Despawned, objectID) }

func (r hallRef) ZoneSpawn(kind string, index int) (scripting.Location, bool) {
	pts := r.h.ZoneSpawns[kind]
	if index < 0 || index >= len(pts) {
		return scripting.Location{}, false
	}
	return pts[index], true
}

func (r hallRef) TeleportClan(clanID int32, loc scripting.Location) {
	r.h.Teleported = append(r.h.Teleported, Action{Kind: "clan", Object: clanID, Value: loc.X})
}

func (r hallRef) TeleportToTown(objectID int32) {
	r.h.Teleported = append(r.h.Teleported, Action{Kind: "town", Object: objectID})
}

func (r hallRef) CheckTeleportZone(index int) {
	r.step("check_teleport_zone " + strconv.Itoa(index))
}

func (r hallRef) ShowSiegeInfo(p scripting.PlayerRef) {
	r.h.SiegeInfo = append(r.h.SiegeInfo, p.ObjectID())
}

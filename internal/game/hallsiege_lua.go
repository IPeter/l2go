package game

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/world"
)

// The clan halls as the hall siege scripts see them (scripting.ClanHallWorld
// and ClanHallRef), and the engine's calls into the scripts: each
// contestable hall has a script named after its Mobius class, asked through
// on_event at the siege's steps.

var (
	_ scripting.ClanHallWorld = (*ScriptOutput)(nil)
	_ scripting.ClanHallRef   = hallScriptRef{}
)

// hallSiegeScripts are the scripts of the contestable halls (Mobius
// dp/scripts/conquerablehalls/).
var hallSiegeScripts = map[int32]string{
	21: "FortressOfResistance",
	34: "DevastatedCastle",
	35: "BanditStronghold",
	62: "RainbowSpringsChateau",
	63: "WildBeastReserve",
	64: "FortressOfTheDead",
}

// hallScriptCall runs the hall's script's on_event with "siege_<event>" and
// returns its text answer, "" for none.
func (s *Server) hallScriptCall(h *ClanHall, event string) string {
	name := hallSiegeScripts[h.ID]
	if s.scripts == nil || name == "" || !s.scripts.HasScript(name) {
		return ""
	}
	reply, ok := s.scripts.Event(context.Background(), name, "siege_"+event, nil, nil)
	if !ok {
		return ""
	}
	return reply.Content
}

// hallScriptReplaces asks the script whether it does a step itself.
func (s *Server) hallScriptReplaces(h *ClanHall, event string) bool {
	return s.hallScriptCall(h, event) == "replace"
}

// hallScriptEvent tells the script about a step.
func (s *Server) hallScriptEvent(h *ClanHall, event string) { s.hallScriptCall(h, event) }

// hallScriptWinner asks the script for the winning clan (getWinner), 0 for
// none.
func (s *Server) hallScriptWinner(h *ClanHall) int32 {
	id, err := strconv.ParseInt(s.hallScriptCall(h, hallEventWinner), 10, 32)
	if err != nil {
		return 0
	}
	return int32(id)
}

// ScriptClanHall implements scripting.ClanHallWorld.
func (o *ScriptOutput) ScriptClanHall(id int) scripting.ClanHallRef {
	s := o.srv.Load()
	if s == nil {
		return nil
	}
	h := s.ClanHallByID(int32(id)) //nolint:gosec // hall IDs are small
	if h == nil {
		return nil
	}
	return hallScriptRef{s: s, h: h}
}

// NpcClanHall implements scripting.ClanHallWorld.
func (o *ScriptOutput) NpcClanHall(npc scripting.NpcRef) scripting.ClanHallRef {
	s, n := o.srv.Load(), npcOfRef(npc)
	if s == nil || n == nil {
		return nil
	}
	h := s.npcClanHall(n)
	if h == nil {
		return nil
	}
	return hallScriptRef{s: s, h: h}
}

// ClanInfo implements scripting.ClanHallWorld.
func (o *ScriptOutput) ClanInfo(clanID int32) (scripting.ClanInfo, bool) {
	s := o.srv.Load()
	if s == nil {
		return scripting.ClanInfo{}, false
	}
	cl := s.Clan(clanID)
	if cl == nil {
		return scripting.ClanInfo{}, false
	}
	return scripting.ClanInfo{ID: cl.ID, Name: cl.Name, Level: cl.Level, Members: cl.MemberCount(), LeaderID: cl.LeaderID,
		LeaderName: cl.LeaderName(), CastleID: cl.CastleID, HallID: cl.HideoutID, Dissolving: cl.DissolvingExpiry > 0}, true
}

// hallScriptRef is a hall for the scripts.
type hallScriptRef struct {
	s *Server
	h *ClanHall
}

func (r hallScriptRef) HallID() int      { return int(r.h.ID) }
func (r hallScriptRef) HallName() string { return r.h.Name }
func (r hallScriptRef) OwnerID() int32   { return r.h.ownerID }
func (r hallScriptRef) IsSiegable() bool { return r.h.siege != nil }

func (r hallScriptRef) SiegeDate() int64 {
	if r.h.siege == nil || r.h.siege.nextSiege.IsZero() {
		return 0
	}
	return r.h.siege.nextSiege.Unix()
}

func (r hallScriptRef) SiegeStatus() string {
	if r.h.siege == nil {
		return residence.HallRegistering.String()
	}
	return r.h.siege.status.String()
}

func (r hallScriptRef) SetSiegeStatus(status string) {
	if hs := r.h.siege; hs != nil {
		for _, st := range []residence.HallSiegeStatus{residence.HallRegistering, residence.HallWaitingBattle, residence.HallRunning} {
			if st.String() == status {
				hs.status = st
			}
		}
	}
}

func (r hallScriptRef) Attackers() []int32 {
	if r.h.siege == nil {
		return nil
	}
	var out []int32
	for _, sc := range r.h.siege.reg.Attackers() {
		out = append(out, sc.ClanID)
	}
	return out
}

func (r hallScriptRef) IsAttacker(id int32) bool {
	return r.h.siege != nil && r.h.siege.reg.IsAttacker(id)
}

func (r hallScriptRef) AddAttacker(id int32) {
	if hs := r.h.siege; hs != nil && id != 0 && !hs.reg.IsAttacker(id) {
		hs.reg.AddAttacker(id)
		r.h.save()
	}
}

func (r hallScriptRef) RemoveAttacker(id int32) {
	if hs := r.h.siege; hs != nil && hs.reg.IsAttacker(id) {
		hs.reg.Remove(id)
		r.h.save()
	}
}

func (r hallScriptRef) ClearAttackers() {
	if hs := r.h.siege; hs != nil {
		hs.reg.Clear()
		r.h.save()
	}
}

// Free is SiegableHall.free with the owner's setHideoutId(0), saved.
func (r hallScriptRef) Free() {
	if r.h.ownerID != 0 {
		r.s.freeHall(r.h)
	}
}

func (r hallScriptRef) BanishForeigners()      { r.s.banishForeigners(r.h) }
func (r hallScriptRef) BanishNonParticipants() { r.s.banishNonParticipants(r.h) }

func (r hallScriptRef) OpenCloseDoor(doorID int, open bool) { r.s.openCloseHallDoor(r.h, doorID, open) }

func (r hallScriptRef) SetZoneActive(active bool) {
	if r.h.siege != nil {
		r.s.setHallZoneActive(r.h, active)
	}
}

func (r hallScriptRef) UpdateNextSiege() {
	if r.h.siege != nil {
		r.s.updateHallNextSiege(r.h)
		r.h.save()
	}
}

func (r hallScriptRef) CloseRegistration() {
	if r.h.siege != nil {
		r.s.closeHallRegistration(r.h)
	}
}

// Schedule arms the siege task with a step.
func (r hallScriptRef) Schedule(step string, ms int64) {
	h, s := r.h, r.s
	if h.siege == nil {
		return
	}
	d := time.Duration(ms) * time.Millisecond
	switch step {
	case "prepare_owner":
		s.scheduleHallTask(h, d, func() { s.hallPrepareOwner(h) })
	case "start_siege":
		s.scheduleHallTask(h, d, func() { s.hallStartSiegeDefault(h) })
	case "end_siege":
		s.scheduleHallTask(h, d, func() { s.hallEndSiege(h) })
	}
}

func (r hallScriptRef) CancelSiegeTask() {
	if r.h.siege != nil {
		r.s.cancelHallTask(r.h)
	}
}

func (r hallScriptRef) StartSiegeDefault() {
	if r.h.siege != nil {
		r.s.hallStartSiegeDefault(r.h)
	}
}

func (r hallScriptRef) CancelSiege() {
	if r.h.siege != nil {
		r.s.cancelHallSiege(r.h)
	}
}

// EndSiege ends the siege; accomplished is Mobius _missionAccomplished.
func (r hallScriptRef) EndSiege(accomplished bool) {
	if r.h.siege != nil {
		r.h.siege.missionAccomplished = accomplished
		r.s.hallEndSiege(r.h)
	}
}

func (r hallScriptRef) Var(name string) (string, bool) {
	if r.h.siege == nil {
		return "", false
	}
	v, ok := r.h.siege.vars[name]
	return v, ok
}

// SetVar keeps a value with the hall, saved at once.
func (r hallScriptRef) SetVar(name, value string) {
	hs := r.h.siege
	if hs == nil {
		return
	}
	if value == "" {
		delete(hs.vars, name)
	} else {
		if hs.vars == nil {
			hs.vars = make(map[string]string)
		}
		hs.vars[name] = value
	}
	r.h.save()
}

func (r hallScriptRef) VarNames() []string {
	if r.h.siege == nil {
		return nil
	}
	var out []string
	for k := range r.h.siege.vars {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func (r hallScriptRef) Spawn(npcID int, loc scripting.Location, respawnMs int64) scripting.NpcRef {
	if r.h.siege == nil {
		return nil
	}
	pos := world.Position{X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z), Heading: int32(loc.Heading)} //nolint:gosec // world coordinates
	n := r.s.hallSpawnNpc(r.h, npcID, pos, time.Duration(respawnMs)*time.Millisecond)
	if n == nil {
		return nil
	}
	return npcRef{n: n}
}

func (r hallScriptRef) Despawn(objectID int32) {
	if r.h.siege != nil {
		r.s.hallDespawnNpc(r.h, world.ObjectID(objectID))
	}
}

func (r hallScriptRef) ZoneSpawn(kind string, index int) (scripting.Location, bool) {
	if r.h.zone == nil {
		return scripting.Location{}, false
	}
	pts := zoneSpawns(r.h.zone, kind)
	if index < 0 || index >= len(pts) {
		return scripting.Location{}, false
	}
	return scripting.Location{X: pts[index].X, Y: pts[index].Y, Z: pts[index].Z}, true
}

// TeleportClan sends the clan's members online to loc (Bandit Stronghold's
// owner at the start).
func (r hallScriptRef) TeleportClan(clanID int32, loc scripting.Location) {
	cl := r.s.Clan(clanID)
	if cl == nil {
		return
	}
	for _, p := range cl.OnlineMembers() {
		r.s.teleportAround(p, data.Location{X: loc.X, Y: loc.Y, Z: loc.Z}, 0)
	}
}

// TeleportToTown sends a player in the game to town.
func (r hallScriptRef) TeleportToTown(objectID int32) {
	if p, ok := r.s.Player(world.ObjectID(objectID)); ok && p.InWorld() {
		r.s.teleportToTown(p)
	}
}

func (r hallScriptRef) CheckTeleportZone(index int) {
	if r.h.siege != nil {
		r.s.checkHallTeleportZone(r.h, index)
	}
}

func (r hallScriptRef) ShowSiegeInfo(p scripting.PlayerRef) {
	if pl := playerOfRef(p); pl != nil && r.h.siege != nil {
		r.s.showHallSiegeInfo(pl, r.h)
	}
}

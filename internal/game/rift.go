package game

import (
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// The Dimensional Rift (Plan 14 M11l, Mobius DimensionalRiftManager,
// DimensionalRift and DimensionalRiftRoom, the Rift bypass handler): a
// party of at least RiftMinPartySize in the waiting room pays dimension
// fragments per member and enters a free room of the chosen area; the room's
// monsters appear RiftSpawnDelay later; every 480-600 s (1.5 times longer
// in the boss room) the party jumps to another free room, at most
// MaxRiftJumps rooms, then goes back to the waiting room. The leader may
// jump or leave early at the room's guide.

// riftFragmentID is DIMENSIONAL_FRAGMENT_ITEM_ID.
const riftFragmentID = 7079

// riftCommands are the Rift bypass handler's commands.
var riftCommands = map[string]bool{"enterrift": true, "changeriftroom": true, "exitrift": true}

// riftState is the rift on the Server. Loop-only.
type riftState struct {
	rooms   map[int]map[int]*riftRoom
	parties map[*Party]*rift
}

// riftRoom is a room (Mobius DimensionalRiftRoom) with its spawn points,
// placed at random inside the room when the server starts.
type riftRoom struct {
	data   *data.RiftRoom
	points []riftPoint
	inside bool
	active bool
	npcs   map[*Npc]*riftPoint
}

type riftPoint struct {
	npcID int
	pos   world.Position
	delay time.Duration
}

// rift is a party in the rift (Mobius DimensionalRift).
type rift struct {
	typ       int
	party     *Party
	room      int
	jump      int
	boss      bool
	completed []int
	dead      map[*Player]bool
	revived   map[*Player]bool

	teleTimer, spawnTimer, quakeTimer world.TimerID
}

// setupRift places the rooms' spawns (DimensionalRiftManager.loadSpawns,
// whose points Rnd chooses once) and registers the hooks.
func (s *Server) setupRift() error {
	f := &s.sevenSigns.rift
	f.rooms = make(map[int]map[int]*riftRoom)
	f.parties = make(map[*Party]*rift)
	s.OnPlayerEnter(s.riftEnter)
	s.OnDie(s.riftMemberDied)
	s.OnPlayerLeave(s.riftLeave)
	d := s.sevenSigns.data
	if d == nil {
		return nil
	}
	for i := range d.Rift {
		rd := &d.Rift[i]
		m := f.rooms[rd.Type]
		if m == nil {
			m = make(map[int]*riftRoom)
			f.rooms[rd.Type] = m
		}
		room := &riftRoom{data: rd, npcs: make(map[*Npc]*riftPoint)}
		for _, sp := range rd.Spawns {
			for range sp.Count {
				x := rd.XMin + data.RiftSpawnMargin + s.rnd.Get(rd.XMax-rd.XMin-2*data.RiftSpawnMargin+1)
				y := rd.YMin + data.RiftSpawnMargin + s.rnd.Get(rd.YMax-rd.YMin-2*data.RiftSpawnMargin+1)
				room.points = append(room.points, riftPoint{npcID: sp.NpcID, delay: time.Duration(sp.Delay) * time.Second,
					pos: world.Position{X: int32(x), Y: int32(y), Z: int32(rd.TeleZ)}}) //nolint:gosec // world coordinates
			}
		}
		m[rd.Room] = room
	}
	return nil
}

func (s *Server) riftRoom(typ, room int) *riftRoom { return s.sevenSigns.rift.rooms[typ][room] }

// riftPeace is checkIfInPeaceZone: the waiting room.
func (s *Server) riftPeace(pos world.Position) bool {
	r := s.riftRoom(data.RiftTypeWaiting, 0)
	return r != nil && r.data.Contains(int(pos.X), int(pos.Y), int(pos.Z))
}

// inRiftZone is checkIfInRiftZone(x, y, z, false): the rift but not its
// waiting room.
func (s *Server) inRiftZone(pos world.Position) bool {
	r := s.riftRoom(data.RiftTypeWaiting, 1)
	return r != nil && r.data.Contains(int(pos.X), int(pos.Y), int(pos.Z)) && !s.riftPeace(pos)
}

func (s *Server) riftTeleport(p *Player, typ, room int) {
	r := s.riftRoom(typ, room)
	if r == nil {
		return
	}
	pos := world.Position{X: int32(r.data.TeleX), Y: int32(r.data.TeleY), Z: int32(r.data.TeleZ)} //nolint:gosec // world coordinates
	if err := s.Teleport(p, pos); err != nil {
		s.log.Warn("game: rift teleport failed", append(p.logArgs(), "err", err)...)
	}
}

// riftEnter is EnterWorld's rule: a player who logged out inside the rift
// goes back to the waiting room.
func (s *Server) riftEnter(p *Player) {
	if s.inRiftZone(p.Position()) {
		s.riftTeleport(p, data.RiftTypeWaiting, 0)
	}
}

// riftPage shows a rift page with the NPC's name (showHtmlFile).
func (s *Server) riftPage(p *Player, n *Npc, file string, count int) {
	pg := s.pageFile(p, n.ID(), sevenSignsHTML+"rift/"+file).replaceObjectID(n.ID()).replace("%npc_name%", n.Template.Name)
	if count >= 0 {
		pg = pg.replace("%count%", strconv.Itoa(count))
	}
	s.sendPage(p, pg)
}

// riftCost is getNeededItems.
func (s *Server) riftCost(typ int) int {
	c := s.sevenSigns.cfg.Rift
	switch typ {
	case 1:
		return c.RecruitCost
	case 2:
		return c.SoldierCost
	case 3:
		return c.OfficerCost
	case 4:
		return c.CaptainCost
	case 5:
		return c.CommanderCost
	case 6:
		return c.HeroCost
	}
	return -1
}

func (s *Server) riftMaxJumps() int {
	if m := s.sevenSigns.cfg.Rift.MaxJumps; m >= 1 && m <= 8 {
		return m
	}
	return 4
}

// bypassRift is the Rift bypass handler.
func (s *Server) bypassRift(p *Player, n *Npc, word, command string) {
	f := &s.sevenSigns.rift
	pt := p.Party()
	var r *rift
	if pt != nil {
		r = f.parties[pt]
	}
	switch word {
	case "enterrift":
		if len(command) <= 10 {
			return
		}
		typ, err := strconv.Atoi(strings.TrimSpace(command[10:]))
		if err != nil || s.riftCost(typ) < 0 {
			return
		}
		s.riftStart(p, typ, n)
	case "changeriftroom":
		if r == nil {
			s.riftPage(p, n, "Cheater.htm", -1)
			return
		}
		s.riftManualJump(r, p, n)
	case "exitrift":
		if r == nil {
			s.riftPage(p, n, "Cheater.htm", -1)
			return
		}
		if pt.Leader() != p {
			s.riftPage(p, n, "NotPartyLeader.htm", -1)
			return
		}
		for _, m := range pt.Members() {
			s.riftTeleport(m, data.RiftTypeWaiting, 0)
		}
		s.killRift(r)
	}
}

// freeRiftRooms is getFreeRooms: the rooms of the area without a party, in
// room order.
func (s *Server) freeRiftRooms(typ int) []int {
	var out []int
	for _, id := range sortedKeys(s.sevenSigns.rift.rooms[typ]) {
		if !s.sevenSigns.rift.rooms[typ][id].inside {
			out = append(out, id)
		}
	}
	return out
}

// riftStart is DimensionalRiftManager.start.
func (s *Server) riftStart(p *Player, typ int, n *Npc) {
	f := &s.sevenSigns.rift
	pt := p.Party()
	switch {
	case pt == nil:
		s.riftPage(p, n, "NoParty.htm", -1)
		return
	case pt.Leader() != p:
		s.riftPage(p, n, "NotPartyLeader.htm", -1)
		return
	case f.parties[pt] != nil:
		s.riftPage(p, n, "Cheater.htm", -1)
		return
	case pt.Len() < s.sevenSigns.cfg.Rift.MinPartySize:
		s.riftPage(p, n, "SmallParty.htm", s.sevenSigns.cfg.Rift.MinPartySize)
		return
	}
	rooms := f.rooms[typ]
	busy := 0
	for _, r := range rooms {
		if r.inside {
			busy++
		}
	}
	if busy >= len(rooms)-1 {
		p.SendMessage("Rift is full. Try later.")
		return
	}
	for _, m := range pt.Members() {
		if !s.riftPeace(m.Position()) {
			s.riftPage(p, n, "NotInWaitingRoom.htm", -1)
			return
		}
	}
	cost := s.riftCost(typ)
	for _, m := range pt.Members() {
		if m.inv.Inventory == nil || m.inv.Count(riftFragmentID, -1, true) < int64(cost) {
			s.riftPage(p, n, "NoFragments.htm", cost)
			return
		}
	}
	for _, m := range pt.Members() {
		it := m.inv.ByItemID(riftFragmentID)
		if it == nil || !s.DestroyItem(m, it, int64(cost), false) {
			s.riftPage(p, n, "NoFragments.htm", cost)
			return
		}
	}
	free := s.freeRiftRooms(typ)
	room := free[s.rnd.Get(len(free))] // Rnd.get(1, size) - 1
	r := &rift{typ: typ, party: pt, room: room, jump: 1, dead: map[*Player]bool{}, revived: map[*Player]bool{}}
	f.parties[pt] = r
	rooms[room].inside = true
	for _, m := range pt.Members() {
		s.riftTeleport(m, typ, room)
	}
	s.riftSpawnTimer(r)
	s.riftTeleTimer(r)
}

// riftSpawnTimer is createSpawnTimer: the room's monsters come after the
// spawn delay.
func (s *Server) riftSpawnTimer(r *rift) {
	s.cancelTimer(&r.spawnTimer)
	room := s.riftRoom(r.typ, r.room)
	r.spawnTimer = s.loop.After(time.Duration(s.sevenSigns.cfg.Rift.SpawnDelay)*time.Millisecond, func() {
		r.spawnTimer = 0
		s.riftRoomSpawn(room)
	})
}

// riftRoomSpawn is DimensionalRiftRoom.spawn: every point spawns and
// respawns after its delay while the room is active.
func (s *Server) riftRoomSpawn(room *riftRoom) {
	if room == nil {
		return
	}
	room.active = true
	for i := range room.points {
		s.riftSpawnPoint(room, &room.points[i])
	}
}

func (s *Server) riftSpawnPoint(room *riftRoom, pt *riftPoint) {
	if !room.active {
		return
	}
	if n := s.spawnSiegeNpc(pt.npcID, pt.pos, int32(s.rnd.Get(65536)), ""); n != nil { //nolint:gosec // headings
		room.npcs[n] = pt
	}
}

// riftRoomUnspawn is DimensionalRiftRoom.unspawn.
func (s *Server) riftRoomUnspawn(room *riftRoom) {
	if room == nil {
		return
	}
	room.active = false
	for n := range room.npcs {
		if n.InWorld() {
			s.DespawnNpc(n)
		}
	}
	clear(room.npcs)
}

// riftJumpTime is calcTimeToNextJump.
func (s *Server) riftJumpTime(r *rift) time.Duration {
	c := s.sevenSigns.cfg.Rift
	sec := c.AutoJumpsDelayMin + s.rnd.Get(c.AutoJumpsDelayMax-c.AutoJumpsDelayMin+1)
	ms := int64(sec) * 1000
	if r.boss {
		ms = int64(float64(ms) * c.BossRoomTimeMultiply)
	}
	return time.Duration(ms) * time.Millisecond
}

// riftTeleTimer is createTeleporterTimer(true): the jump, with an
// earthquake 7 s before it.
func (s *Server) riftTeleTimer(r *rift) {
	s.cancelTimer(&r.teleTimer)
	s.cancelTimer(&r.quakeTimer)
	d := s.riftJumpTime(r)
	r.teleTimer = s.loop.After(d, func() {
		r.teleTimer = 0
		s.riftAutoJump(r)
	})
	r.quakeTimer = s.loop.After(max(d-7*time.Second, 0), func() {
		r.quakeTimer = 0
		if r.party == nil {
			return
		}
		for _, m := range r.party.Members() {
			if !r.revived[m] {
				s.sendEarthquake(m, 65, 9)
			}
		}
	})
}

// riftAutoJump is the teleporter task: the next room while jumps are left
// and someone lives, otherwise back to the waiting room.
func (s *Server) riftAutoJump(r *rift) {
	if r.party == nil {
		return
	}
	s.riftRoomUnspawn(s.riftRoom(r.typ, r.room))
	if room := s.riftRoom(r.typ, r.room); room != nil {
		room.inside = false
	}
	if r.jump < s.riftMaxJumps() && r.party.Len() > len(r.dead) {
		r.jump++
		s.riftNextRoom(r, true)
		s.riftTeleTimer(r)
		s.riftSpawnTimer(r)
		return
	}
	for _, m := range r.party.Members() {
		if !r.revived[m] {
			s.riftTeleport(m, data.RiftTypeWaiting, 0)
		}
	}
	s.killRift(r)
}

// riftNextRoom is teleportToNextRoom for every member: a free room not done
// yet (any free one when all are done), boss room or not.
func (s *Server) riftNextRoom(r *rift, skipRevived bool) {
	r.completed = append(r.completed, r.room)
	free := s.freeRiftRooms(r.typ)
	var fresh []int
	for _, id := range free {
		if !containsInt(r.completed, id) {
			fresh = append(fresh, id)
		}
	}
	if len(fresh) == 0 {
		fresh = free
	}
	if len(fresh) == 0 {
		return
	}
	r.room = fresh[s.rnd.Get(len(fresh))]
	room := s.riftRoom(r.typ, r.room)
	room.inside = true
	r.boss = room.data.Boss
	for _, m := range r.party.Members() {
		if skipRevived && r.revived[m] {
			continue
		}
		s.riftTeleport(m, r.typ, r.room)
	}
}

// riftManualJump is manualTeleport: the leader moves the party on early.
func (s *Server) riftManualJump(r *rift, p *Player, n *Npc) {
	if r.party.Leader() != p {
		s.riftPage(p, n, "NotPartyLeader.htm", -1)
		return
	}
	if r.jump >= s.riftMaxJumps() {
		s.riftPage(p, n, "AlreadyTeleported.htm", -1)
		return
	}
	r.jump++
	room := s.riftRoom(r.typ, r.room)
	s.riftRoomUnspawn(room)
	room.inside = false
	s.riftNextRoom(r, false)
	s.riftSpawnTimer(r)
	s.riftTeleTimer(r)
}

// killRift is killRift: the party is out, the room empty, the timers gone.
func (s *Server) killRift(r *rift) {
	f := &s.sevenSigns.rift
	if r.party != nil {
		delete(f.parties, r.party)
	}
	r.party = nil
	if room := s.riftRoom(r.typ, r.room); room != nil {
		s.riftRoomUnspawn(room)
		room.inside = false
	}
	s.cancelTimer(&r.teleTimer)
	s.cancelTimer(&r.spawnTimer)
	s.cancelTimer(&r.quakeTimer)
}

// sendEarthquake sends p an Earthquake at its position.
func (s *Server) sendEarthquake(p *Player, intensity, seconds int32) {
	pos := p.Position()
	p.Send(serverpackets.NewEarthquakePacket(serverpackets.Earthquake{X: pos.X, Y: pos.Y, Z: pos.Z, Intensity: intensity, Duration: seconds}))
}

// riftMemberDied is memberDead for a member; a rift monster comes back after
// its respawn delay while its room is active (Spawn.startRespawn).
func (s *Server) riftMemberDied(victim, _ Creature) {
	if n, ok := victim.(*Npc); ok {
		for _, typ := range s.sevenSigns.rift.rooms {
			for _, room := range typ {
				if pt, ok := room.npcs[n]; ok {
					delete(room.npcs, n)
					s.loop.After(pt.delay, func() { s.riftSpawnPoint(room, pt) })
					return
				}
			}
		}
		return
	}
	p, ok := victim.(*Player)
	if !ok || p.Party() == nil {
		return
	}
	if r := s.sevenSigns.rift.parties[p.Party()]; r != nil {
		r.dead[p] = true
	}
}

// riftLeave is partyMemberExited for a member who leaves the game: below
// the minimum party size everyone goes back.
func (s *Server) riftLeave(p *Player) {
	pt := p.Party()
	if pt == nil {
		return
	}
	r := s.sevenSigns.rift.parties[pt]
	if r == nil {
		return
	}
	delete(r.dead, p)
	delete(r.revived, p)
	left := pt.Len() - 1
	if left < s.sevenSigns.cfg.Rift.MinPartySize || left == 1 {
		for _, m := range pt.Members() {
			if m != p {
				s.riftTeleport(m, data.RiftTypeWaiting, 0)
			}
		}
		s.killRift(r)
	}
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

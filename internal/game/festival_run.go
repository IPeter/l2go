package game

import (
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/sevensigns"
	"github.com/IPeter/l2go/internal/world"
)

// The festival cycles (Mobius SevenSignsFestival.FestivalManager and
// DarknessFestival) as timers on the loop: every cycle (38 minutes) after
// the first start (2 minutes), the sign-up runs 19 minutes (cycle −
// festival − 1 minute; again and again while no party signed up), then the
// parties go into their arenas, the witch appears, and 2, 5, 9 and 12
// minutes later the monsters spawn and swarm, the chests at 15, the end at
// 18: the parties go back to where they stood and their leaders register
// the Blood Offerings at the guide. The guides' links are Mobius's Festival
// bypass handler (bypassFestival).

// Mobius SevenSignsFestival constants.
const (
	festivalMaxOffset      = 230 // FESTIVAL_MAX_OFFSET_X and _Y
	festivalDefaultRespawn = 60  // FESTIVAL_DEFAULT_RESPAWN, seconds
	festivalDawnChatGuide  = 31127
	festivalDuskChatGuide  = 31137
)

// festivalRun is the running festival manager. Loop-only.
type festivalRun struct {
	active bool // the manager is scheduled (Mobius _managerScheduledTask)
	tick   world.TimerID
	tickAt time.Time
	phase  world.TimerID
	// busy is set from a tick to the end of its festival (Mobius's
	// synchronized run).
	busy bool

	initialized, inProgress bool
	nextCycleStart          time.Time
	nextFestivalStart       time.Time

	// participants and previous are by cabal (index 1 Dusk, 2 Dawn), then
	// festival; a nil list is Mobius's null entry.
	participants [3]map[int][]int32
	previous     [3]map[int][]int32
	instances    map[int]*darknessFestival // 10+id for Dusk, 20+id for Dawn
	// mobs are the festival's monsters (FestivalMonster), with their offering
	// bonus.
	mobs map[*Npc]*festivalMob
}

// festivalMob is a festival monster: its arena, spawn and offering bonus.
type festivalMob struct {
	fest    *darknessFestival
	spawn   data.FestivalSpawn
	respawn time.Duration
	bonus   int
}

// darknessFestival is one arena (Mobius DarknessFestival).
type darknessFestival struct {
	cabal        sevensigns.Cabal
	level        int
	challenged   bool
	start        data.FestivalSpawn
	witchSpawn   data.FestivalSpawn
	witch        *Npc
	mobs         []*Npc
	participants []int32
	original     map[int32]world.Position
}

// setupFestival registers the festival monsters' death hook.
func (s *Server) setupFestival() error {
	r := &s.sevenSigns.festival.run
	r.instances = make(map[int]*darknessFestival)
	r.mobs = make(map[*Npc]*festivalMob)
	for c := range r.participants {
		r.participants[c] = make(map[int][]int32)
		r.previous[c] = make(map[int][]int32)
	}
	s.OnDie(s.festivalMobDied)
	return nil
}

func (s *Server) festivalCfg() (cycle, length, signup time.Duration) {
	c := &s.sevenSigns.cfg
	cycle = time.Duration(c.FestivalCycleLength) * time.Millisecond
	length = time.Duration(c.FestivalLength) * time.Millisecond
	return cycle, length, cycle - length - time.Minute
}

// startFestivalManager is startFestivalManager: the first cycle after
// AltFestivalManagerStart, then one every cycle.
func (s *Server) startFestivalManager() {
	r := &s.sevenSigns.festival.run
	if r.active || s.sevenSigns.data == nil {
		return
	}
	r.active = true
	start := time.Duration(s.sevenSigns.cfg.FestivalManagerStart) * time.Millisecond
	_, _, signup := s.festivalCfg()
	r.nextFestivalStart = s.Now().Add(start + signup)
	r.tickAt = s.Now().Add(start)
	r.tick = s.loop.After(start, s.festivalTick)
	s.log.Info("game: the first festival of darkness cycle begins", "in", start)
}

// stopFestivalManager is getFestivalManagerSchedule().cancel(false): no new
// cycle starts; a running festival ends at its time.
func (s *Server) stopFestivalManager() {
	r := &s.sevenSigns.festival.run
	r.active = false
	s.cancelTimer(&r.tick)
}

// festivalTick is FestivalManager.run.
func (s *Server) festivalTick() {
	r := &s.sevenSigns.festival.run
	r.tick = 0
	if !r.active {
		return
	}
	cycle, _, signup := s.festivalCfg()
	st := s.sevenSigns.state
	switch {
	case st.IsSealValidation(), s.sevenSigns.next.Sub(s.Now()) < cycle:
		s.festivalNextTick()
		return
	case s.minsToNextFestival() == 2:
		s.festivalShout("The main event will start in 2 minutes. Please register now.")
	}
	r.busy = true
	r.phase = s.loop.After(signup, s.festivalSignupEnd)
}

// festivalNextTick schedules the next fixed-rate run; a late one runs at
// once.
func (s *Server) festivalNextTick() {
	r := &s.sevenSigns.festival.run
	r.busy = false
	if !r.active {
		return
	}
	cycle, _, _ := s.festivalCfg()
	r.tickAt = r.tickAt.Add(cycle)
	r.tick = s.loop.After(max(r.tickAt.Sub(s.Now()), 0), s.festivalTick)
}

// festivalSignupEnd ends the sign-up: the previous parties may no longer
// register, stray monsters go, and the festival starts when a party signed
// up (otherwise another sign-up runs).
func (s *Server) festivalSignupEnd() {
	r := &s.sevenSigns.festival.run
	clear(r.previous[sevensigns.CabalDawn])
	clear(r.previous[sevensigns.CabalDusk])
	for _, f := range r.instances {
		s.festivalUnspawn(f)
	}
	if len(r.participants[sevensigns.CabalDusk]) == 0 && len(r.participants[sevensigns.CabalDawn]) == 0 {
		cycle, _, signup := s.festivalCfg()
		r.nextCycleStart = s.Now().Add(cycle)
		r.nextFestivalStart = s.Now().Add(cycle - signup)
		r.phase = s.loop.After(cycle-signup, s.festivalSignupEnd)
		return
	}
	s.festivalInitiate()
}

// festivalInitiate is the festival's initiation: an arena for each party,
// the parties inside, the witches; the monsters come at the first spawn.
func (s *Server) festivalInitiate() {
	r := &s.sevenSigns.festival.run
	clear(r.instances)
	for id := range sevensigns.FestivalCount {
		if r.participants[sevensigns.CabalDusk][id] != nil {
			r.instances[10+id] = s.newDarknessFestival(sevensigns.CabalDusk, id)
		}
		if r.participants[sevensigns.CabalDawn][id] != nil {
			r.instances[20+id] = s.newDarknessFestival(sevensigns.CabalDawn, id)
		}
	}
	cycle, _, _ := s.festivalCfg()
	r.initialized = true
	r.nextFestivalStart = s.Now().Add(cycle)
	s.festivalShout("The main event is now starting.")
	c := &s.sevenSigns.cfg
	ms := func(v int) time.Duration { return time.Duration(v) * time.Millisecond }
	steps := []struct {
		at time.Duration
		fn func()
	}{
		{ms(c.FestivalFirstSpawn), func() {
			r.inProgress = true
			for _, f := range s.festivalInstances() {
				s.spawnFestivalMonsters(f, festivalDefaultRespawn, 0)
				s.festivalWitchSay(f, "The main event is now starting.")
			}
		}},
		{ms(c.FestivalFirstSwarm), func() {
			for _, f := range s.festivalInstances() {
				s.festivalSwarm(f)
			}
		}},
		{ms(c.FestivalSecondSpawn), func() {
			end := (c.FestivalLength - c.FestivalSecondSpawn) / 60000
			for _, f := range s.festivalInstances() {
				s.spawnFestivalMonsters(f, festivalDefaultRespawn/2, 2)
				if end == 2 {
					s.festivalWitchSay(f, "The Festival of Darkness will end in two minutes.")
				} else {
					s.festivalWitchSay(f, "The Festival of Darkness will end in "+strconv.Itoa(end)+" minute(s).")
				}
			}
		}},
		{ms(c.FestivalSecondSwarm), func() {
			for _, f := range s.festivalInstances() {
				s.festivalSwarm(f)
			}
		}},
		{ms(c.FestivalChestSpawn), func() {
			for _, f := range s.festivalInstances() {
				s.spawnFestivalMonsters(f, festivalDefaultRespawn, 3)
				s.festivalWitchSay(f, "The chests have spawned! Be quick, the festival will end soon.")
			}
		}},
		{ms(c.FestivalLength), s.festivalTerminate},
	}
	var run func(i int, at time.Duration)
	run = func(i int, at time.Duration) {
		if i >= len(steps) {
			return
		}
		r.phase = s.loop.After(steps[i].at-at, func() {
			r.phase = 0
			steps[i].fn()
			run(i+1, steps[i].at)
		})
	}
	run(0, 0)
}

// festivalInstances returns the arenas in Mobius's map order (Dusk 10-14,
// then Dawn 20-24).
func (s *Server) festivalInstances() []*darknessFestival {
	r := &s.sevenSigns.festival.run
	var out []*darknessFestival
	for _, k := range sortedKeys(r.instances) {
		out = append(out, r.instances[k])
	}
	return out
}

// festivalTerminate ends the festival: everyone goes back, the parties'
// leaders may register their scores, and sign-ups open again.
func (s *Server) festivalTerminate() {
	r := &s.sevenSigns.festival.run
	r.inProgress = false
	for _, f := range s.festivalInstances() {
		s.festivalEnd(f)
	}
	clear(r.participants[sevensigns.CabalDawn])
	clear(r.participants[sevensigns.CabalDusk])
	r.initialized = false
	s.festivalShout("That will do! I'll move you to the outside soon.")
	s.festivalNextTick()
}

// newDarknessFestival is DarknessFestival's constructor and festivalInit:
// the party goes to the arena (a random point around its centre, without
// buffs and stray offerings) and the witch appears.
func (s *Server) newDarknessFestival(c sevensigns.Cabal, id int) *darknessFestival {
	r := &s.sevenSigns.festival.run
	fd := s.sevenSigns.data.Festival
	f := &darknessFestival{cabal: c, level: id, participants: r.participants[c][id], original: make(map[int32]world.Position)}
	if c == sevensigns.CabalDawn {
		f.witchSpawn, f.start = fd.DawnWitch[id], fd.DawnPlayer[id]
	} else {
		f.witchSpawn, f.start = fd.DuskWitch[id], fd.DuskPlayer[id]
	}
	for _, oid := range f.participants {
		p, ok := s.Player(world.ObjectID(oid))
		if !ok {
			continue
		}
		f.original[oid] = p.Position()
		x, y := s.festivalOffset(f.start.X, f.start.Y)
		if err := s.Teleport(p, world.Position{X: int32(x), Y: int32(y), Z: int32(f.start.Z)}); err != nil { //nolint:gosec // world coordinates
			s.log.Warn("game: festival teleport failed", append(p.logArgs(), "err", err)...)
		}
		p.Effects().StopAllExceptStayAfterDeath()
		if p.inv.Inventory != nil {
			if it := p.inv.ByItemID(sevensigns.OfferingID); it != nil {
				s.DestroyItem(p, it, it.Count(), true)
			}
		}
	}
	pos := world.Position{X: int32(f.witchSpawn.X), Y: int32(f.witchSpawn.Y), Z: int32(f.witchSpawn.Z)} //nolint:gosec // world coordinates
	f.witch = s.spawnSiegeNpc(f.witchSpawn.NpcID, pos, int32(f.witchSpawn.Heading), "")                //nolint:gosec // headings
	s.festivalWitchSay(f, "The main event will start in 2 minutes. Please register now.")
	return f
}

// festivalOffset is the random point around a centre: both coordinates up
// or both down by Rnd.get(230) (Mobius Rnd.nextBoolean, then two draws).
func (s *Server) festivalOffset(x, y int) (int, int) {
	if s.rnd.Get(2) == 1 {
		return x + s.rnd.Get(festivalMaxOffset), y + s.rnd.Get(festivalMaxOffset)
	}
	return x - s.rnd.Get(festivalMaxOffset), y - s.rnd.Get(festivalMaxOffset)
}

// spawnFestivalMonsters is spawnFestivalMonsters: kind 0 the primary
// monsters, 1 the same without the archers (a raised challenge, offerings
// ×2), 2 the archers, 3 the chests (offerings ×5).
func (s *Server) spawnFestivalMonsters(f *darknessFestival, respawn, kind int) {
	fd := s.sevenSigns.data.Festival
	dawn := f.cabal == sevensigns.CabalDawn
	var list []data.FestivalSpawn
	switch kind {
	case 0, 1:
		list = pick(dawn, fd.DawnPrimary, fd.DuskPrimary)[f.level]
	case 2:
		list = pick(dawn, fd.DawnSecondary, fd.DuskSecondary)[f.level]
	case 3:
		list = pick(dawn, fd.DawnChest, fd.DuskChest)[f.level]
	default:
		return
	}
	bonus := 1
	switch kind {
	case 1:
		bonus = 2
	case 3:
		bonus = 5
	}
	for _, sp := range list {
		if kind == 1 && sevensigns.IsFestivalArcher(sp.NpcID) {
			continue
		}
		s.spawnFestivalMob(f, &festivalMob{fest: f, spawn: sp, respawn: time.Duration(respawn) * time.Second, bonus: bonus})
	}
}

func pick[T any](dawn bool, a, b T) T {
	if dawn {
		return a
	}
	return b
}

// spawnFestivalMob spawns one festival monster, facing a random way.
func (s *Server) spawnFestivalMob(f *darknessFestival, m *festivalMob) {
	pos := world.Position{X: int32(m.spawn.X), Y: int32(m.spawn.Y), Z: int32(m.spawn.Z)} //nolint:gosec // world coordinates
	n := s.spawnSiegeNpc(m.spawn.NpcID, pos, int32(s.rnd.Get(65536)), "")              //nolint:gosec // headings
	if n == nil {
		return
	}
	f.mobs = append(f.mobs, n)
	s.sevenSigns.festival.run.mobs[n] = m
}

// festivalMobDied is FestivalMonster.doItemDrop: a player's kill gives the
// leader of the player's party the offering bonus in Blood Offerings; the
// monster comes back after its respawn delay while the festival runs.
func (s *Server) festivalMobDied(victim, killer Creature) {
	n, ok := victim.(*Npc)
	if !ok {
		return
	}
	r := &s.sevenSigns.festival.run
	m, ok := r.mobs[n]
	if !ok {
		return
	}
	delete(r.mobs, n)
	if p, ok := killer.(*Player); ok {
		if pt := p.Party(); pt != nil {
			if leader := pt.Leader(); leader != nil {
				if _, err := s.AddItem(leader, sevensigns.OfferingID, int64(m.bonus), 0, false, false); err != nil {
					s.log.Warn("game: no blood offering for the festival party", append(leader.logArgs(), "err", err)...)
				}
			}
		}
	}
	f := m.fest
	s.loop.After(m.respawn, func() {
		if s.sevenSigns.festival.run.instances[festivalKey(f)] == f && f.witch != nil && f.witch.InWorld() {
			s.spawnFestivalMob(f, m)
		}
	})
}

func festivalKey(f *darknessFestival) int {
	if f.cabal == sevensigns.CabalDawn {
		return 20 + f.level
	}
	return 10 + f.level
}

// festivalSwarm is moveMonstersToCenter: the idle monsters run to a random
// point around the arena's centre.
func (s *Server) festivalSwarm(f *darknessFestival) {
	for _, n := range f.mobs {
		if !n.InWorld() || n.Dead {
			continue
		}
		if in := n.Intention(); in != NpcIdle && in != NpcActive {
			continue
		}
		x, y := s.festivalOffset(f.start.X, f.start.Y)
		s.SetNpcRunning(n, true)
		s.NpcMoveTo(n, world.Position{X: int32(x), Y: int32(y), Z: int32(f.start.Z), Heading: int32(s.rnd.Get(65536))}) //nolint:gosec // world coordinates
	}
}

// festivalEnd is festivalEnd: the party goes back and becomes the previous
// participants, the arena empties.
func (s *Server) festivalEnd(f *darknessFestival) {
	r := &s.sevenSigns.festival.run
	if len(f.participants) > 0 {
		for _, oid := range f.participants {
			p, ok := s.Player(world.ObjectID(oid))
			if !ok {
				continue
			}
			s.relocateFestivalPlayer(f, p, false)
			p.SendMessage("The festival has ended. Your party leader must now register your score before the next festival takes place.")
		}
		r.previous[f.cabal][f.level] = f.participants
	}
	f.participants = nil
	s.festivalUnspawn(f)
}

// festivalUnspawn is unspawnMobs.
func (s *Server) festivalUnspawn(f *darknessFestival) {
	r := &s.sevenSigns.festival.run
	if f.witch != nil && f.witch.InWorld() {
		s.DespawnNpc(f.witch)
	}
	for _, n := range f.mobs {
		delete(r.mobs, n)
		if n.InWorld() {
			s.DespawnNpc(n)
		}
	}
	f.mobs = nil
}

// relocateFestivalPlayer is relocatePlayer: back to where the player stood
// (town when that is not known).
func (s *Server) relocateFestivalPlayer(f *darknessFestival, p *Player, removing bool) {
	pos, ok := f.original[p.ObjectID()]
	if removing {
		delete(f.original, p.ObjectID())
	}
	if ok {
		if err := s.Teleport(p, pos); err != nil {
			s.teleportToTown(p)
		}
	} else {
		s.teleportToTown(p)
	}
	p.SendMessage("You have been removed from the festival arena.")
}

// festivalShout is sendMessageToAll: both chat guides shout to the oracles.
func (s *Server) festivalShout(text string) {
	var dawn, dusk *Npc
	for x := range s.world.Objects() {
		if n, ok := x.(*Npc); ok && n.Template != nil {
			switch n.Template.ID {
			case festivalDawnChatGuide:
				dawn = n
			case festivalDuskChatGuide:
				dusk = n
			}
		}
	}
	if dawn == nil || dusk == nil {
		return
	}
	for _, n := range []*Npc{dawn, dusk} {
		s.npcSayType(n, serverpackets.ChatShout, text)
	}
}

// npcSayType broadcasts NpcSay of the type around n.
func (s *Server) npcSayType(n *Npc, typ serverpackets.ChatType, text string) {
	s.world.Broadcast(n, serverpackets.NewNpcSayPacket(serverpackets.NpcSay{
		ObjectID: int32(n.ID()), //nolint:gosec // object IDs fit in int32
		Type:     typ,
		NpcID:    int32(n.Template.DisplayID), //nolint:gosec // NPC IDs fit in int32
		Text:     text,
	}), false)
}

// festivalWitchSay is sendMessageToParticipants: the witch speaks to the
// arena.
func (s *Server) festivalWitchSay(f *darknessFestival, text string) {
	if len(f.participants) == 0 || f.witch == nil || !f.witch.InWorld() {
		return
	}
	s.npcSayType(f.witch, serverpackets.ChatGeneral, text)
}

// characterName is CharInfoTable.getNameById for a festival party: the
// player online, else a clan member's name, else "".
func (s *Server) characterName(id int32) string {
	if p, ok := s.Player(world.ObjectID(id)); ok {
		return p.Name()
	}
	for _, cl := range s.Clans() {
		if m := cl.Member(id); m != nil {
			return m.Name()
		}
	}
	return ""
}

// minsToNextFestival is getMinsToNextFestival.
func (s *Server) minsToNextFestival() int {
	if s.sevenSigns.state.IsSealValidation() {
		return -1
	}
	return int(s.sevenSigns.festival.run.nextFestivalStart.Sub(s.Now())/time.Minute) + 1
}

// minsToNextCycle is getMinsToNextCycle.
func (s *Server) minsToNextCycle() int {
	if s.sevenSigns.state.IsSealValidation() {
		return -1
	}
	return int(s.sevenSigns.festival.run.nextCycleStart.Sub(s.Now()) / time.Minute)
}

// ----- participants -----

// festivalOf is getFestivalForPlayer: the cabal and festival the player
// takes part in, CabalNull and -1 for none.
func (s *Server) festivalOf(p *Player) (sevensigns.Cabal, int) {
	r := &s.sevenSigns.festival.run
	for _, c := range []sevensigns.Cabal{sevensigns.CabalDawn, sevensigns.CabalDusk} {
		for id := range sevensigns.FestivalCount {
			for _, oid := range r.participants[c][id] {
				if oid == p.ObjectID() {
					return c, id
				}
			}
		}
	}
	return sevensigns.CabalNull, -1
}

// IsFestivalParticipant is Player.isFestivalParticipant. Loop-only.
func (s *Server) IsFestivalParticipant(p *Player) bool {
	if st := s.sevenSigns.state; st == nil || st.IsSealValidation() || !s.sevenSigns.festival.run.active {
		return false
	}
	c, _ := s.festivalOf(p)
	return c != sevensigns.CabalNull
}

// setFestivalParticipants is setParticipants: the party's members (nil for
// none).
func (s *Server) setFestivalParticipants(c sevensigns.Cabal, id int, pt *Party) {
	r := &s.sevenSigns.festival.run
	var ids []int32
	if pt != nil {
		for _, m := range pt.Members() {
			ids = append(ids, m.ObjectID())
		}
	}
	r.participants[c][id] = ids
}

// clearFestivalParties forgets every participant.
func (s *Server) clearFestivalParties() {
	r := &s.sevenSigns.festival.run
	for c := range r.participants {
		clear(r.participants[c])
		clear(r.previous[c])
	}
}

// festivalLeave is Player.deleteMe for a participant: the player leaves the
// festival's list (and the arena), as updateParticipants does.
func (s *Server) festivalLeave(p *Player) {
	r := &s.sevenSigns.festival.run
	c, id := s.festivalOf(p)
	if c == sevensigns.CabalNull {
		return
	}
	list := r.participants[c][id]
	out := list[:0:0]
	for _, oid := range list {
		if oid != p.ObjectID() {
			out = append(out, oid)
		}
	}
	r.participants[c][id] = out
	if f := r.instances[festivalKeyOf(c, id)]; f != nil {
		delete(f.original, p.ObjectID())
	}
}

func festivalKeyOf(c sevensigns.Cabal, id int) int {
	if c == sevensigns.CabalDawn {
		return 20 + id
	}
	return 10 + id
}

// ----- the guides -----

// festivalGuide is what FestivalGuide knows of a guide or witch NPC: its
// festival, cabal and the seal stones a party offers.
func festivalGuide(npcID int) (id int, c sevensigns.Cabal, blue, green, red int) {
	stones := [sevensigns.FestivalCount][3]int{{900, 540, 270}, {1500, 900, 450}, {3000, 1800, 900}, {4500, 2700, 1350}, {6000, 3600, 1800}}
	switch {
	case npcID >= 31127 && npcID <= 31131:
		id, c = npcID-31127, sevensigns.CabalDawn
	case npcID >= 31132 && npcID <= 31136:
		id, c = npcID-31132, sevensigns.CabalDawn
	case npcID >= 31137 && npcID <= 31141:
		id, c = npcID-31137, sevensigns.CabalDusk
	case npcID >= 31142 && npcID <= 31146:
		id, c = npcID-31142, sevensigns.CabalDusk
	default:
		return sevensigns.FestivalCount - 1, sevensigns.CabalNull, 0, 0, 0
	}
	st := stones[id]
	return id, c, st[0], st[1], st[2]
}

// festivalGuidePage is FestivalGuide.showChatWindow(player, value, suffix,
// isDescription).
func (s *Server) festivalGuidePage(p *Player, n *Npc, value int, suffix string, desc bool) {
	id, _, blue, green, red := festivalGuide(n.Template.ID)
	name := sevenSignsHTML + "festival/"
	if desc {
		name += "desc_"
	} else {
		name += "festival_"
	}
	name += strconv.Itoa(value) + suffix + ".htm"
	pg := s.pageFile(p, n.ID(), name).replaceObjectID(n.ID()).replace("%festivalType%", sevensigns.FestivalName(id)).
		replace("%cycleMins%", strconv.Itoa(s.minsToNextCycle()))
	if !desc && strconv.Itoa(value)+suffix == "2b" {
		pg = pg.replace("%minFestivalPartyMembers%", strconv.Itoa(s.sevenSigns.cfg.FestivalMinPlayers))
	}
	book := s.sevenSigns.festival.book
	if value == 5 && book != nil {
		var b strings.Builder
		for i := range sevensigns.FestivalCount {
			dawn, dusk := book.HighestScoreData(sevensigns.CabalDawn, i).Score, book.HighestScoreData(sevensigns.CabalDusk, i).Score
			winner := "Children of Dusk"
			if dawn > dusk {
				winner = "Children of Dawn"
			} else if dawn == dusk {
				winner = "None"
			}
			b.WriteString(`<tr><td width="100" align="center">` + sevensigns.FestivalName(i) + `</td><td align="center" width="35">` + strconv.Itoa(dusk) +
				`</td><td align="center" width="35">` + strconv.Itoa(dawn) + `</td><td align="center" width="130">` + winner + `</td></tr>`)
		}
		pg = pg.replace("%statsTable%", b.String())
	}
	if value == 6 && book != nil {
		var b strings.Builder
		for i := range sevensigns.FestivalCount {
			b.WriteString(`<tr><td align="center" width="150">` + sevensigns.FestivalName(i) + `</td><td align="center" width="150">` + strconv.Itoa(book.Bonus[i]) + `</td></tr>`)
		}
		pg = pg.replace("%bonusTable%", b.String())
	}
	if value == 1 {
		pg = pg.replace("%blueStoneNeeded%", strconv.Itoa(blue)).replace("%greenStoneNeeded%", strconv.Itoa(green)).replace("%redStoneNeeded%", strconv.Itoa(red))
	}
	s.sendPage(p, pg)
	p.SendActionFailed()
}

// bypassFestival is Mobius's Festival bypass handler ("festival <n> ...",
// "festivaldesc <n>") at a festival guide or witch.
func (s *Server) bypassFestival(p *Player, n *Npc, command string) {
	st := s.sevenSigns.state
	if st == nil || s.sevenSigns.festival.book == nil {
		return
	}
	lower := strings.ToLower(command)
	if strings.HasPrefix(lower, "festivaldesc") {
		if len(command) > 13 {
			if v, err := strconv.Atoi(command[13:]); err == nil {
				s.festivalGuidePage(p, n, v, "", true)
			}
		}
		return
	}
	if len(command) < 10 {
		return
	}
	val, err := strconv.Atoi(command[9:10])
	if err != nil {
		return
	}
	id, cabal, blue, green, red := festivalGuide(n.Template.ID)
	r := &s.sevenSigns.festival.run
	book := s.sevenSigns.festival.book
	pt := p.Party()
	switch val {
	case 1: // become a participant
		switch {
		case st.IsSealValidation():
			s.festivalGuidePage(p, n, 2, "a", false)
		case r.initialized:
			p.SendMessage("You cannot sign up while a festival is in progress.")
		case pt == nil:
			s.festivalGuidePage(p, n, 2, "b", false)
		case !pt.IsLeader(p):
			s.festivalGuidePage(p, n, 2, "c", false)
		case pt.Len() < s.sevenSigns.cfg.FestivalMinPlayers:
			s.festivalGuidePage(p, n, 2, "b", false)
		case pt.Level() > sevensigns.FestivalMaxLevel(id, s.MaxLevel()):
			s.festivalGuidePage(p, n, 2, "d", false)
		case s.IsFestivalParticipant(p):
			s.setFestivalParticipants(cabal, id, pt)
			s.festivalGuidePage(p, n, 2, "f", false)
		default:
			s.festivalGuidePage(p, n, 1, "", false)
		}
	case 2: // the seal stones
		if len(command) <= 11 {
			return
		}
		stone, err := strconv.Atoi(command[11:])
		if err != nil {
			return
		}
		count := map[int]int{sevensigns.StoneBlueID: blue, sevensigns.StoneGreenID: green, sevensigns.StoneRedID: red}[stone]
		if count <= 0 || !s.DestroyItemByItemID(p, int32(stone), int64(count), true) { //nolint:gosec // item IDs
			return
		}
		s.setFestivalParticipants(cabal, id, pt)
		book.AddBonus(id, stone, count)
		s.saveSevenSigns(false, false)
		s.festivalGuidePage(p, n, 2, "e", false)
	case 3: // register the score
		prev := r.previous[cabal][id]
		switch {
		case st.IsSealValidation():
			s.festivalGuidePage(p, n, 3, "a", false)
			return
		case r.inProgress:
			p.SendMessage("You cannot register a score while a festival is in progress.")
			return
		case pt == nil, len(prev) == 0, !containsID(prev, p.ObjectID()), prev[0] != p.ObjectID():
			s.festivalGuidePage(p, n, 3, "b", false)
			return
		}
		it := p.inv.ByItemID(sevensigns.OfferingID)
		if it == nil {
			p.SendMessage("You do not have any blood offerings to contribute.")
			return
		}
		score := int(it.Count()) * sevensigns.OfferingValue
		if !s.DestroyItem(p, it, it.Count(), false) {
			return
		}
		var names []string
		for _, oid := range prev {
			names = append(names, s.characterName(oid))
		}
		best, points := book.SetFinalScore(cabal, id, score, names, s.Now().UnixMilli())
		if points > 0 {
			st.AddFestivalScore(cabal, points)
		}
		if best {
			s.saveSevenSigns(false, true)
		}
		p.SendSystemMessage(serverpackets.SystemMessageID(sevensigns.MsgContributionIncreased), serverpackets.NumberParam(int32(score))) //nolint:gosec // scores
		if best {
			s.festivalGuidePage(p, n, 3, "c", false)
		} else {
			s.festivalGuidePage(p, n, 3, "d", false)
		}
	case 4: // the high scores
		s.festivalHighScores(p, n, id)
	case 8: // raise the challenge
		if pt == nil || !r.inProgress {
			return
		}
		if !pt.IsLeader(p) {
			s.festivalGuidePage(p, n, 8, "a", false)
			return
		}
		f := r.instances[festivalKeyOf(cabal, id)]
		if f != nil && !f.challenged {
			f.challenged = true
			s.spawnFestivalMonsters(f, festivalDefaultRespawn, 1)
			s.festivalGuidePage(p, n, 8, "b", false)
		} else {
			s.festivalGuidePage(p, n, 8, "c", false)
		}
	case 9: // leave the festival
		if pt == nil {
			return
		}
		if pt.IsLeader(p) {
			s.festivalPartyLeaves(p)
		} else if pt.Len() > s.sevenSigns.cfg.FestivalMinPlayers {
			s.removePartyMember(pt, p, partyLeaveExpelled)
		} else {
			p.SendMessage("Only the party leader can leave a festival when a party has minimum number of members.")
		}
	case 0: // the accumulated bonus
		if !st.IsSealValidation() {
			p.SendMessage("Bonuses cannot be paid during the competition period.")
			return
		}
		// Mobius takes the share from the bonus but gives the player nothing.
		if book.DistribBonus(p.Name(), st.PlayerCabal(p.ObjectID()), st.HighestCabal()) > 0 {
			s.saveSevenSigns(false, false)
			s.festivalGuidePage(p, n, 0, "a", false)
		} else {
			s.festivalGuidePage(p, n, 0, "b", false)
		}
	default:
		s.festivalGuidePage(p, n, val, "", false)
	}
}

// festivalPartyLeaves is updateParticipants(leader, null): the whole party
// leaves the festival (and the arena while it runs).
func (s *Server) festivalPartyLeaves(p *Player) {
	r := &s.sevenSigns.festival.run
	if !s.IsFestivalParticipant(p) {
		return
	}
	c, id := s.festivalOf(p)
	if r.initialized {
		if f := r.instances[festivalKeyOf(c, id)]; f != nil {
			for _, oid := range r.participants[c][id] {
				if m, ok := s.Player(world.ObjectID(oid)); ok {
					s.relocateFestivalPlayer(f, m, true)
				}
			}
		}
	}
	r.participants[c][id] = nil
}

// festivalHighScores is the guide's fourth page: this cycle's best scores
// of both cabals and the best ever, with the date as Mobius writes it (the
// month from 0).
func (s *Server) festivalHighScores(p *Player, n *Npc, id int) {
	book := s.sevenSigns.festival.book
	date := func(ms int64) string {
		t := s.Calendar().In(time.UnixMilli(ms))
		return strconv.Itoa(t.Year()) + "/" + strconv.Itoa(int(t.Month())-1) + "/" + strconv.Itoa(t.Day())
	}
	var b strings.Builder
	b.WriteString("<html><body>Festival Guide:<br>These are the top scores of the week, for the ")
	b.WriteString(sevensigns.FestivalName(id) + " festival.<br>")
	dawn, dusk := book.HighestScoreData(sevensigns.CabalDawn, id), book.HighestScoreData(sevensigns.CabalDusk, id)
	for _, e := range []struct {
		label string
		d     sevensigns.FestivalScore
	}{{"Dawn", dawn}, {"Dusk", dusk}} {
		if e.d.Score > 0 {
			b.WriteString(e.label + ": " + date(e.d.Date) + ". Score " + strconv.Itoa(e.d.Score) + "<br>" + e.d.Members + "<br>")
		} else {
			b.WriteString(e.label + ": No record exists. Score 0<br>")
		}
	}
	if best, ok := book.OverallHighest(id); ok && best.Score > 0 {
		side := "Children of Dusk"
		if best.Cabal == sevensigns.CabalDawn {
			side = "Children of Dawn"
		}
		b.WriteString("Consecutive top scores: " + date(best.Date) + ". Score " + strconv.Itoa(best.Score) + "<br>Affilated side: " + side + "<br>" + best.Members + "<br>")
	} else {
		b.WriteString("Consecutive top scores: No record exists. Score 0<br>")
	}
	b.WriteString(`<a action="bypass -h npc_` + strconv.Itoa(int(n.ID())) + `_Chat 0">Go back.</a></body></html>`)
	s.sendPage(p, newPage(n.ID(), b.String()))
}

func containsID(list []int32, id int32) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

package game

import (
	"context"
	"maps"
	"math"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// Clan halls (Plan 14 M11j): the 38 auctionable halls of clanhall.sql and
// the 6 contestable halls of siegable_clanhall.sql (Mobius ClanHallTable,
// AuctionableHall, SiegableHall and CHSiegeManager). A hall's owner is the
// clan whose HideoutID names it. An auctionable hall is bought at an
// auctioneer (auction.go) and costs a weekly lease from the clan warehouse;
// both kinds rent functions at their manager (clanhall_funcs.go,
// clanhall_npc.go). Contestable halls change hands in sieges (hallsiege.go).
// Every transition is saved at once, with the clans and the clan warehouse
// items it changed, in one transaction (X4).

func init() { registerFeature("clanhalls", setupClanHalls) }

// hallLoadTimeout bounds reading the halls and auctions at startup.
const hallLoadTimeout = 30 * time.Second

// Nearness of a hall's NPCs (Mobius getNearbyClanHall): an auctionable
// hall's manager or doorman stands within 500 of its zone, a contestable
// hall's NPCs within 10,000.
const (
	hallNpcRange      = 500
	hallSiegeNpcRange = 10000
)

// hallFeature is the clan halls on the Server (Server.halls). Loop-only,
// except data and store, which New sets.
type hallFeature struct {
	data  *data.ClanHalls
	store storage.ClanHallStore

	cfg      config.ClanHalls
	siegeCfg config.HallSiege

	halls []*ClanHall // in ID order
	byID  map[int32]*ClanHall

	// auctions are the running auctions in Mobius's list order; pending the
	// sales a seller has prepared but not confirmed (Auctioneer
	// _pendingAuctions), by hall.
	auctions []*hallAuction
	pending  map[int32]*residence.Auction

	// npcHall caches which hall an NPC belongs to (Mobius _clanHallId), 0
	// for none.
	npcHall map[world.ObjectID]int32
}

// ClanHall is a clan hall on the loop (Mobius ClanHall with AuctionableHall
// or SiegableHall). Loop-only.
type ClanHall struct {
	srv  *Server
	Data *data.ClanHall
	ID   int32
	Name string

	ownerID   int32
	paidUntil int64 // Unix milliseconds
	paid      bool
	functions map[int]*hallFunction
	feeTimer  world.TimerID

	// zone is the hall's ClanHallZone (or SiegableHallZone); siegeZone the
	// battlefield of a contestable hall; teleZones its
	// ResidenceHallTeleportZones by residenceZoneId.
	zone, siegeZone *data.Zone
	teleZones       map[int]*data.Zone

	// siege is a contestable hall's siege (hallsiege.go), nil for an
	// auctionable hall.
	siege *hallSiege
}

// setupClanHalls makes the halls from the data and their stored state, the
// NPC auctions of the free halls, and starts the lease, function and auction
// tasks once the loop runs.
func setupClanHalls(s *Server) error {
	f := &s.halls
	f.cfg, f.siegeCfg = s.gs.ClanHalls, s.gs.HallSiege
	if len(f.cfg.HpRegenFees) == 0 {
		f.cfg = config.DefaultClanHalls()
	}
	if f.siegeCfg.MaxAttackers == 0 {
		f.siegeCfg = config.DefaultHallSiege()
	}
	f.byID = make(map[int32]*ClanHall)
	f.pending = make(map[int32]*residence.Auction)
	f.npcHall = make(map[world.ObjectID]int32)
	s.OnPlayerEnter(s.hallPlayerEntered)
	if err := s.registerHallGM(); err != nil { // clanhall_gm.go
		return err
	}
	if f.data == nil {
		return nil
	}
	var stored []storage.ClanHall
	var auctions []storage.Auction
	if f.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), hallLoadTimeout)
		defer cancel()
		var err error
		if stored, err = f.store.LoadClanHalls(ctx); err != nil {
			return err
		}
		if auctions, err = f.store.LoadAuctions(ctx); err != nil {
			return err
		}
	}
	byID := make(map[int32]storage.ClanHall, len(stored))
	for _, h := range stored {
		byID[h.ID] = h
	}
	for i := range f.data.Halls {
		h := s.newClanHall(&f.data.Halls[i])
		if st, ok := byID[h.ID]; ok {
			h.restore(st)
		}
		f.halls = append(f.halls, h)
		f.byID[h.ID] = h
	}
	// The owners come from the clans (Mobius clanhall.ownerId and
	// clan.setHideoutId).
	for _, cl := range s.Clans() {
		if h := f.byID[cl.HideoutID]; h != nil {
			h.ownerID = cl.ID
		}
	}
	for _, a := range auctions {
		if f.byID[a.ID] != nil {
			f.auctions = append(f.auctions, newHallAuction(a))
		}
	}
	for _, h := range f.halls {
		if h.ownerID == 0 { // Mobius loads the functions of owned halls only
			h.functions = make(map[int]*hallFunction)
		}
	}
	s.loop.After(0, func() {
		var b storage.Batch
		for _, h := range f.halls {
			s.startHallTasks(h, &b)
		}
		// ClanHallTable.load: a free auctionable hall with a lease and no
		// auction gets its NPC auction (initNPC).
		for _, h := range f.halls {
			if h.siege == nil && h.ownerID == 0 && h.Data.Lease > 0 && s.hallAuction(h.ID) == nil {
				if a := s.newNpcAuction(h); a != nil {
					f.auctions = append(f.auctions, a)
				}
			}
		}
		for _, a := range f.auctions {
			if s.startAuctionTask(a) {
				b.Auctions = append(b.Auctions, a.document())
			}
		}
		s.SaveJoint(nil, b)
	})
	return nil
}

// newClanHall makes a hall of the data without an owner, with its zones.
func (s *Server) newClanHall(d *data.ClanHall) *ClanHall {
	h := &ClanHall{srv: s, Data: d, ID: int32(d.ID), Name: d.Name, functions: make(map[int]*hallFunction), //nolint:gosec // hall IDs are small
		teleZones: make(map[int]*data.Zone)}
	if w := s.worldData; w != nil {
		for i := range w.Zones {
			z := &w.Zones[i]
			switch z.Type {
			case "ClanHallZone", "SiegableHallZone":
				if zoneIntParam(z.Params, "clanHallId") == float64(h.ID) {
					h.zone = z
				}
			case "SiegeZone":
				if zoneIntParam(z.Params, "clanHallId") == float64(h.ID) {
					h.siegeZone = z
				}
			case "ResidenceHallTeleportZone":
				if zoneIntParam(z.Params, "residenceId") == float64(h.ID) {
					h.teleZones[int(zoneIntParam(z.Params, "residenceZoneId"))] = z
				}
			}
		}
	}
	if d.Siegable {
		h.siege = newHallSiege(h) // hallsiege.go
	}
	return h
}

// restore takes the stored state.
func (h *ClanHall) restore(st storage.ClanHall) {
	h.paidUntil, h.paid = st.PaidUntil, st.Paid
	for _, f := range st.Functions {
		h.functions[int(f.Type)] = &hallFunction{HallFunction: residence.HallFunction{Type: int(f.Type), Level: int(f.Level),
			Lease: int(f.Lease), Rate: f.Rate, EndTime: f.EndTime}, fromWarehouse: true}
	}
	if h.siege != nil {
		h.siege.restore(st)
	}
}

// document returns the hall's stored state.
func (h *ClanHall) document() storage.ClanHall {
	st := storage.ClanHall{ID: h.ID, PaidUntil: h.paidUntil, Paid: h.paid}
	for _, t := range sortedKeys(h.functions) {
		f := h.functions[t]
		st.Functions = append(st.Functions, storage.ClanHallFunction{Type: int32(f.Type), Level: int32(f.Level), //nolint:gosec // small
			Lease: int32(f.Lease), Rate: f.Rate, EndTime: f.EndTime}) //nolint:gosec // fees fit
	}
	if h.siege != nil {
		h.siege.document(&st)
	}
	return st
}

// save writes the hall with the clans given in one transaction (X4).
func (h *ClanHall) save(clans ...*Clan) {
	b := storage.Batch{ClanHalls: []storage.ClanHall{h.document()}}
	for _, cl := range clans {
		if cl != nil {
			b.Clans = append(b.Clans, cl.document())
		}
	}
	h.srv.SaveJoint(nil, b)
}

// ClanHalls returns the halls in ID order. Loop-only.
func (s *Server) ClanHalls() []*ClanHall { return s.halls.halls }

// ClanHallByID returns the hall with the ID, or nil. Loop-only.
func (s *Server) ClanHallByID(id int32) *ClanHall { return s.halls.byID[id] }

// ClanHallByOwner returns the hall a clan owns, or nil (Mobius
// getAbstractHallByOwner). Loop-only.
func (s *Server) ClanHallByOwner(c *Clan) *ClanHall {
	if c == nil || c.HideoutID == 0 {
		return nil
	}
	if h := s.halls.byID[c.HideoutID]; h != nil && h.ownerID == c.ID {
		return h
	}
	return nil
}

// OwnerID returns the owner clan's ID, 0 for none.
func (h *ClanHall) OwnerID() int32 { return h.ownerID }

// Owner returns the owner clan, or nil.
func (h *ClanHall) Owner() *Clan { return h.srv.Clan(h.ownerID) }

// Siegable reports whether the hall is contestable.
func (h *ClanHall) Siegable() bool { return h.siege != nil }

// Lease returns an auctionable hall's weekly lease (0 for a contestable
// hall, Mobius ClanHall.getLease).
func (h *ClanHall) Lease() int {
	if h.siege != nil {
		return 0
	}
	return h.Data.Lease
}

// Grade returns an auctionable hall's grade (0 for a contestable one).
func (h *ClanHall) Grade() int {
	if h.siege != nil {
		return 0
	}
	return h.Data.Grade
}

// PaidUntil returns when the lease is paid until.
func (h *ClanHall) PaidUntil() time.Time { return msTime(h.paidUntil) }

// Contains reports whether pos is in the hall's zone (Mobius checkIfInZone).
func (h *ClanHall) Contains(pos world.Position) bool {
	return h.zone != nil && h.zone.Shape.Contains(int(pos.X), int(pos.Y), int(pos.Z))
}

// ----- zones and NPCs -----

// zoneDistance is Mobius ZoneForm.getDistanceToZone: the distance to the
// nearest corner of a polygon or rectangle, or to a cylinder's edge.
func zoneDistance(z *data.Zone, x, y int) float64 {
	sh := z.Shape
	pts := sh.Points
	if sh.Type == data.ShapeCylinder && len(pts) > 0 {
		return math.Hypot(float64(pts[0].X-x), float64(pts[0].Y-y)) - float64(sh.Radius)
	}
	if sh.Type == data.ShapeCuboid && len(pts) == 2 {
		pts = []data.Point{pts[0], {X: pts[0].X, Y: pts[1].Y}, pts[1], {X: pts[1].X, Y: pts[0].Y}}
	}
	best := math.MaxFloat64
	for _, p := range pts {
		best = min(best, math.Hypot(float64(p.X-x), float64(p.Y-y)))
	}
	return best
}

// npcClanHall is Mobius getClanHall of the hall NPCs: the first owned, then
// free, auctionable hall whose zone is within 500 of the NPC, else a
// contestable hall within 10,000 (CHSiegeManager.getNearbyClanHall). The
// answer is kept for the NPC.
func (s *Server) npcClanHall(n *Npc) *ClanHall {
	if id, ok := s.halls.npcHall[n.ID()]; ok {
		return s.halls.byID[id]
	}
	h := s.nearbyHall(n.Position(), false)
	if h == nil {
		h = s.nearbyHall(n.Position(), true)
	}
	var id int32
	if h != nil {
		id = h.ID
	}
	s.halls.npcHall[n.ID()] = id
	return h
}

// nearbyHall finds an auctionable hall within 500 (owned ones first, as
// ClanHallTable.getNearbyClanHall), or with siegable a contestable one
// within 10,000.
func (s *Server) nearbyHall(pos world.Position, siegable bool) *ClanHall {
	x, y := int(pos.X), int(pos.Y)
	if siegable {
		for _, h := range s.halls.halls {
			if h.siege != nil && h.zone != nil && zoneDistance(h.zone, x, y) < hallSiegeNpcRange {
				return h
			}
		}
		return nil
	}
	for _, owned := range []bool{true, false} {
		for _, h := range s.halls.halls {
			if h.siege == nil && (h.ownerID != 0) == owned && h.zone != nil && zoneDistance(h.zone, x, y) < hallNpcRange {
				return h
			}
		}
	}
	return nil
}

// hallSpawnLoc is a point of the hall zone's spawns of a kind (ZoneRespawn
// getSpawnLoc and its other, chaotic, banish and challenger twins), random
// with RandomRespawnInTown.
func (s *Server) hallSpawnLoc(z *data.Zone, kind string) (data.Location, bool) {
	if z == nil {
		return data.Location{}, false
	}
	pts := zoneSpawns(z, kind)
	if len(pts) == 0 {
		return data.Location{}, false
	}
	if s.combatSettings().RandomRespawnInTown {
		return pts[s.rnd.Get(len(pts))], true
	}
	return pts[0], true
}

// zoneSpawns returns a zone's spawn points of a kind.
func zoneSpawns(z *data.Zone, kind string) []data.Location {
	var pts []data.Location
	for _, sp := range z.Spawns {
		if sp.Type == kind {
			pts = append(pts, data.Location{X: sp.X, Y: sp.Y, Z: sp.Z})
		}
	}
	return pts
}

// banishForeigners is ResidenceZone.banishForeigners: everyone in the hall
// zone not of the owner clan goes to the banish point (the normal one
// without), 50 around.
func (s *Server) banishForeigners(h *ClanHall) {
	if h.zone == nil {
		return
	}
	loc, ok := s.hallSpawnLoc(h.zone, data.ZoneSpawnBanish)
	if !ok {
		if loc, ok = s.hallSpawnLoc(h.zone, data.ZoneSpawnNormal); !ok {
			return
		}
	}
	for _, p := range s.luaPlayersInZone(h.zone) {
		if p.ClanID() == 0 || p.ClanID() != h.ownerID {
			s.teleportAround(p, loc, maxOffsetOnTeleport)
		}
	}
}

// hallDoors returns the hall's doors (Mobius getDoors: the doors whose
// clanhall_id names the hall).
func (s *Server) hallDoors(h *ClanHall) []*Door {
	var out []*Door
	for _, d := range s.Doors() {
		if d.Template != nil && d.Template.ClanHallID == int(h.ID) {
			out = append(out, d)
		}
	}
	return out
}

// openCloseHallDoors is ClanHall.openCloseDoors.
func (s *Server) openCloseHallDoors(h *ClanHall, open bool) {
	for _, d := range s.hallDoors(h) {
		if open {
			s.openDoor(d)
		} else {
			s.closeDoor(d)
		}
	}
}

// openCloseHallDoor is ClanHall.openCloseDoor(doorId, open).
func (s *Server) openCloseHallDoor(h *ClanHall, doorID int, open bool) {
	for _, d := range s.hallDoors(h) {
		if d.Template.ID != doorID {
			continue
		}
		if open {
			s.openDoor(d)
		} else {
			s.closeDoor(d)
		}
	}
}

// ----- owner -----

// setHallOwner is ClanHallTable.setOwner with ClanHall.setOwner: an owned
// hall is freed first; the clan gets the hall, its members PledgeShowInfoUpdate,
// and an auctionable hall's lease is due at once (initialyzeTask(true)).
// Saved with the clans in one transaction.
func (s *Server) setHallOwner(h *ClanHall, cl *Clan) {
	if cl == nil {
		return
	}
	h.save(s.setHallOwnerState(h, cl)...) // auction.go
	s.afterHallOwner(h)
}

// freeHall is ClanHallTable.setFree (and SiegableHall.free): the owner
// loses the hall, its functions stop and the lease is forgotten; saved with
// the old owner.
func (s *Server) freeHall(h *ClanHall) {
	old := s.freeHallState(h)
	h.save(old)
}

// freeHallState frees the hall without saving and returns the old owner.
func (s *Server) freeHallState(h *ClanHall) *Clan {
	old := h.Owner()
	if old != nil && old.HideoutID == h.ID {
		old.HideoutID = 0
	}
	h.ownerID = 0
	for _, f := range h.functions {
		if f.timer != 0 {
			s.loop.Cancel(f.timer)
		}
	}
	h.functions = make(map[int]*hallFunction)
	h.paidUntil, h.paid = 0, false
	if h.feeTimer != 0 {
		s.loop.Cancel(h.feeTimer)
		h.feeTimer = 0
	}
	return old
}

// ----- lease -----

// startHallTasks starts a hall's tasks once the loop runs: the lease
// (initialyzeTask(false)) and the functions of an owned hall, a contestable
// hall's siege.
func (s *Server) startHallTasks(h *ClanHall, b *storage.Batch) {
	if h.ownerID != 0 {
		if h.siege == nil {
			s.scheduleLease(h, residence.LeaseFirstWait(s.Now().UnixMilli(), h.paidUntil, h.paid, false))
		}
		for _, t := range sortedKeys(h.functions) {
			f := h.functions[t]
			s.scheduleHallFunction(h, f, residence.FunctionFirstWait(f.HallFunction, s.Now().UnixMilli()))
		}
	}
	if h.siege != nil {
		s.startHallSiege(h, b) // hallsiege.go
	}
}

// scheduleLease (re)arms the lease task.
func (s *Server) scheduleLease(h *ClanHall, d time.Duration) {
	if h.feeTimer != 0 {
		s.loop.Cancel(h.feeTimer)
	}
	h.feeTimer = s.loop.After(d, func() {
		h.feeTimer = 0
		s.hallLeaseTask(h)
	})
}

// hallLeaseTask is AuctionableHall.FeeTask.run: the warehouse pays the
// week, or the clan is warned, and a week overdue it loses the hall, which
// goes back to an NPC auction. The hall, the clan and its warehouse are
// saved in one transaction.
func (s *Server) hallLeaseTask(h *ClanHall) {
	cl := h.Owner()
	if h.siege != nil || cl == nil {
		return
	}
	wh := s.ClanWarehouse(cl)
	st := residence.NextLeaseStep(s.Now().UnixMilli(), h.paidUntil, wh.Adena(), h.Data.Lease)
	if st.Revoke {
		s.freeHallState(h)
		b := storage.Batch{Clans: []storage.Clan{cl.document()}, ClanHalls: []storage.ClanHall{h.document()}}
		if s.hallAuction(h.ID) == nil {
			if a := s.newNpcAuction(h); a != nil {
				s.halls.auctions = append(s.halls.auctions, a)
				s.startAuctionTask(a)
				b.Auctions = append(b.Auctions, a.document())
			}
		}
		s.SaveJoint(nil, b)
		broadcastToClan(cl, clanSysMsg(serverpackets.SystemMessageID(residence.MsgHallLeaseRevoked)))
		s.broadcastClanInfo(cl)
		return
	}
	h.paidUntil, h.paid = st.PaidUntil, st.Paid
	var b storage.Batch
	if st.Pay && h.Data.Lease > 0 {
		if _, err := wh.DestroyByItemID(adenaID, int64(h.Data.Lease)); err != nil {
			s.log.Error("game: clan hall lease not taken", "hall", h.ID, "clan", cl.Name, "err", err)
		}
		if ch := wh.Tracker().Take(); !noItemChanges(ch) {
			b.Items = []storage.OwnerItemChanges{ownerItemChanges(cl.ID, ch)}
		}
		wh.TakeUpdates()
	}
	if st.Save || st.Pay {
		b.ClanHalls = []storage.ClanHall{h.document()}
		s.SaveJoint(nil, b)
	}
	if st.Warn {
		broadcastToClan(cl, clanSysMsg(serverpackets.SystemMessageID(residence.MsgHallLeaseUnpaid), serverpackets.NumberParam(int32(h.Data.Lease)))) //nolint:gosec // leases fit
	}
	s.scheduleLease(h, st.Wait)
}

// hallPlayerEntered is the clan hall part of Mobius EnterWorld: a member of
// a clan whose lease is unpaid is reminded; a member of a clan in a hall
// siege gets its siege state.
func (s *Server) hallPlayerEntered(p *Player) {
	cl := p.Clan()
	if cl == nil {
		return
	}
	if h := s.ClanHallByOwner(cl); h != nil && h.siege == nil && !h.paid {
		p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgHallLeaseUnpaid), serverpackets.NumberParam(int32(h.Data.Lease))) //nolint:gosec // leases fit
	}
	s.hallSiegePlayerEntered(p) // hallsiege.go
}

// hallZoneEntered is ClanHallZone.onEnter: an auctionable hall shows its
// furniture (AgitDecoInfo).
func (s *Server) hallZoneEntered(p *Player, z *data.Zone) {
	if z.Type != "ClanHallZone" {
		return
	}
	h := s.halls.byID[int32(zoneIntParam(z.Params, "clanHallId"))]
	if h == nil || h.siege != nil {
		return
	}
	p.Send(s.agitDecoPacket(h))
}

// agitDecoPacket is AgitDecoInfo of an auctionable hall.
func (s *Server) agitDecoPacket(h *ClanHall) []byte {
	lvl := func(t int) int {
		if f := h.functions[t]; f != nil {
			return f.Level
		}
		return 0
	}
	return serverpackets.NewAgitDecoInfoPacket(serverpackets.AgitDeco{HallID: h.ID, Grade: h.Grade(),
		HP: lvl(residence.HallFuncRestoreHP), MP: lvl(residence.HallFuncRestoreMP), Exp: lvl(residence.HallFuncRestoreExp),
		Teleport: lvl(residence.HallFuncTeleport), Curtains: lvl(residence.HallFuncCurtains), ItemCreate: lvl(residence.HallFuncItemCreate),
		Support: lvl(residence.HallFuncSupport), FrontPlatform: lvl(residence.HallFuncFrontPlatform)})
}

// hallRestartLocation is MapRegionData.getTeleToLocation(CLANHALL): the
// spawn of the clan's hall zone.
func (s *Server) hallRestartLocation(p *Player) (data.Location, bool) {
	h := s.ClanHallByOwner(p.Clan())
	if h == nil {
		return data.Location{}, false
	}
	return s.hallSpawnLoc(h.zone, data.ZoneSpawnNormal)
}

// hallRestart is the "to clan hall" choice of RequestRestartPoint: the
// clan's hall, with the exp restore function's percent back. It reports
// whether the player has a hall.
func (s *Server) hallRestart(p *Player) bool {
	cl := p.Clan()
	if cl == nil || cl.HideoutID == 0 {
		s.log.Warn("game: RequestRestartPoint to a clan hall without one", p.logArgs()...)
		return true
	}
	loc, ok := s.hallRestartLocation(p)
	if !ok {
		if loc, ok = s.restartLocation(p); !ok {
			return true
		}
	}
	s.RevivePlayer(p)
	if h := s.ClanHallByOwner(cl); h != nil {
		if f := h.functions[residence.HallFuncRestoreExp]; f != nil {
			s.RestoreExp(p, float64(f.Level))
		}
	}
	s.teleportAround(p, loc, maxOffsetOnTeleport)
	return true
}

// hallParam is a hall name parameter of a system message.
func hallParam(h *ClanHall) serverpackets.SystemMessageParam { return serverpackets.TextParam(h.Name) }

// hallVarsClone copies a hall script's values.
func hallVarsClone(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	return maps.Clone(m)
}

// idText formats an ID.
func idText(v int32) string { return strconv.Itoa(int(v)) }

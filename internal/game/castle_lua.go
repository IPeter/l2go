package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
)

// The castles as the castle NPC scripts see them (Plan 14 M11i,
// scripting.NpcCastle, CastleRef, PlayerClanPrivileges and
// CastleNpcActions): npc:castle() with the owner, tax, treasury, siege,
// doors and zones, player:has_clan_privilege, and the merchant window and
// map-region shout of the castle NPCs. Everything runs on the loop, while a
// handler runs.

var (
	_ scripting.NpcCastle            = npcRef{}
	_ scripting.PlayerClanPrivileges = (*Player)(nil)
	_ scripting.CastleRef            = castleScriptRef{}
	_ scripting.CastleNpcActions     = (*ScriptOutput)(nil)
)

// Mobius SevenSigns constants the castle scripts read.
const (
	luaSealStrife           = 3 // SevenSigns.SEAL_STRIFE
	luaPeriodCompetition    = 1 // SevenSigns.PERIOD_COMPETITION
	luaPeriodSealValidation = 3 // SevenSigns.PERIOD_SEAL_VALIDATION
	luaOustOffset           = 200
	luaMaxTaxPercent        = 100
)

// ScriptCastle returns the castle of the NPC (Mobius Npc.getCastle), nil
// without one.
func (r npcRef) ScriptCastle() scripting.CastleRef {
	s := r.n.srv
	if s == nil {
		return nil
	}
	c := s.npcCastle(r.n)
	if c == nil {
		return nil
	}
	return castleScriptRef{s: s, c: c}
}

// HasClanPrivilege is Mobius Player.hasAccess(ClanAccess.valueOf(name)).
func (p *Player) HasClanPrivilege(name string) (has, ok bool) {
	for a := social.AccessNone; a <= social.AccessCastleManageFunctions; a++ {
		if a.String() == name {
			return p.HasClanAccess(a), true
		}
	}
	return false, false
}

// ShowBuyWindow is Mobius Merchant.showBuyWindow(player, listId, applyTax):
// with applyTax the window shows the merchant's tax (showBuyWindow), else
// none; the purchase charges the merchant's tax either way, as Mobius
// RequestBuyItem does. Nothing happens for an NPC that is not a merchant.
func (o *ScriptOutput) ShowBuyWindow(npc scripting.NpcRef, p scripting.PlayerRef, listID int, applyTax bool) {
	s, pl, ok := o.player(p)
	n := npcOfRef(npc)
	if !ok || n == nil || !n.InWorld() || !s.npcIs(n, "Merchant") {
		return
	}
	if applyTax {
		s.showBuyWindow(pl, n, listID)
		return
	}
	l := s.buyList(listID)
	if l == nil || !l.AllowsNpc(n.Template.ID) {
		s.log.Warn("game: buylist not found for the NPC", "list", listID, "npc", n.Template.ID)
		pl.SendActionFailed()
		return
	}
	s.BlockInventory(pl)
	pl.Send(serverpackets.NewBuyListPacket(s.buyListPacket(pl, l, 0)))
	pl.SendActionFailed()
}

// NpcSayToMapRegion sends NpcSay to every player in the game in the NPC's
// map region (CastleTeleporter's loop over World.getPlayers with
// MapRegionData.getMapRegionLocId).
func (o *ScriptOutput) NpcSayToMapRegion(npc scripting.NpcRef, chatType int, text string) {
	s := o.srv.Load()
	n := npcOfRef(npc)
	if s == nil || n == nil || !n.InWorld() || s.worldData == nil {
		return
	}
	pos := n.Position()
	region := s.worldData.MapRegionLocID(int(pos.X), int(pos.Y))
	pkt := serverpackets.NewNpcSayPacket(serverpackets.NpcSay{
		ObjectID: int32(n.ID()),                    //nolint:gosec // object IDs fit in int32
		Type:     serverpackets.ChatType(chatType), //nolint:gosec // small
		NpcID:    int32(n.Template.DisplayID),      //nolint:gosec // NPC IDs fit in int32
		Text:     text,
	})
	for p := range s.Players() {
		if p.state != stateInGame {
			continue
		}
		pp := p.Position()
		if s.worldData.MapRegionLocID(int(pp.X), int(pp.Y)) == region {
			p.Send(pkt)
		}
	}
}

// castleScriptRef is a castle for the scripts.
type castleScriptRef struct {
	s *Server
	c *Castle
}

func (r castleScriptRef) CastleID() int      { return int(r.c.ID) }
func (r castleScriptRef) CastleName() string { return r.c.Name }
func (r castleScriptRef) OwnerID() int32     { return r.c.ownerID }

// OwnerClan returns the owner clan, nil without one.
func (r castleScriptRef) OwnerClan() scripting.ClanRef {
	cl := r.c.Owner()
	if cl == nil {
		return nil
	}
	return clanScriptRef{s: r.s, c: cl}
}

func (r castleScriptRef) TaxPercent() int { return r.c.TaxPercent() }

// SetTaxPercent is Castle.setTaxPercent, saved; the Lua side keeps it within
// 0-100.
func (r castleScriptRef) SetTaxPercent(p int) {
	r.c.SetTaxPercent(max(0, min(p, luaMaxTaxPercent)))
}

// MaxTax is the chamberlain's getTaxLimit for the Seal of Strife's owner.
func (r castleScriptRef) MaxTax() int { return residence.MaxTax(r.s.sevenSignsSealOwner()) }

func (r castleScriptRef) Treasury() int64 { return r.c.Treasury() }

// DepositToTreasury is the chamberlain's deposit (takeItems then
// Castle.addToTreasuryNoTax), the player and the castle saved in one
// transaction (decision X4). Deviation: Mobius takes the adena even when the
// treasury refuses it (a castle without an owner, for a GM); here nothing
// moves then.
func (r castleScriptRef) DepositToTreasury(p scripting.PlayerRef, amount int64) bool {
	c := r.c
	pl := playerOfRef(p)
	if pl == nil || amount <= 0 || pl.inv.Adena() < amount {
		return false
	}
	v, ok := residence.AddNoTax(c.ownerID > 0, c.treasury, amount)
	if !ok || !pl.TakeItems(int(adenaID), amount) {
		return false
	}
	c.treasury = v
	r.s.SaveJoint([]*Player{pl}, storage.Batch{Castles: []storage.Castle{c.document()}})
	return true
}

// WithdrawFromTreasury is the chamberlain's withdraw
// (Castle.addToTreasuryNoTax(-amount) then giveAdena), saved in one
// transaction. Deviation: Mobius gives the adena even when the treasury
// refuses (a castle without an owner); here nothing moves then.
func (r castleScriptRef) WithdrawFromTreasury(p scripting.PlayerRef, amount int64) bool {
	c := r.c
	pl := playerOfRef(p)
	if pl == nil || amount <= 0 || amount > c.treasury {
		return false
	}
	v, ok := residence.AddNoTax(c.ownerID > 0, c.treasury, -amount)
	if !ok || !pl.GiveItems(int(adenaID), amount) {
		return false
	}
	c.treasury = v
	r.s.SaveJoint([]*Player{pl}, storage.Batch{Castles: []storage.Castle{c.document()}})
	return true
}

// SiegeDate is the next siege's start in Unix seconds, 0 for none.
func (r castleScriptRef) SiegeDate() int64 {
	if d := r.c.SiegeDate(); !d.IsZero() {
		return d.Unix()
	}
	return 0
}

func (r castleScriptRef) IsSiegeInProgress() bool  { return r.c.InProgress() }
func (r castleScriptRef) IsRegistrationOver() bool { return r.c.siege.registrationOver }

// IsZoneActive is SiegeZone.isActive.
func (r castleScriptRef) IsZoneActive() bool {
	return r.c.siegeZone != nil && r.s.siegeZoneActive(r.c.siegeZone)
}

func (r castleScriptRef) ControlTowerCount() int { return r.c.siege.ctCount }

// Function is Castle.getFunction; the end date is in the server's time zone
// (Mobius Calendar).
func (r castleScriptRef) Function(kind int) (scripting.CastleFunction, bool) {
	f, ok := r.c.functions[kind]
	if !ok {
		return scripting.CastleFunction{}, false
	}
	end := r.s.Calendar().In(msTime(f.EndTime))
	out := scripting.CastleFunction{Level: f.Level, Lease: f.Lease, Rate: f.Rate, EndTime: f.EndTime}
	if f.EndTime != 0 {
		out.EndDay, out.EndMonth, out.EndYear = end.Day(), int(end.Month()), end.Year()
	}
	return out, true
}

// SealOwner is SevenSigns.getSealOwner (sevensigns_castle.go).
func (r castleScriptRef) SealOwner(seal int) int { return r.s.sevenSignsSealOwnerOf(seal) }

// SevenSignsPeriod is SevenSigns.getCurrentPeriod (sevensigns_castle.go).
func (r castleScriptRef) SevenSignsPeriod() int { return r.s.sevenSignsPeriod() }

// OpenCloseDoor is Castle.openCloseDoor: for a player of the owner clan, a
// door of the castle (Castle.getDoor) opens or closes.
func (r castleScriptRef) OpenCloseDoor(p scripting.PlayerRef, doorID int, open bool) {
	pl := playerOfRef(p)
	if pl == nil || pl.ClanID() != r.c.ownerID || doorID <= 0 {
		return
	}
	d, ok := r.s.Door(doorID)
	if !ok || !r.c.Contains(d.Position()) {
		return
	}
	if open {
		r.s.openDoor(d)
	} else {
		r.s.closeDoor(d)
	}
}

// BanishForeigners is ResidenceZone.banishForeigners: everyone in the castle
// zone who is not of the owner clan goes to its banish point (the normal
// one without), with a random offset.
func (r castleScriptRef) BanishForeigners() {
	c, s := r.c, r.s
	if c.castleZone == nil {
		return
	}
	loc, ok := c.spawnLoc(data.ZoneSpawnBanish)
	if !ok {
		if loc, ok = c.spawnLoc(data.ZoneSpawnNormal); !ok {
			return
		}
	}
	for _, p := range s.luaPlayersInZone(c.castleZone) {
		if p.ClanID() != c.ownerID {
			s.teleportAround(p, loc, maxOffsetOnTeleport)
		}
	}
}

// OustAllPlayers is ResidenceTeleportZone.oustAllPlayers: everyone in the
// castle's teleport zone goes to its spawn, 200 around.
func (r castleScriptRef) OustAllPlayers() {
	c, s := r.c, r.s
	z := c.teleZone
	if z == nil {
		return
	}
	var pts []data.Location
	for _, sp := range z.Spawns {
		if sp.Type == data.ZoneSpawnNormal {
			pts = append(pts, data.Location{X: sp.X, Y: sp.Y, Z: sp.Z})
		}
	}
	if len(pts) == 0 {
		return
	}
	for _, p := range s.luaPlayersInZone(z) {
		loc := pts[0]
		if s.combatSettings().RandomRespawnInTown {
			loc = pts[s.rnd.Get(len(pts))]
		}
		s.teleportAround(p, loc, luaOustOffset)
	}
}

// ShowSiegeInfo is Siege.listRegisterClan: the SiegeInfo window.
func (r castleScriptRef) ShowSiegeInfo(p scripting.PlayerRef) {
	if pl := playerOfRef(p); pl != nil {
		r.s.ShowSiegeInfo(pl, r.c)
	}
}

// luaPlayersInZone returns the players in the game inside z.
func (s *Server) luaPlayersInZone(z *data.Zone) []*Player {
	var out []*Player
	for p := range s.Players() {
		if p.state != stateInGame || !p.InWorld() {
			continue
		}
		pos := p.Position()
		if z.Shape.Contains(int(pos.X), int(pos.Y), int(pos.Z)) {
			out = append(out, p)
		}
	}
	return out
}

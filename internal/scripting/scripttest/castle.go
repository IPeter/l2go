package scripttest

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/social"
)

// Castle is a fake castle, npc:castle() of the NPCs whose Castle it is.
// Change its fields between calls; the scripts' changes land in them.
type Castle struct {
	ID   int
	Name string
	// Owner is the owner clan, nil for none.
	Owner      *Clan
	TaxPercent int
	Treasury   int64
	// SiegeDate is the next siege's start; zero for none.
	SiegeDate                    time.Time
	InProgress, RegistrationOver bool
	// ZoneActive is castle:is_zone_active().
	ZoneActive    bool
	ControlTowers int
	// Functions are the rented castle functions by type.
	Functions map[int]scripting.CastleFunction
	// StrifeOwner is the Seal of Strife's owner (0 none, 1 Dusk, 2 Dawn);
	// Period the Seven Signs period (3 is seal validation, what the server
	// has until Seven Signs exist).
	StrifeOwner, Period int
	// Doors are the castle's doors by door ID: true open.
	Doors map[int]bool
	// DoorUpgrades are the HP ratios set_door_upgrade gave the doors
	// (castle_doors.go); a door of Doors without one has ratio 1.
	DoorUpgrades map[int]int

	// What the scripts did: Banished and Ousted count banish_foreigners()
	// and oust_all_players(); SiegeInfo lists the players
	// show_siege_info(player) showed the window to.
	Banished, Ousted int
	SiegeInfo        []int32
}

var (
	_ scripting.NpcCastle            = npcRef{}
	_ scripting.PlayerClanPrivileges = playerRef{}
	_ scripting.CastleRef            = castleRef{}
	_ scripting.CastleNpcActions     = (*World)(nil)
)

// ScriptCastle returns the NPC's Castle.
func (r npcRef) ScriptCastle() scripting.CastleRef {
	if r.n.Castle == nil {
		return nil
	}
	return castleRef{r.n.Castle}
}

// HasClanPrivilege reports whether the name is in ClanPrivileges, true for
// every privilege of a clan leader; ok is false for a name Mobius's
// ClanAccess does not have.
func (r playerRef) HasClanPrivilege(name string) (has, ok bool) {
	for a := social.AccessNone; a <= social.AccessCastleManageFunctions; a++ {
		if a.String() == name {
			return r.IsClanLeader() || slices.Contains(r.p.ClanPrivileges, name), true
		}
	}
	return false, false
}

// ShowBuyWindow records a "buy_window" action: Object the NPC, Target the
// player, Value the list, Level 1 when the window shows the tax.
func (w *World) ShowBuyWindow(npc scripting.NpcRef, p scripting.PlayerRef, listID int, applyTax bool) {
	tax := 0
	if applyTax {
		tax = 1
	}
	w.Actions = append(w.Actions, Action{Kind: "buy_window", Object: npc.ObjectID(), Target: p.ObjectID(), Value: listID, Level: tax})
}

// NpcSayToMapRegion records a "region_say" action: Object the NPC, Value
// the chat type, Text the text.
func (w *World) NpcSayToMapRegion(npc scripting.NpcRef, chatType int, text string) {
	w.Actions = append(w.Actions, Action{Kind: "region_say", Object: npc.ObjectID(), Value: chatType, Text: text})
}

type castleRef struct{ c *Castle }

func (r castleRef) CastleID() int      { return r.c.ID }
func (r castleRef) CastleName() string { return r.c.Name }

func (r castleRef) OwnerID() int32 {
	if r.c.Owner == nil {
		return 0
	}
	return r.c.Owner.ID
}

func (r castleRef) OwnerClan() scripting.ClanRef {
	if r.c.Owner == nil {
		return nil
	}
	return clanRef{r.c.Owner}
}

func (r castleRef) TaxPercent() int     { return r.c.TaxPercent }
func (r castleRef) SetTaxPercent(p int) { r.c.TaxPercent = p }
func (r castleRef) MaxTax() int         { return residence.MaxTax(r.c.StrifeOwner) }
func (r castleRef) Treasury() int64     { return r.c.Treasury }

// DepositToTreasury moves adena (item 57) from the fake player to Treasury
// when the castle has an owner.
func (r castleRef) DepositToTreasury(p scripting.PlayerRef, amount int64) bool {
	pr, ok := p.(playerRef)
	if !ok || amount <= 0 || r.c.Owner == nil || pr.p.Items[57] < amount {
		return false
	}
	pr.TakeItems(57, amount)
	r.c.Treasury += amount
	return true
}

// WithdrawFromTreasury moves adena from Treasury to the fake player when
// the castle has an owner and enough.
func (r castleRef) WithdrawFromTreasury(p scripting.PlayerRef, amount int64) bool {
	pr, ok := p.(playerRef)
	if !ok || amount <= 0 || r.c.Owner == nil || r.c.Treasury < amount {
		return false
	}
	r.c.Treasury -= amount
	pr.p.addItem(57, amount)
	return true
}

func (r castleRef) SiegeDate() int64 {
	if r.c.SiegeDate.IsZero() {
		return 0
	}
	return r.c.SiegeDate.Unix()
}

func (r castleRef) IsSiegeInProgress() bool  { return r.c.InProgress }
func (r castleRef) IsRegistrationOver() bool { return r.c.RegistrationOver }
func (r castleRef) IsZoneActive() bool       { return r.c.ZoneActive }
func (r castleRef) ControlTowerCount() int   { return r.c.ControlTowers }

func (r castleRef) Function(kind int) (scripting.CastleFunction, bool) {
	f, ok := r.c.Functions[kind]
	return f, ok
}

// SealOwner is StrifeOwner for the Seal of Strife (3), else 0.
func (r castleRef) SealOwner(seal int) int {
	if seal == 3 {
		return r.c.StrifeOwner
	}
	return 0
}

func (r castleRef) SevenSignsPeriod() int { return r.c.Period }

// OpenCloseDoor changes a door in Doors when the player's clan owns the
// castle.
func (r castleRef) OpenCloseDoor(p scripting.PlayerRef, doorID int, open bool) {
	pr, ok := p.(playerRef)
	if !ok || pr.p.Clan == nil || pr.p.Clan != r.c.Owner {
		return
	}
	if _, ok := r.c.Doors[doorID]; ok {
		r.c.Doors[doorID] = open
	}
}

func (r castleRef) BanishForeigners() { r.c.Banished++ }
func (r castleRef) OustAllPlayers()   { r.c.Ousted++ }

func (r castleRef) ShowSiegeInfo(p scripting.PlayerRef) {
	r.c.SiegeInfo = append(r.c.SiegeInfo, p.ObjectID())
}

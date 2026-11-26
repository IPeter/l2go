package game

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// The castle as the castle NPC scripts see it (castle_lua.go): an NPC near
// Gludio belongs to it, the lord's tax and vault, the privileges and the
// siege window.
func TestCastleScriptRef(t *testing.T) {
	e := newSiegeEnv(t, nil, siegeTestStart)
	c := e.gludio()
	sess, p, clan := e.siegeLeader("alice", "Alice", "Lions")
	art := npcSpawnedOne(t, e, siegeTestArtefact)
	firstSiege := time.Date(2026, 12, 6, 16, 0, 0, 0, time.UTC)
	e.onLoop(func() {
		e.srv.setCastleOwner(c, clan)
		ref := npcRef{art}.ScriptCastle()
		if ref == nil {
			t.Fatal("the artefact has no castle")
		}
		if ref.CastleID() != 1 || ref.CastleName() != "Gludio" || ref.OwnerID() != clan.ID || ref.OwnerClan().ClanName() != "Lions" {
			t.Errorf("castle %d %q owned by %d", ref.CastleID(), ref.CastleName(), ref.OwnerID())
		}
		if has, ok := p.HasClanPrivilege("CASTLE_VAULT"); !has || !ok {
			t.Errorf("the leader lacks CASTLE_VAULT: %v %v", has, ok)
		}
		if _, ok := p.HasClanPrivilege("CASTLE_NOTHING"); ok {
			t.Error("an unknown privilege is known")
		}
		// A new Seven Signs state: the competition, no seal owner.
		if ref.MaxTax() != 15 || ref.SealOwner(3) != 0 || ref.SevenSignsPeriod() != 1 {
			t.Errorf("max tax %d, strife owner %d, period %d", ref.MaxTax(), ref.SealOwner(3), ref.SevenSignsPeriod())
		}
		if ref.SiegeDate() != firstSiege.Unix() || ref.IsSiegeInProgress() || ref.IsRegistrationOver() || ref.IsZoneActive() {
			t.Errorf("siege date %d, in progress %v", ref.SiegeDate(), ref.IsSiegeInProgress())
		}
		ref.SetTaxPercent(12)
		if c.TaxPercent() != 12 || ref.TaxPercent() != 12 {
			t.Errorf("tax %d after setting 12", c.TaxPercent())
		}

		// The vault moves adena both ways, never more than there is.
		if !p.GiveItems(adenaID, 1000) {
			t.Fatal("no adena given")
		}
		if !ref.DepositToTreasury(p, 600) || c.Treasury() != 600 || p.Adena() != 400 {
			t.Errorf("deposit: treasury %d, adena %d", c.Treasury(), p.Adena())
		}
		if ref.DepositToTreasury(p, 600) || c.Treasury() != 600 {
			t.Errorf("a deposit beyond the adena: treasury %d", c.Treasury())
		}
		if !ref.WithdrawFromTreasury(p, 100) || c.Treasury() != 500 || p.Adena() != 500 {
			t.Errorf("withdraw: treasury %d, adena %d", c.Treasury(), p.Adena())
		}
		if ref.WithdrawFromTreasury(p, 501) || c.Treasury() != 500 || p.Adena() != 500 {
			t.Errorf("a withdrawal beyond the treasury: treasury %d, adena %d", c.Treasury(), p.Adena())
		}
		if _, ok := ref.Function(2); ok {
			t.Error("a castle function without renting one")
		}
	})
	sess.take(t)
	e.onLoop(func() { npcRef{art}.ScriptCastle().ShowSiegeInfo(p) })
	if info := only[testclient.SiegeInfo](sess.take(t)); len(info) != 1 || info[0].SiegeDate != int32(firstSiege.Unix()) {
		t.Errorf("siege window %+v", info)
	}
}

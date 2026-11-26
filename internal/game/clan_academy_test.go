package game

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/testclient"
)

// Class IDs of the graduation tests.
// The members start as Human Fighters (class 0).
const (
	classGladiator = 2 // second class: THIRD_CLASS_GROUP
	classKnight    = 4 // first class (Human Fighter → Human Knight)
)

// Academy graduation: an academy member who changes into a second-transfer
// class earns its clan 650 reputation if it joined at level 16 or below,
// 190 from level 39, 650 − (level − 16) × 20 between; it leaves the clan
// without a join penalty, the members see it expelled, and it gets the
// Academy Circlet. The clan and the graduate are stored.
func TestClanAcademyGraduation(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	c := e.foundClan(a, "Dragons")
	e.onLoop(func() { c.Level = 5 })

	for i, tc := range []struct {
		level int32
		gain  int
	}{
		{15, 650}, {16, 650}, {17, 630}, {25, 470}, {38, 210}, {39, 190}, {40, 190},
	} {
		account := "pupil" + string(rune('a'+i))
		sess, p := e.enterClanChar(clanChar{account: account, name: "Pupil" + string(rune('A'+i)), level: tc.level})
		e.inviteToClan(sessA, a, sess, p, social.UnitAcademy, 1)
		var before int
		e.onLoop(func() { before = c.Reputation })
		sessA.take(t)
		sess.take(t)
		e.onLoop(func() { e.srv.ChangeClass(p, classGladiator) })
		got := sess.take(t)
		clanWantMsg(t, "graduate", got, int32(academyMsgGraduated))
		clanWantMsg(t, "graduate sees the expulsion", got, int32(clanMsgMemberExpelled))
		leader := sessA.take(t)
		clanWantMsg(t, "leader sees the expulsion", leader, int32(clanMsgMemberExpelled))
		if dels := only[testclient.PledgeShowMemberListDelete](leader); len(dels) != 1 || dels[0].Name != p.Name() {
			t.Errorf("level %d: PledgeShowMemberListDelete %+v", tc.level, dels)
		}
		e.onLoop(func() {
			if gain := c.Reputation - before; gain != tc.gain {
				t.Errorf("joined at %d: reputation +%d, want +%d", tc.level, gain, tc.gain)
			}
			if p.Clan() != nil || p.char.ClanID != 0 || p.char.LvlJoinedAcademy != 0 || p.char.ClanJoinExpiry != 0 || c.Member(p.ObjectID()) != nil {
				t.Errorf("joined at %d: still in the clan or penalized: clan %d joined %d expiry %d", tc.level, p.char.ClanID, p.char.LvlJoinedAcademy, p.char.ClanJoinExpiry)
			}
			if p.char.ClassID != classGladiator {
				t.Errorf("class %d, want the Gladiator", p.char.ClassID)
			}
		})
		if n := e.itemCount(p, academyCirclet); n != 1 {
			t.Errorf("joined at %d: %d circlets", tc.level, n)
		}
		e.advance(inventoryUpdateDelay)
		if ius := only[testclient.InventoryUpdate](sess.take(t)); len(ius) == 0 {
			t.Errorf("joined at %d: no InventoryUpdate for the circlet", tc.level)
		}
		if ch := e.storedCharacter(p.ObjectID()); ch.ClanID != 0 || ch.LvlJoinedAcademy != 0 || ch.PledgeType != 0 || ch.ClanJoinExpiry != 0 {
			t.Errorf("stored graduate clan %d joined %d unit %d expiry %d", ch.ClanID, ch.LvlJoinedAcademy, ch.PledgeType, ch.ClanJoinExpiry)
		}
	}
	doc, err := e.storedClan(c.ID)
	if want := 650 + 650 + 630 + 470 + 210 + 190 + 190; err != nil || doc.Reputation != int32(want) {
		t.Errorf("stored reputation %d, %v; want %d", doc.Reputation, err, want)
	}
}

// No graduation: a first-class change of an academy member, and a
// second-class change of a member of the main clan.
func TestClanAcademyNoGraduation(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessP, pupil := e.enterClanChar(clanChar{account: "pupil", name: "Pupil", level: 20})
	sessM, member := e.enterClanChar(clanChar{account: "member", name: "Member", level: 39})
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessP, pupil, social.UnitAcademy, 1)
	e.inviteToClan(sessA, a, sessM, member, social.UnitMain, 1)

	e.onLoop(func() {
		e.srv.ChangeClass(pupil, classKnight)
		e.srv.ChangeClass(member, classGladiator)
		if pupil.Clan() != c || !pupil.IsAcademyMember() || member.Clan() != c || c.Reputation != 0 {
			t.Errorf("pupil clan %v academy %v, member clan %v, reputation %d", pupil.Clan(), pupil.IsAcademyMember(), member.Clan(), c.Reputation)
		}
	})
	for _, id := range sysMsgs(sessP.take(t)) {
		if id == int32(academyMsgGraduated) {
			t.Error("the first class change graduated")
		}
	}
	if slices.Contains(sysMsgs(sessM.take(t)), int32(academyMsgGraduated)) || e.itemCount(member, academyCirclet) != 0 {
		t.Error("a main clan member graduated")
	}
}

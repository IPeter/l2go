package game

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// flushInfo sends the coalesced UserInfo and CharInfo now.
func (e *testEnv) flushInfo() { e.onLoop(e.srv.world.FlushUpdates) }

// clanExists reports, on the loop, whether the clan is in the registry.
func (e *testEnv) clanExists(id int32) bool {
	var ok bool
	e.onLoop(func() { ok = e.srv.Clan(id) != nil })
	return ok
}

// containsInOrder reports whether want appears in names in order.
func containsInOrder(names []string, want ...string) bool {
	i := 0
	for _, n := range names {
		if i < len(want) && n == want[i] {
			i++
		}
	}
	return i == len(want)
}

// A character founds a clan at a village master: level 10, a valid and
// unique name, no clan, no create penalty. The leader gets Mobius's
// packets, every privilege, grade 1 and pledge class 1; the clan and the
// leader are stored.
func TestClanCreateAtVillageMaster(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sess, p := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 9})
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))

	clanWantMsg(t, "level 9", e.vmCommand(sess, p, vm, "create_clan Dragons"), int32(clanMsgCreateCriteria))
	e.onLoop(func() { e.setPlayerLevel(p, 10) })
	for _, tc := range []struct {
		cmd  string
		want int32
	}{
		{"create_clan", int32(clanMsgCreateClanName)},
		{"create_clan Two Words", int32(clanMsgNameInvalid)},
		{"create_clan a", int32(clanMsgNameInvalid)},
		{"create_clan Bad-Name", int32(clanMsgNameInvalid)},
		{"create_clan Abcdefghijklmnopq", int32(clanMsgNameLength)},
	} {
		clanWantMsg(t, tc.cmd, e.vmCommand(sess, p, vm, tc.cmd), tc.want)
	}

	ps := e.vmCommand(sess, p, vm, "create_clan Dragons")
	if names := testclient.Names(ps); !containsInOrder(names, "PledgeShowInfoUpdate", "PledgeShowMemberListAll", "PledgeShowMemberListUpdate", "SystemMessage") {
		t.Fatalf("creation packets %q", names)
	}
	clanWantMsg(t, "created", ps, int32(clanMsgClanCreated))
	all := only[testclient.PledgeShowMemberListAll](ps)[0]
	if all.ClanName != "Dragons" || all.LeaderName != "Leader" || len(all.Members) != 1 || all.Members[0].ObjectID != p.ObjectID() {
		t.Errorf("clan window %+v", all)
	}
	var c *Clan
	e.onLoop(func() {
		c = p.Clan()
		if c == nil || c.LeaderID != p.ObjectID() || c.Level != 0 || p.ClanPrivileges() != social.AllPrivileges || p.PowerGrade() != 1 || p.PledgeClass() != 1 {
			t.Errorf("after creation: clan %+v privileges %x grade %d class %d", c, p.ClanPrivileges(), p.PowerGrade(), p.PledgeClass())
		}
	})
	sess.take(t)
	e.flushInfo()
	ui := clanUserInfo(t, sess.take(t))
	if ui.ClanID != c.ID || ui.Relation != 0x40 || ui.ClanPrivileges != int32(social.AllPrivileges) || ui.PledgeClass != 1 {
		t.Errorf("UserInfo clan id %d relation %#x privileges %#x class %d", ui.ClanID, ui.Relation, ui.ClanPrivileges, ui.PledgeClass)
	}
	doc, err := e.storedClan(c.ID)
	if err != nil || doc.Name != "Dragons" || doc.LeaderID != p.ObjectID() || len(doc.Ranks) != social.Ranks {
		t.Errorf("stored clan %+v, %v", doc, err)
	}
	if ch := e.storedCharacter(p.ObjectID()); ch.ClanID != c.ID || ch.PowerGrade != 1 {
		t.Errorf("stored leader clan %d grade %d", ch.ClanID, ch.PowerGrade)
	}

	clanWantMsg(t, "second clan", e.vmCommand(sess, p, vm, "create_clan Other"), int32(clanMsgFailedToCreate))
	sess2, q := e.enterClanChar(clanChar{account: "other", name: "Other", level: 10})
	ps = e.vmCommand(sess2, q, vm, "create_clan dragons")
	clanWantMsg(t, "taken name", ps, int32(clanMsgS1AlreadyExists))
	e.onLoop(func() { q.char.ClanCreateExpiry = e.srv.Now().Add(time.Hour).UnixMilli() })
	clanWantMsg(t, "create penalty", e.vmCommand(sess2, q, vm, "create_clan Tigers"), int32(clanMsgWait10Days))
}

// Every level-up step: 0→1 and 1→2 cost SP and adena, 2→3, 3→4 and 4→5 SP
// and an item, 5→6, 6→7 and 7→8 reputation with enough members; level 8 is
// the last. Success shows skill 5103 on the leader.
func TestClanLevelUpSteps(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, func(gs *config.GameServer) {
		gs.Clan.Level6Members, gs.Clan.Level7Members, gs.Clan.Level8Members = 1, 2, 2
	})
	sess, p := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	c := e.foundClan(p, "Dragons")

	clanWantMsg(t, "no SP", e.vmCommand(sess, p, vm, "increase_clan_level"), int32(clanMsgLevelUpConditions))
	level := func() int {
		var l int
		e.onLoop(func() { l = c.Level })
		return l
	}
	steps := []struct {
		sp    int32
		adena int64
		item  int32
		rep   int
		msgs  []int32
	}{
		{sp: 20000, adena: 650000, msgs: []int32{int32(sysMsgAdenaDisappeared), int32(clanMsgSPDecreased)}},
		{sp: 100000, adena: 2500000, msgs: []int32{int32(sysMsgAdenaDisappeared), int32(clanMsgSPDecreased)}},
		{sp: 350000, item: social.ItemBloodMark, msgs: []int32{int32(clanMsgSPDecreased), int32(clanMsgS1HasDisappeared)}},
		{sp: 1000000, item: social.ItemAllianceManifesto, msgs: []int32{int32(clanMsgSPDecreased), int32(clanMsgS1HasDisappeared)}},
		{sp: 2500000, item: social.ItemSealOfAspiration, msgs: []int32{int32(clanMsgSPDecreased), int32(clanMsgS1HasDisappeared), int32(clanMsgReputationAbove5)}},
		{rep: 5000, msgs: []int32{int32(clanMsgReputationDeducted)}},
		{rep: 10000, msgs: []int32{int32(clanMsgReputationDeducted)}},
		{rep: 20000, msgs: []int32{int32(clanMsgReputationDeducted)}},
	}
	for i, st := range steps {
		from := i
		// One short first: the conditions are not met.
		e.onLoop(func() {
			p.char.SP = st.sp - 1
			if st.adena > 0 {
				e.srv.AddItem(p, adenaID, st.adena, -1, false, false)
			}
			if st.item != 0 {
				e.srv.AddItem(p, st.item, 1, -1, false, false)
			}
			c.Reputation = st.rep - 1
			if st.sp == 0 {
				p.char.SP = 0
			}
		})
		if st.sp > 0 || st.rep > 0 {
			clanWantMsg(t, "short", e.vmCommand(sess, p, vm, "increase_clan_level"), int32(clanMsgLevelUpConditions))
		}
		if from >= 6 {
			// Levels 7 and 8 need two members.
			e.onLoop(func() {
				if c.MemberCount() < 2 {
					c.addMember(&ClanMember{data: storage.ClanMember{ObjectID: 999999, Name: "Ghost", Level: 20}})
				}
			})
		}
		e.onLoop(func() {
			p.char.SP = st.sp
			c.Reputation = st.rep
		})
		ps := e.vmCommand(sess, p, vm, "increase_clan_level")
		if got := level(); got != from+1 {
			t.Fatalf("step %d→%d: level %d; messages %v", from, from+1, got, sysMsgs(ps))
		}
		for _, m := range st.msgs {
			clanWantMsg(t, "step", ps, m)
		}
		clanWantMsg(t, "step", ps, int32(clanMsgSkillLevelIncreased))
		uses := only[testclient.MagicSkillUse](ps)
		if len(uses) != 1 || uses[0].SkillID != classChangeSkill {
			t.Errorf("step %d: MagicSkillUse %+v", from, uses)
		}
		if infos := only[testclient.PledgeShowInfoUpdate](ps); len(infos) == 0 || infos[len(infos)-1].Level != int32(from+1) {
			t.Errorf("step %d: PledgeShowInfoUpdate %+v", from, infos)
		}
		var sp int32
		var adena int64
		e.onLoop(func() { sp, adena = p.char.SP, p.inv.Adena() })
		if sp != 0 || adena != 0 {
			t.Errorf("step %d: SP %d adena %d left", from, sp, adena)
		}
		if st.item != 0 && e.itemCount(p, st.item) != 0 {
			t.Errorf("step %d: item %d left", from, st.item)
		}
	}
	ps := e.vmCommand(sess, p, vm, "increase_clan_level")
	if level() != social.MaxClanLevel || clanHasMsg(ps, int32(clanMsgLevelUpConditions)) {
		t.Errorf("level 8 stays without a message: level %d, %v", level(), sysMsgs(ps))
	}
	if doc, err := e.storedClan(c.ID); err != nil || doc.Level != social.MaxClanLevel {
		t.Errorf("stored clan level %d, %v", doc.Level, err)
	}
}

// A member who is not the leader, and a dissolving clan, cannot level up.
func TestClanLevelUpRefused(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sess, p := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40, sp: 20000, adena: 650000})
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	c := e.foundClan(p, "Dragons")
	e.onLoop(func() { c.DissolvingExpiry = e.srv.Now().Add(time.Hour).UnixMilli() })
	clanWantMsg(t, "dissolving", e.vmCommand(sess, p, vm, "increase_clan_level"), int32(clanMsgNoLevelUpDissolving))
	sess2, q := e.enterClanChar(clanChar{account: "memb", name: "Member", level: 20})
	e.inviteToClan(sess, p, sess2, q, 0, 1)
	clanWantMsg(t, "member", e.vmCommand(sess2, q, vm, "increase_clan_level"), int32(clanMsgNotAuthorized))
}

// Invitation, withdrawal and dismissal with their penalties, online and
// offline, the join conditions and the member limits.
func TestClanMembersAndPenalties(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	sessC, cc := e.enterClanChar(clanChar{account: "carl", name: "Carl", level: 20})
	c := e.foundClan(a, "Dragons")
	sessA.take(t)

	// B declines, then accepts.
	ps := e.inviteToClan(sessA, a, sessB, b, 0, 0)
	clanWantMsg(t, "declined", ps, int32(clanMsgYouDidNotRespond))
	clanWantMsg(t, "declined (inviter)", sessA.take(t), int32(clanMsgS1DidNotRespond))
	ps = e.inviteToClan(sessA, a, sessB, b, 0, 1)
	if names := testclient.Names(ps); !containsInOrder(names, "JoinPledge", "PledgeShowMemberListUpdate", "PledgeSkillList", "SystemMessage", "PledgeShowInfoUpdate", "PledgeShowMemberListAll") {
		t.Errorf("join packets %q", names)
	}
	clanWantMsg(t, "joined", ps, int32(clanMsgEnteredClan))
	psA := sessA.take(t)
	clanWantMsg(t, "joined (leader)", psA, int32(clanMsgS1JoinedClan))
	if adds := only[testclient.PledgeShowMemberListAdd](psA); len(adds) != 1 || adds[0].Name != "Bob" {
		t.Errorf("leader's PledgeShowMemberListAdd %+v", adds)
	}
	e.onLoop(func() {
		if b.Clan() != c || b.PowerGrade() != social.RankNewMember || c.MemberCount() != 2 {
			t.Errorf("Bob: clan %v grade %d members %d", b.Clan(), b.PowerGrade(), c.MemberCount())
		}
	})

	// The conditions.
	e.onLoop(func() { b.char.SP = 0 })
	ps = e.clanSend(sessB, clientpackets.OpRequestJoinPledge, clanInts(cc.ObjectID(), 0))
	clanWantMsg(t, "no privilege", ps, int32(clanMsgNotAuthorized))
	ps = e.clanSend(sessA, clientpackets.OpRequestJoinPledge, clanInts(a.ObjectID(), 0))
	clanWantMsg(t, "self", ps, int32(clanMsgCannotAskYourself))
	ps = e.clanSend(sessA, clientpackets.OpRequestJoinPledge, clanInts(b.ObjectID(), 0))
	clanWantMsg(t, "already a member", ps, int32(clanMsgS1WorkingWithAnotherClan))
	ps = e.clanSend(sessA, clientpackets.OpRequestJoinPledge, clanInts(123456789, 0))
	clanWantMsg(t, "nobody", ps, int32(clanMsgWrongTarget))

	// B withdraws: the join penalty for a day.
	ps = e.clanSend(sessB, clientpackets.OpRequestWithdrawalPledge, nil)
	if names := testclient.Names(ps); !slices.Contains(names, "PledgeShowMemberListDeleteAll") {
		t.Errorf("withdrawal packets %q", names)
	}
	clanWantMsg(t, "withdrew", ps, int32(clanMsgYouWithdrew))
	clanWantMsg(t, "withdrew", ps, int32(clanMsgWaitAfterLeaving))
	psA = sessA.take(t)
	clanWantMsg(t, "withdrew (leader)", psA, int32(clanMsgS1Withdrew))
	if dels := only[testclient.PledgeShowMemberListDelete](psA); len(dels) != 1 || dels[0].Name != "Bob" {
		t.Errorf("leader's delete %+v", dels)
	}
	day := social.DayMillis
	e.onLoop(func() {
		if b.Clan() != nil || b.char.ClanID != 0 || b.char.ClanJoinExpiry != e.srv.Now().UnixMilli()+day {
			t.Errorf("Bob after leaving: clan %v id %d expiry %d", b.Clan(), b.char.ClanID, b.char.ClanJoinExpiry)
		}
	})
	ps = e.clanSend(sessA, clientpackets.OpRequestJoinPledge, clanInts(b.ObjectID(), 0))
	clanWantMsg(t, "join penalty", ps, int32(clanMsgS1JoinPenalty))
	if ch := e.storedCharacter(b.ObjectID()); ch.ClanID != 0 || ch.ClanJoinExpiry == 0 {
		t.Errorf("stored Bob clan %d expiry %d", ch.ClanID, ch.ClanJoinExpiry)
	}
	e.advance(24*time.Hour + time.Second)
	e.inviteToClan(sessA, a, sessB, b, 0, 1)

	// The leader cannot leave.
	clanWantMsg(t, "leader leaves", e.clanSend(sessA, clientpackets.OpRequestWithdrawalPledge, nil), int32(clanMsgLeaderCannotWithdraw))

	// C joins and is dismissed online: C's join penalty and the clan's.
	e.inviteToClan(sessA, a, sessC, cc, 0, 1)
	sessC.take(t)
	clanWantMsg(t, "dismiss self", e.clanSend(sessA, clientpackets.OpRequestOustPledgeMember, clanStrings("leader")), int32(clanMsgCannotDismissYourself))
	clanWantMsg(t, "member dismisses", e.clanSend(sessB, clientpackets.OpRequestOustPledgeMember, clanStrings("Carl")), int32(clanMsgNotAuthorized))
	ps = e.clanSend(sessA, clientpackets.OpRequestOustPledgeMember, clanStrings("Carl"))
	clanWantMsg(t, "dismissed (leader)", ps, int32(clanMsgExpelSucceeded))
	clanWantMsg(t, "dismissed (leader)", ps, int32(clanMsgClanWaitAfterDismiss))
	psC := sessC.take(t)
	clanWantMsg(t, "dismissed", psC, int32(clanMsgRecentlyDismissed))
	e.onLoop(func() {
		if cc.Clan() != nil || cc.char.ClanJoinExpiry == 0 || c.MemberPenaltyExpiry != cc.char.ClanJoinExpiry {
			t.Errorf("Carl after dismissal: clan %v expiry %d clan penalty %d", cc.Clan(), cc.char.ClanJoinExpiry, c.MemberPenaltyExpiry)
		}
	})
	sessD, d := e.enterClanChar(clanChar{account: "dave", name: "Dave", level: 20})
	ps = e.clanSend(sessA, clientpackets.OpRequestJoinPledge, clanInts(d.ObjectID(), 0))
	clanWantMsg(t, "clan penalty", ps, int32(clanMsgClanWaitAfterDismiss))
	if doc, err := e.storedClan(c.ID); err != nil || doc.MemberPenaltyExpiry == 0 {
		t.Errorf("stored clan penalty %d, %v", doc.MemberPenaltyExpiry, err)
	}

	// After a day D joins, logs out and is dismissed offline: a patch.
	e.advance(24*time.Hour + time.Second)
	e.inviteToClan(sessA, a, sessD, d, 0, 1)
	e.clanLogout(sessD, d)
	sessA.take(t)
	ps = e.clanSend(sessA, clientpackets.OpRequestOustPledgeMember, clanStrings("Dave"))
	clanWantMsg(t, "offline dismissal", ps, int32(clanMsgExpelSucceeded))
	if ch := e.storedCharacter(d.ObjectID()); ch.ClanID != 0 || ch.PowerGrade != 0 || ch.ClanJoinExpiry == 0 {
		t.Errorf("stored Dave after the offline dismissal: clan %d grade %d expiry %d", ch.ClanID, ch.PowerGrade, ch.ClanJoinExpiry)
	}
	_, d = e.clanLogin("dave")
	e.onLoop(func() {
		if d.Clan() != nil {
			t.Errorf("Dave is still in %v", d.Clan())
		}
	})
}

// The member limits: 10 in the main clan at level 0; the academy takes
// members of level 40 or below before their second class, with grade 9.
func TestClanLimitsAndAcademy(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	_, old := e.enterClanChar(clanChar{account: "old", name: "Old", level: 41})
	c := e.foundClan(a, "Dragons")
	e.onLoop(func() {
		for i := range 9 {
			c.addMember(&ClanMember{data: storage.ClanMember{ObjectID: int32(900000 + i), Name: "Ghost" + string(rune('a'+i))}})
		}
	})
	ps := e.clanSend(sessA, clientpackets.OpRequestJoinPledge, clanInts(b.ObjectID(), 0))
	clanWantMsg(t, "full", ps, int32(clanMsgS1IsFull))
	ps = e.clanSend(sessA, clientpackets.OpRequestJoinPledge, clanInts(old.ObjectID(), social.UnitAcademy))
	clanWantMsg(t, "too old for the academy", ps, int32(clanMsgS1NotForAcademy))
	clanWantMsg(t, "too old for the academy", ps, int32(clanMsgAcademyRequirements))
	e.inviteToClan(sessA, a, sessB, b, social.UnitAcademy, 1)
	e.onLoop(func() {
		if b.PowerGrade() != social.RankAcademy || b.PledgeType() != social.UnitAcademy || b.char.LvlJoinedAcademy != 20 || !b.IsAcademyMember() {
			t.Errorf("academy member grade %d unit %d joined %d", b.PowerGrade(), b.PledgeType(), b.char.LvlJoinedAcademy)
		}
	})
	// Leaving the academy brings no penalty.
	e.clanSend(sessB, clientpackets.OpRequestWithdrawalPledge, nil)
	e.onLoop(func() {
		if b.char.ClanJoinExpiry != 0 {
			t.Errorf("academy leaver got the join penalty %d", b.char.ClanJoinExpiry)
		}
	})
}

// Ranks and privileges: the leader sets rank privileges (rank 9 keeps only
// its three bits), members of the rank get them; grades, the grade list,
// member info, and titles from clan level 3.
func TestClanRanksAndTitles(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, 0, 1)

	privs := social.Privileges(0).With(social.AccessInviteMember, social.AccessAssignTitle)
	e.clanSend(sessB, clientpackets.OpRequestPledgePower, clanInts(5, clientpackets.PledgePowerSet, int32(privs)))
	e.onLoop(func() {
		if c.Ranks.Of(5) != 0 {
			t.Errorf("a member set rank privileges %x", c.Ranks.Of(5))
		}
	})
	e.clanSend(sessA, clientpackets.OpRequestPledgePower, clanInts(5, clientpackets.PledgePowerSet, int32(privs)))
	e.flushInfo()
	psB := sessB.take(t)
	if ui := clanUserInfo(t, psB); ui.ClanPrivileges != int32(privs) {
		t.Errorf("Bob's privileges %#x, want %#x", ui.ClanPrivileges, privs)
	}
	ps := e.clanSend(sessB, clientpackets.OpRequestPledgePower, clanInts(5, clientpackets.PledgePowerShow))
	if m := only[testclient.ManagePledgePower](ps); len(m) != 1 || m[0].Privileges != int32(privs) {
		t.Errorf("ManagePledgePower %+v", m)
	}
	e.clanSend(sessA, clientpackets.OpRequestPledgePower, clanInts(9, clientpackets.PledgePowerSet, int32(social.AllPrivileges)))
	e.onLoop(func() {
		if c.Ranks.Of(9) != social.Rank9Allowed {
			t.Errorf("rank 9 = %#x, want %#x", c.Ranks.Of(9), social.Rank9Allowed)
		}
	})

	ps = e.clanSendEx(sessA, clientpackets.ExOpRequestPledgePowerGradeList, nil)
	if g := only[testclient.PledgePowerGradeList](ps); len(g) != 1 || len(g[0].Grades) != 9 {
		t.Errorf("grade list %+v", g)
	}
	// Bob to rank 3: the privileges of rank 3 (none).
	ps = e.clanSendEx(sessA, clientpackets.ExOpRequestPledgeSetMemberPowerGrade, append(clanStrings("Bob"), clanInts(3)...))
	if info := only[testclient.PledgeReceivePowerInfo](ps); len(info) != 1 || info[0].PowerGrade != 3 || info[0].Name != "Bob" {
		t.Errorf("power info %+v", info)
	}
	e.onLoop(func() {
		if b.PowerGrade() != 3 || b.ClanPrivileges() != 0 {
			t.Errorf("Bob grade %d privileges %#x", b.PowerGrade(), b.ClanPrivileges())
		}
	})
	ps = e.clanSendEx(sessA, clientpackets.ExOpRequestPledgeMemberInfo, append(clanInts(0), clanStrings("Bob")...))
	if mi := only[testclient.PledgeReceiveMemberInfo](ps); len(mi) != 1 || mi[0].UnitName != "Dragons" || mi[0].PowerGrade != 3 {
		t.Errorf("member info %+v", mi)
	}
	ps = e.clanSendEx(sessA, clientpackets.ExOpRequestPledgeMemberPowerInfo, append(clanInts(0), clanStrings("Leader")...))
	if info := only[testclient.PledgeReceivePowerInfo](ps); len(info) != 1 || info[0].PowerGrade != 1 {
		t.Errorf("leader power info %+v", info)
	}

	// Titles need clan level 3.
	ps = e.clanSend(sessA, clientpackets.OpRequestGiveNickName, clanStrings("Bob", "Knight"))
	clanWantMsg(t, "level 0", ps, int32(clanMsgTitleNeedsLevel3))
	e.onLoop(func() { e.srv.ChangeClanLevel(c, 3) })
	sessB.take(t)
	e.clanSend(sessA, clientpackets.OpRequestGiveNickName, clanStrings("Bob", "Knight of the long title"))
	clanWantMsg(t, "title", sessB.take(t), int32(clanMsgTitleChanged))
	e.onLoop(func() {
		if b.Title() != "Knight of the long t" {
			t.Errorf("Bob's title %q", b.Title())
		}
	})
	clanWantMsg(t, "not a member", e.clanSend(sessA, clientpackets.OpRequestGiveNickName, clanStrings("Nobody", "X")), int32(clanMsgTargetMustBeMember))
	clanWantMsg(t, "no privilege", e.clanSend(sessB, clientpackets.OpRequestGiveNickName, clanStrings("Leader", "X")), int32(clanMsgNotAuthorized))
	e.clanLogout(sessB, b)
	clanWantMsg(t, "offline", e.clanSend(sessA, clientpackets.OpRequestGiveNickName, clanStrings("Bob", "X")), int32(clanMsgPlayerNotOnline))
	// The offline member's grade is patched.
	e.clanSendEx(sessA, clientpackets.ExOpRequestPledgeSetMemberPowerGrade, append(clanStrings("Bob"), clanInts(4)...))
	if ch := e.storedCharacter(b.ObjectID()); ch.PowerGrade != 4 || ch.Title != "Knight of the long t" {
		t.Errorf("stored Bob grade %d title %q", ch.PowerGrade, ch.Title)
	}
}

// Dissolution: refused in an alliance or at war, then requested (7 days,
// the leader loses exp); recovery cancels it; a second request removes the
// clan after 7 days, with the create penalty for the leader and offline
// members patched out.
func TestClanDissolveAndRecover(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, 0, 1)

	clanWantMsg(t, "member", e.vmCommand(sessB, b, vm, "dissolve_clan"), int32(clanMsgNotAuthorized))
	e.onLoop(func() { c.AllyID = c.ID })
	clanWantMsg(t, "alliance", e.vmCommand(sessA, a, vm, "dissolve_clan"), int32(clanMsgDissolveInAlliance))
	e.onLoop(func() { c.AllyID, c.WarsDeclared = 0, []int32{1} })
	clanWantMsg(t, "war", e.vmCommand(sessA, a, vm, "dissolve_clan"), int32(clanMsgDissolveAtWar))
	e.onLoop(func() { c.WarsDeclared = nil })

	var exp int64
	e.onLoop(func() { exp = a.char.Exp })
	e.vmCommand(sessA, a, vm, "dissolve_clan")
	e.onLoop(func() {
		if want := e.srv.Now().UnixMilli() + 7*social.DayMillis; c.DissolvingExpiry != want {
			t.Errorf("dissolving expiry %d, want %d", c.DissolvingExpiry, want)
		}
		if a.char.Exp >= exp {
			t.Errorf("the leader kept its exp %d", a.char.Exp)
		}
	})
	clanWantMsg(t, "again", e.vmCommand(sessA, a, vm, "dissolve_clan"), int32(clanMsgAlreadyDissolving))
	e.vmCommand(sessA, a, vm, "recover_clan")
	if doc, err := e.storedClan(c.ID); err != nil || doc.DissolvingExpiry != 0 {
		t.Errorf("stored expiry %d after recovery, %v", doc.DissolvingExpiry, err)
	}
	e.advance(7*24*time.Hour + time.Minute)
	if !e.clanExists(c.ID) {
		t.Fatal("a recovered clan was removed")
	}

	e.vmCommand(sessA, a, vm, "dissolve_clan")
	e.clanLogout(sessB, b)
	sessA.take(t)
	e.advance(7*24*time.Hour - time.Minute)
	if !e.clanExists(c.ID) {
		t.Fatal("the clan went before 7 days")
	}
	e.advance(2 * time.Minute)
	var gone bool
	e.onLoop(func() { gone = e.srv.Clan(c.ID) == nil && a.Clan() == nil })
	if !gone {
		t.Fatal("the clan is still there after 7 days")
	}
	ps := sessA.take(t)
	clanWantMsg(t, "dispersed", ps, int32(clanMsgClanDispersed))
	if !slices.Contains(testclient.Names(ps), "PledgeShowMemberListDeleteAll") {
		t.Errorf("leader packets %q", testclient.Names(ps))
	}
	e.onLoop(func() {
		if a.char.ClanCreateExpiry <= e.srv.Now().UnixMilli()+9*social.DayMillis || a.char.ClanJoinExpiry != 0 {
			t.Errorf("leader penalties create %d join %d", a.char.ClanCreateExpiry, a.char.ClanJoinExpiry)
		}
	})
	if _, err := e.storedClan(c.ID); err == nil {
		t.Error("the clan is still stored")
	}
	if ch := e.storedCharacter(b.ObjectID()); ch.ClanID != 0 {
		t.Errorf("offline Bob still in clan %d", ch.ClanID)
	}
}

// The leader change waits for the Wednesday 06:30 reset; with
// leaderInstantActivation it happens at once.
func TestClanLeaderChangeOnWednesday(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil) // Thursday 13:00
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, 0, 1)

	clanWantMsg(t, "unknown", e.vmCommand(sessA, a, vm, "change_clan_leader Nobody"), int32(clanMsgS1DoesNotExist))
	clanWantMsg(t, "member", e.vmCommand(sessB, b, vm, "change_clan_leader Bob"), int32(clanMsgNotAuthorized))
	ps := e.vmCommand(sessA, a, vm, "change_clan_leader Bob")
	if pg := onePage(t, ps); !strings.Contains(pg.HTML, "LEADER SUCCESS") {
		t.Errorf("page %q", pg.HTML)
	}
	if pg := onePage(t, e.vmCommand(sessA, a, vm, "change_clan_leader Bob")); !strings.Contains(pg.HTML, "LEADER IN PROGRESS") {
		t.Errorf("page %q", pg.HTML)
	}
	if pg := onePage(t, e.vmCommand(sessA, a, vm, "cancel_clan_leader_change")); !strings.Contains(pg.HTML, "LEADER CANCELED") {
		t.Errorf("page %q", pg.HTML)
	}
	e.vmCommand(sessA, a, vm, "change_clan_leader Bob")
	if doc, err := e.storedClan(c.ID); err != nil || doc.NewLeaderID != b.ObjectID() {
		t.Errorf("stored new leader %d, %v", doc.NewLeaderID, err)
	}

	// Thursday 13:00 → Wednesday 06:29: nothing yet.
	e.advance(5*24*time.Hour + 17*time.Hour + 29*time.Minute)
	e.onLoop(func() {
		if e.srv.Now().Weekday() != time.Wednesday || c.LeaderID != a.ObjectID() {
			t.Errorf("%v: leader %d", e.srv.Now(), c.LeaderID)
		}
	})
	sessB.take(t)
	e.advance(time.Minute)
	e.onLoop(func() {
		if c.LeaderID != b.ObjectID() || c.NewLeaderID != 0 || !b.IsClanLeader() || a.IsClanLeader() ||
			b.ClanPrivileges() != social.AllPrivileges || a.ClanPrivileges() != 0 {
			t.Errorf("after the reset: leader %d new %d, privileges %#x / %#x", c.LeaderID, c.NewLeaderID, b.ClanPrivileges(), a.ClanPrivileges())
		}
	})
	clanWantMsg(t, "transferred", sessB.take(t), int32(clanMsgLeaderTransferred))
	if doc, err := e.storedClan(c.ID); err != nil || doc.LeaderID != b.ObjectID() {
		t.Errorf("stored leader %d, %v", doc.LeaderID, err)
	}

	// Instant activation.
	e2 := newClanEnv(t, nil, time.Time{}, func(gs *config.GameServer) { gs.Clan.LeaderInstantActivation = true })
	sessA2, a2 := e2.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB2, b2 := e2.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	vm2 := e2.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	c2 := e2.foundClan(a2, "Dragons")
	e2.inviteToClan(sessA2, a2, sessB2, b2, 0, 1)
	e2.vmCommand(sessA2, a2, vm2, "change_clan_leader Bob")
	e2.onLoop(func() {
		if c2.LeaderID != b2.ObjectID() {
			t.Errorf("instant activation: leader %d", c2.LeaderID)
		}
	})
}

// Entering the world: PledgeSkillList, the clan window and
// PledgeStatusChanged after the first UserInfo; the members online learn
// that the member logged in and, later, out. Clan chat skips members who
// block the speaker.
func TestClanEnterWorldAndChat(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, 0, 1)
	e.clanLogout(sessB, b)
	sessA.take(t)

	sessB = newSession("bob")
	b = e.enter(sessB, 0)
	names := testclient.Names(sessB.take(t))
	if !containsInOrder(names, "UserInfo", "PledgeSkillList", "PledgeShowMemberListAll", "PledgeStatusChanged", "SendMacroList") {
		t.Errorf("enter world packets %q", names)
	}
	psA := sessA.take(t)
	clanWantMsg(t, "logged in", psA, int32(clanMsgMemberLoggedIn))
	if ups := only[testclient.PledgeShowMemberListUpdate](psA); len(ups) != 1 || ups[0].ObjectID != b.ObjectID() {
		t.Errorf("online update %+v", ups)
	}
	e.onLoop(func() {
		if b.Clan() != c || b.PowerGrade() != social.RankNewMember {
			t.Errorf("Bob after login: clan %v grade %d", b.Clan(), b.PowerGrade())
		}
	})

	// Clan chat.
	_, x := e.enterClanChar(clanChar{account: "xeno", name: "Xeno", level: 20})
	sessA.take(t)
	sessB.take(t)
	e.srv.Handle(sessA, clientpackets.OpSay2, chatBody(4, "hello clan", ""))
	e.sync()
	if says := only[testclient.CreatureSay](sessB.take(t)); len(says) != 1 || says[0].Text != "hello clan" {
		t.Errorf("Bob's clan chat %+v", says)
	}
	e.onLoop(func() { b.social.blocks = append(b.social.blocks, socialEntry{id: a.ObjectID(), name: a.Name()}) })
	e.srv.Handle(sessA, clientpackets.OpSay2, chatBody(4, "again", ""))
	e.sync()
	if says := only[testclient.CreatureSay](sessB.take(t)); len(says) != 0 {
		t.Errorf("a blocker got clan chat %+v", says)
	}
	_ = x

	// Logging out.
	e.clanLogout(sessB, b)
	if ups := only[testclient.PledgeShowMemberListUpdate](sessA.take(t)); len(ups) != 1 || ups[0].ObjectID != 0 {
		t.Errorf("offline update %+v", ups)
	}
}

// The registry comes back after a restart: clans, members, ranks and the
// pending leader change; an alliance whose leader clan is gone is dropped;
// an expired dissolution is re-armed for 5 minutes.
func TestClanRegistryAfterRestart(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	_, z := e.enterClanChar(clanChar{account: "zed", name: "Zed", level: 40})
	c := e.foundClan(a, "Dragons")
	gone := e.foundClan(z, "Gone")
	e.inviteToClan(sessA, a, sessB, b, 0, 1)
	e.clanSend(sessA, clientpackets.OpRequestPledgePower, clanInts(5, clientpackets.PledgePowerSet, 6))
	e.onLoop(func() {
		e.srv.ChangeClanLevel(c, 3)
		c.AllyID, c.AllyName = 424242, "Lost"
		gone.DissolvingExpiry = e.srv.Now().UnixMilli() + 1000
		e.srv.saveClan(c)
		e.srv.saveClan(gone)
	})
	e.flushClanWrites(c.ID, gone.ID, a.ObjectID(), b.ObjectID())
	e.stop()

	e2 := newClanEnv(t, e, testStart.Add(time.Hour), nil)
	var c2 *Clan
	e2.onLoop(func() {
		c2 = e2.srv.Clan(c.ID)
		if c2 == nil || c2.Name != "Dragons" || c2.Level != 3 || c2.MemberCount() != 2 || c2.LeaderID != a.ObjectID() || c2.Ranks.Of(5) != 6 {
			t.Errorf("restored clan %+v", c2)
			return
		}
		if c2.AllyID != 0 || c2.AllyName != "" {
			t.Errorf("the alliance of a missing leader clan stayed: %d %q", c2.AllyID, c2.AllyName)
		}
		if e2.srv.ClanByName("dragons") != c2 {
			t.Error("no clan by name")
		}
	})
	if !e2.clanExists(gone.ID) {
		t.Fatal("the dissolving clan went at once")
	}
	e2.advance(5*time.Minute + time.Second)
	if e2.clanExists(gone.ID) {
		t.Error("the dissolving clan stayed after 5 minutes")
	}
	_, b2 := e2.clanLogin("bob")
	e2.onLoop(func() {
		if b2.Clan() != c2 || b2.ClanPrivileges() != 6 {
			t.Errorf("Bob after the restart: clan %v privileges %#x", b2.Clan(), b2.ClanPrivileges())
		}
	})
}

// Clan ranks are recalculated every 20 minutes: clans of level 3 and more,
// by level, then reputation.
func TestClanRanksEvery20Minutes(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	_, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	_, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 40})
	ca := e.foundClan(a, "Dragons")
	cb := e.foundClan(b, "Tigers")
	e.onLoop(func() {
		ca.Level, cb.Level = 3, 4
		ca.Rank, cb.Rank = 0, 0
	})
	e.advance(20 * time.Minute)
	e.onLoop(func() {
		if ca.Rank != 2 || cb.Rank != 1 {
			t.Errorf("ranks %d and %d, want 2 and 1", ca.Rank, cb.Rank)
		}
	})
}

// The GM commands: //pledge create, setlevel, rep, info and dismiss;
// //clan_info, //clan_show_pending, //clan_force_pending and
// //clan_changeleader.
func TestClanGMCommands(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessG, gm := e.enterClanChar(clanChar{account: "gm", name: "Gm", level: 40, access: storage.AccessLevelAdmin})
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	gmRun := func(line string) []testclient.Packet {
		t.Helper()
		sessG.take(t)
		e.onLoop(func() { e.srv.RunGMCommand(gm, line) })
		return sessG.take(t)
	}
	e.onLoop(func() { gm.Target = a })
	gmRun("pledge create Dragons")
	var c *Clan
	e.onLoop(func() { c = a.Clan() })
	if c == nil || c.LeaderID != a.ObjectID() {
		t.Fatalf("//pledge create: %+v", c)
	}
	ps := gmRun("pledge rep 100")
	if says := only[testclient.CreatureSay](ps); len(says) == 0 || !strings.Contains(says[0].Text, "level 5") {
		t.Errorf("//pledge rep below level 5: %+v", says)
	}
	gmRun("pledge setlevel 5")
	gmRun("pledge rep 100")
	e.onLoop(func() {
		if c.Level != 5 || c.Reputation != 100 {
			t.Errorf("level %d reputation %d", c.Level, c.Reputation)
		}
	})
	if u := only[testclient.Unknown](gmRun("pledge info")); len(u) != 1 || u[0].Opcode != 0x90 {
		t.Errorf("//pledge info: %+v", u)
	}
	if pg := onePage(t, gmRun("clan_info")); !strings.Contains(pg.HTML, "Dragons Leader 5 100 1") {
		t.Errorf("//clan_info page %q", pg.HTML)
	}
	e.inviteToClan(sessA, a, sessB, b, 0, 1)
	e.onLoop(func() { c.NewLeaderID = b.ObjectID() })
	if pg := onePage(t, gmRun("clan_show_pending")); !strings.Contains(pg.HTML, "admin_clan_force_pending "+itoa32(c.ID)) {
		t.Errorf("//clan_show_pending %q", pg.HTML)
	}
	gmRun("clan_force_pending " + itoa32(c.ID))
	e.onLoop(func() {
		if c.LeaderID != b.ObjectID() {
			t.Errorf("forced leader %d", c.LeaderID)
		}
	})
	gmRun("clan_changeleader Leader")
	e.onLoop(func() {
		if c.LeaderID != a.ObjectID() {
			t.Errorf("changed leader %d", c.LeaderID)
		}
	})
	e.onLoop(func() { gm.Target = b })
	ps = gmRun("pledge dismiss")
	clanWantMsg(t, "not a leader", ps, int32(clanMsgS1IsNotClanLeader))
	e.onLoop(func() { gm.Target = a })
	gmRun("pledge dismiss")
	if e.clanExists(c.ID) {
		t.Error("//pledge dismiss left the clan")
	}
}

func itoa32(v int32) string { return strconv.Itoa(int(v)) }

// /clanpenalty lists the penalties; clan members and leaders cannot be
// deleted.
func TestClanPenaltyAndDeletion(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	c := e.foundClan(a, "Dragons")
	pg := onePage(t, e.clanSend(sessB, clientpackets.OpBypassUserCmd, clanInts(userCmdClanPenalty)))
	if !strings.Contains(pg.HTML, "No penalty is imposed.") {
		t.Errorf("no penalty page %q", pg.HTML)
	}
	e.inviteToClan(sessA, a, sessB, b, 0, 1)
	for _, tc := range []struct {
		id   int32
		want ClanDeletion
	}{{a.ObjectID(), ClanDeletionLeader}, {b.ObjectID(), ClanDeletionMember}} {
		got, err := e.srv.CharacterClanDeletion(e.ctx, tc.id, c.ID)
		if err != nil || got != tc.want {
			t.Errorf("deletion of %d = %v, %v; want %v", tc.id, got, err, tc.want)
		}
	}
	if got, _ := e.srv.CharacterClanDeletion(e.ctx, b.ObjectID(), 0); got != ClanDeletionAllowed {
		t.Errorf("deletion without a clan = %v", got)
	}
	e.clanSend(sessB, clientpackets.OpRequestWithdrawalPledge, nil)
	pg = onePage(t, e.clanSend(sessB, clientpackets.OpBypassUserCmd, clanInts(userCmdClanPenalty)))
	if !strings.Contains(pg.HTML, "Unable to join a clan.") || !strings.Contains(pg.HTML, "2026-11-20") {
		t.Errorf("penalty page %q", pg.HTML)
	}
}

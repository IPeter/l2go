package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The clan quests with clans (Plan 12 Phase 7): Q501, Q503 and Q508–Q510
// read the clan, its level and reputation, and the clan leader's quest
// state. Every top-level name here starts with clanq.

// clanqLoad loads one quest with the generated constant tables
// (SystemMessageId).
func clanqLoad(t *testing.T, name string) *scripttest.Harness {
	t.Helper()
	consts, err := data.LoadConstants(b2DataDir)
	if err != nil {
		t.Fatal(err)
	}
	tables := make(map[string]map[string]any)
	for _, tb := range consts.Tables {
		m := make(map[string]any, tb.Len())
		for k, v := range tb.Ints {
			m[k] = v
		}
		for k, v := range tb.Strings {
			m[k] = v
		}
		tables[tb.Name] = m
	}
	return scripttest.LoadWithTables(t, b3cScriptsDir, tables, "quests/"+name+"/"+name+".lua")
}

// clanqClan returns a clan of the level led by leader, with members (the
// leader included) in the game.
func clanqClan(level int, leader *scripttest.Player, members ...*scripttest.Player) *scripttest.Clan {
	c := &scripttest.Clan{ID: 0x30000001, Name: "Dragons", Level: level, Leader: leader}
	c.Members = append([]*scripttest.Player{leader}, members...)
	for _, m := range c.Members {
		m.Clan = c
	}
	return c
}

// A Clan's Reputation (Q508): only the leader of a clan of level 5 starts
// it; a member's raid kill near the leader gives the leader the item; the
// leader turns it in for reputation, with the clan quest message.
func TestClanqQ508Reputation(t *testing.T) {
	const q508 = "Q00508_AClansReputation"
	h := clanqLoad(t, q508)
	w := h.World()
	eric := &scripttest.Npc{ObjectID: 0x60000001, ID: 30868, Name: "Sir Eric Rodemai"}
	leader := &scripttest.Player{ObjectID: 0x10000001, Name: "Leader", Level: 60}
	member := &scripttest.Player{ObjectID: 0x10000002, Name: "Member", Level: 60}
	clan := clanqClan(4, leader, member)

	reply, ok := h.Talk(q508, member, eric)
	b3cPage(t, "member", reply, ok, "30868-0a.htm")
	reply, ok = h.Talk(q508, leader, eric)
	b3cPage(t, "level 4", reply, ok, "30868-0b.htm")
	b3cQuest(t, "level 4", leader, q508, scripting.StateCreated, 0)

	clan.Level = 5
	reply, ok = h.Talk(q508, leader, eric)
	b3cPage(t, "level 5", reply, ok, "30868-0c.htm")
	reply, ok = h.Event(q508, "1", leader, eric)
	b3cPage(t, "raid 1", reply, ok, "30868-1.htm")
	b3cQuest(t, "raid 1", leader, q508, scripting.StateStarted, 1)

	// The wrong raid gives nothing; Themis near the leader gives the scale.
	h.Kill(member, &scripttest.Npc{ObjectID: 0x60000002, ID: 25140, Name: "Hekaton Prime"})
	b3cItems(t, "wrong raid", leader, map[int]int64{8277: 0})
	themis := &scripttest.Npc{ObjectID: 0x60000003, ID: 25252, Name: "Palibati Queen Themis", X: 100}
	w.ClearRecords()
	h.Kill(member, themis)
	b3cItems(t, "Themis", leader, map[int]int64{8277: 1})
	b3cSounds(t, "Themis", w, leader, b3cSoundMiddle)

	// A leader out of range (and a member without the leader's state) gets
	// nothing more.
	leader.X = 5000
	h.Kill(member, themis)
	b3cItems(t, "far leader", leader, map[int]int64{8277: 1})
	leader.X = 0

	h.SetRand(rnd.NewScripted(t, rnd.Between(65, 100, 70)))
	reply, ok = h.Talk(q508, leader, eric)
	b3cPage(t, "reward", reply, ok, "30868-1b.htm")
	if clan.Reputation != 70 || clan.PledgeInfoUpdates != 1 {
		t.Errorf("reputation %d, info updates %d", clan.Reputation, clan.PledgeInfoUpdates)
	}
	b3cItems(t, "reward", leader, map[int]int64{8277: 0})
	if len(w.Messages) != 1 || w.Messages[0].Player != leader.ObjectID || !slices.Equal(w.Messages[0].Params, []string{"70"}) {
		t.Errorf("messages %+v", w.Messages)
	}
}

// The Clan's Prestige (Q509) needs clan level 6; Q510 needs level 5 and
// pays 50 per claw a clan member brought the leader.
func TestClanqQ509Q510(t *testing.T) {
	const q509, q510 = "Q00509_AClansFame", "Q00510_AClansPrestige"
	valdis := &scripttest.Npc{ObjectID: 0x60000001, ID: 31331, Name: "Valdis"}
	leader := &scripttest.Player{ObjectID: 0x10000001, Name: "Leader", Level: 60}
	member := &scripttest.Player{ObjectID: 0x10000002, Name: "Member", Level: 60}
	clan := clanqClan(5, leader, member)

	h := clanqLoad(t, q509)
	reply, ok := h.Talk(q509, leader, valdis)
	b3cPage(t, "Q509 level 5", reply, ok, "31331-0b.htm")
	clan.Level = 6
	reply, ok = h.Talk(q509, leader, valdis)
	b3cPage(t, "Q509 level 6", reply, ok, "31331-0c.htm")

	clan.Level = 5
	h = clanqLoad(t, q510)
	reply, ok = h.Talk(q510, leader, valdis)
	b3cPage(t, "Q510 level 5", reply, ok, "31331-1.htm")
	h.Event(q510, "31331-3.htm", leader, valdis)
	for i := range 2 {
		h.Kill(member, &scripttest.Npc{ObjectID: int32(0x60000010 + i), ID: 22215, Name: "Tyrannosaurus"})
	}
	b3cItems(t, "claws", leader, map[int]int64{8767: 2})
	reply, ok = h.Talk(q510, leader, valdis)
	b3cPage(t, "Q510 reward", reply, ok, "31331-7.htm")
	if clan.Reputation != 100 {
		t.Errorf("reputation %d, want 100", clan.Reputation)
	}
}

// Proof of Clan Alliance (Q501): the leader of a level 3 clan starts it; a
// member's offering counts on the leader's state; Kalis resets the trial
// when the leader is no longer poisoned (player:buff_info_by_abnormal_type).
func TestClanqQ501Poison(t *testing.T) {
	const q501 = "Q00501_ProofOfClanAlliance"
	h := clanqLoad(t, q501)
	kristof := &scripttest.Npc{ObjectID: 0x60000001, ID: 30756, Name: "Sir Kristof Rodemai"}
	statue := &scripttest.Npc{ObjectID: 0x60000002, ID: 30757, Name: "Statue of Offering"}
	kalis := &scripttest.Npc{ObjectID: 0x60000003, ID: 30759, Name: "Kalis"}
	leader := &scripttest.Player{ObjectID: 0x10000001, Name: "Leader", Level: 60}
	member := &scripttest.Player{ObjectID: 0x10000002, Name: "Member", Level: 45}
	clanqClan(3, leader, member)

	reply, ok := h.Talk(q501, leader, kristof)
	b3cPage(t, "start", reply, ok, "30756-04.htm")
	h.Event(q501, "30756-07.htm", leader, kristof)
	reply, ok = h.Talk(q501, leader, kalis)
	b3cPage(t, "Kalis", reply, ok, "30759-01.htm")
	h.Event(q501, "30759-03.htm", leader, kalis)
	b3cQuest(t, "trial", leader, q501, scripting.StateStarted, 2)

	reply, ok = h.Talk(q501, member, statue)
	b3cPage(t, "statue", reply, ok, "30757-01.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 7)))
	reply, ok = h.Event(q501, "30757-03.htm", member, statue)
	b3cPage(t, "offering", reply, ok, "30757-04.htm")
	if got := leader.Quest(q501).Vars()["symbols"]; got != "1" {
		t.Errorf("leader's symbols %q, want 1", got)
	}
	b3cItems(t, "offering", member, map[int]int64{3837: 1})

	// The poison trial: poisoned without the herbs, then cured too early.
	leader.Give(3837, 3)
	h.Event(q501, "30759-07.htm", leader, kalis)
	leader.Abnormals = []string{"FATAL_POISON"}
	reply, ok = h.Talk(q501, leader, kalis)
	b3cPage(t, "poisoned", reply, ok, "30759-10.htm")
	leader.Abnormals = nil
	reply, ok = h.Talk(q501, leader, kalis)
	b3cPage(t, "cured", reply, ok, "30759-09.htm")
	b3cQuest(t, "cured", leader, q501, scripting.StateStarted, 1)
}

// Pursuit of Clan Ambition (Q503): the leader of a level 4 clan starts it;
// at Martien the members in the game get the quest, and they see the
// leader's progress; finishing it takes it from them again.
func TestClanqQ503Members(t *testing.T) {
	const q503 = "Q00503_PursuitOfClanAmbition"
	h := clanqLoad(t, q503)
	gustav := &scripttest.Npc{ObjectID: 0x60000001, ID: 30760, Name: "Sir Gustav Athebaldt"}
	martien := &scripttest.Npc{ObjectID: 0x60000002, ID: 30645, Name: "Martien"}
	leader := &scripttest.Player{ObjectID: 0x10000001, Name: "Leader", Level: 60}
	member := &scripttest.Player{ObjectID: 0x10000002, Name: "Member", Level: 60}
	offline := &scripttest.Player{ObjectID: 0x10000003, Name: "Away", Level: 60, Offline: true}
	clan := clanqClan(3, leader, member, offline)

	reply, ok := h.Talk(q503, leader, gustav)
	b3cPage(t, "level 3", reply, ok, "30760-02.htm")
	clan.Level = 4
	reply, ok = h.Talk(q503, member, gustav)
	b3cPage(t, "member", reply, ok, "30760-04t.htm")
	reply, ok = h.Talk(q503, leader, gustav)
	b3cPage(t, "level 4", reply, ok, "30760-04.htm")
	h.Event(q503, "30760-08.htm", leader, gustav)
	b3cQuest(t, "started", leader, q503, scripting.StateStarted, 1)

	h.Event(q503, "30645-03.htm", leader, martien)
	b3cQuest(t, "Martien", leader, q503, scripting.StateStarted, 2)
	b3cQuest(t, "member's state", member, q503, scripting.StateStarted, 1)
	b3cNoQuest(t, "offline member", offline, q503)
	reply, ok = h.Talk(q503, member, martien)
	b3cPage(t, "member at Martien", reply, ok, "30645-01.htm")

	// Without the leader in the game the member sees nothing of it.
	leader.Offline = true
	reply, ok = h.Talk(q503, member, martien)
	if reply.Content == "30645-01.htm" {
		t.Errorf("member at Martien without the leader: %+v, %v", reply, ok)
	}
	leader.Offline = false

	leader.Give(3869, 1)
	h.Event(q503, "30760-20.htm", leader, gustav)
	b3cItems(t, "finished", leader, map[int]int64{3869: 0, 3870: 1})
	b3cNoQuest(t, "member after the end", member, q503)
}

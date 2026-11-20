package game

import (
	"maps"
	"slices"
	"strconv"
	"testing"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// knownLevels returns the levels of the skills p knows, by ID.
func (e *testEnv) knownLevels(p *Player) map[int]int {
	e.t.Helper()
	out := map[int]int{}
	e.onLoop(func() {
		for _, sk := range p.Skills() {
			out[sk.ID] = sk.Level
		}
	})
	return out
}

// storedLevels returns the saved skill rows of p, by ID.
func (e *testEnv) storedLevels(p *Player) map[int]int {
	e.t.Helper()
	out := map[int]int{}
	e.onLoop(func() {
		for _, r := range p.skill.stored {
			out[int(r.id)] = int(r.level)
		}
	})
	return out
}

// AutoLearnSkills (Mobius rewardSkills → giveAvailableSkills): a Human
// Fighter that reaches level 20 knows what //give_all_skills gives at that
// level, saved, with ShortcutInit and "You have learned N new skills.";
// one that logs in at level 20 knows them too. Without it the level-up
// brings only the auto-get skills.
func TestAutoLearnSkills(t *testing.T) {
	// Off: the level-up brings the auto-get skills; //give_all_skills then
	// gives what AutoLearnSkills should.
	off := newLearnEnv(t)
	gmSess, gm := off.enterLearner(learner{account: "gm", name: "Master", class: classHumanFighter, level: 20, access: storage.AccessLevelAdmin})
	sess, p := off.enterLearner(learner{account: "acc", name: "Pupil", class: classHumanFighter, level: 1})
	off.onLoop(func() { off.srv.AddLevel(p, 19) })
	autoGet := off.knownLevels(p)
	for _, s := range only[testclient.SystemMessage](sess.take(t)) {
		if len(s.Texts()) > 0 {
			t.Errorf("level-up without AutoLearnSkills said %q", s.Texts())
		}
	}
	gmSess.take(t)
	off.onLoop(func() {
		gm.Target = p
		off.srv.RunGMCommand(gm, "give_all_skills")
	})
	want := off.knownLevels(p)
	if len(want) <= len(autoGet) {
		t.Fatalf("//give_all_skills gave nothing beyond the auto-get skills %v", autoGet)
	}
	t.Logf("level 20 Human Fighter: auto-get %v, //give_all_skills %v", autoGet, want)
	newSkills := 0
	for id := range want {
		if _, ok := autoGet[id]; !ok {
			newSkills++
		}
	}

	// On: the same level-up gives all of them.
	on := newLearnEnv(t, func(gs *config.GameServer) { gs.World.AutoLearnSkills = true })
	sess, p = on.enterLearner(learner{account: "acc", name: "Pupil", class: classHumanFighter, level: 1})
	atOne := on.knownLevels(p)
	on.onLoop(func() { on.srv.AddLevel(p, 19) })
	if got := on.knownLevels(p); !maps.Equal(got, want) {
		t.Errorf("level 20 with AutoLearnSkills knows %v\nwant (//give_all_skills) %v", got, want)
	}
	if got := on.storedLevels(p); !maps.Equal(got, want) {
		t.Errorf("saved skills %v\nwant %v", got, want)
	}
	ps := sess.take(t)
	learned := 0
	for id := range want {
		if _, ok := atOne[id]; !ok {
			learned++
		}
	}
	msg := "You have learned " + strconv.Itoa(learned) + " new skills."
	var texts []string
	for _, s := range only[testclient.SystemMessage](ps) {
		texts = append(texts, s.Texts()...)
	}
	if !slices.Contains(texts, msg) {
		t.Errorf("level-up said %q, want %q", texts, msg)
	}
	if len(only[testclient.ShortcutInit](ps)) == 0 {
		t.Errorf("level-up sent %q, want ShortcutInit", testclient.Names(ps))
	}
	on.advance(skillListDelay)
	if lists := only[testclient.SkillList](sess.take(t)); len(lists) != 1 || len(lists[0].Skills) != len(want) {
		t.Errorf("SkillLists after the level-up %+v, want one of %d skills", lists, len(want))
	}
	if learned < newSkills {
		t.Errorf("learned %d skills at level 20, want at least the %d beyond auto-get", learned, newSkills)
	}

	// A level-20 Human Fighter logging in gets them too, without a message.
	_, q := on.enterLearner(learner{account: "acc2", name: "Elder", class: classHumanFighter, level: 20})
	if got := on.knownLevels(q); !maps.Equal(got, want) {
		t.Errorf("login at level 20 with AutoLearnSkills knows %v\nwant %v", got, want)
	}
	if got := on.storedLevels(q); !maps.Equal(got, want) {
		t.Errorf("saved skills at login %v\nwant %v", got, want)
	}
}

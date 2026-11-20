package skills_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/skilltest"
	"github.com/IPeter/l2go/internal/stats"
)

func TestBuffLandsShowsAndWearsOff(t *testing.T) {
	f := newFixture(t)
	cat := f.catalog(t, buffSkill(1068, "PA_UP", 1, "1.08", map[string]string{"abnormalTime": "3"}))
	might := skilltest.Skill(t, cat, 1068)
	caster, target := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2)

	might.ApplyTo(f.env, caster, target)

	base := 100.0
	if got, want := target.PAtk(nil), base*1.08; got != want {
		t.Fatalf("P.Atk with Might = %v, want %v", got, want)
	}
	if got := target.MessageIDs(); !slices.Equal(got, []sp.SystemMessageID{skills.MsgEffectsOfS1FlowThroughYou}) {
		t.Fatalf("messages = %v, want the flow-through message", got)
	}
	if len(target.Icons) != 0 {
		t.Fatalf("icons sent at once, want them 300 ms later")
	}
	f.sched.Advance(300 * time.Millisecond)
	want := skills.IconUpdate{SendAbnormal: true,
		Icons: []sp.AbnormalStatusEntry{{SkillID: 1068, SkillLevel: 1, Seconds: 3}},
		Party: []sp.AbnormalStatusEntry{{SkillID: 1068, SkillLevel: 1, Seconds: 3}}}
	if len(target.Icons) != 1 || !iconsEqual(target.Icons[0], want) {
		t.Fatalf("icons = %+v, want %+v", target.Icons, want)
	}

	// The buff clock counts once at the start and every second after: a
	// 3 s buff ends when the count passes 3, at 3 s.
	f.sched.Advance(2699 * time.Millisecond)
	if target.EL.BySkillID(1068) == nil {
		t.Fatalf("Might ended before 3 s")
	}
	target.Reset()
	f.sched.Advance(time.Millisecond)
	if target.EL.BySkillID(1068) != nil {
		t.Fatalf("Might still on at 3 s")
	}
	if got := target.MessageIDs(); !slices.Equal(got, []sp.SystemMessageID{skills.MsgS1HasWornOff}) {
		t.Fatalf("messages = %v, want worn off", got)
	}
	if got := target.PAtk(nil); got != 100 {
		t.Fatalf("P.Atk after Might = %v, want 100", got)
	}
	if target.EL.ByAbnormal(data.AbnormalPaUp) != nil {
		t.Fatalf("PA_UP still stacked")
	}
}

func iconsEqual(a, b skills.IconUpdate) bool {
	return a.SendAbnormal == b.SendAbnormal && slices.Equal(a.Icons, b.Icons) && slices.Equal(a.Party, b.Party) && a.Summon == b.Summon
}

func TestStackingByAbnormalLevel(t *testing.T) {
	const oldID, newID = 100, 101
	replaced := []sp.SystemMessageID{skills.MsgS1HasWornOff, skills.MsgEffectsOfS1FlowThroughYou}
	tests := []struct {
		name     string
		old, new int // abnormal levels
		stacked  int // skill ID in use after
		messages []sp.SystemMessageID
	}{
		{"lower level dropped", 2, 1, oldID, nil},
		{"equal level replaces", 2, 2, newID, replaced},
		{"higher level replaces", 2, 3, newID, replaced},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			cat := f.catalog(t,
				buffSkill(oldID, "PA_UP", tt.old, "1.1", nil),
				buffSkill(newID, "PA_UP", tt.new, "1.2", nil))
			p := skilltest.NewPlayer(f.env, 1)
			skilltest.Skill(t, cat, oldID).ApplyTo(f.env, p, p)
			p.Reset()
			skilltest.Skill(t, cat, newID).ApplyTo(f.env, p, p)
			if got := p.EL.ByAbnormal(data.AbnormalPaUp).Skill().ID; got != tt.stacked {
				t.Errorf("stacked skill = %d, want %d", got, tt.stacked)
			}
			if got := len(p.EL.Buffs()); got != 1 {
				t.Errorf("%d buffs, want 1", got)
			}
			if got := p.MessageIDs(); !slices.Equal(got, tt.messages) {
				t.Errorf("messages = %v, want %v", got, tt.messages)
			}
		})
	}
}

func TestNoneTypeRecastMovesToTheEnd(t *testing.T) {
	f := newFixture(t)
	cat := f.catalog(t, buffSkill(200, "NONE", 0, "1.1", nil), buffSkill(201, "NONE", 0, "1.2", nil))
	p := skilltest.NewPlayer(f.env, 1)
	a, b := skilltest.Skill(t, cat, 200), skilltest.Skill(t, cat, 201)
	a.ApplyTo(f.env, p, p)
	b.ApplyTo(f.env, p, p)
	p.Reset()
	a.ApplyTo(f.env, p, p)
	if got := skillIDs(p.EL.Buffs()); !slices.Equal(got, []int{201, 200}) {
		t.Fatalf("buffs = %v, want [201 200]", got)
	}
	if got := p.MessageIDs(); !slices.Equal(got, []sp.SystemMessageID{skills.MsgS1HasWornOff, skills.MsgEffectsOfS1FlowThroughYou}) {
		t.Fatalf("messages = %v", got)
	}
	base := 100.0
	if got, want := p.PAtk(nil), base*1.2*1.1; got != want {
		t.Fatalf("P.Atk = %v, want %v (no stats left behind)", got, want)
	}
}

func TestHerbHidesAndRestores(t *testing.T) {
	f := newFixture(t)
	cat := f.catalog(t,
		buffSkill(300, "PA_UP", 1, "1.1", nil),
		buffSkill(301, "PA_UP", 2, "1.5", map[string]string{"abnormalInstant": "true", "abnormalTime": "2"}))
	p := skilltest.NewPlayer(f.env, 1)
	skilltest.Skill(t, cat, 300).ApplyTo(f.env, p, p)
	skilltest.Skill(t, cat, 301).ApplyTo(f.env, p, p)

	old := p.EL.BySkillID(300)
	if old.InUse() || p.EL.HiddenCount() != 1 || p.EL.BuffCount() != 1 {
		t.Fatalf("in use %v, hidden %d, count %d: want the old buff hidden", old.InUse(), p.EL.HiddenCount(), p.EL.BuffCount())
	}
	if got, want := p.PAtk(nil), 150.0; got != want {
		t.Fatalf("P.Atk under the herb = %v, want %v", got, want)
	}
	f.sched.Advance(300 * time.Millisecond)
	last := p.Icons[len(p.Icons)-1]
	if len(last.Icons) != 1 || last.Icons[0].SkillID != 301 {
		t.Fatalf("icons = %+v, want only the herb", last.Icons)
	}
	// The herb joined a running clock at 0 s: it counts at 1, 2 and 3 s.
	f.sched.Advance(2700 * time.Millisecond)
	if p.EL.BySkillID(301) != nil {
		t.Fatalf("herb still on")
	}
	base := 100.0
	if !old.InUse() || p.EL.HiddenCount() != 0 || p.EL.ByAbnormal(data.AbnormalPaUp) != old {
		t.Fatalf("the hidden buff did not come back")
	}
	if got, want := p.PAtk(nil), base*1.1; got != want {
		t.Fatalf("P.Atk after the herb = %v, want %v", got, want)
	}
}

func TestBuffLimitEndsTheOldest(t *testing.T) {
	for _, inspiration := range []int{0, 1} {
		f := newFixture(t)
		f.env.Config.MaxBuffAmount = 2
		cat := f.catalog(t, buffSkill(400, "NONE", 0, "1", nil), buffSkill(401, "NONE", 0, "1", nil), buffSkill(402, "NONE", 0, "1", nil))
		p := skilltest.NewPlayer(f.env, 1)
		p.Known[skills.DivineInspiration] = inspiration
		for _, id := range []int{400, 401, 402} {
			if id == 402 {
				p.Reset()
			}
			skilltest.Skill(t, cat, id).ApplyTo(f.env, p, p)
			f.sched.Advance(time.Second)
		}
		want, msgs := []int{401, 402}, []sp.SystemMessageID{skills.MsgEffectOfS1Removed, skills.MsgEffectsOfS1FlowThroughYou}
		if inspiration > 0 {
			want, msgs = []int{400, 401, 402}, []sp.SystemMessageID{skills.MsgEffectsOfS1FlowThroughYou}
		}
		if got := skillIDs(p.EL.Buffs()); !slices.Equal(got, want) {
			t.Errorf("Divine Inspiration %d: buffs = %v, want %v", inspiration, got, want)
		}
		if got := p.MessageIDs(); !slices.Equal(got, msgs) {
			t.Errorf("Divine Inspiration %d: messages = %v, want %v", inspiration, got, msgs)
		}
	}
}

func TestDanceLimitAndPotionsOutsideTheLimit(t *testing.T) {
	f := newFixture(t)
	f.env.Config.MaxBuffAmount = 2 // the buff and one dance
	f.env.Config.MaxDanceAmount = 1
	dance := func(id int) data.SkillTemplate {
		return buffSkill(id, "NONE", 0, "1", map[string]string{"isMagic": "3"})
	}
	cat := f.catalog(t,
		buffSkill(400, "NONE", 0, "1", nil),
		dance(500), dance(501),
		buffSkill(600, "HP_RECOVER", 1, "1", map[string]string{"abnormalTime": "15"}))
	p := skilltest.NewPlayer(f.env, 1)
	skilltest.Skill(t, cat, 400).ApplyTo(f.env, p, p)
	f.sched.Advance(time.Second)
	skilltest.Skill(t, cat, 500).ApplyTo(f.env, p, p)
	f.sched.Advance(time.Second)
	skilltest.Skill(t, cat, 501).ApplyTo(f.env, p, p)
	if got := skillIDs(p.EL.Dances()); !slices.Equal(got, []int{501}) {
		t.Fatalf("dances = %v, want the newest only", got)
	}
	if got := skillIDs(p.EL.Buffs()); !slices.Equal(got, []int{400}) {
		t.Fatalf("buffs = %v, the dance limit must not end a buff", got)
	}
	skilltest.Skill(t, cat, 600).ApplyTo(f.env, p, p)
	if got := skillIDs(p.EL.Buffs()); !slices.Equal(got, []int{400, 600}) {
		t.Fatalf("buffs = %v, a potion must not end a buff", got)
	}
	f.sched.Advance(300 * time.Millisecond)
	if len(p.ShortBuffs) == 0 || p.ShortBuffs[len(p.ShortBuffs)-1] != (skills.ShortBuff{SkillID: 600, SkillLevel: 1, Seconds: 15}) {
		t.Fatalf("short buffs = %+v", p.ShortBuffs)
	}
	last := p.Icons[len(p.Icons)-1]
	for _, e := range last.Icons {
		if e.SkillID == 600 {
			t.Fatalf("the potion is among the icons")
		}
	}
	if p.EL.BuffCount() != 1 {
		t.Fatalf("buff count %d, want 1 (the short buff does not count)", p.EL.BuffCount())
	}
}

func TestPassivesReplaceBySkillID(t *testing.T) {
	f := newFixture(t)
	tmpl := data.SkillTemplate{
		ID: 700, Name: "Mastery", Levels: 2,
		Tables: map[string][]string{"#v": {"10", "20"}},
		Values: map[string]string{"operateType": "P"},
		Effects: skilltest.General(data.EffectDef{Name: "Buff",
			Funcs: []data.StatFuncDef{{Stat: data.StatPowerAttack, Op: data.OpAdd, Order: 30, Value: "#v"}}}),
	}
	cat := f.catalog(t, tmpl)
	p := skilltest.NewPlayer(f.env, 1)
	p.Dead = true // passives start even on the dead
	l1, _ := cat.Skill(700, 1)
	l2, _ := cat.Skill(700, 2)
	l1.AddPassive(f.env, p)
	if got := p.PAtk(nil); got != 110 {
		t.Fatalf("P.Atk with level 1 = %v, want 110", got)
	}
	l2.AddPassive(f.env, p)
	if got := p.PAtk(nil); got != 120 {
		t.Fatalf("P.Atk with level 2 = %v, want 120", got)
	}
	if n := len(p.EL.Passives()); n != 1 {
		t.Fatalf("%d passives, want 1", n)
	}
	if mods := p.Mods.Of(stats.PAtk); len(mods) != 1 || mods[0].Owner != (stats.Owner{Kind: stats.OwnerSkill, ID: 700}) {
		t.Fatalf("modifiers = %+v, want one owned by skill 700", mods)
	}
	if len(p.Messages) != 0 {
		t.Fatalf("passives sent %v", p.MessageIDs())
	}
	p.EL.StopSkill(skills.FinishNormal, 700)
	if got := p.PAtk(nil); got != 100 || len(p.Messages) != 0 {
		t.Fatalf("after removal P.Atk %v, messages %v", got, p.MessageIDs())
	}
}

func TestNothingLandsOnTheDeadOrBlocked(t *testing.T) {
	f := newFixture(t)
	cat := f.catalog(t, buffSkill(100, "PA_UP", 1, "1.1", nil))
	sk := skilltest.Skill(t, cat, 100)
	dead := skilltest.NewPlayer(f.env, 1)
	dead.Dead = true
	sk.ApplyTo(f.env, dead, dead)
	blocked := skilltest.NewPlayer(f.env, 2)
	blocked.EL.BlockAbnormals(data.AbnormalPaUp)
	sk.ApplyTo(f.env, blocked, blocked)
	if !dead.EL.IsEmpty() || !blocked.EL.IsEmpty() {
		t.Fatalf("effects landed: dead %v, blocked %v", dead.EL.Effects(), blocked.EL.Effects())
	}
	if !blocked.EL.UnblockAbnormals(data.AbnormalPaUp) {
		t.Fatalf("PA_UP was not blocked")
	}
	sk.ApplyTo(f.env, blocked, blocked)
	if blocked.EL.IsEmpty() {
		t.Fatalf("unblocked buff did not land")
	}
}

func TestTicksAndToggles(t *testing.T) {
	f := newFixture(t)
	toggle := skilltest.Template(800, map[string]string{"operateType": "T"},
		skilltest.General(probeDef(nil, "1")))
	ticking := skilltest.Template(801, map[string]string{"operateType": "A2", "abnormalTime": "10"},
		skilltest.General(probeDef(map[string]string{"keep": "false"}, "3")))
	cat := f.catalog(t, toggle, ticking)
	p := skilltest.NewPlayer(f.env, 1)

	skilltest.Skill(t, cat, 801).ApplyTo(f.env, p, p)
	f.sched.Advance(1997 * time.Millisecond)
	if f.log.ticks != 0 {
		t.Fatalf("ticked before 3 × 666 ms")
	}
	f.sched.Advance(time.Millisecond)
	if f.log.ticks != 1 {
		t.Fatalf("%d ticks at 1998 ms, want 1", f.log.ticks)
	}
	f.sched.Advance(1998 * time.Millisecond)
	if f.log.ticks != 2 || p.EL.BySkillID(801) == nil {
		t.Fatalf("a non-toggle that stops ticking ends only with its time")
	}

	*f.log = probeLog{}
	p.Reset()
	skilltest.Skill(t, cat, 800).ApplyTo(f.env, p, p)
	if got := skillIDs(p.EL.Toggles()); !slices.Equal(got, []int{800}) {
		t.Fatalf("toggles = %v", got)
	}
	f.sched.Advance(666 * time.Millisecond)
	if p.EL.BySkillID(800) != nil {
		t.Fatalf("the toggle did not end when its tick said so")
	}
	if f.log.starts != 1 || f.log.ticks != 1 || f.log.exits != 1 {
		t.Fatalf("probe = %+v", *f.log)
	}
	if got := p.MessageIDs(); !slices.Contains(got, skills.MsgS1HasBeenAborted) {
		t.Fatalf("messages = %v, want aborted", got)
	}
}

func TestEndEffectsFlagsAndVisuals(t *testing.T) {
	f := newFixture(t)
	tmpl := skilltest.Template(900, map[string]string{"operateType": "A2", "abnormalTime": "5", "abnormalVisualEffect": "STUN", "isDebuff": "true"},
		map[data.EffectScope][]data.EffectDef{
			data.ScopeGeneral: {probeDef(nil, "")},
			data.ScopeEnd:     {probeDef(map[string]string{"instant": "true"}, "")},
		})
	cat := f.catalog(t, tmpl)
	sk := skilltest.Skill(t, cat, 900)
	for _, ft := range []skills.FinishType{skills.FinishNormal, skills.FinishRemoved, skills.FinishSilent} {
		*f.log = probeLog{}
		m := skilltest.NewMonster(f.env, 10)
		sk.ApplyTo(f.env, m, m)
		if !m.EL.IsAffected(skills.FlagStunned) || m.EL.Visual().Normal == 0 || m.VisualUpdates == 0 {
			t.Fatalf("flags %b, visual %+v, updates %d", m.EL.Flags(), m.EL.Visual(), m.VisualUpdates)
		}
		if got := skillIDs(m.EL.Debuffs()); !slices.Equal(got, []int{900}) {
			t.Fatalf("debuffs = %v", got)
		}
		m.EL.Remove(ft, m.EL.BySkillID(900))
		if m.EL.IsAffected(skills.FlagStunned) || m.EL.Visual().Normal != 0 {
			t.Fatalf("finish %d: flags or visual left", ft)
		}
		// GENERAL starts once; END runs its instant effect (OnStart) on
		// a normal or silent end, not on a removal.
		wantStarts := 2
		if ft == skills.FinishRemoved {
			wantStarts = 1
		}
		if f.log.starts != wantStarts || f.log.exits != 1 {
			t.Errorf("finish %d: probe %+v, want %d starts", ft, *f.log, wantStarts)
		}
	}
}

func TestStopAllExceptStayAfterDeath(t *testing.T) {
	f := newFixture(t)
	cat := f.catalog(t,
		buffSkill(100, "PA_UP", 1, "1.1", nil),
		buffSkill(101, "PD_UP", 1, "1.1", map[string]string{"stayAfterDeath": "true"}))
	p := skilltest.NewPlayer(f.env, 1)
	skilltest.Skill(t, cat, 100).ApplyTo(f.env, p, p)
	skilltest.Skill(t, cat, 101).ApplyTo(f.env, p, p)
	p.EL.StopAllExceptStayAfterDeath()
	if got := skillIDs(p.EL.Buffs()); !slices.Equal(got, []int{101}) {
		t.Fatalf("buffs = %v, want [101]", got)
	}
	p.EL.StopAll()
	if !p.EL.IsEmpty() || p.EL.ByAbnormal(data.AbnormalPdUp) != nil {
		t.Fatalf("StopAll left %v", skillIDs(p.EL.Effects()))
	}
}

func TestIconsCoalesce(t *testing.T) {
	f := newFixture(t)
	cat := f.catalog(t, buffSkill(100, "PA_UP", 1, "1.1", nil), buffSkill(101, "PD_UP", 1, "1.1", nil))
	p := skilltest.NewPlayer(f.env, 1)
	skilltest.Skill(t, cat, 100).ApplyTo(f.env, p, p)
	f.sched.Advance(200 * time.Millisecond)
	skilltest.Skill(t, cat, 101).ApplyTo(f.env, p, p)
	f.sched.Advance(100 * time.Millisecond)
	if len(p.Icons) != 1 || len(p.Icons[0].Icons) != 2 {
		t.Fatalf("icons = %+v, want one update with both buffs", p.Icons)
	}
	p.EL.UpdateIcons(true)
	f.sched.Advance(300 * time.Millisecond)
	if last := p.Icons[len(p.Icons)-1]; last.SendAbnormal || len(last.Party) != 2 {
		t.Fatalf("party-only update = %+v", last)
	}
}

func TestToStore(t *testing.T) {
	f := newFixture(t)
	cat := f.catalog(t,
		buffSkill(100, "PA_UP", 1, "1.1", nil),
		buffSkill(101, "LIFE_FORCE_OTHERS", 1, "1", nil),
		buffSkill(102, "NONE", 0, "1", map[string]string{"isMagic": "3"}),
		skilltest.Template(103, map[string]string{"operateType": "T"}, skilltest.General(probeDef(map[string]string{"keep": "true"}, "100"))))
	p := skilltest.NewPlayer(f.env, 1)
	for _, id := range []int{100, 101, 102, 103} {
		skilltest.Skill(t, cat, id).ApplyTo(f.env, p, p)
	}
	if got := skillIDs(p.EL.ToStore()); !slices.Equal(got, []int{100, 102}) {
		t.Fatalf("stored = %v, want [100 102]", got)
	}
	f.env.Config.StoreDances, f.env.Config.StoreToggles = false, true
	if got := skillIDs(p.EL.ToStore()); !slices.Equal(got, []int{100, 103}) {
		t.Fatalf("stored = %v, want [100 103]", got)
	}
}

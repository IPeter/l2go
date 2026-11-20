package game

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// classDarkFighter is the Dark Elf fighter (Dark Elves know Shadow Sense).
const classDarkFighter = 31

// The fake clock starts at game 06:00; game midnight is 3 real hours later
// and the next 06:00 another 1.5 hours on.
const (
	toMidnight = 3 * time.Hour
	toDawn     = 90 * time.Minute
)

// systemMessages returns the system messages of id in ps.
func systemMessages(ps []testclient.Packet, id int32) []testclient.SystemMessage {
	var out []testclient.SystemMessage
	for _, m := range only[testclient.SystemMessage](ps) {
		if m.ID == id {
			out = append(out, m)
		}
	}
	return out
}

// At game midnight a Dark Elf who knows Shadow Sense is told it works now
// and gains its accuracy; at 06:00 it is told the effect is gone and loses
// it. Other players hear nothing.
func TestShadowSenseAtMidnightAndDawn(t *testing.T) {
	e := newSkillsEnv(t, false)
	c := e.createCharacter("dark", "Dark", classDarkFighter, storage.AccessLevelPlayer)
	c.Skills = []storage.CharacterSkill{{ID: shadowSenseSkill, Level: 1}}
	if err := e.store.Update(e.ctx, &c); err != nil {
		t.Fatalf("Update: %v", err)
	}
	e.createCharacter("human", "Human", classHumanFighter, storage.AccessLevelPlayer)
	dark, human := newSession("dark"), newSession("human")
	p := e.enter(dark, 0)
	e.enter(human, 0)
	dark.take(t)
	human.take(t)
	accuracy := func() int32 {
		var a int32
		e.onLoop(func() { a = p.stats.View().Accuracy })
		return a
	}
	day := accuracy()

	e.advance(toMidnight)
	ps := dark.take(t)
	msgs := systemMessages(ps, int32(sysMsgMidnightEffect))
	if len(msgs) != 1 || len(msgs[0].Params) != 1 || msgs[0].Params[0].Type != testclient.SMParamSkillName ||
		msgs[0].Params[0].Numbers[0] != shadowSenseSkill {
		t.Fatalf("at midnight: %+v; want one \"midnight\" message naming Shadow Sense", msgs)
	}
	if got := accuracy(); got != day+3 {
		t.Errorf("accuracy at night %d, want %d + 3", got, day)
	}
	if infos := only[testclient.UserInfo](ps); len(infos) == 0 || infos[len(infos)-1].Accuracy != day+3 {
		t.Errorf("no UserInfo with the night accuracy %d", day+3)
	}
	if got := systemMessages(human.take(t), int32(sysMsgMidnightEffect)); len(got) != 0 {
		t.Errorf("the human fighter got %+v", got)
	}

	e.advance(toDawn)
	ps = dark.take(t)
	if msgs := systemMessages(ps, int32(sysMsgDawnEffect)); len(msgs) != 1 {
		t.Fatalf("at dawn: %d \"dawn\" messages, want 1", len(msgs))
	}
	if got := accuracy(); got != day {
		t.Errorf("accuracy at dawn %d, want %d", got, day)
	}
}

// on_day_night_change runs with true at game midnight and false at 06:00,
// after the day/night spawns switched.
func TestDayNightChangeScriptEvent(t *testing.T) {
	near := world.Position{X: tiVillage.X + 100, Y: tiVillage.Y, Z: tiVillage.Z}
	e := newDialogEnv(t, dialogEnvOptions{scripts: map[string]string{
		"ai/day_night.lua": `
local s = script.new("DayNightTest")
s:on_day_night_change(function(is_night)
  local n = s:add_spawn(30001, { x = -83890, y = 243336, z = -3750 })
  if n then n:say(is_night and "night" or "day") end
end)`,
	}})
	sess, _ := e.enterChar(testChar{account: "alice", name: "Alice", pos: near})
	says := func() []string {
		var out []string
		for _, s := range only[testclient.NpcSay](sess.take(t)) {
			out = append(out, s.Text)
		}
		return out
	}
	e.advance(toMidnight - time.Second)
	if got := says(); len(got) != 0 {
		t.Fatalf("before midnight: %q", got)
	}
	e.advance(time.Second)
	if got := says(); len(got) != 1 || got[0] != "night" {
		t.Fatalf("at midnight: %q, want [night]", got)
	}
	e.advance(toDawn)
	if got := says(); len(got) != 1 || got[0] != "day" {
		t.Fatalf("at 06:00: %q, want [day]", got)
	}
}

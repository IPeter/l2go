package scripting_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

func loadErr(t *testing.T, files map[string]string) error {
	t.Helper()
	_, err := scripting.Check(context.Background(), scripting.Options{Dir: scripttest.WriteScripts(t, files)})
	if err == nil {
		t.Fatal("the scripts loaded, want an error")
	}
	return err
}

func TestQuestRegistration(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"quests/q.lua": `
local q = quest.new(7, "Q7", "Seven")
q:add_start_npc(100, { 101 }, 100)
q:add_talk_id(102, 102)
q:add_kill_id({ 200, 201 }, 200)
q:register_quest_items(5, 0, 6, 5)
q:requires("clans", "pets", "clans")
q:set_custom(true)
q:set_on_enter_world(true)
q:on_talk(function(npc, player) return "default-" .. npc:id() .. ".htm" end)
q:on_talk(103, function(npc, player) return "bound.htm" end)
local s = script.new("Plain")
s:on_talk(102, function() return "plain.htm" end)
assert(q:id() == 7 and q:name() == "Q7" and q:description() == "Seven" and q:is_custom())
assert(s:id() == -1)
assert(#q:get_registered_item_ids() == 2)
`})
	e := h.Engine()
	info, ok := e.Quest("Q7")
	want := scripting.QuestInfo{
		ID: 7, Name: "Q7", Description: "Seven", Custom: true, OnEnterWorld: true,
		Requires: []string{"clans", "pets"}, StartNpcs: []int{100, 101}, TalkNpcs: []int{100, 101, 102, 103},
		QuestItems: []int{5, 6}, Dir: info.Dir, RelDir: "quests",
	}
	if !ok || !slices.Equal(info.Requires, want.Requires) || !slices.Equal(info.StartNpcs, want.StartNpcs) ||
		!slices.Equal(info.TalkNpcs, want.TalkNpcs) || !slices.Equal(info.QuestItems, want.QuestItems) ||
		info.ID != 7 || !info.Custom || !info.OnEnterWorld || info.RelDir != "quests" {
		t.Errorf("Quest(Q7) = %+v, want %+v", info, want)
	}
	if qs := e.Quests(); len(qs) != 1 || qs[0].Name != "Q7" {
		t.Errorf("Quests() = %v; plain scripts are not quests", qs)
	}
	if got := e.TalkScripts(102); !slices.Equal(got, []string{"Q7", "Plain"}) {
		t.Errorf("TalkScripts(102) = %v", got)
	}
	if got := e.StartQuests(101); !slices.Equal(got, []string{"Q7"}) {
		t.Errorf("StartQuests(101) = %v", got)
	}
	if e.HasKill(201) {
		t.Error("HasKill is true for kill IDs without a kill handler")
	}

	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	for id, page := range map[int]string{100: "default-100.htm", 102: "default-102.htm", 103: "bound.htm"} {
		reply, ok := h.Talk("Q7", p, &scripttest.Npc{ObjectID: 2, ID: id})
		wantPage(t, "talk "+strconv.Itoa(id), reply, ok, page)
	}
	if reply, ok := h.Talk("Q7", p, &scripttest.Npc{ObjectID: 2, ID: 104}); ok {
		t.Errorf("an unregistered NPC answered %+v", reply)
	}
}

func TestQuestRegistrationErrors(t *testing.T) {
	for name, tt := range map[string]struct {
		files map[string]string
		want  string
	}{
		"quest id twice": {map[string]string{"a.lua": `quest.new(1, "A", "") quest.new(1, "B", "")`}, "quest ID 1 is already used by A"},
		"bad quest id":   {map[string]string{"a.lua": `quest.new(0, "A", "")`}, "the quest ID must be a positive integer"},
		"bad feature":    {map[string]string{"a.lua": `quest.new(1, "A", ""):requires("flying")`}, `unknown feature "flying"`},
		"first talk owned": {map[string]string{
			"a.lua": `quest.new(1, "A", ""):add_first_talk_id(5)`,
			"b.lua": `quest.new(2, "B", ""):add_first_talk_id(6, 5)`,
		}, "NPC 5 already has a first-talk handler, from script A"},
		"handler twice":    {map[string]string{"a.lua": "local q = quest.new(1, \"A\", \"\")\nq:on_kill(function() end)\nq:on_kill(function() end)"}, "already has a kill handler"},
		"bad id":           {map[string]string{"a.lua": `quest.new(1, "A", ""):add_kill_id("x")`}, "NPC ID or list of NPC IDs expected"},
		"world at load":    {map[string]string{"a.lua": `time.now_ms()`}, "do it in q:on_start"},
		"unknown constant": {map[string]string{"a.lua": `local x = State.FINISHED`}, "unknown constant State.FINISHED"},
		"constant write":   {map[string]string{"a.lua": `QuestSound.X = 1`}, "QuestSound is read-only"},
		"randomseed gone":  {map[string]string{"a.lua": `if math.randomseed == nil then error("randomseed is gone") end`}, "randomseed is gone"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := loadErr(t, tt.files); !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

// The same object is the same Lua value within a run (Java identity).
func TestRefIdentity(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"q.lua": `
local q = quest.new(5, "Q5", "")
q:add_talk_id(30001)
q:on_talk(function(npc, player)
  local st = q:get_quest_state(player, true)
  local checks = {
    st == q:get_quest_state(player, false),
    st == player:get_quest_state("Q5"),
    st:get_player() == player,
    world.find_object(npc:object_id()) == npc,
    world.get_player(player:object_id()) == player,
    world.find_object(12345) == nil,
    world.get_player(npc:object_id()) == nil,
  }
  local out = {}
  for i, c in ipairs(checks) do out[i] = tostring(c) end
  return table.concat(out, ",")
end)`})
	reply, _ := h.Talk("Q5", &scripttest.Player{ObjectID: 1, Name: "Hero"}, &scripttest.Npc{ObjectID: 2, ID: 30001})
	if reply.Content != "true,true,true,true,true,true,true" {
		t.Errorf("identity checks = %q", reply.Content)
	}
}

// QuestState methods over the fake model, Mobius rules included.
func TestQuestStateMethods(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"q.lua": `
local q = quest.new(5, "Q5", "")
q:register_quest_items(10)
q:on_event(function(event, npc, player)
  local st = q:get_quest_state(player, true)
  local out = {}
  local function add(v) out[#out + 1] = tostring(v) end
  if event == "basics" then
    add(st:get_state()) add(st:is_created()) add(st:get_cond())
    st:set_cond(3) -- ignored before the start
    add(st:get_cond())
    st:start_quest()
    add(st:is_started()) add(st:get_cond()) add(st:is_cond(1))
    st:set("count", 5) add(st:get("count")) add(st:get_int("count")) add(st:is_set("count"))
    st:set("word", "x") add(st:get_int("word"))
    st:unset("count") add(st:get("count")) add(st:is_set("count"))
    st:set_cond(3, true) add(st:get("__compltdStateFlags"))
    st:set_memo_state(4) add(st:get_memo_state()) add(st:is_memo_state(4))
    st:set_memo_state_ex(2, 9) add(st:get_memo_state_ex(2)) add(st:is_memo_state_ex(2, 9))
    add(st:get_quest_name())
  elseif event == "exit" then
    st:start_quest()
    q:give_items(player, 10, 3)
    st:exit_quest(QuestType.DAILY, true)
    add(st:is_completed()) add(q:get_quest_items_count(player, 10)) add(st:get("restartTime"))
    add(st:is_now_available())
  elseif event == "available" then
    add(st:is_now_available())
  end
  return table.concat(out, ",")
end)`})
	w := h.World()
	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	reply, _ := h.Event("Q5", "basics", p, nil)
	want := strings.Join([]string{
		"0", "true", "0", "0", "true", "1", "true", "5", "5", "true", "0", "nil", "false",
		"-2147483643", // 0x80000001 | 1 | 4: step 2 was skipped
		"4", "true", "9", "true", "Q5",
	}, ",")
	if reply.Content != want {
		t.Errorf("basics:\n got %s\nwant %s", reply.Content, want)
	}
	if got := w.SoundsOf(p); !slices.Equal(got, []string{soundAccept, soundMiddle}) {
		t.Errorf("sounds = %v", got)
	}

	// Daily quests restart at 06:30 server time: Start is 05:00, so today.
	p2 := &scripttest.Player{ObjectID: 2, Name: "Daily"}
	reply, _ = h.Event("Q5", "exit", p2, nil)
	restart := scripttest.Start.Add(90 * time.Minute).UnixMilli()
	if want := "true,0," + strconv.FormatInt(restart, 10) + ",false"; reply.Content != want {
		t.Errorf("exit = %q, want %q", reply.Content, want)
	}
	h.Advance(90 * time.Minute)
	if reply, _ := h.Event("Q5", "available", p2, nil); reply.Content != "true" {
		t.Errorf("available at 06:30 = %q", reply.Content)
	}
}

// Quest timers: Mobius keys, one-shot removal before the handler, repeats,
// cancellation and the reply to the timer's player.
func TestQuestTimers(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"q.lua": `
local q = quest.new(5, "Q5", "")
local fired = {}
q:on_event(function(event, npc, player)
  if event == "start" then
    q:start_quest_timer("once", 1000, npc, player)
    q:start_quest_timer("once", 5000, npc, player) -- same key: ignored
    q:start_quest_timer("once", 2000, nil, player) -- npc nil: another key
    q:start_quest_timer("tick", 500, nil, nil, true)
    return tostring(q:get_quest_timer("once", npc, player) ~= nil) .. "," .. tostring(q:get_quest_timer("once", nil, nil) ~= nil)
  elseif event == "once" then
    fired[#fired + 1] = "once:" .. tostring(npc ~= nil)
    if npc ~= nil then
      -- removed before this handler: it can start again
      fired[#fired + 1] = "again:" .. tostring(q:get_quest_timer("once", npc, player) == nil)
    end
    return "timer.htm"
  elseif event == "tick" then
    fired[#fired + 1] = "tick"
    if #fired > 6 then q:cancel_quest_timers("tick") end
  elseif event == "fired" then
    return table.concat(fired, " ")
  elseif event == "long" then
    q:start_quest_timer("long", 60000, npc, player)
    q:start_quest_timer("long", 60000, nil, player)
  end
end)`})
	w := h.World()
	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	n := &scripttest.Npc{ObjectID: 2, ID: 30001}
	if reply, _ := h.Event("Q5", "start", p, n); reply.Content != "true,false" {
		t.Errorf("start = %q", reply.Content)
	}
	h.Advance(10 * time.Second)
	reply, _ := h.Event("Q5", "fired", p, n)
	if want := "tick once:true again:true tick tick once:false tick"; reply.Content != want {
		t.Errorf("fired = %q, want %q", reply.Content, want)
	}
	// The one-shot replies went to the player, with the NPC when there was one.
	if len(w.Replies) != 2 || w.Replies[0].Npc != 2 || w.Replies[1].Npc != 0 || w.Replies[0].Reply.Content != "timer.htm" {
		t.Errorf("replies = %+v", w.Replies)
	}
	if w.Pending() != 0 {
		t.Errorf("%d timers left", w.Pending())
	}

	// A player leaving and an NPC going cancel their timers.
	h.Event("Q5", "long", p, n)
	h.Engine().NpcGone(2)
	if w.Pending() != 1 {
		t.Errorf("after NpcGone: %d timers, want 1", w.Pending())
	}
	h.Engine().PlayerLeft(1)
	if w.Pending() != 0 {
		t.Errorf("after PlayerLeft: %d timers, want 0", w.Pending())
	}
	// A reload cancels every timer.
	h.Event("Q5", "long", p, n)
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	if w.Pending() != 0 {
		t.Errorf("after a reload: %d timers, want 0", w.Pending())
	}
}

// rnd follows Mobius Rnd through the engine's source; math.random too.
func TestRnd(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"q.lua": `
local s = script.new("R")
s:on_event(function(event)
  local out = {}
  local function add(v) out[#out + 1] = tostring(v) end
  add(rnd(0)) add(rnd(-5)) add(rnd(5, 5)) add(rnd(9, 2)) -- no draws
  add(rnd(10)) add(rnd(1, 6)) add(rnd.bool()) add(rnd.double()) add(rnd.double(100))
  add(rnd.entry({ "a", "b", "c" })) add(rnd.entry({}))
  add(math.random(6)) add(math.random(3, 4)) add(math.random())
  return table.concat(out, ",")
end)`})
	h.SetRand(rnd.NewScripted(t,
		rnd.Get(10, 7), rnd.Between(1, 6, 6), rnd.Bool(true), rnd.Float64(0.5), rnd.GetFloat64(100, 42.5),
		rnd.Get(3, 1), rnd.Get(6, 0), rnd.Between(3, 4, 4), rnd.Float64(0.25),
	))
	reply, _ := h.Event("R", "x", &scripttest.Player{ObjectID: 1}, nil)
	if want := "0,0,5,9,7,6,true,0.5,42.5,b,nil,1,4,0.25"; reply.Content != want {
		t.Errorf("rnd = %q, want %q", reply.Content, want)
	}
}

func TestConstants(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"q.lua": `
local s = script.new("C")
s:on_event(function(event)
  return table.concat({
    State.CREATED, State.STARTED, State.COMPLETED, QuestType.DAILY, QuestSound.ITEMSOUND_QUEST_ACCEPT,
    Paperdoll.RHAND, ChatType.NPC_GENERAL, Race.DARK_ELF, tostring(Config.ALT_PARTY_RANGE),
    ClassId.FIGHTER, ClassId.WARRIOR, ClassId.level(ClassId.WARRIOR), ClassId.race(ClassId.ELVEN_FIGHTER),
    tostring(ClassId.parent(ClassId.WARRIOR)), tostring(ClassId.parent(ClassId.FIGHTER)),
    tostring(ClassId.is_mage(ClassId.MAGE)), quest.DF_REWARD_37[4],
  }, ",")
end)`})
	reply, _ := h.Event("C", "x", &scripttest.Player{ObjectID: 1}, nil)
	if want := "0,1,2,2,ItemSound.quest_accept,7,0,2,1500,0,1,1,1,0,nil,true,50"; reply.Content != want {
		t.Errorf("constants = %q, want %q", reply.Content, want)
	}
}

// give_item_randomly draws as Mobius: the amount (when it varies), then the
// chance.
func TestGiveItemRandomly(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"q.lua": `
local q = quest.new(5, "Q5", "")
q:on_event(function(event, npc, player)
  return tostring(q:give_item_randomly(player, npc, 100, 1, 3, 5, 0.5, true))
end)`})
	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	h.SetRand(rnd.NewScripted(t,
		rnd.Between(1, 3, 3), rnd.Float64(0.4), // 3 given, limit not reached
		rnd.Between(1, 3, 3), rnd.Float64(0.9), // chance missed
		rnd.Between(1, 3, 3), rnd.Float64(0.1), // capped at the limit of 5
	))
	for i, want := range []string{"false", "false", "true", "true"} {
		if reply, _ := h.Event("Q5", "x", p, nil); reply.Content != want {
			t.Errorf("call %d = %q, want %s", i, reply.Content, want)
		}
	}
	if p.Items[100] != 5 {
		t.Errorf("items = %d, want 5", p.Items[100])
	}
	if got := h.World().SoundsOf(p); !slices.Equal(got, []string{soundItemGet, soundMiddle}) {
		t.Errorf("sounds = %v", got)
	}
}

// The world events reach the scripts registered for the ID, in load order,
// with their arguments.
func TestWorldEvents(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{
		"a.lua": `
local q = quest.new(1, "A", "")
q:add_kill_id(500)
q:add_attack_id(500)
q:add_spawn_id(500)
q:add_item_talk_id(57)
q:add_enter_zone_id(9)
q:set_on_enter_world(true)
q:on_kill(function(npc, killer, is_summon) globals.set("kill", "A:" .. npc:id() .. ":" .. killer:name() .. ":" .. tostring(is_summon)) end)
q:on_attack(function(npc, attacker, damage, is_summon, skill) globals.set("attack", damage .. ":" .. tostring(skill)) end)
q:on_spawn(function(npc) globals.set("spawn", npc:object_id()) end)
q:on_item_talk(function(item, player) return "item-" .. item:id() .. "-" .. item:count() .. ".htm" end)
q:on_enter_zone(function(creature, zone) globals.set("zone", creature:name() .. "@" .. zone:id() .. ":" .. zone:type()) end)
q:on_enter_world(function(player) globals.set("enter", player:name()) end)
q:on_start(function() globals.set("started", globals.get("started", "") .. "A") end)`,
		"b.lua": `
local q = quest.new(2, "B", "")
q:on_kill(500, function(npc, killer) globals.set("kill", globals.get("kill") .. "+B") end)
q:on_start(function() globals.set("started", globals.get("started", "") .. "B") end)`,
	})
	w := h.World()
	h.Start()
	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	n := &scripttest.Npc{ObjectID: 2, ID: 500}
	h.Kill(p, n)
	h.Attack(p, n, 77)
	ctx := context.Background()
	if err := h.Engine().Spawn(ctx, n.Ref()); err != nil {
		t.Fatal(err)
	}
	if err := h.Engine().EnterZone(ctx, p.Ref(), scripting.Zone{ID: 9, Type: "peace"}); err != nil {
		t.Fatal(err)
	}
	h.EnterWorld(p)
	replies := h.Engine().ItemTalk(ctx, scripting.Item{ObjectID: 3, ID: 57, Count: 10}, p.Ref())
	if len(replies) != 1 || replies[0].Content != "item-57-10.htm" {
		t.Errorf("item talk = %+v", replies)
	}
	for k, want := range map[string]string{
		"kill": "A:500:Hero:false+B", "attack": "77:nil", "spawn": "2", "zone": "Hero@9:peace",
		"enter": "Hero", "started": "AB",
	} {
		if got, _ := w.Global(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	// A reload runs on_start again.
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	if got, _ := w.Global("started"); got != "ABAB" {
		t.Errorf("started after reload = %q", got)
	}
}

// A failing world event handler is logged and returned; the next script
// still runs.
func TestWorldEventErrors(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{
		"a.lua": `quest.new(1, "A", ""):on_kill(500, function() error("boom") end)`,
		"b.lua": `quest.new(2, "B", ""):on_kill(500, function() globals.set("b", "ran") end)`,
	})
	err := h.Engine().Kill(context.Background(), (&scripttest.Npc{ObjectID: 2, ID: 500}).Ref(), (&scripttest.Player{ObjectID: 1}).Ref(), false)
	var se *scripting.Error
	if !errors.As(err, &se) || !strings.Contains(se.Msg, "boom") {
		t.Errorf("err = %v", err)
	}
	if got, _ := h.World().Global("b"); got != "ran" {
		t.Error("the second script did not run")
	}
}

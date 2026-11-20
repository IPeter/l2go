package scripting_test

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The scripted AI API of Plan 13, Phase 10.6.

func said(h *scripttest.Harness) []string {
	var out []string
	for _, s := range h.Said() {
		out = append(out, s.Text)
	}
	return out
}

// Asynchronous events raised while a handler runs wait for it, then run in
// order; on_npc_hate runs at once.
func TestAIEventsQueue(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
s:add_event_received_id(30001, 30002)
s:add_spawn_id(30003)
s:on_event(function(event, npc, player)
  npc:say("before")
  npc:broadcast_event("HELLO", 500, player)
  s:add_spawn(30003, npc)
  npc:say("after")
  return ""
end)
s:on_event_received(function(event, sender, receiver, reference)
  receiver:say(event .. " from " .. sender:object_id() .. " for " .. reference:name())
  if receiver:id() == 30001 then
    receiver:send_event("AGAIN", sender, nil)
  end
end)
s:on_spawn(function(npc) npc:say("spawned " .. npc:id()) end)
`})
	h.SpawnEvents(true)
	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	a := &scripttest.Npc{ObjectID: 10, ID: 30002}
	b := &scripttest.Npc{ObjectID: 11, ID: 30001, X: 100}
	c := &scripttest.Npc{ObjectID: 12, ID: 30002, X: 200}
	h.World().Add(b, c)
	h.Event("A", "go", p, a)
	want := []string{
		"before", "after",
		"HELLO from 10 for Hero", "HELLO from 10 for Hero", "spawned 30003",
		"AGAIN from 11 for nil",
	}
	got := said(h)
	// AGAIN has no reference: the handler fails on reference:name(); the
	// others ran.
	if len(got) != 5 || !slices.Equal(got, want[:5]) {
		t.Errorf("said = %q, want %q", got, want[:5])
	}
	if h.Engine().QueuedEvents() != 0 {
		t.Errorf("%d events left in the queue", h.Engine().QueuedEvents())
	}
}

// Each queued event has its own time limit: one that loops forever does
// not stop the next.
func TestAIEventsTimeoutEach(t *testing.T) {
	h2 := scripttest.LoadSource(t, map[string]string{"b.lua": `
local s = script.new("B")
s:add_event_received_id(30001, 30002)
s:on_event(function(event, npc, player)
  npc:broadcast_event("GO", 1000, nil)
  return ""
end)
s:on_event_received(function(event, sender, receiver)
  if receiver:id() == 30001 then
    while true do end
  end
  receiver:say("done")
end)
`})
	loop := &scripttest.Npc{ObjectID: 11, ID: 30001, X: 10}
	ok := &scripttest.Npc{ObjectID: 12, ID: 30002, X: 20}
	h2.World().Add(loop, ok)
	start := time.Now()
	h2.Event("B", "go", &scripttest.Player{ObjectID: 1}, &scripttest.Npc{ObjectID: 10, ID: 30000})
	if got := said(h2); !slices.Equal(got, []string{"done"}) {
		t.Errorf("said = %q", got)
	}
	if time.Since(start) > 3*time.Second {
		t.Errorf("took %v", time.Since(start))
	}
	timedOut := false
	for _, l := range h2.Logs() {
		if l.Level == slog.LevelWarn && strings.Contains(l.Msg, "failed") {
			timedOut = true
		}
	}
	if !timedOut {
		t.Errorf("no handler failure logged")
	}
}

func TestNpcHateVeto(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
s:on_npc_hate(30001, function(npc, player, is_summon)
  npc:say("asked")
  return player:level() >= 10
end)
local t = script.new("B")
t:add_npc_hate_id(30001)
t:on_npc_hate(function(npc, player) npc:say("second") end)
`})
	n := &scripttest.Npc{ObjectID: 2, ID: 30001}
	if h.NpcHate(n, &scripttest.Player{ObjectID: 1, Level: 5}) {
		t.Errorf("level 5: hate allowed")
	}
	if got := said(h); !slices.Equal(got, []string{"asked"}) {
		t.Errorf("after a veto: said %q, want the second script skipped", got)
	}
	h.ClearSaid()
	if !h.NpcHate(n, &scripttest.Player{ObjectID: 1, Level: 20}) {
		t.Errorf("level 20: hate vetoed")
	}
	if got := said(h); !slices.Equal(got, []string{"asked", "second"}) {
		t.Errorf("said %q", got)
	}
	if !h.NpcHate(&scripttest.Npc{ObjectID: 3, ID: 30009}, &scripttest.Player{ObjectID: 1}) {
		t.Errorf("an NPC without handlers: hate vetoed")
	}
}

// The other AI events reach their handlers with Mobius's arguments.
func TestAIEventArguments(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
local ID = 30001
s:on_faction_call(ID, function(npc, caller, attacker, is_summon)
  npc:say("faction " .. caller:object_id() .. " " .. attacker:name() .. " " .. tostring(is_summon))
end)
s:on_creature_see(ID, function(npc, creature) npc:say("see " .. creature:name()) end)
s:on_move_finished(ID, function(npc) npc:say("move") end)
s:on_node_arrived(ID, function(npc) npc:say("node") end)
s:on_route_finished(ID, function(npc) npc:say("route") end)
s:on_teleport(ID, function(npc) npc:say("teleport") end)
s:on_day_night_change(function(is_night) world.find_object(2):say("night " .. tostring(is_night)) end)
local other = script.new("B")
other:on_day_night_change(function(is_night) world.find_object(2):say("B " .. tostring(is_night)) end)
`})
	n := &scripttest.Npc{ObjectID: 2, ID: 30001}
	caller := &scripttest.Npc{ObjectID: 3, ID: 30002}
	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	h.FactionCall(n, caller, p)
	h.CreatureSee(n, p)
	h.MoveFinished(n)
	h.NodeArrived(n)
	h.RouteFinished(n)
	if err := h.Engine().Teleported(t.Context(), n.Ref()); err != nil {
		t.Fatal(err)
	}
	h.DayNightChange(true)
	want := []string{"faction 3 Hero false", "see Hero", "move", "node", "route", "teleport", "night true", "B true"}
	if got := said(h); !slices.Equal(got, want) {
		t.Errorf("said =\n%q\nwant\n%q", got, want)
	}
	e := h.Engine()
	for _, c := range []struct {
		event string
		id    int
		want  bool
	}{
		{"on_move_finished", 30001, true}, {"move_finished", 30002, false}, {"day_night_change", 0, true},
		{"npc_hate", 30001, false}, {"faction_call", 30001, true}, {"no_such_event", 30001, false},
	} {
		if got := e.Listens(c.event, c.id); got != c.want {
			t.Errorf("Listens(%s, %d) = %v", c.event, c.id, got)
		}
	}
}

// Npc values are handles: kept across handlers, invalid once the NPC is
// gone, usable until the end of the run it left in.
func TestNpcHandles(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
local kept
s:on_event(function(event, npc, player)
  if event == "keep" then
    kept = npc
    return ""
  end
  if event == "same" then
    return tostring(kept == npc) .. " " .. tostring(kept:valid()) .. " " .. kept:name()
  end
  if event == "valid" then
    return tostring(kept:valid())
  end
  if event == "use" then
    return kept:name()
  end
  if event == "delete" then
    npc:delete()
    return tostring(npc:valid()) .. " " .. npc:name()
  end
end)
`})
	p := &scripttest.Player{ObjectID: 1}
	guard := &scripttest.Npc{ObjectID: 2, ID: 30001, Name: "Guard"}
	other := &scripttest.Npc{ObjectID: 3, ID: 30002, Name: "Other"}
	h.Event("A", "keep", p, guard)
	if r, _ := h.Event("A", "same", p, guard); r.Content != "true true Guard" {
		t.Errorf("same = %q", r.Content)
	}
	if r, _ := h.Event("A", "use", p, other); r.Content != "Guard" {
		t.Errorf("use from another run = %q", r.Content)
	}
	h.Despawn(guard)
	if r, _ := h.Event("A", "valid", p, other); r.Content != "false" {
		t.Errorf("valid after despawn = %q", r.Content)
	}
	r, ok := h.Engine().Event(t.Context(), "A", "use", p.Ref(), other.Ref())
	if ok || r.Err == nil || !strings.Contains(r.Err.Error(), "npc:valid()") {
		t.Errorf("use after despawn = %+v, %v", r, ok)
	}
	if r, _ := h.Event("A", "delete", p, other); r.Content != "false Other" {
		t.Errorf("delete = %q", r.Content)
	}
	// A new NPC with the same object ID but another template is not the
	// kept one.
	h.Event("A", "keep", p, &scripttest.Npc{ObjectID: 4, ID: 30003, Name: "Old"})
	h.World().Add(&scripttest.Npc{ObjectID: 4, ID: 30004, Name: "New"})
	if r, _ := h.Event("A", "valid", p, other); r.Content != "false" {
		t.Errorf("reused object ID: valid = %q", r.Content)
	}
}

// A boss script reloaded mid-fight finds its boss instead of spawning a
// second one, and its timers run once, from the new scripts.
func TestReloadBossMidFight(t *testing.T) {
	src := `
local s = script.new("Boss")
local BOSS = 29001
s:on_start(function(ev)
  local boss = grandboss.boss(BOSS)
  if ev.reload and boss ~= nil then
    boss:say("re-armed")
  else
    boss = s:add_spawn(BOSS, { x = 1, y = 2, z = 3 })
    grandboss.add_boss(boss)
    grandboss.set_status(BOSS, 2)
  end
  s:start_quest_timer("action", 10000, boss, nil, true)
end)
s:on_event(function(event, npc)
  if event == "action" then
    npc:say("VERSION")
  end
end)
`
	dir := scripttest.WriteScripts(t, map[string]string{"boss.lua": strings.ReplaceAll(src, "VERSION", "v1")})
	h := scripttest.Load(t, dir)
	h.Start()
	if n := len(h.World().Spawned); n != 1 {
		t.Fatalf("%d spawns", n)
	}
	h.Advance(10 * time.Second)
	scripttest.WriteScript(t, dir, "boss.lua", strings.ReplaceAll(src, "VERSION", "v2"))
	h.Advance(5 * time.Second)
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	h.Advance(10 * time.Second)
	if n := len(h.World().Spawned); n != 1 {
		t.Errorf("%d spawns after the reload, want 1", n)
	}
	want := []string{"v1", "re-armed", "v2"}
	if got := said(h); !slices.Equal(got, want) {
		t.Errorf("said = %q, want %q", got, want)
	}
	if st, _ := h.World().BossStatus(29001); st != 2 {
		t.Errorf("status = %d", st)
	}
}

// The NPC control methods reach the world's NpcControl.
func TestNpcControlMethods(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
s:on_event(function(event, npc, player)
  if event == "read" then
    local parts = {}
    for _, e in ipairs(npc:hate_list()) do
      parts[#parts + 1] = e.target:object_id() .. ":" .. e.hate .. ":" .. e.damage
    end
    return table.concat({
      tostring(npc:is_running()), tostring(npc:is_core_ai_disabled()), tostring(npc:is_invul()),
      tostring(npc:is_mortal()), npc:intention(), tostring(npc:is_moving()), tostring(npc:is_in_combat()),
      tostring(npc:is_casting_now()), table.concat(parts, ","), npc:most_hated():object_id(),
    }, " ")
  end
  npc:set_walking()
  npc:disable_core_ai()
  npc:set_immobilized(true)
  npc:set_invul(true)
  npc:set_mortal(false)
  npc:set_random_walking(false)
  npc:set_heading(1234)
  npc:set_intention(Intention.MOVE_TO, { x = 5, y = 6, z = 7 })
  npc:follow(player)
  npc:stop_follow()
  npc:stop_moving()
  npc:add_damage_hate(player, 10, 100)
  npc:reduce_hate(player, 40)
  npc:stop_hating(player)
  npc:add_damage_hate(player, 1, 5)
  npc:clear_aggro_list()
  npc:return_home()
  return ""
end)
`})
	p := &scripttest.Player{ObjectID: 1}
	home := scripting.Location{X: 9, Y: 9}
	n := &scripttest.Npc{ObjectID: 2, ID: 30001, Running: true, Spawn: &home}
	h.Event("A", "go", p, n)
	if n.Running || !n.Walking || !n.CoreAIDisabled || !n.Immobilized || !n.Invul || !n.Immortal || !n.NoRandomWalk || n.Heading != 1234 {
		t.Errorf("npc = %+v", n)
	}
	if n.Dest == nil || *n.Dest != home || len(n.Hate) != 0 {
		t.Errorf("after return_home: dest %v, hate %v", n.Dest, n.Hate)
	}
	kinds := []string{}
	for _, a := range h.World().Actions {
		kinds = append(kinds, a.Kind)
	}
	want := []string{"walk", "core_ai", "immobilized", "invul", "mortal", "random_walk", "heading", "intention", "follow",
		"stop_follow", "stop_moving", "hate", "reduce_hate", "stop_hating", "hate", "clear_aggro", "return_home"}
	if !slices.Equal(kinds, want) {
		t.Errorf("actions = %q", kinds)
	}

	other := &scripttest.Npc{ObjectID: 3, ID: 30002}
	n.Hate = []scripttest.Hate{{Target: other, Hate: 50, Damage: 5}, {Target: p, Hate: 20, Damage: 7}}
	n.Running, n.Intention, n.InCombat = true, "ATTACK", true
	h.World().Add(other)
	r, _ := h.Event("A", "read", p, n)
	if want := "true true true false ATTACK true true false 3:50:5,1:20:7 3"; r.Content != want {
		t.Errorf("read = %q, want %q", r.Content, want)
	}
}

// NPC variables live on the engine: kept across handlers and reloads,
// gone with the NPC.
func TestNpcVariables(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
s:on_event(function(event, npc)
  if event == "set" then
    npc:set_var("state", 2)
    npc:set_var("flag", true)
    npc:set_var("name", "x")
    npc:set_var("gone", 1)
    npc:unset_var("gone")
    return ""
  end
  return table.concat({ npc:var_int("state"), npc:var_int("missing", 7), tostring(npc:var_bool("flag")),
    tostring(npc:var("name")), npc:var("missing", "d"), tostring(npc:has_var("gone")), tostring(npc:has_var("state")) }, " ")
end)
`})
	n := &scripttest.Npc{ObjectID: 2, ID: 30001}
	h.Event("A", "set", nil, n)
	const want = "2 7 true x d false true"
	if r, _ := h.Event("A", "get", nil, n); r.Content != want {
		t.Errorf("get = %q", r.Content)
	}
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	if r, _ := h.Event("A", "get", nil, n); r.Content != want {
		t.Errorf("after reload = %q", r.Content)
	}
	h.Despawn(n)
	n2 := &scripttest.Npc{ObjectID: 2, ID: 30001}
	if r, _ := h.Event("A", "get", nil, n2); r.Content != "0 7 false nil d false false" {
		t.Errorf("respawned = %q", r.Content)
	}
}

func TestNpcSpawnDataParamsAndMinions(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
s:on_event(function(event, npc)
  if event == "minion" then
    local m = npc:add_minion(30010)
    return m:leader():object_id() .. " " .. #npc:minions() .. " " .. tostring(npc:has_minions())
  end
  local sp = npc:spawn_location()
  return table.concat({ sp.x, sp.y, tostring(npc:stays_in_spawn_loc()), npc:ai_value("fleeX"), tostring(npc:ai_value("none")),
    tostring(npc:has_ai_value("fleeX")), npc:param("Skill01_ID", "?"), npc:param_int("Count", 3), npc:param_int("Missing", 4) }, " ")
end)
`})
	sp := scripting.Location{X: 10, Y: 20}
	n := &scripttest.Npc{ObjectID: 2, ID: 30001, X: 10, Y: 20, Spawn: &sp,
		AIData: map[string]int{"fleeX": 99}, Params: map[string]string{"Skill01_ID": "4001", "Count": "5"}}
	if r, _ := h.Event("A", "data", nil, n); r.Content != "10 20 true 99 nil true 4001 5 4" {
		t.Errorf("data = %q", r.Content)
	}
	if r, _ := h.Event("A", "minion", nil, n); r.Content != "2 1 true" {
		t.Errorf("minion = %q", r.Content)
	}
	if len(n.Minions) != 1 || n.Minions[0].ID != 30010 {
		t.Errorf("minions = %+v", n.Minions)
	}
}

func TestWorldScanModule(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
local function ids(list)
  local out = {}
  for _, o in ipairs(list) do out[#out + 1] = o:object_id() end
  return table.concat(out, ",")
end
s:on_event(function(event, npc, player)
  local v = world.valid_location(npc, { x = 100, y = 0, z = 0 })
  return table.concat({ ids(world.npcs_by_id(30002)), ids(world.npcs_around(npc, 150)), ids(world.players_around(npc)),
    ids(world.visible_objects(npc, "Monster", 1000)), ids(world.visible_objects(npc, "Creature", 1000)),
    v.x, string.format("%.0f", util.calculate_angle_from(npc, { x = 0, y = -10 })),
    string.format("%.0f", npc:calculate_distance_2d(player)) }, " ")
end)
`})
	p := &scripttest.Player{ObjectID: 1, X: 30, Y: 40}
	n := &scripttest.Npc{ObjectID: 2, ID: 30001}
	h.World().Add(
		&scripttest.Npc{ObjectID: 3, ID: 30002, X: 100},
		&scripttest.Npc{ObjectID: 4, ID: 30002, X: 900, Types: []string{"Monster"}},
		&scripttest.Npc{ObjectID: 5, ID: 30003, X: 5000},
	)
	h.World().SetGeo(func(from, to scripting.Location) scripting.Location { to.X /= 2; return to })
	r, _ := h.Event("A", "x", p, n)
	if want := "3,4 3 1 4 1,3,4 50 270 50"; r.Content != want {
		t.Errorf("scan = %q, want %q", r.Content, want)
	}
}

func TestZonesDoorsAndGameTime(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
s:on_event(function(event, npc, player)
  if event == "zone" then
    local n = #zones.players_inside(12001)
    local inside = tostring(zones.is_inside(12001, player)) .. tostring(player:is_inside_zone(12001)) .. tostring(npc:is_inside_zone(12001))
    zones.allow_entry(12001, player, 30)
    zones.remove_player(12001, player)
    zones.move_players(12001, { x = 1, y = 2, z = 3 })
    zones.oust_all(12001)
    return n .. " " .. inside
  end
  if event == "door" then
    local a = doors.open(17220001)
    local open = doors.is_open(17220001)
    local b = doors.close(17220001)
    return table.concat({ tostring(a), tostring(open), tostring(b), tostring(doors.is_open(17220001)),
      tostring(doors.open(1)), tostring(doors.is_open(1)) }, " ")
  end
  return table.concat({ time.game_time(), time.game_hour(), tostring(time.is_night()), time.game_ticks() }, " ")
end)
`})
	p := &scripttest.Player{ObjectID: 1}
	n := &scripttest.Npc{ObjectID: 2, ID: 30001}
	z := h.World().AddZone(12001)
	z.Players = []*scripttest.Player{p}
	z.Oust = scripting.Location{X: 7, Y: 8, Z: 9}
	if r, _ := h.Event("A", "zone", p, n); r.Content != "1 truetruefalse" {
		t.Errorf("zone = %q", r.Content)
	}
	if p.X != 7 || len(z.Players) != 0 || len(z.Allowed) != 0 {
		t.Errorf("after oust: player at %d, zone %+v", p.X, z)
	}
	h.World().AddDoor(17220001, false)
	if r, _ := h.Event("A", "door", p, n); r.Content != "true true true false false nil" {
		t.Errorf("door = %q", r.Content)
	}
	h.World().SetGameTime(5*60 + 59)
	if r, _ := h.Event("A", "time", p, n); r.Content != "359 5 true 35900" {
		t.Errorf("time = %q", r.Content)
	}
	h.Advance(10 * time.Second)
	if r, _ := h.Event("A", "time", p, n); r.Content != "360 6 false 36000" {
		t.Errorf("time at 06:00 = %q", r.Content)
	}
}

func TestGrandBossModule(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
s:on_event(function(event, npc)
  if event == "set" then
    grandboss.set_status(29001, 1)
    grandboss.set_info(29001, { respawn_time = 123456789, x = 1, y = 2, z = 3, heading = 4, hp = 10.5, mp = 20 })
    grandboss.add_boss(npc)
    return ""
  end
  local info = grandboss.info(29001)
  local boss = grandboss.boss(29001)
  return table.concat({ grandboss.status(29001), info.respawn_time, info.x, info.hp, info.mp,
    tostring(boss and boss:object_id()), tostring(grandboss.status(1)), tostring(grandboss.info(1)) }, " ")
end)
`})
	queen := &scripttest.Npc{ObjectID: 2, ID: 29001}
	h.Event("A", "set", nil, queen)
	if r, _ := h.Event("A", "get", nil, queen); r.Content != "1 123456789 1 10.5 20 2 nil nil" {
		t.Errorf("get = %q", r.Content)
	}
	h.Despawn(queen)
	if r, _ := h.Event("A", "get", nil, &scripttest.Npc{ObjectID: 3, ID: 1}); !strings.Contains(r.Content, " 20 nil ") {
		t.Errorf("after the boss left = %q", r.Content)
	}
	if b := h.World().GrandBoss(29001); b.Saves != 2 {
		t.Errorf("saves = %d", b.Saves)
	}
}

func TestCinematicsModule(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
s:on_event(function(event, npc, player)
  cinematics.special_camera(npc, npc, 1200, 180, -5, 0, 10000, 15000, 0, 0, 1, 0)
  cinematics.earthquake(12001, npc, 20, 10)
  cinematics.sound(player, "BS02_A", npc)
  cinematics.screen_message(player, "Hello", 5000)
  npc:magic_skill_use(player, 4339, 1, 6000, 0)
  return ""
end)
`})
	p := &scripttest.Player{ObjectID: 1}
	n := &scripttest.Npc{ObjectID: 2, ID: 29001, X: 5, Y: 6, Z: 7}
	h.Event("A", "x", p, n)
	got := h.World().Cinematics()
	if len(got) != 4 {
		t.Fatalf("cinematics = %+v", got)
	}
	cam := scripting.Camera{Force: 1200, Angle1: 180, Angle2: -5, Range: 10000, Duration: 15000, Wide: 1}
	if got[0].Kind != "camera" || got[0].Camera != cam || got[0].Target != 2 || got[0].To.Around == nil {
		t.Errorf("camera = %+v", got[0])
	}
	if got[1].Kind != "earthquake" || got[1].To.Zone != 12001 || got[1].Loc.X != 5 || got[1].Value != 20 || got[1].Time != 10 {
		t.Errorf("earthquake = %+v", got[1])
	}
	if got[2].Kind != "sound" || got[2].To.Player == nil || got[2].Text != "BS02_A" || got[2].Target != 2 {
		t.Errorf("sound = %+v", got[2])
	}
	if got[3].Kind != "screen" || got[3].Text != "Hello" || got[3].Value != 2 || got[3].Time != 5000 {
		t.Errorf("screen = %+v", got[3])
	}
}

// Player and Skill additions.
func TestPlayerAndSkillAdditions(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
s:on_event(function(event, npc, player)
  local summon = player:summon()
  return table.concat({ player:hp(), player:max_hp(), tostring(summon and summon:object_id()),
    skills.get(1177, 1):target_type(), skills.get(27, 1):target_type() }, " ")
end)
`})
	h.World().SetSkillTargetType(27, "UNLOCKABLE")
	p := &scripttest.Player{ObjectID: 1, HP: 50, MaxHP: 100}
	if r, _ := h.Event("A", "x", p, nil); r.Content != "50 100 nil ONE UNLOCKABLE" {
		t.Errorf("got %q", r.Content)
	}
	p.Summon = &scripttest.Npc{ObjectID: 9, ID: 12077}
	if r, _ := h.Event("A", "x", p, nil); !strings.HasPrefix(r.Content, "50 100 9 ") {
		t.Errorf("with a summon: %q", r.Content)
	}
}

// bareWorld is a quest world without any AI capability.
type bareWorld struct{ scripting.QuestWorld }

// Without the world's support the AI calls say so; valid_location falls
// back to the destination.
func TestAICallsWithoutSupport(t *testing.T) {
	dir := scripttest.WriteScripts(t, map[string]string{"a.lua": `
local s = script.new("A")
s:on_event(function(event, npc)
  if event == "geo" then
    return tostring(world.valid_location(npc, { x = 5, y = 6, z = 7 }).x)
  end
  if event == "valid" then
    return tostring(npc:valid())
  end
  local calls = {
    disable_core_ai = function() npc:disable_core_ai(true) end,
    minions = function() npc:minions() end,
    zones = function() zones.oust_all(1) end,
    doors = function() doors.open(1) end,
    grandboss = function() grandboss.status(1) end,
    cinematics = function() cinematics.earthquake(npc, npc, 1, 1) end,
    game_time = function() time.game_time() end,
    is_box = function() npc:is_box() end,
    scan = function() world.npcs_by_id(1) end,
    spawn = function() npc:spawn_location() end,
  }
  calls[event]()
end)
`})
	w := scripttest.NewWorld()
	e, err := scripting.New(t.Context(), scripting.Options{Dir: dir, World: bareWorld{w}})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	n := &scripttest.Npc{ObjectID: 2, ID: 30001}
	w.Add(n)
	ctx := context.Background()
	if r, _ := e.Event(ctx, "A", "geo", nil, n.Ref()); r.Content != "5" {
		t.Errorf("valid_location = %+v", r)
	}
	if r, _ := e.Event(ctx, "A", "valid", nil, n.Ref()); r.Content != "true" {
		t.Errorf("valid = %+v", r)
	}
	for _, ev := range []string{"disable_core_ai", "minions", "zones", "doors", "grandboss", "cinematics", "game_time", "is_box", "scan"} {
		r, _ := e.Event(ctx, "A", ev, nil, n.Ref())
		if r.Err == nil || !strings.Contains(r.Err.Error(), "cannot do that yet") {
			t.Errorf("%s: %v", ev, r.Err)
		}
	}
	// spawn_location needs the NPC's spawn data, which the bare NPC has.
	if r, _ := e.Event(ctx, "A", "spawn", nil, n.Ref()); r.Err != nil {
		t.Errorf("spawn: %v", r.Err)
	}
}

// The new handlers register like the M8 ones; a second handler of a kind
// is an error, and on_start gets the reload flag.
func TestAIHandlerRegistration(t *testing.T) {
	err := loadErr(t, map[string]string{"a.lua": `
local s = script.new("A")
s:on_move_finished(function() end)
s:on_move_finished(function() end)
`})
	if !strings.Contains(err.Error(), "already has a move-finished handler") {
		t.Errorf("err = %v", err)
	}
	h := scripttest.LoadSource(t, map[string]string{"a.lua": `
local s = script.new("A")
s:on_start(function(ev) globals.set("reload", ev.reload) end)
`})
	h.Start()
	if v, _ := h.World().Global("reload"); v != "false" {
		t.Errorf("reload at start = %q", v)
	}
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	if v, _ := h.World().Global("reload"); v != "true" {
		t.Errorf("reload after a reload = %q", v)
	}
}

package scripting_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The NPC AI methods reach the world's NpcActions with their arguments.
func TestNpcAIMethods(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"q.lua": `
local q = quest.new(5, "Q5", "")
q:on_event(function(event, npc, player)
  if event == "status" then
    return npc:level() .. "," .. npc:hp() .. "," .. npc:max_hp()
  end
  npc:set_running()
  npc:add_damage_hate(player, 0, 999)
  npc:attack(player)
  npc:remove_attacker(player)
  npc:abort_attack()
  npc:move_to({ x = 1, y = 2, z = 3 })
  npc:move_to(4, 5, 6)
  npc:stop_respawn()
  npc:start_respawn()
  npc:cast(skills.get(4126, 1))
  npc:cast(skills.get(5089, 2), player)
  npc:cast(nil)
  npc:set_target(player)
  npc:teleport(7, 8, 9)
  npc:drop_item(player, 57, 10)
  npc:magic_skill_use(player, 4339, 1, 6000, 1)
  npc:die()
  return tostring(world.npc_exists(30001)) .. "," .. tostring(world.npc_exists(30002))
end)`})
	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	n := &scripttest.Npc{ObjectID: 2, ID: 30001, Level: 40, HP: 150.5, MaxHP: 300}
	if reply, _ := h.Event("Q5", "status", p, n); reply.Content != "40,150.5,300" {
		t.Errorf("status = %q", reply.Content)
	}
	reply, _ := h.Event("Q5", "go", p, n)
	if reply.Content != "true,false" {
		t.Errorf("npc_exists = %q, want true,false", reply.Content)
	}
	want := []scripttest.Action{
		{Kind: "run", Object: 2},
		{Kind: "hate", Object: 2, Target: 1, Value: 999},
		{Kind: "attack", Object: 2, Target: 1},
		{Kind: "remove_attacker", Object: 2, Target: 1},
		{Kind: "abort_attack", Object: 2},
		{Kind: "move", Object: 2, Loc: scripting.Location{X: 1, Y: 2, Z: 3}},
		{Kind: "move", Object: 2, Loc: scripting.Location{X: 4, Y: 5, Z: 6}},
		{Kind: "respawn", Object: 2, Value: 0},
		{Kind: "respawn", Object: 2, Value: 1},
		{Kind: "cast", Object: 2, Value: 4126, Level: 1},
		{Kind: "cast", Object: 2, Target: 1, Value: 5089, Level: 2},
		{Kind: "set_target", Object: 2, Target: 1},
		{Kind: "npc_teleport", Object: 2, Loc: scripting.Location{X: 7, Y: 8, Z: 9}},
		{Kind: "drop_item", Object: 2, Target: 1, Value: 57, Count: 10},
		{Kind: "magic_skill_use", Object: 2, Target: 1, Value: 4339, Level: 1, HitTime: 6 * time.Second},
		{Kind: "die", Object: 2, Target: 2},
	}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Errorf("actions =\n%+v\nwant\n%+v", got, want)
	}
	if !n.Running || !n.Respawn || !n.Dead || n.X != 7 {
		t.Errorf("npc = %+v, want running, respawning, dead, at x 7", n)
	}
}

// The Player methods of skills, the weapon, the target, noblesse, PKs and
// the class change reach the ref.
func TestPlayerAIMethods(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"q.lua": `
local q = quest.new(5, "Q5", "")
q:on_event(function(event, npc, player)
  if event == "start" then
    q:get_quest_state(player, true):set_state(State.STARTED)
    return ""
  end
  local out = {}
  local function add(v) out[#out + 1] = tostring(v) end
  add(player:active_weapon())
  local target = player:target()
  add(target and target:object_id())
  add(player:is_noble())
  player:set_noble(true)
  add(player:is_noble())
  add(player:is_affected_by_skill(4082))
  skills.apply_effects(skills.get(4082, 1), npc, player)
  add(player:is_affected_by_skill(4082))
  add(player:is_affected_by_skill(skills.get(4082, 1)))
  add(player:pk_kills())
  player:set_pk_kills(player:pk_kills() - 2)
  player:update_user_info()
  add(#player:active_quests())
  player:set_class_id(ClassId.WARRIOR)
  player:set_base_class(ClassId.WARRIOR)
  player:broadcast_user_info()
  player:magic_skill_use(player, 4546, 2)
  return table.concat(out, ",")
end)`})
	n := &scripttest.Npc{ObjectID: 2, ID: 30001}
	p := &scripttest.Player{ObjectID: 1, Name: "Hero", Target: n, PkKills: 5}
	p.Equip(7, 3471)
	h.Event("Q5", "start", p, n)
	reply, _ := h.Event("Q5", "go", p, n)
	if want := "3471,2,false,true,false,true,true,5,1"; reply.Content != want {
		t.Errorf("player = %q, want %q", reply.Content, want)
	}
	if !p.Noble || p.PkKills != 3 || p.ClassID != 1 || p.BaseClass != 1 || p.UserInfos != 2 {
		t.Errorf("player = %+v", p)
	}
	last := h.World().Actions[len(h.World().Actions)-1]
	if want := (scripttest.Action{Kind: "magic_skill_use", Object: 1, Target: 1, Value: 4546, Level: 2}); last != want {
		t.Errorf("last action = %+v, want %+v", last, want)
	}
	p.Weapon, p.Target = 0, nil
	p.Equipped = nil
	if reply, _ := h.Event("Q5", "go", p, n); reply.Content[:7] != "nil,nil" {
		t.Errorf("without weapon and target = %q", reply.Content)
	}
}

// skills.get finds levels that exist; items.get returns the template's
// fields; the item functions take Mobius's ItemHolder as {id, count}.
func TestSkillsAndItems(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"q.lua": `
local q = quest.new(5, "Q5", "")
q:on_event(function(event, npc, player)
  local sk = skills.get(4082)
  local item = items.get(57)
  q:give_items(player, { id = 100, count = 3 })
  q:reward_items(player, { id = 101, count = 2 })
  q:take_items(player, { id = 100, count = 1 })
  return table.concat({ sk:id(), sk:level(), tostring(skills.get(1, 1)), item.id, item.name, tostring(item.stackable),
    tostring(items.get(5).stackable) }, ",")
end)`})
	w := h.World()
	w.ItemNames[57] = "Adena"
	w.StackableItems = []int{57}
	w.MissingSkills = []int{1}
	p := &scripttest.Player{ObjectID: 1}
	reply, _ := h.Event("Q5", "x", p, nil)
	if want := "4082,1,nil,57,Adena,true,false"; reply.Content != want {
		t.Errorf("skills and items = %q, want %q", reply.Content, want)
	}
	if p.Items[100] != 2 || p.Items[101] != 2 {
		t.Errorf("items = %v, want 100: 2, 101: 2", p.Items)
	}
}

// The world calls raise errors while the scripts load, and a bad
// ItemHolder is an argument error.
func TestAIMethodErrors(t *testing.T) {
	err := loadErr(t, map[string]string{"q.lua": `local s = skills.get(4082, 1)`})
	if !strings.Contains(err.Error(), "skills.get") {
		t.Errorf("skills.get at load: %v", err)
	}
	h := scripttest.LoadSource(t, map[string]string{"q.lua": `
local q = quest.new(5, "Q5", "")
q:on_event(function(event, npc, player)
  q:give_items(player, { id = 100 })
end)`})
	reply, _ := h.Engine().Event(t.Context(), "Q5", "x", (&scripttest.Player{ObjectID: 1}).Ref(), nil)
	if reply.Err == nil || !strings.Contains(reply.Err.Error(), "{id = integer, count = integer}") {
		t.Errorf("bad ItemHolder: %v", reply.Err)
	}
}

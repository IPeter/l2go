-- Pronghorns and Frost Buffalos split, ported from Mobius
-- ai/areas/FrozenLabyrinth/FrozenLabyrinth.
--
-- The first physical skill that hits one makes it vanish into six
-- Pronghorn Spirits or Lost Buffalos, which attack the player.

local s = script.new("FrozenLabyrinth")

-- Monsters
local PRONGHORN_SPIRIT = 22087
local PRONGHORN = 22088
local LOST_BUFFALO = 22093
local FROST_BUFFALO = 22094

s:add_attack_id(PRONGHORN, FROST_BUFFALO)

-- add_attack_desire is Mobius Quest.addAttackDesire(npc, target): hate
-- 999, run, attack.
local function add_attack_desire(npc, target)
  npc:add_damage_hate(target, 0, 999)
  npc:set_running()
  npc:attack(target)
end

s:on_attack(function(npc, attacker, damage, is_summon, skill)
  if npc:is_script_value(0) and skill ~= nil and not skill:is_magic() then
    local spawn_id = npc:id() == PRONGHORN and PRONGHORN_SPIRIT or LOST_BUFFALO
    local diff = 0
    for _ = 1, 6 do
      local x = diff < 60 and npc:x() + diff or npc:x()
      local y = diff >= 60 and npc:y() + (diff - 40) or npc:y()
      local monster = s:add_spawn(spawn_id, { x = x, y = y, z = npc:z(), heading = npc:heading() }, false, 0)
      if monster ~= nil then
        add_attack_desire(monster, attacker)
      end
      diff = diff + 20
    end
    npc:set_script_value(1)
    npc:delete()
  end
end)

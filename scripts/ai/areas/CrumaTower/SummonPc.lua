-- Porta and Perum pull their attackers in, ported from Mobius
-- ai/areas/CrumaTower/SummonPc.
--
-- Hit from afar, the monster may cast Summon PC (4161) at the attacker;
-- when the cast lands it teleports the attacker to itself. The variable
-- "attacked" marks a summon in flight.
--
-- Mobius checks the skill's MP and HP cost against the monster's MP and HP
-- and that the skill is not disabled; npc:check_do_cast_conditions does
-- those checks (and the mute ones).

local s = script.new("SummonPc")

-- NPCs
local PORTA = 20213
local PERUM = 20221

-- Skill
local SUMMON_PC_ID = 4161

s:add_attack_id(PORTA, PERUM)
s:add_spell_finished_id(PORTA, PERUM)

s:on_attack(function(npc, attacker, damage, is_summon)
  local chance = rnd(100)
  local attacked = npc:var_bool("attacked", false)
  local summon_pc = skills.get(SUMMON_PC_ID, 1)
  if npc:calculate_distance_3d(attacker) > 300 and not attacked then
    if chance < 50 then
      -- Mobius casts twice here; the second cast marks the summon.
      if npc:check_do_cast_conditions(summon_pc) then
        npc:set_target(attacker)
        npc:cast(summon_pc)
      end
      if npc:check_do_cast_conditions(summon_pc) then
        npc:set_target(attacker)
        npc:cast(summon_pc)
        npc:set_var("attacked", true)
      end
    end
  elseif npc:calculate_distance_3d(attacker) > 100 and not attacked then
    local most_hated = npc:most_hated()
    if most_hated ~= nil then
      local is_most_hated = most_hated:object_id() == attacker:object_id()
      if ((is_most_hated and chance < 50) or chance < 10) and npc:check_do_cast_conditions(summon_pc) then
        npc:set_target(attacker)
        npc:cast(summon_pc)
        npc:set_var("attacked", true)
      end
    end
  end
end)

s:on_spell_finished(function(npc, player, skill)
  if skill:id() == SUMMON_PC_ID and not npc:is_dead() and npc:var_bool("attacked", false) then
    player:teleport(npc)
    npc:set_var("attacked", false)
  end
end)

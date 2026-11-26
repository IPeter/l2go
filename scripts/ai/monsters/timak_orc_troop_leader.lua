-- The Timak Orc Troop Leader, ported from Mobius
-- ai/others/TimakOrcTroopLeader (Mobius).
--
-- Hit, the leader may (template parameter SummonPrivateRate, in percent)
-- call back the first of its Privates that is missing, while it has fewer
-- than 3 alive, and shouts. (Mobius also skips a leader in the middle of a
-- teleport; NPC teleports here are instant.)

local s = script.new("TimakOrcTroopLeader")

local TIMAK_ORC_TROOP_LEADER = 20767

local ON_ATTACK_MSG = {
  "Come out, you children of darkness!",
  "Show yourselves!",
  "Destroy the enemy, my brothers!",
  "Forces of darkness! Follow me!",
}

local MAX_MINIONS = 3

s:add_attack_id(TIMAK_ORC_TROOP_LEADER)

-- has_minion reports whether one of minions is of the template.
local function has_minion(minions, id)
  for _, minion in ipairs(minions) do
    if minion:id() == id then
      return true
    end
  end
  return false
end

s:on_attack(function(npc, attacker, damage, is_summon)
  if not npc:is_type("Monster") then
    return
  end
  if rnd(1, 100) > npc:param_int("SummonPrivateRate", 0) then
    return
  end
  local minions = npc:minions()
  if #minions >= MAX_MINIONS then
    return
  end
  for _, private in ipairs(npc:param_minions("Privates")) do
    if not has_minion(minions, private.id) then
      npc:broadcast_say(ChatType.NPC_GENERAL, rnd.entry(ON_ATTACK_MSG))
      npc:add_minion(private.id)
      return
    end
  end
end)

-- Fairy trees, ported from Mobius ai/others/FairyTrees (Charus).
--
-- The trees stand still. A tree killed from within 1,500 calls up 20 Souls
-- of Tree Guardian for 30 s, which go for the killer (or its summon); each
-- poisons it at once with a coin flip.

local s = script.new("FairyTrees")

local SOUL_GUARDIAN = 27189 -- Soul of Tree Guardian

local MOBS = {
  27185, -- Fairy Tree of Wind
  27186, -- Fairy Tree of Star
  27187, -- Fairy Tree of Twilight
  27188, -- Fairy Tree of Abyss
}

local VENOMOUS_POISON = { id = 4243, level = 1 }

local MIN_DISTANCE = 1500
local GUARDIANS = 20
local GUARDIAN_LIFE_MS = 30000

s:add_kill_id(MOBS)
s:add_spawn_id(MOBS)

-- add_attack_desire is Mobius addAttackDesire with its default desire.
local function add_attack_desire(npc, target)
  npc:add_damage_hate(target, 0, 999)
  npc:set_running()
  npc:attack(target)
end

s:on_kill(function(npc, killer, is_summon)
  if npc:calculate_distance_3d(killer) > MIN_DISTANCE then
    return
  end
  local attacker = killer
  if is_summon then
    attacker = killer:summon() or killer
  end
  for _ = 1, GUARDIANS do
    local guardian = s:add_spawn(SOUL_GUARDIAN, npc, false, GUARDIAN_LIFE_MS)
    if guardian ~= nil then
      add_attack_desire(guardian, attacker)
      if rnd.bool() then
        guardian:set_target(attacker)
        guardian:cast(skills.get(VENOMOUS_POISON.id, VENOMOUS_POISON.level))
      end
    end
  end
end)

s:on_spawn(function(npc)
  npc:set_random_walking(false)
  npc:set_immobilized(true)
end)

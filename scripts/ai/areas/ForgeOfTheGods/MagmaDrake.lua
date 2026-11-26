-- A dying Magma Drake hatches five drakes, ported from Mobius
-- ai/areas/ForgeOfTheGods/MagmaDrake.
--
-- They spawn where it died and run at its killer (the killer's summon when
-- the summon made the kill).

local s = script.new("MagmaDrake")

-- NPCs
local MAGMA_DRAKE = 21657
local MAGMA_DRAKE_MINION = 21393

s:add_kill_id(MAGMA_DRAKE)

s:on_kill(function(npc, killer, is_summon)
  local attacker = killer
  if is_summon then
    attacker = killer:summon() or killer
  end
  local pos = { x = npc:x(), y = npc:y(), z = npc:z() + 20, heading = npc:heading() }
  for _ = 1, 5 do
    local drake = s:add_spawn(MAGMA_DRAKE_MINION, pos, false, 0, true)
    if drake ~= nil then
      drake:set_running()
      drake:add_damage_hate(attacker, 0, 500)
      drake:attack(attacker)
    end
  end
end)

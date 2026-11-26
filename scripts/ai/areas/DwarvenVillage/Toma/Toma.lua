-- Master Toma moves every 30 minutes, ported from Mobius
-- ai/areas/DwarvenVillage/Toma/Toma.
--
-- He stands at one of three places, picked at random, and every 30
-- minutes leaves for another pick. His page is in
-- data/html/scripts/ai/areas/DwarvenVillage/Toma/.
--
-- Restart-safe: Toma is found in the world instead of kept in a variable,
-- so a reload replaces the Toma it finds rather than adding a second one.

local s = script.new("Toma")

-- NPC
local TOMA = 30556

-- Locations
local LOCATIONS = {
  { x = 151680, y = -174891, z = -1782 },
  { x = 154153, y = -220105, z = -3402 },
  { x = 178834, y = -184336, z = -355, heading = 41400 },
}

-- Misc
local TELEPORT_DELAY = 1800000 -- 30 minutes

s:add_first_talk_id(TOMA)

local function respawn()
  for _, toma in ipairs(world.npcs_by_id(TOMA)) do
    toma:delete()
  end
  s:add_spawn(TOMA, rnd.entry(LOCATIONS), false, TELEPORT_DELAY)
end

s:on_start(function(ev)
  respawn()
  s:start_quest_timer("RESPAWN_TOMA", TELEPORT_DELAY, nil, nil, true)
end)

s:on_event(function(event, npc, player)
  if event == "RESPAWN_TOMA" then
    respawn()
  end
  return nil
end)

s:on_first_talk(function(npc, player)
  return "30556.htm"
end)

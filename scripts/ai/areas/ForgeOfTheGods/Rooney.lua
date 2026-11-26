-- Rooney wanders the Forge of the Gods, ported from Mobius
-- ai/areas/ForgeOfTheGods/Rooney.
--
-- She stands at one of 39 places. When she sees a player she greets
-- them, then talks once a minute and after five minutes moves to another
-- place. Her script value counts the minutes (0: waiting for a player).
--
-- Restart-safe: on_start spawns her only when she is not in the world,
-- and after a reload starts her minute timer again if she was counting.

local s = script.new("Rooney")

-- NPC
local ROONEY = 32049

-- Locations
local LOCATIONS = {
  { x = 175937, y = -112167, z = -5550 },
  { x = 178896, y = -112425, z = -5860 },
  { x = 180628, y = -115992, z = -6135 },
  { x = 183010, y = -114753, z = -6135 },
  { x = 184496, y = -116773, z = -6135 },
  { x = 181857, y = -109491, z = -5865 },
  { x = 178917, y = -107633, z = -5853 },
  { x = 178804, y = -110080, z = -5853 },
  { x = 182221, y = -106806, z = -6025 },
  { x = 186488, y = -109715, z = -5915 },
  { x = 183847, y = -119231, z = -3113 },
  { x = 185193, y = -120342, z = -3113 },
  { x = 188047, y = -120867, z = -3113 },
  { x = 189734, y = -120471, z = -3113 },
  { x = 188754, y = -118940, z = -3313 },
  { x = 190022, y = -116803, z = -3313 },
  { x = 188443, y = -115814, z = -3313 },
  { x = 186421, y = -114614, z = -3313 },
  { x = 185188, y = -113307, z = -3313 },
  { x = 187378, y = -112946, z = -3313 },
  { x = 189815, y = -113425, z = -3313 },
  { x = 189301, y = -111327, z = -3313 },
  { x = 190289, y = -109176, z = -3313 },
  { x = 187783, y = -110478, z = -3313 },
  { x = 185889, y = -109990, z = -3313 },
  { x = 181881, y = -109060, z = -3695 },
  { x = 183570, y = -111344, z = -3675 },
  { x = 182077, y = -112567, z = -3695 },
  { x = 180127, y = -112776, z = -3698 },
  { x = 179155, y = -108629, z = -3695 },
  { x = 176282, y = -109510, z = -3698 },
  { x = 176071, y = -113163, z = -3515 },
  { x = 179376, y = -117056, z = -3640 },
  { x = 179760, y = -115385, z = -3640 },
  { x = 177950, y = -119691, z = -4140 },
  { x = 177037, y = -120820, z = -4340 },
  { x = 181125, y = -120148, z = -3702 },
  { x = 182212, y = -117969, z = -3352 },
  { x = 186074, y = -118154, z = -3312 },
}

-- What she says at minutes 1 to 4.
local SPEECH = {
  "Hurry hurry",
  "I am not that type of person who stays in one place for a long time",
  "It's hard for me to keep standing like this",
  "Why don't I go that way this time",
}

s:add_creature_see_id(ROONEY)

s:on_start(function(ev)
  local rooney = world.npcs_by_id(ROONEY)[1]
  if rooney == nil then
    s:add_spawn(ROONEY, rnd.entry(LOCATIONS), false, 0)
  elseif not rooney:is_script_value(0) then
    s:start_quest_timer("TELEPORT", 60000, rooney, nil)
  end
end)

s:on_event(function(event, npc, player)
  if event == "TELEPORT" and npc ~= nil and npc:valid() then
    local minute = npc:get_script_value()
    local text = SPEECH[minute]
    if text == nil then
      npc:teleport(rnd.entry(LOCATIONS))
      npc:set_script_value(0)
      return nil
    end
    npc:broadcast_say(ChatType.NPC_GENERAL, text)
    npc:set_script_value(minute + 1)
    s:start_quest_timer("TELEPORT", 60000, npc, nil)
  end
  return nil
end)

s:on_creature_see(function(npc, creature)
  if creature:is_player() and npc:is_script_value(0) then
    npc:broadcast_say(ChatType.NPC_GENERAL, "Welcome!")
    s:start_quest_timer("TELEPORT", 60000, npc, nil)
    npc:set_script_value(1)
  end
end)

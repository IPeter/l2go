-- The Tower of Insolence dimension vortexes, ported from Mobius
-- ai/areas/TowerOfInsolence/ToIVortex (Plim; High Five by pmq, reworked by
-- xban1x).
--
-- A vortex takes a player to a floor for one dimension stone: green for
-- floors 1 to 3, blue for 4 to 6, red for 7 to 10. Keplon, Euclie and
-- Pithgon sell the stones for 100,000 adena each.

local s = script.new("ToIVortex")

local KEPLON, EUCLIE, PITHGON = 30949, 30950, 30951
local DIMENSION_VORTEX_1, DIMENSION_VORTEX_2, DIMENSION_VORTEX_3 = 30952, 30953, 30954

-- Items
local ADENA = 57
local GREEN_DIMENSION_STONE = 4401
local BLUE_DIMENSION_STONE = 4402
local RED_DIMENSION_STONE = 4403
local STONE_PRICE = 100000

-- The floors: where each leads and the stone it takes.
local TOI_FLOORS = {
  ["1"] = { loc = { x = 114356, y = 13423, z = -5096 }, stone = GREEN_DIMENSION_STONE },
  ["2"] = { loc = { x = 114666, y = 13380, z = -3608 }, stone = GREEN_DIMENSION_STONE },
  ["3"] = { loc = { x = 111982, y = 16028, z = -2120 }, stone = GREEN_DIMENSION_STONE },
  ["4"] = { loc = { x = 114636, y = 13413, z = -640 }, stone = BLUE_DIMENSION_STONE },
  ["5"] = { loc = { x = 114152, y = 19902, z = 928 }, stone = BLUE_DIMENSION_STONE },
  ["6"] = { loc = { x = 117131, y = 16044, z = 1944 }, stone = BLUE_DIMENSION_STONE },
  ["7"] = { loc = { x = 113026, y = 17687, z = 2952 }, stone = RED_DIMENSION_STONE },
  ["8"] = { loc = { x = 115571, y = 13723, z = 3960 }, stone = RED_DIMENSION_STONE },
  ["9"] = { loc = { x = 114649, y = 14144, z = 4976 }, stone = RED_DIMENSION_STONE },
  ["10"] = { loc = { x = 118507, y = 16605, z = 5984 }, stone = RED_DIMENSION_STONE },
}

-- The stones for sale.
local DIMENSION_TRADE = {
  GREEN = GREEN_DIMENSION_STONE,
  BLUE = BLUE_DIMENSION_STONE,
  RED = RED_DIMENSION_STONE,
}

local NPCS = { KEPLON, EUCLIE, PITHGON, DIMENSION_VORTEX_1, DIMENSION_VORTEX_2, DIMENSION_VORTEX_3 }
s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

s:on_event(function(event, npc, player)
  if npc == nil or player == nil then
    return nil
  end
  local floor = TOI_FLOORS[event]
  if floor ~= nil then
    if not s:has_quest_items(player, floor.stone) then
      return "no-stones.htm"
    end
    s:take_items(player, floor.stone, 1)
    -- Mobius teleports with a random offset; player:teleport has none.
    player:teleport(floor.loc)
  elseif DIMENSION_TRADE[event] ~= nil then
    if player:adena() < STONE_PRICE then
      return npc:id() .. "no-adena.htm"
    end
    s:take_items(player, ADENA, STONE_PRICE)
    s:give_items(player, DIMENSION_TRADE[event], 1)
  end
  return nil
end)

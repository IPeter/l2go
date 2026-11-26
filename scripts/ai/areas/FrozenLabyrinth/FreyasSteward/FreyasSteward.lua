-- Freya's Steward, ported from Mobius
-- ai/areas/FrozenLabyrinth/FreyasSteward/FreyasSteward.
--
-- She sends characters of level 82 and above on to Freya; nobody else.
-- Her pages are in data/html/scripts/ai/areas/FrozenLabyrinth/FreyasSteward/.

local s = script.new("FreyasSteward")

-- NPC
local FREYAS_STEWARD = 32029

-- Location
local TELEPORT_LOC = { x = 103045, y = -124361, z = -2768 }

-- Misc
local MIN_LEVEL = 82

s:add_start_npc(FREYAS_STEWARD)
s:add_first_talk_id(FREYAS_STEWARD)
s:add_talk_id(FREYAS_STEWARD)

s:on_event(function(event, npc, player)
  if event:upper() == "ENTER" then
    if player:level() >= MIN_LEVEL then
      player:teleport(TELEPORT_LOC)
      return nil
    end
    return "32029-1.html" -- This is beyond you. Wait for the right time.
  end
  return nil
end)

s:on_first_talk(function(npc, player)
  return "32029.html"
end)

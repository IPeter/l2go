-- Mozella, the Cruma Tower entrance, ported from Mobius
-- ai/areas/CrumaTower/CrumaTower.
--
-- Characters up to level 55 may enter: one of two spots inside, at
-- random. Her pages are in data/html/scripts/ai/areas/CrumaTower/.

local s = script.new("CrumaTower")

-- NPC
local MOZELLA = 30483

-- Locations
local TELEPORT_LOC1 = { x = 17776, y = 113968, z = -11671 }
local TELEPORT_LOC2 = { x = 17680, y = 113968, z = -11671 }

-- Misc
local MAX_LEVEL = 55

s:add_first_talk_id(MOZELLA)
s:add_start_npc(MOZELLA)
s:add_talk_id(MOZELLA)

s:on_event(function(event, npc, player)
  if event == "ENTER" then
    if player:level() <= MAX_LEVEL then
      player:teleport(rnd.bool() and TELEPORT_LOC1 or TELEPORT_LOC2)
      return nil
    end
    return "30483-1.html" -- Characters who are level 56 and above cannot enter the Cruma Tower.
  end
  return nil
end)

s:on_first_talk(function(npc, player)
  return npc:id() .. ".html"
end)

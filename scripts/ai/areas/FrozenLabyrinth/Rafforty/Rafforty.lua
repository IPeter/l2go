-- Rafforty blesses Kirunak's necklace, ported from Mobius
-- ai/areas/FrozenLabyrinth/Rafforty/Rafforty.
--
-- A necklace and a bottle of ice become a blessed necklace. His page is
-- in data/html/scripts/ai/areas/FrozenLabyrinth/Rafforty/ (Mobius has
-- only the first one; the others show nothing there too).

local s = script.new("Rafforty")

-- NPC
local RAFFORTY = 32020

-- Items
local NECKLACE = 16025
local BLESSED_NECKLACE = 16026
local BOTTLE = 16027

s:add_start_npc(RAFFORTY)
s:add_first_talk_id(RAFFORTY)
s:add_talk_id(RAFFORTY)

s:on_event(function(event, npc, player)
  local htmltext = event
  if event == "32020-01.html" then
    if not s:has_quest_items(player, NECKLACE) then
      htmltext = "32020-02.html"
    end
  elseif event == "32020-04.html" then
    if not s:has_quest_items(player, BOTTLE) then
      htmltext = "32020-05.html"
    end
  elseif event == "32020-07.html" then
    if not s:has_quest_items(player, BOTTLE, NECKLACE) then
      return "32020-08.html"
    end
    s:take_items(player, NECKLACE, 1)
    s:take_items(player, BOTTLE, 1)
    s:give_items(player, BLESSED_NECKLACE, 1)
  end
  return htmltext
end)

s:on_first_talk(function(npc, player)
  return npc:id() .. ".html"
end)

-- Mobius events/HeavyMedal (a LongTimeEvent, lib/long_time_event.lua):
-- monsters drop medals, Roy the Cat trades them for prizes by the player's
-- collector level, and Winnie the Cat raises that level with a coin toss
-- for glittering medals (the badges 6399-6402 are the levels). Talking to
-- a cat starts the event's quest state, as in Mobius. Its dates, drops and
-- spawns are config.xml's (data/events/longtimeevents.json, the config's
-- events.dates on top; Mobius's own drop period ends before it starts, so
-- the medals drop only with dates from the config); the pages are Mobius's
-- (data/html/scripts/events/HeavyMedal/).

local lte = require("lib.long_time_event")

local CAT_ROY = 31228
local CAT_WINNIE = 31229
local GLITTERING_MEDAL = 6393

local WIN_CHANCE = 50

-- The glittering medals a toss costs and the badge of each level, by level
-- + 1 (Mobius MEDALS[level], BADGES[level]).
local MEDALS = { 5, 10, 20, 40 }
local BADGES = { 6399, 6400, 6401, 6402 }

local s = lte.new("HeavyMedal")
s:add_start_npc(CAT_ROY)
s:add_start_npc(CAT_WINNIE)
s:add_talk_id(CAT_ROY)
s:add_talk_id(CAT_WINNIE)
s:add_first_talk_id(CAT_ROY)
s:add_first_talk_id(CAT_WINNIE)

-- check_level is the player's collector level: the highest badge held.
local function check_level(player)
  if s:has_quest_items(player, 6402) then
    return 4
  elseif s:has_quest_items(player, 6401) then
    return 3
  elseif s:has_quest_items(player, 6400) then
    return 2
  elseif s:has_quest_items(player, 6399) then
    return 1
  end
  return 0
end

lte.on_event(s, function(event, npc, player)
  local htmltext = event
  local level = check_level(player)
  local lower = event:lower()
  if lower == "game" or lower == "heads" or lower == "tails" then
    if MEDALS[level + 1] == nil then
      -- Level 4 has no next toss: Mobius's MEDALS[4] is out of bounds and
      -- its handler fails, showing nothing.
      return nil
    end
  end
  if lower == "game" then
    if s:get_quest_items_count(player, GLITTERING_MEDAL) < MEDALS[level + 1] then
      htmltext = "31229-no.htm"
    else
      htmltext = "31229-game.htm"
    end
  elseif lower == "heads" or lower == "tails" then
    if s:get_quest_items_count(player, GLITTERING_MEDAL) < MEDALS[level + 1] then
      htmltext = "31229-" .. lower .. "-10.htm"
    else
      s:take_items(player, GLITTERING_MEDAL, MEDALS[level + 1])
      if rnd(100) > WIN_CHANCE then
        level = 0
      else
        if level > 0 then
          s:take_items(player, BADGES[level], -1)
        end
        s:give_items(player, BADGES[level + 1], 1)
        s:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        level = level + 1
      end
      htmltext = "31229-" .. lower .. "-" .. level .. ".htm"
    end
  elseif lower == "talk" then
    if npc == nil then
      return nil
    end
    htmltext = npc:id() .. "-lvl-" .. level .. ".htm"
  end
  return htmltext
end)

s:on_first_talk(function(npc, player)
  if s:get_quest_state(player, false) == nil then
    s:new_quest_state(player)
  end
  return npc:id() .. ".htm"
end)

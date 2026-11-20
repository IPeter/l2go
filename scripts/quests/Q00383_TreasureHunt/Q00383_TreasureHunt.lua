-- Translated by questport from quests/Q00383_TreasureHunt (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(383, "Q00383_TreasureHunt", "Searching for Treasure")

-- NPCs
local ESPEN = 30890
local PIRATE_CHEST = 31148
-- Items
local PIRATE_TREASURE_MAP = 5915
local THIEF_KEY = 1661

q:add_start_npc(ESPEN)
q:add_talk_id(ESPEN, PIRATE_CHEST)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30890-04.htm" then
    -- Sell the map.
    if q:has_quest_items(player, PIRATE_TREASURE_MAP) then
      q:take_items(player, PIRATE_TREASURE_MAP, 1)
      q:give_adena(player, 1000, true)
    else
      htmltext = "30890-06.htm"
    end
  elseif event == "30890-07.htm" then
    -- Listen the story.
    if q:has_quest_items(player, PIRATE_TREASURE_MAP) then
      st:start_quest()
    else
      htmltext = "30890-06.htm"
    end
  elseif event == "30890-11.htm" then
    -- Decipher the map.
    if q:has_quest_items(player, PIRATE_TREASURE_MAP) then
      st:set_cond(2, true)
      q:take_items(player, PIRATE_TREASURE_MAP, 1)
    else
      htmltext = "30890-06.htm"
    end
  elseif event == "31148-02.htm" then
    if q:has_quest_items(player, THIEF_KEY) then
      q:take_items(player, THIEF_KEY, 1)
      -- Adena reward.
      local i1 = 0
      local i0 = rnd(100)
      if i0 < 5 then
        q:give_items(player, 2450, 1)
      elseif i0 < 6 then
        q:give_items(player, 2451, 1)
      elseif i0 < 18 then
        q:give_items(player, 956, 1)
      elseif i0 < 28 then
        q:give_items(player, 952, 1)
      else
        i1 = i1 + 500
      end
      i0 = rnd(1000)
      if i0 < 25 then
        q:give_items(player, 4481, 1)
      elseif i0 < 50 then
        q:give_items(player, 4482, 1)
      elseif i0 < 75 then
        q:give_items(player, 4483, 1)
      elseif i0 < 100 then
        q:give_items(player, 4484, 1)
      elseif i0 < 125 then
        q:give_items(player, 4485, 1)
      elseif i0 < 150 then
        q:give_items(player, 4486, 1)
      elseif i0 < 175 then
        q:give_items(player, 4487, 1)
      elseif i0 < 200 then
        q:give_items(player, 4488, 1)
      elseif i0 < 225 then
        q:give_items(player, 4489, 1)
      elseif i0 < 250 then
        q:give_items(player, 4490, 1)
      elseif i0 < 275 then
        q:give_items(player, 4491, 1)
      elseif i0 < 300 then
        q:give_items(player, 4492, 1)
      else
        i1 = i1 + 300
      end
      i0 = rnd(100)
      if i0 < 4 then
        q:give_items(player, 1337, 1)
      elseif i0 < 8 then
        q:give_items(player, 1338, 2)
      elseif i0 < 12 then
        q:give_items(player, 1339, 2)
      elseif i0 < 16 then
        q:give_items(player, 3447, 2)
      elseif i0 < 20 then
        q:give_items(player, 3450, 1)
      elseif i0 < 25 then
        q:give_items(player, 3453, 1)
      elseif i0 < 27 then
        q:give_items(player, 3456, 1)
      else
        i1 = i1 + 500
      end
      i0 = rnd(100)
      if i0 < 20 then
        q:give_items(player, 4408, 1)
      elseif i0 < 40 then
        q:give_items(player, 4409, 1)
      elseif i0 < 60 then
        q:give_items(player, 4418, 1)
      elseif i0 < 80 then
        q:give_items(player, 4419, 1)
      else
        i1 = i1 + 500
      end
      q:give_adena(player, i1, true)
      st:exit_quest(true, true)
    else
      htmltext = "31148-03.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = (player:level() < 42 or not q:has_quest_items(player, PIRATE_TREASURE_MAP)) and "30890-01.htm" or "30890-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ESPEN then
      if cond == 1 then
        htmltext = "30890-07a.htm"
      else
        htmltext = "30890-12.htm"
      end
    elseif sw2 == PIRATE_CHEST then
      if cond == 2 then
        htmltext = "31148-01.htm"
      end
    end
  end
  return htmltext
end)

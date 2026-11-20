-- Translated by questport from quests/Q00033_MakeAPairOfDressShoes (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(33, "Q00033_MakeAPairOfDressShoes", "Make a Pair of Dress Shoes")

-- NPCs
local WOODLEY = 30838
local IAN = 30164
local LEIKAR = 31520
-- Items
local LEATHER = 1882
local THREAD = 1868
local ADENA = 57
-- Rewards
local DRESS_SHOES_BOX = 7113

q:add_start_npc(WOODLEY)
q:add_talk_id(WOODLEY, IAN, LEIKAR)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30838-1.htm" then
    st:start_quest()
  elseif event == "31520-1.htm" then
    st:set_cond(2, true)
  elseif event == "30838-3.htm" then
    st:set_cond(3, true)
  elseif event == "30838-5.htm" then
    if q:get_quest_items_count(player, LEATHER) >= 200 and q:get_quest_items_count(player, THREAD) >= 600 and q:get_quest_items_count(player, ADENA) >= 200000 then
      st:set_cond(4, true)
      q:take_items(player, ADENA, 200000)
      q:take_items(player, LEATHER, 200)
      q:take_items(player, THREAD, 600)
    else
      htmltext = "30838-4a.htm"
    end
  elseif event == "30164-1.htm" then
    if q:get_quest_items_count(player, ADENA) >= 300000 then
      st:set_cond(5, true)
      q:take_items(player, ADENA, 300000)
    else
      htmltext = "30164-1a.htm"
    end
  elseif event == "30838-7.htm" then
    q:give_items(player, DRESS_SHOES_BOX, 1)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() >= 60 then
      local fwear = player:get_quest_state("Q00037_MakeFormalWear")
      if fwear ~= nil and fwear:is_cond(7) then
        htmltext = "30838-0.htm"
      else
        htmltext = "30838-0a.htm"
      end
    else
      htmltext = "30838-0b.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == WOODLEY then
      if cond == 1 then
        htmltext = "30838-1.htm"
      elseif cond == 2 then
        htmltext = "30838-2.htm"
      elseif cond == 3 then
        if q:get_quest_items_count(player, LEATHER) >= 200 and q:get_quest_items_count(player, THREAD) >= 600 and q:get_quest_items_count(player, ADENA) >= 200000 then
          htmltext = "30838-4.htm"
        else
          htmltext = "30838-4a.htm"
        end
      elseif cond == 4 then
        htmltext = "30838-5a.htm"
      elseif cond == 5 then
        htmltext = "30838-6.htm"
      end
    elseif sw2 == LEIKAR then
      if cond == 1 then
        htmltext = "31520-0.htm"
      elseif cond > 1 then
        htmltext = "31520-1a.htm"
      end
    elseif sw2 == IAN then
      if cond == 4 then
        htmltext = "30164-0.htm"
      elseif cond == 5 then
        htmltext = "30164-2.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

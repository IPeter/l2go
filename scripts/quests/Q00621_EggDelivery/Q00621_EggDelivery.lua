-- Translated by questport from quests/Q00621_EggDelivery (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(621, "Q00621_EggDelivery", "Egg Delivery")

-- NPCs
local JEREMY = 31521
local PULIN = 31543
local NAFF = 31544
local CROCUS = 31545
local KUBER = 31546
local BEOLIN = 31547
local VALENTINE = 31584
-- Items
local BOILED_EGGS = 7195
local FEE_OF_BOILED_EGG = 7196
-- Rewards
local HASTE_POTION = 1062
local RECIPES = {6847, 6849, 6851}

q:register_quest_items(BOILED_EGGS, FEE_OF_BOILED_EGG)
q:add_start_npc(JEREMY)
q:add_talk_id(JEREMY, PULIN, NAFF, CROCUS, KUBER, BEOLIN, VALENTINE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31521-02.htm" then
    st:start_quest()
    q:give_items(player, BOILED_EGGS, 5)
  elseif event == "31543-02.htm" then
    st:set_cond(2)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, BOILED_EGGS, 1)
    q:give_items(player, FEE_OF_BOILED_EGG, 1)
  elseif event == "31544-02.htm" then
    st:set_cond(3)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, BOILED_EGGS, 1)
    q:give_items(player, FEE_OF_BOILED_EGG, 1)
  elseif event == "31545-02.htm" then
    st:set_cond(4)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, BOILED_EGGS, 1)
    q:give_items(player, FEE_OF_BOILED_EGG, 1)
  elseif event == "31546-02.htm" then
    st:set_cond(5)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, BOILED_EGGS, 1)
    q:give_items(player, FEE_OF_BOILED_EGG, 1)
  elseif event == "31547-02.htm" then
    st:set_cond(6)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, BOILED_EGGS, 1)
    q:give_items(player, FEE_OF_BOILED_EGG, 1)
  elseif event == "31521-06.htm" then
    if q:get_quest_items_count(player, FEE_OF_BOILED_EGG) < 5 then
      htmltext = "31521-08.htm"
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_GIVEUP)
      st:exit_quest(true)
    else
      st:set_cond(7, true)
      q:take_items(player, FEE_OF_BOILED_EGG, 5)
    end
  elseif event == "31584-02.htm" then
    if rnd(5) < 1 then
      q:reward_items(player, RECIPES[rnd(3) + 1], 1)
      st:exit_quest(true, true)
    else
      q:give_adena(player, 18800, true)
      q:reward_items(player, HASTE_POTION, 1)
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 68 and "31521-03.htm" or "31521-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == JEREMY then
      if cond == 1 then
        htmltext = "31521-04.htm"
      elseif cond == 6 then
        htmltext = "31521-05.htm"
      elseif cond == 7 then
        htmltext = "31521-07.htm"
      end
    elseif sw2 == PULIN then
      if cond == 1 and q:get_quest_items_count(player, BOILED_EGGS) == 5 then
        htmltext = "31543-01.htm"
      elseif cond > 1 then
        htmltext = "31543-03.htm"
      end
    elseif sw2 == NAFF then
      if cond == 2 and q:get_quest_items_count(player, BOILED_EGGS) == 4 then
        htmltext = "31544-01.htm"
      elseif cond > 2 then
        htmltext = "31544-03.htm"
      end
    elseif sw2 == CROCUS then
      if cond == 3 and q:get_quest_items_count(player, BOILED_EGGS) == 3 then
        htmltext = "31545-01.htm"
      elseif cond > 3 then
        htmltext = "31545-03.htm"
      end
    elseif sw2 == KUBER then
      if cond == 4 and q:get_quest_items_count(player, BOILED_EGGS) == 2 then
        htmltext = "31546-01.htm"
      elseif cond > 4 then
        htmltext = "31546-03.htm"
      end
    elseif sw2 == BEOLIN then
      if cond == 5 and q:get_quest_items_count(player, BOILED_EGGS) == 1 then
        htmltext = "31547-01.htm"
      elseif cond > 5 then
        htmltext = "31547-03.htm"
      end
    elseif sw2 == VALENTINE then
      if cond == 7 then
        htmltext = "31584-01.htm"
      end
    end
  end
  return htmltext
end)

-- Translated by questport from quests/Q00622_SpecialtyLiquorDelivery (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(622, "Q00622_SpecialtyLiquorDelivery", "Specialty Liquor Delivery")

-- NPCs
local JEREMY = 31521
local PULIN = 31543
local NAFF = 31544
local CROCUS = 31545
local KUBER = 31546
local BEOLIN = 31547
local LIETTA = 31267
-- Items
local SPECIAL_DRINK = 7197
local FEE_OF_SPECIAL_DRINK = 7198
-- Rewards
local ADENA = 57
local HASTE_POTION = 1062
local RECIPES = {6847, 6849, 6851}

q:register_quest_items(SPECIAL_DRINK, FEE_OF_SPECIAL_DRINK)
q:add_start_npc(JEREMY)
q:add_talk_id(JEREMY, PULIN, NAFF, CROCUS, KUBER, BEOLIN, LIETTA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31521-02.htm" then
    st:start_quest()
    q:give_items(player, SPECIAL_DRINK, 5)
  elseif event == "31547-02.htm" then
    st:set_cond(2, true)
    q:take_items(player, SPECIAL_DRINK, 1)
    q:give_items(player, FEE_OF_SPECIAL_DRINK, 1)
  elseif event == "31546-02.htm" then
    st:set_cond(3, true)
    q:take_items(player, SPECIAL_DRINK, 1)
    q:give_items(player, FEE_OF_SPECIAL_DRINK, 1)
  elseif event == "31545-02.htm" then
    st:set_cond(4, true)
    q:take_items(player, SPECIAL_DRINK, 1)
    q:give_items(player, FEE_OF_SPECIAL_DRINK, 1)
  elseif event == "31544-02.htm" then
    st:set_cond(5, true)
    q:take_items(player, SPECIAL_DRINK, 1)
    q:give_items(player, FEE_OF_SPECIAL_DRINK, 1)
  elseif event == "31543-02.htm" then
    st:set_cond(6, true)
    q:take_items(player, SPECIAL_DRINK, 1)
    q:give_items(player, FEE_OF_SPECIAL_DRINK, 1)
  elseif event == "31521-06.htm" then
    st:set_cond(7, true)
    q:take_items(player, FEE_OF_SPECIAL_DRINK, 5)
  elseif event == "31267-02.htm" then
    if rnd(5) < 1 then
      q:give_items(player, rnd.entry(RECIPES), 1)
    else
      q:reward_items(player, ADENA, 18800)
      q:reward_items(player, HASTE_POTION, 1)
    end
    st:exit_quest(true, true)
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
      if cond < 6 then
        htmltext = "31521-04.htm"
      elseif cond == 6 then
        htmltext = "31521-05.htm"
      elseif cond == 7 then
        htmltext = "31521-06.htm"
      end
    elseif sw2 == BEOLIN then
      if cond == 1 and q:get_quest_items_count(player, SPECIAL_DRINK) == 5 then
        htmltext = "31547-01.htm"
      elseif cond > 1 then
        htmltext = "31547-03.htm"
      end
    elseif sw2 == KUBER then
      if cond == 2 and q:get_quest_items_count(player, SPECIAL_DRINK) == 4 then
        htmltext = "31546-01.htm"
      elseif cond > 2 then
        htmltext = "31546-03.htm"
      end
    elseif sw2 == CROCUS then
      if cond == 3 and q:get_quest_items_count(player, SPECIAL_DRINK) == 3 then
        htmltext = "31545-01.htm"
      elseif cond > 3 then
        htmltext = "31545-03.htm"
      end
    elseif sw2 == NAFF then
      if cond == 4 and q:get_quest_items_count(player, SPECIAL_DRINK) == 2 then
        htmltext = "31544-01.htm"
      elseif cond > 4 then
        htmltext = "31544-03.htm"
      end
    elseif sw2 == PULIN then
      if cond == 5 and q:get_quest_items_count(player, SPECIAL_DRINK) == 1 then
        htmltext = "31543-01.htm"
      elseif cond > 5 then
        htmltext = "31543-03.htm"
      end
    elseif sw2 == LIETTA then
      if cond == 7 then
        htmltext = "31267-01.htm"
      end
    end
  end
  return htmltext
end)

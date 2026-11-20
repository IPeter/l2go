-- Translated by questport from quests/Q00046_OnceMoreInTheArmsOfTheMotherTree (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(46, "Q00046_OnceMoreInTheArmsOfTheMotherTree", "Once More In the Arms of the Mother Tree")

-- NPCs
local GALLADUCCI = 30097
local GENTLER = 30094
local SANDRA = 30090
local DUSTIN = 30116
-- Items
local ORDER_DOCUMENT_1 = 7563
local ORDER_DOCUMENT_2 = 7564
local ORDER_DOCUMENT_3 = 7565
local MAGIC_SWORD_HILT = 7568
local GEMSTONE_POWDER = 7567
local PURIFIED_MAGIC_NECKLACE = 7566
local MARK_OF_TRAVELER = 7570
local SCROLL_OF_ESCAPE_SPECIAL = 7555

q:register_quest_items(ORDER_DOCUMENT_1, ORDER_DOCUMENT_2, ORDER_DOCUMENT_3, MAGIC_SWORD_HILT, GEMSTONE_POWDER, PURIFIED_MAGIC_NECKLACE)
q:add_start_npc(GALLADUCCI)
q:add_talk_id(GALLADUCCI, GENTLER, SANDRA, DUSTIN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30097-03.htm" then
    st:start_quest()
    q:give_items(player, ORDER_DOCUMENT_1, 1)
  elseif event == "30094-02.htm" then
    st:set_cond(2, true)
    q:take_items(player, ORDER_DOCUMENT_1, 1)
    q:give_items(player, MAGIC_SWORD_HILT, 1)
  elseif event == "30097-06.htm" then
    st:set_cond(3, true)
    q:take_items(player, MAGIC_SWORD_HILT, 1)
    q:give_items(player, ORDER_DOCUMENT_2, 1)
  elseif event == "30090-02.htm" then
    st:set_cond(4, true)
    q:take_items(player, ORDER_DOCUMENT_2, 1)
    q:give_items(player, GEMSTONE_POWDER, 1)
  elseif event == "30097-09.htm" then
    st:set_cond(5, true)
    q:take_items(player, GEMSTONE_POWDER, 1)
    q:give_items(player, ORDER_DOCUMENT_3, 1)
  elseif event == "30116-02.htm" then
    st:set_cond(6, true)
    q:take_items(player, ORDER_DOCUMENT_3, 1)
    q:give_items(player, PURIFIED_MAGIC_NECKLACE, 1)
  elseif event == "30097-12.htm" then
    q:take_items(player, MARK_OF_TRAVELER, -1)
    q:take_items(player, PURIFIED_MAGIC_NECKLACE, 1)
    q:reward_items(player, SCROLL_OF_ESCAPE_SPECIAL, 1)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() == Race.ELF and player:level() >= 3 then
      if q:has_quest_items(player, MARK_OF_TRAVELER) then
        htmltext = "30097-02.htm"
      else
        htmltext = "30097-01.htm"
      end
    else
      htmltext = "30097-01a.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == GALLADUCCI then
      if cond == 1 then
        htmltext = "30097-04.htm"
      elseif cond == 2 then
        htmltext = "30097-05.htm"
      elseif cond == 3 then
        htmltext = "30097-07.htm"
      elseif cond == 4 then
        htmltext = "30097-08.htm"
      elseif cond == 5 then
        htmltext = "30097-10.htm"
      elseif cond == 6 then
        htmltext = "30097-11.htm"
      end
    elseif sw2 == GENTLER then
      if cond == 1 then
        htmltext = "30094-01.htm"
      elseif cond > 1 then
        htmltext = "30094-03.htm"
      end
    elseif sw2 == SANDRA then
      if cond == 3 then
        htmltext = "30090-01.htm"
      elseif cond > 3 then
        htmltext = "30090-03.htm"
      end
    elseif sw2 == DUSTIN then
      if cond == 5 then
        htmltext = "30116-01.htm"
      elseif cond == 6 then
        htmltext = "30116-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

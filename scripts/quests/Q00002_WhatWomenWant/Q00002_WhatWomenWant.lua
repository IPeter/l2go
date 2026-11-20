-- Translated by questport from quests/Q00002_WhatWomenWant (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(2, "Q00002_WhatWomenWant", "What Women Want")

-- NPCs
local ARUJIEN = 30223
local MIRABEL = 30146
local HERBIEL = 30150
local GREENIS = 30157
-- Items
local ARUJIEN_LETTER_1 = 1092
local ARUJIEN_LETTER_2 = 1093
local ARUJIEN_LETTER_3 = 1094
local POETRY_BOOK = 689
local GREENIS_LETTER = 693
-- Rewards
local MYSTICS_EARRING = 113

q:register_quest_items(ARUJIEN_LETTER_1, ARUJIEN_LETTER_2, ARUJIEN_LETTER_3, POETRY_BOOK, GREENIS_LETTER)
q:add_start_npc(ARUJIEN)
q:add_talk_id(ARUJIEN, MIRABEL, HERBIEL, GREENIS)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30223-04.htm" then
    st:start_quest()
    q:give_items(player, ARUJIEN_LETTER_1, 1)
  elseif event == "30223-08.htm" then
    st:set_cond(4, true)
    q:take_items(player, ARUJIEN_LETTER_3, 1)
    q:give_items(player, POETRY_BOOK, 1)
  elseif event == "30223-09.htm" then
    q:take_items(player, ARUJIEN_LETTER_3, 1)
    q:give_adena(player, 450, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF and player:race() ~= Race.HUMAN then
      htmltext = "30223-00.htm"
    elseif player:level() < 2 then
      htmltext = "30223-01.htm"
    else
      htmltext = "30223-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ARUJIEN then
      if q:has_quest_items(player, ARUJIEN_LETTER_1) then
        htmltext = "30223-05.htm"
      elseif q:has_quest_items(player, ARUJIEN_LETTER_3) then
        htmltext = "30223-07.htm"
      elseif q:has_quest_items(player, ARUJIEN_LETTER_2) then
        htmltext = "30223-06.htm"
      elseif q:has_quest_items(player, POETRY_BOOK) then
        htmltext = "30223-11.htm"
      elseif q:has_quest_items(player, GREENIS_LETTER) then
        htmltext = "30223-10.htm"
        q:take_items(player, GREENIS_LETTER, 1)
        q:give_items(player, MYSTICS_EARRING, 1)
        st:exit_quest(false, true)
      end
    elseif sw2 == MIRABEL then
      if cond == 1 then
        htmltext = "30146-01.htm"
        st:set_cond(2, true)
        q:take_items(player, ARUJIEN_LETTER_1, 1)
        q:give_items(player, ARUJIEN_LETTER_2, 1)
      elseif cond > 1 then
        htmltext = "30146-02.htm"
      end
    elseif sw2 == HERBIEL then
      if cond == 2 then
        htmltext = "30150-01.htm"
        st:set_cond(3, true)
        q:take_items(player, ARUJIEN_LETTER_2, 1)
        q:give_items(player, ARUJIEN_LETTER_3, 1)
      elseif cond > 2 then
        htmltext = "30150-02.htm"
      end
    elseif sw2 == GREENIS then
      if cond < 4 then
        htmltext = "30157-01.htm"
      elseif cond == 4 then
        htmltext = "30157-02.htm"
        st:set_cond(5, true)
        q:take_items(player, POETRY_BOOK, 1)
        q:give_items(player, GREENIS_LETTER, 1)
      elseif cond == 5 then
        htmltext = "30157-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

-- Translated by questport from quests/Q00363_SorrowfulSoundOfFlute (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(363, "Q00363_SorrowfulSoundOfFlute", "Sorrowful Sound of Flute")

-- NPCs
local NANARIN = 30956
local OPIX = 30595
local ALDO = 30057
local RANSPO = 30594
local HOLVAS = 30058
local BARBADO = 30959
local POITAN = 30458
-- Items
local NANARIN_FLUTE = 4319
local BLACK_BEER = 4320
local CLOTHES = 4318
-- Reward
local THEME_OF_SOLITUDE = 4420

q:register_quest_items(NANARIN_FLUTE, BLACK_BEER, CLOTHES)
q:add_start_npc(NANARIN)
q:add_talk_id(NANARIN, OPIX, ALDO, RANSPO, HOLVAS, BARBADO, POITAN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30956-02.htm" then
    st:start_quest()
  elseif event == "30956-05.htm" then
    st:set_cond(3, true)
    q:give_items(player, CLOTHES, 1)
  elseif event == "30956-06.htm" then
    st:set_cond(3, true)
    q:give_items(player, NANARIN_FLUTE, 1)
  elseif event == "30956-07.htm" then
    st:set_cond(3, true)
    q:give_items(player, BLACK_BEER, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 15 and "30956-03.htm" or "30956-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == NANARIN then
      if cond == 1 then
        htmltext = "30956-02.htm"
      elseif cond == 2 then
        htmltext = "30956-04.htm"
      elseif cond == 3 then
        htmltext = "30956-08.htm"
      elseif cond == 4 then
        if st:get_int("success") == 1 then
          htmltext = "30956-09.htm"
          q:give_items(player, THEME_OF_SOLITUDE, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_FINISH)
        else
          htmltext = "30956-10.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_GIVEUP)
        end
        st:exit_quest(true)
      end
    elseif sw2 == OPIX or sw2 == POITAN or sw2 == ALDO or sw2 == RANSPO or sw2 == HOLVAS then
      htmltext = npc:id() .. "-01.htm"
      if cond == 1 then
        st:set_cond(2, true)
      end
    elseif sw2 == BARBADO then
      if cond == 3 then
        st:set_cond(4, true)
        if q:has_quest_items(player, NANARIN_FLUTE) then
          htmltext = "30959-02.htm"
          st:set("success", "1")
        else
          htmltext = "30959-01.htm"
        end
        q:take_items(player, BLACK_BEER, -1)
        q:take_items(player, CLOTHES, -1)
        q:take_items(player, NANARIN_FLUTE, -1)
      elseif cond == 4 then
        htmltext = "30959-03.htm"
      end
    end
  end
  return htmltext
end)

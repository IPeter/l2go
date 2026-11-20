-- Translated by questport from quests/Q00017_LightAndDarkness (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(17, "Q00017_LightAndDarkness", "Light and Darkness")

-- NPCs
local HIERARCH = 31517
local SAINT_ALTAR_1 = 31508
local SAINT_ALTAR_2 = 31509
local SAINT_ALTAR_3 = 31510
local SAINT_ALTAR_4 = 31511
-- Items
local BLOOD_OF_SAINT = 7168

q:register_quest_items(BLOOD_OF_SAINT)
q:add_start_npc(HIERARCH)
q:add_talk_id(HIERARCH, SAINT_ALTAR_1, SAINT_ALTAR_2, SAINT_ALTAR_3, SAINT_ALTAR_4)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31517-04.htm" then
    st:start_quest()
    q:give_items(player, BLOOD_OF_SAINT, 4)
  elseif event == "31508-02.htm" then
    if q:has_quest_items(player, BLOOD_OF_SAINT) then
      st:set_cond(2, true)
      q:take_items(player, BLOOD_OF_SAINT, 1)
    else
      htmltext = "31508-03.htm"
    end
  elseif event == "31509-02.htm" then
    if q:has_quest_items(player, BLOOD_OF_SAINT) then
      st:set_cond(3, true)
      q:take_items(player, BLOOD_OF_SAINT, 1)
    else
      htmltext = "31509-03.htm"
    end
  elseif event == "31510-02.htm" then
    if q:has_quest_items(player, BLOOD_OF_SAINT) then
      st:set_cond(4, true)
      q:take_items(player, BLOOD_OF_SAINT, 1)
    else
      htmltext = "31510-03.htm"
    end
  elseif event == "31511-02.htm" then
    if q:has_quest_items(player, BLOOD_OF_SAINT) then
      st:set_cond(5, true)
      q:take_items(player, BLOOD_OF_SAINT, 1)
    else
      htmltext = "31511-03.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 61 and "31517-03.htm" or "31517-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == HIERARCH then
      if cond == 5 then
        htmltext = "31517-07.htm"
        q:add_exp_and_sp(player, 105527, 0)
        st:exit_quest(false, true)
      else
        if q:has_quest_items(player, BLOOD_OF_SAINT) then
          htmltext = "31517-05.htm"
        else
          htmltext = "31517-06.htm"
          st:exit_quest(true)
        end
      end
    elseif sw2 == SAINT_ALTAR_1 then
      if cond == 1 then
        htmltext = "31508-01.htm"
      elseif cond > 1 then
        htmltext = "31508-04.htm"
      end
    elseif sw2 == SAINT_ALTAR_2 then
      if cond == 2 then
        htmltext = "31509-01.htm"
      elseif cond > 2 then
        htmltext = "31509-04.htm"
      end
    elseif sw2 == SAINT_ALTAR_3 then
      if cond == 3 then
        htmltext = "31510-01.htm"
      elseif cond > 3 then
        htmltext = "31510-04.htm"
      end
    elseif sw2 == SAINT_ALTAR_4 then
      if cond == 4 then
        htmltext = "31511-01.htm"
      elseif cond > 4 then
        htmltext = "31511-04.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

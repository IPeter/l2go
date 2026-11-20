-- Translated by questport from quests/Q00124_MeetingTheElroki (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(124, "Q00124_MeetingTheElroki", "Meeting the Elroki")

-- NPCs
local MARQUEZ = 32113
local MUSHIKA = 32114
local ASAMAH = 32115
local KARAKAWEI = 32117
local MANTARASA = 32118

q:add_start_npc(MARQUEZ)
q:add_talk_id(MARQUEZ, MUSHIKA, ASAMAH, KARAKAWEI, MANTARASA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32113-03.htm" then
    st:start_quest()
  elseif event == "32113-04.htm" then
    st:set_cond(2, true)
  elseif event == "32114-02.htm" then
    st:set_cond(3, true)
  elseif event == "32115-04.htm" then
    st:set_cond(4, true)
  elseif event == "32117-02.htm" then
    if st:is_cond(4) then
      st:set("progress", "1")
    end
  elseif event == "32117-03.htm" then
    st:set_cond(5, true)
  elseif event == "32118-02.htm" then
    st:set_cond(6, true)
    q:give_items(player, 8778, 1) -- Egg
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 75 and "32113-01a.htm" or "32113-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == MARQUEZ then
      if cond == 1 then
        htmltext = "32113-03.htm"
      elseif cond > 1 then
        htmltext = "32113-04a.htm"
      end
    elseif sw2 == MUSHIKA then
      if cond == 2 then
        htmltext = "32114-01.htm"
      elseif cond > 2 then
        htmltext = "32114-03.htm"
      end
    elseif sw2 == ASAMAH then
      if cond == 3 then
        htmltext = "32115-01.htm"
      elseif cond == 6 then
        htmltext = "32115-05.htm"
        q:take_items(player, 8778, -1)
        q:give_adena(player, 71318, true)
        st:exit_quest(false, true)
      end
    elseif sw2 == KARAKAWEI then
      if cond == 4 then
        htmltext = "32117-01.htm"
        if st:get_int("progress") == 1 then
          htmltext = "32117-02.htm"
        end
      elseif cond > 4 then
        htmltext = "32117-04.htm"
      end
    elseif sw2 == MANTARASA then
      if cond == 5 then
        htmltext = "32118-01.htm"
      elseif cond > 5 then
        htmltext = "32118-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    if npc:id() == ASAMAH then
      htmltext = "32115-06.htm"
    else
      htmltext = q:get_already_completed_msg(player)
    end
  end
  return htmltext
end)

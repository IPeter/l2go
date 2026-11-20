-- Translated by questport from quests/Q00031_SecretBuriedInTheSwamp (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(31, "Q00031_SecretBuriedInTheSwamp", "Secret Buried in the Swamp")

-- NPCs
local ABERCROMBIE = 31555
local FORGOTTEN_MONUMENT_1 = 31661
local FORGOTTEN_MONUMENT_2 = 31662
local FORGOTTEN_MONUMENT_3 = 31663
local FORGOTTEN_MONUMENT_4 = 31664
local CORPSE_OF_DWARF = 31665
-- Item
local KRORIN_JOURNAL = 7252

q:register_quest_items(KRORIN_JOURNAL)
q:add_start_npc(ABERCROMBIE)
q:add_talk_id(ABERCROMBIE, CORPSE_OF_DWARF, FORGOTTEN_MONUMENT_1, FORGOTTEN_MONUMENT_2, FORGOTTEN_MONUMENT_3, FORGOTTEN_MONUMENT_4)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31555-01.htm" then
    st:start_quest()
  elseif event == "31665-01.htm" then
    st:set_cond(2, true)
    q:give_items(player, KRORIN_JOURNAL, 1)
  elseif event == "31555-04.htm" then
    st:set_cond(3, true)
  elseif event == "31661-01.htm" then
    st:set_cond(4, true)
  elseif event == "31662-01.htm" then
    st:set_cond(5, true)
  elseif event == "31663-01.htm" then
    st:set_cond(6, true)
  elseif event == "31664-01.htm" then
    st:set_cond(7, true)
  elseif event == "31555-07.htm" then
    q:take_items(player, KRORIN_JOURNAL, 1)
    q:give_adena(player, 40000, true)
    q:add_exp_and_sp(player, 130000, 0)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 66 and "31555-00a.htm" or "31555-00.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ABERCROMBIE then
      if cond == 1 then
        htmltext = "31555-02.htm"
      elseif cond == 2 then
        htmltext = "31555-03.htm"
      elseif cond > 2 and cond < 7 then
        htmltext = "31555-05.htm"
      elseif cond == 7 then
        htmltext = "31555-06.htm"
      end
    elseif sw2 == CORPSE_OF_DWARF then
      if cond == 1 then
        htmltext = "31665-00.htm"
      elseif cond > 1 then
        htmltext = "31665-02.htm"
      end
    elseif sw2 == FORGOTTEN_MONUMENT_1 then
      if cond == 3 then
        htmltext = "31661-00.htm"
      elseif cond > 3 then
        htmltext = "31661-02.htm"
      end
    elseif sw2 == FORGOTTEN_MONUMENT_2 then
      if cond == 4 then
        htmltext = "31662-00.htm"
      elseif cond > 4 then
        htmltext = "31662-02.htm"
      end
    elseif sw2 == FORGOTTEN_MONUMENT_3 then
      if cond == 5 then
        htmltext = "31663-00.htm"
      elseif cond > 5 then
        htmltext = "31663-02.htm"
      end
    elseif sw2 == FORGOTTEN_MONUMENT_4 then
      if cond == 6 then
        htmltext = "31664-00.htm"
      elseif cond > 6 then
        htmltext = "31664-02.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

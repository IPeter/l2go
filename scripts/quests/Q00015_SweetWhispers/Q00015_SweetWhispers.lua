-- Translated by questport from quests/Q00015_SweetWhispers (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(15, "Q00015_SweetWhispers", "Sweet Whispers")

-- NPCs
local VLADIMIR = 31302
local HIERARCH = 31517
local MYSTERIOUS_NECRO = 31518

q:add_start_npc(VLADIMIR)
q:add_talk_id(VLADIMIR, HIERARCH, MYSTERIOUS_NECRO)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31302-01.htm" then
    st:start_quest()
  elseif event == "31518-01.htm" then
    st:set_cond(2, true)
  elseif event == "31517-01.htm" then
    q:add_exp_and_sp(player, 60217, 0)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 60 and "31302-00a.htm" or "31302-00.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == VLADIMIR then
      htmltext = "31302-01a.htm"
    elseif sw2 == MYSTERIOUS_NECRO then
      if cond == 1 then
        htmltext = "31518-00.htm"
      elseif cond == 2 then
        htmltext = "31518-01a.htm"
      end
    elseif sw2 == HIERARCH then
      if cond == 2 then
        htmltext = "31517-00.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

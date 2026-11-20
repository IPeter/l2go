-- Translated by questport from quests/Q00121_PavelTheGiant (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(121, "Q00121_PavelTheGiant", "Pavel the Giant")

-- NPCs
local NEWYEAR = 31961
local YUMI = 32041

q:add_start_npc(NEWYEAR)
q:add_talk_id(NEWYEAR, YUMI)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31961-2.htm" then
    st:start_quest()
  elseif event == "32041-2.htm" then
    q:add_exp_and_sp(player, 10000, 0)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 46 and "31961-1a.htm" or "31961-1.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == NEWYEAR then
      htmltext = "31961-2a.htm"
    elseif sw2 == YUMI then
      htmltext = "32041-1.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

-- Translated by questport from quests/Q00122_OminousNews (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(122, "Q00122_OminousNews", "Ominous News")

-- NPCs
local MOIRA = 31979
local KARUDA = 32017

q:add_start_npc(MOIRA)
q:add_talk_id(MOIRA, KARUDA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31979-03.htm" then
    st:start_quest()
  elseif event == "32017-02.htm" then
    q:give_adena(player, 1695, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 20 and "31979-01.htm" or "31979-02.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == MOIRA then
      htmltext = "31979-03.htm"
    elseif sw2 == KARUDA then
      htmltext = "32017-01.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

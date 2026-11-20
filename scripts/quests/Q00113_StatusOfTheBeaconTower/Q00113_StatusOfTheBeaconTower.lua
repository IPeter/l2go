-- Translated by questport from quests/Q00113_StatusOfTheBeaconTower (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(113, "Q00113_StatusOfTheBeaconTower", "Status of the Beacon Tower")

-- NPCs
local MOIRA = 31979
local TORRANT = 32016
-- Item
local BOX = 8086

q:register_quest_items(BOX)
q:add_start_npc(MOIRA)
q:add_talk_id(MOIRA, TORRANT)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = player:get_quest_state(q:name())
  if st == nil then
    return htmltext
  end
  if event == "31979-02.htm" then
    st:start_quest()
    q:give_items(player, BOX, 1)
  elseif event == "32016-02.htm" then
    q:take_items(player, BOX, 1)
    q:give_adena(player, 21578, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 40 and "31979-00.htm" or "31979-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == MOIRA then
      htmltext = "31979-03.htm"
    elseif sw2 == TORRANT then
      htmltext = "32016-01.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

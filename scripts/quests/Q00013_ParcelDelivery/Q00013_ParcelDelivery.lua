-- Translated by questport from quests/Q00013_ParcelDelivery (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(13, "Q00013_ParcelDelivery", "Parcel Delivery")

-- NPCs
local FUNDIN = 31274
local VULCAN = 31539
-- Item
local PACKAGE = 7263

q:register_quest_items(PACKAGE)
q:add_start_npc(FUNDIN)
q:add_talk_id(FUNDIN, VULCAN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31274-2.htm" then
    st:start_quest()
    q:give_items(player, PACKAGE, 1)
  elseif event == "31539-1.htm" then
    q:take_items(player, PACKAGE, 1)
    q:give_adena(player, 82656, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 74 and "31274-1.htm" or "31274-0.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == FUNDIN then
      htmltext = "31274-2.htm"
    elseif sw2 == VULCAN then
      htmltext = "31539-0.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

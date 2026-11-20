-- Translated by questport from quests/Q00009_IntoTheCityOfHumans (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(9, "Q00009_IntoTheCityOfHumans", "Into the City of Humans")

-- NPCs
local PETUKAI = 30583
local TANAPI = 30571
local TAMIL = 30576
-- Rewards
local MARK_OF_TRAVELER = 7570
local SOE_GIRAN = 7126

q:add_start_npc(PETUKAI)
q:add_talk_id(PETUKAI, TANAPI, TAMIL)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30583-01.htm" then
    st:start_quest()
  elseif event == "30571-01.htm" then
    st:set_cond(2, true)
  elseif event == "30576-01.htm" then
    q:give_items(player, MARK_OF_TRAVELER, 1)
    q:reward_items(player, SOE_GIRAN, 1)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() >= 3 and player:race() == Race.ORC then
      htmltext = "30583-00.htm"
    else
      htmltext = "30583-00a.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == PETUKAI then
      if cond == 1 then
        htmltext = "30583-01a.htm"
      end
    elseif sw2 == TANAPI then
      if cond == 1 then
        htmltext = "30571-00.htm"
      elseif cond == 2 then
        htmltext = "30571-01a.htm"
      end
    elseif sw2 == TAMIL then
      if cond == 2 then
        htmltext = "30576-00.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

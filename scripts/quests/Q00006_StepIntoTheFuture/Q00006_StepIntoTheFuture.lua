-- Translated by questport from quests/Q00006_StepIntoTheFuture (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(6, "Q00006_StepIntoTheFuture", "Step into the Future")

-- NPCs
local ROXXY = 30006
local BAULRO = 30033
local SIR_COLLIN = 30311
-- Items
local BAULRO_LETTER = 7571
-- Rewards
local MARK_TRAVELER = 7570
local SOE_GIRAN = 7559

q:register_quest_items(BAULRO_LETTER)
q:add_start_npc(ROXXY)
q:add_talk_id(ROXXY, BAULRO, SIR_COLLIN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30006-03.htm" then
    st:start_quest()
  elseif event == "30033-02.htm" then
    st:set_cond(2, true)
    q:give_items(player, BAULRO_LETTER, 1)
  elseif event == "30311-02.htm" then
    if q:has_quest_items(player, BAULRO_LETTER) then
      st:set_cond(3, true)
      q:take_items(player, BAULRO_LETTER, 1)
    else
      htmltext = "30311-03.htm"
    end
  elseif event == "30006-06.htm" then
    q:give_items(player, MARK_TRAVELER, 1)
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
    if player:race() ~= Race.HUMAN or player:level() < 3 then
      htmltext = "30006-01.htm"
    else
      htmltext = "30006-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ROXXY then
      if cond == 1 or cond == 2 then
        htmltext = "30006-04.htm"
      elseif cond == 3 then
        htmltext = "30006-05.htm"
      end
    elseif sw2 == BAULRO then
      if cond == 1 then
        htmltext = "30033-01.htm"
      elseif cond == 2 then
        htmltext = "30033-03.htm"
      else
        htmltext = "30033-04.htm"
      end
    elseif sw2 == SIR_COLLIN then
      if cond == 2 then
        htmltext = "30311-01.htm"
      elseif cond == 3 then
        htmltext = "30311-03a.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

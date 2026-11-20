-- Translated by questport from quests/Q00008_AnAdventureBegins (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(8, "Q00008_AnAdventureBegins", "An Adventure Begins")

-- NPCs
local JASMINE = 30134
local ROSELYN = 30355
local HARNE = 30144
-- Items
local ROSELYN_NOTE = 7573
-- Rewards
local SOE_GIRAN = 7559
local MARK_TRAVELER = 7570

q:register_quest_items(ROSELYN_NOTE)
q:add_start_npc(JASMINE)
q:add_talk_id(JASMINE, ROSELYN, HARNE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30134-03.htm" then
    st:start_quest()
  elseif event == "30355-02.htm" then
    st:set_cond(2, true)
    q:give_items(player, ROSELYN_NOTE, 1)
  elseif event == "30144-02.htm" then
    st:set_cond(3, true)
    q:take_items(player, ROSELYN_NOTE, 1)
  elseif event == "30134-06.htm" then
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
    if player:level() >= 3 and player:race() == Race.DARK_ELF then
      htmltext = "30134-02.htm"
    else
      htmltext = "30134-01.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == JASMINE then
      if cond == 1 or cond == 2 then
        htmltext = "30134-04.htm"
      elseif cond == 3 then
        htmltext = "30134-05.htm"
      end
    elseif sw2 == ROSELYN then
      if cond == 1 then
        htmltext = "30355-01.htm"
      elseif cond == 2 then
        htmltext = "30355-03.htm"
      end
    elseif sw2 == HARNE then
      if cond == 2 then
        htmltext = "30144-01.htm"
      elseif cond == 3 then
        htmltext = "30144-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

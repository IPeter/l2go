-- Translated by questport from quests/Q00018_MeetingWithTheGoldenRam (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(18, "Q00018_MeetingWithTheGoldenRam", "Meeting with the Golden Ram")

-- NPCs
local DONAL = 31314
local DAISY = 31315
local ABERCROMBIE = 31555
-- Items
local SUPPLY_BOX = 7245

q:register_quest_items(SUPPLY_BOX)
q:add_start_npc(DONAL)
q:add_talk_id(DONAL, DAISY, ABERCROMBIE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31314-03.htm" then
    st:start_quest()
  elseif event == "31315-02.htm" then
    st:set_cond(2, true)
    q:give_items(player, SUPPLY_BOX, 1)
  elseif event == "31555-02.htm" then
    q:take_items(player, SUPPLY_BOX, 1)
    q:give_adena(player, 15000, true)
    q:add_exp_and_sp(player, 50000, 0)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 66 and "31314-02.htm" or "31314-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == DONAL then
      htmltext = "31314-04.htm"
    elseif sw2 == DAISY then
      if cond == 1 then
        htmltext = "31315-01.htm"
      elseif cond == 2 then
        htmltext = "31315-03.htm"
      end
    elseif sw2 == ABERCROMBIE then
      if cond == 2 then
        htmltext = "31555-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

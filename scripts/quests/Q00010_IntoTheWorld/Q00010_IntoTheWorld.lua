-- Translated by questport from quests/Q00010_IntoTheWorld (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(10, "Q00010_IntoTheWorld", "Into the World")

-- Items
local VERY_EXPENSIVE_NECKLACE = 7574
-- Rewards
local SOE_GIRAN = 7559
local MARK_OF_TRAVELER = 7570
-- NPCs
local REED = 30520
local BALANKI = 30533
local GERALD = 30650

q:register_quest_items(VERY_EXPENSIVE_NECKLACE)
q:add_start_npc(BALANKI)
q:add_talk_id(BALANKI, REED, GERALD)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30533-02.htm" then
    st:start_quest()
  elseif event == "30520-02.htm" then
    st:set_cond(2, true)
    q:give_items(player, VERY_EXPENSIVE_NECKLACE, 1)
  elseif event == "30650-02.htm" then
    st:set_cond(3, true)
    q:take_items(player, VERY_EXPENSIVE_NECKLACE, 1)
  elseif event == "30520-04.htm" then
    st:set_cond(4, true)
  elseif event == "30533-05.htm" then
    q:give_items(player, SOE_GIRAN, 1)
    q:reward_items(player, MARK_OF_TRAVELER, 1)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() >= 3 and player:race() == Race.DWARF then
      htmltext = "30533-01.htm"
    else
      htmltext = "30533-01a.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == BALANKI then
      if cond < 4 then
        htmltext = "30533-03.htm"
      elseif cond == 4 then
        htmltext = "30533-04.htm"
      end
    elseif sw2 == REED then
      if cond == 1 then
        htmltext = "30520-01.htm"
      elseif cond == 2 then
        htmltext = "30520-02a.htm"
      elseif cond == 3 then
        htmltext = "30520-03.htm"
      elseif cond == 4 then
        htmltext = "30520-04a.htm"
      end
    elseif sw2 == GERALD then
      if cond == 2 then
        htmltext = "30650-01.htm"
      elseif cond > 2 then
        htmltext = "30650-04.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

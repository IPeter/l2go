-- Translated by questport from quests/Q00151_CureForFever (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(151, "Q00151_CureForFever", "Cure for Fever Disease")

-- NPCs
local ELIAS = 30050
local YOHANES = 30032
-- Items
local POISON_SAC = 703
local FEVER_MEDICINE = 704

q:register_quest_items(FEVER_MEDICINE, POISON_SAC)
q:add_start_npc(ELIAS)
q:add_talk_id(ELIAS, YOHANES)
q:add_kill_id(20103, 20106, 20108)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30050-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 15 and "30050-01.htm" or "30050-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ELIAS then
      if cond == 1 then
        htmltext = "30050-04.htm"
      elseif cond == 2 then
        htmltext = "30050-05.htm"
      elseif cond == 3 then
        htmltext = "30050-06.htm"
        q:take_items(player, FEVER_MEDICINE, 1)
        q:give_items(player, 102, 1)
        st:exit_quest(false, true)
      end
    elseif sw2 == YOHANES then
      if cond == 2 then
        htmltext = "30032-01.htm"
        st:set_cond(3, true)
        q:take_items(player, POISON_SAC, 1)
        q:give_items(player, FEVER_MEDICINE, 1)
      elseif cond == 3 then
        htmltext = "30032-02.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_cond(1) and rnd(10) < 2 then
    q:give_items(killer, POISON_SAC, 1)
    qs:set_cond(2, true)
  end
end)

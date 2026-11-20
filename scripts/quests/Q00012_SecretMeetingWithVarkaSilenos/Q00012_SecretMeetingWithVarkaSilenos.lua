-- Translated by questport from quests/Q00012_SecretMeetingWithVarkaSilenos (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(12, "Q00012_SecretMeetingWithVarkaSilenos", "Secret Meeting With Varka Silenos")

-- NPCs
local CADMON = 31296
local HELMUT = 31258
local NARAN_ASHANUK = 31378
-- Items
local MUNITIONS_BOX = 7232

q:register_quest_items(MUNITIONS_BOX)
q:add_start_npc(CADMON)
q:add_talk_id(CADMON, HELMUT, NARAN_ASHANUK)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31296-03.htm" then
    st:start_quest()
  elseif event == "31258-02.htm" then
    q:give_items(player, MUNITIONS_BOX, 1)
    st:set_cond(2, true)
  elseif event == "31378-02.htm" then
    q:take_items(player, MUNITIONS_BOX, 1)
    q:add_exp_and_sp(player, 79761, 0)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 74 and "31296-02.htm" or "31296-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == CADMON then
      if cond == 1 then
        htmltext = "31296-04.htm"
      end
    elseif sw2 == HELMUT then
      if cond == 1 then
        htmltext = "31258-01.htm"
      elseif cond == 2 then
        htmltext = "31258-03.htm"
      end
    elseif sw2 == NARAN_ASHANUK then
      if cond == 2 then
        htmltext = "31378-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

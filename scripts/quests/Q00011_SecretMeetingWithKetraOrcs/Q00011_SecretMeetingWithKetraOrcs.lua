-- Translated by questport from quests/Q00011_SecretMeetingWithKetraOrcs (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(11, "Q00011_SecretMeetingWithKetraOrcs", "Secret Meeting With Ketra Orcs")

-- NPCs
local CADMON = 31296
local LEON = 31256
local WAHKAN = 31371
-- Items
local MUNITIONS_BOX = 7231

q:register_quest_items(MUNITIONS_BOX)
q:add_start_npc(CADMON)
q:add_talk_id(CADMON, LEON, WAHKAN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31296-03.htm" then
    st:start_quest()
  elseif event == "31256-02.htm" then
    q:give_items(player, MUNITIONS_BOX, 1)
    st:set_cond(2, true)
  elseif event == "31371-02.htm" then
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
    elseif sw2 == LEON then
      if cond == 1 then
        htmltext = "31256-01.htm"
      elseif cond == 2 then
        htmltext = "31256-03.htm"
      end
    elseif sw2 == WAHKAN then
      if cond == 2 then
        htmltext = "31371-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

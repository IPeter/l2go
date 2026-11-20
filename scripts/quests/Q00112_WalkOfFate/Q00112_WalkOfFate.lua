-- Translated by questport from quests/Q00112_WalkOfFate (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(112, "Q00112_WalkOfFate", "Walk of Fate")

-- NPCs
local LIVINA = 30572
local KARUDA = 32017
-- Rewards
local ENCHANT_D = 956

q:add_start_npc(LIVINA)
q:add_talk_id(LIVINA, KARUDA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30572-02.htm" then
    st:start_quest()
  elseif event == "32017-02.htm" then
    q:give_items(player, ENCHANT_D, 1)
    q:give_adena(player, 4665, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 20 and "30572-00.htm" or "30572-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == LIVINA then
      htmltext = "30572-03.htm"
    elseif sw2 == KARUDA then
      htmltext = "32017-01.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

-- Translated by questport from quests/Q00338_AlligatorHunter (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(338, "Q00338_AlligatorHunter", "Alligator Hunter")

-- Item
local ALLIGATOR_PELT = 4337

q:register_quest_items(ALLIGATOR_PELT)
q:add_start_npc(30892) -- Enverun
q:add_talk_id(30892)
q:add_kill_id(20135) -- Alligator

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30892-02.htm" then
    st:start_quest()
  elseif event == "30892-05.htm" then
    local pelts = q:get_quest_items_count(player, ALLIGATOR_PELT)
    local reward = pelts * 40
    if pelts > 10 then
      reward = reward + 3430
    end
    q:take_items(player, ALLIGATOR_PELT, -1)
    q:give_adena(player, reward, true)
  elseif event == "30892-08.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 40 and "30892-00.htm" or "30892-01.htm"
  elseif sw == State.STARTED then
    htmltext = q:has_quest_items(player, ALLIGATOR_PELT) and "30892-03.htm" or "30892-04.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  q:give_items(player, ALLIGATOR_PELT, 1)
  q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
end)

-- Translated by questport from quests/Q00261_CollectorsDream (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(261, "Q00261_CollectorsDream", "Collector's Dream")

-- Items
local GIANT_SPIDER_LEG = 1087

q:register_quest_items(GIANT_SPIDER_LEG)
q:add_start_npc(30222) -- Alshupes
q:add_talk_id(30222)
q:add_kill_id(20308, 20460, 20466)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30222-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 15 and "30222-01.htm" or "30222-02.htm"
  elseif sw == State.STARTED then
    if st:is_cond(2) then
      htmltext = "30222-05.htm"
      q:take_items(player, GIANT_SPIDER_LEG, -1)
      q:give_adena(player, 1000, true)
      q:add_exp_and_sp(player, 2000, 0)
      st:exit_quest(true, true)
    else
      htmltext = "30222-04.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  q:give_items(player, GIANT_SPIDER_LEG, 1)
  if q:get_quest_items_count(player, GIANT_SPIDER_LEG) >= 8 then
    st:set_cond(2, true)
  else
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)

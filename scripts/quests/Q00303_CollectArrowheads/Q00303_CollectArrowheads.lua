-- Translated by questport from quests/Q00303_CollectArrowheads (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(303, "Q00303_CollectArrowheads", "Collect Arrowheads")

-- Item
local ORCISH_ARROWHEAD = 963

q:register_quest_items(ORCISH_ARROWHEAD)
q:add_start_npc(30029) -- Minia
q:add_talk_id(30029)
q:add_kill_id(20361)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30029-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 10 and "30029-01.htm" or "30029-02.htm"
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30029-04.htm"
    else
      htmltext = "30029-05.htm"
      q:take_items(player, ORCISH_ARROWHEAD, -1)
      q:give_adena(player, 1000, true)
      q:add_exp_and_sp(player, 2000, 0)
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if rnd(10) < 4 then
    q:give_items(player, ORCISH_ARROWHEAD, 1)
    if q:get_quest_items_count(player, ORCISH_ARROWHEAD) < 10 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

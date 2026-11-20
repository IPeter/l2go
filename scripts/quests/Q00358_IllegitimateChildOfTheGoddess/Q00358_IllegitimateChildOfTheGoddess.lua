-- Translated by questport from quests/Q00358_IllegitimateChildOfTheGoddess (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(358, "Q00358_IllegitimateChildOfTheGoddess", "Illegitimate Child of A Goddess")

-- Item
local SCALE = 5868
-- Reward
local REWARD = {6329, 6331, 6333, 6335, 6337, 6339, 5364, 5366}

q:register_quest_items(SCALE)
q:add_start_npc(30862) -- Oltlin
q:add_talk_id(30862)
q:add_kill_id(20672, 20673) -- Trives, Falibati

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30862-05.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 63 and "30862-01.htm" or "30862-02.htm"
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30862-06.htm"
    else
      htmltext = "30862-07.htm"
      q:take_items(player, SCALE, -1)
      q:give_items(player, rnd.entry(REWARD), 1)
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
  if rnd(100) < (npc:id() == 20672 and 68 or 66) then
    q:give_items(player, SCALE, 1)
    if q:get_quest_items_count(player, SCALE) < 108 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

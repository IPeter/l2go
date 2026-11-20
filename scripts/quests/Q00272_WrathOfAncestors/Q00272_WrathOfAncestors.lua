-- Translated by questport from quests/Q00272_WrathOfAncestors (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(272, "Q00272_WrathOfAncestors", "Wrath of Ancestors")

-- Item
local GRAVE_ROBBERS_HEAD = 1474

q:register_quest_items(GRAVE_ROBBERS_HEAD)
q:add_start_npc(30572) -- Livina
q:add_talk_id(30572)
q:add_kill_id(20319, 20320) -- Goblin Grave Robber, Goblin Tomb Raider Leader

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30572-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ORC then
      htmltext = "30572-00.htm"
    elseif player:level() < 5 then
      htmltext = "30572-01.htm"
    else
      htmltext = "30572-02.htm"
    end
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30572-04.htm"
    else
      htmltext = "30572-05.htm"
      q:take_items(player, GRAVE_ROBBERS_HEAD, -1)
      q:give_adena(player, 1500, true)
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
  q:give_items(player, GRAVE_ROBBERS_HEAD, 1)
  if q:get_quest_items_count(player, GRAVE_ROBBERS_HEAD) < 50 then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  else
    st:set_cond(2, true)
  end
end)

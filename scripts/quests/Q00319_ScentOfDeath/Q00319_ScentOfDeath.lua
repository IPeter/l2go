-- Translated by questport from quests/Q00319_ScentOfDeath (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(319, "Q00319_ScentOfDeath", "Scent of Death")

-- Item
local ZOMBIE_SKIN = 1045

q:register_quest_items(ZOMBIE_SKIN)
q:add_start_npc(30138) -- Minaless
q:add_talk_id(30138)
q:add_kill_id(20015, 20020)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30138-04.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 11 and "30138-02.htm" or "30138-03.htm"
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30138-05.htm"
    else
      htmltext = "30138-06.htm"
      q:take_items(player, ZOMBIE_SKIN, -1)
      q:give_adena(player, 3350, true)
      q:reward_items(player, 1060, 1)
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
  if rnd(10) < 2 then
    q:give_items(player, ZOMBIE_SKIN, 1)
    if q:get_quest_items_count(player, ZOMBIE_SKIN) < 5 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

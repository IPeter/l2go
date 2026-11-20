-- Translated by questport from quests/Q00313_CollectSpores (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(313, "Q00313_CollectSpores", "Collect Spores")

-- Item
local SPORE_SAC = 1118

q:register_quest_items(SPORE_SAC)
q:add_start_npc(30150) -- Herbiel
q:add_talk_id(30150)
q:add_kill_id(20509) -- Spore Fungus

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30150-05.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 8 and "30150-02.htm" or "30150-03.htm"
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30150-06.htm"
    else
      htmltext = "30150-07.htm"
      q:take_items(player, SPORE_SAC, -1)
      q:give_adena(player, 3500, true)
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
    q:give_items(player, SPORE_SAC, 1)
    if q:get_quest_items_count(player, SPORE_SAC) < 10 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

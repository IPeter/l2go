-- Translated by questport from quests/Q00295_DreamingOfTheSkies (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(295, "Q00295_DreamingOfTheSkies", "Dreaming of the Skies")

-- Item
local FLOATING_STONE = 1492
-- Reward
local RING_OF_FIREFLY = 1509

q:register_quest_items(FLOATING_STONE)
q:add_start_npc(30536) -- Arin
q:add_talk_id(30536)
q:add_kill_id(20153) -- Magical Weaver

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30536-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 11 and "30536-01.htm" or "30536-02.htm"
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30536-04.htm"
    else
      q:take_items(player, FLOATING_STONE, -1)
      if not q:has_quest_items(player, RING_OF_FIREFLY) then
        htmltext = "30536-05.htm"
        q:give_items(player, RING_OF_FIREFLY, 1)
      else
        htmltext = "30536-06.htm"
        q:give_adena(player, 2400, true)
      end
      q:add_exp_and_sp(player, 0, 500)
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
  q:give_items(player, FLOATING_STONE, rnd(100) > 25 and 1 or 2)
  if q:get_quest_items_count(player, FLOATING_STONE) < 50 then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  else
    st:set_cond(2, true)
  end
end)

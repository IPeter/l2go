-- Translated by questport from quests/Q00659_IdRatherBeCollectingFairyBreath (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(659, "Q00659_IdRatherBeCollectingFairyBreath", "I'd Rather Be Collecting Fairy Breath")

-- NPCs
local GALATEA = 30634
-- Monsters
local SOBBING_WIND = 21023
local BABBLING_WIND = 21024
local GIGGLING_WIND = 21025
-- Item
local FAIRY_BREATH = 8286

q:register_quest_items(FAIRY_BREATH)
q:add_start_npc(GALATEA)
q:add_talk_id(GALATEA)
q:add_kill_id(GIGGLING_WIND, BABBLING_WIND, SOBBING_WIND)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30634-03.htm" then
    st:start_quest()
  elseif event == "30634-06.htm" then
    local count = q:get_quest_items_count(player, FAIRY_BREATH)
    if count > 0 then
      q:take_items(player, FAIRY_BREATH, count)
      if count < 10 then
        q:give_adena(player, count * 50, true)
      else
        q:give_adena(player, count * 50 + 5365, true)
      end
    end
  elseif event == "30634-08.htm" then
    st:exit_quest(true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 26 and "30634-01.htm" or "30634-02.htm"
  elseif sw == State.STARTED then
    htmltext = not q:has_quest_items(player, FAIRY_BREATH) and "30634-04.htm" or "30634-05.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if rnd(10) < 9 then
    q:give_items(player, FAIRY_BREATH, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)

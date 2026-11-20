-- Translated by questport from quests/Q00324_SweetestVenom (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(324, "Q00324_SweetestVenom", "Sweetest Venom")

-- Item
local VENOM_SAC = 1077
-- Drop chances
local CHANCES = {}

CHANCES[20034] = 220000
CHANCES[20038] = 230000
CHANCES[20043] = 250000

q:register_quest_items(VENOM_SAC)
q:add_start_npc(30351) -- Astaron
q:add_talk_id(30351)
q:add_kill_id(20034, 20038, 20043)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30351-04.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 18 and "30351-02.htm" or "30351-03.htm"
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30351-05.htm"
    else
      htmltext = "30351-06.htm"
      q:take_items(player, VENOM_SAC, -1)
      q:give_adena(player, 5810, true)
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
  if rnd(1000000) < CHANCES[npc:id()] then
    q:give_items(player, VENOM_SAC, 1)
    if q:get_quest_items_count(player, VENOM_SAC) < 10 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

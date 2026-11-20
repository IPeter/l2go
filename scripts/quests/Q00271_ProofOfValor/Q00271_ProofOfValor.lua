-- Translated by questport from quests/Q00271_ProofOfValor (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(271, "Q00271_ProofOfValor", "Proof of Valor")

-- Item
local KASHA_WOLF_FANG = 1473
-- Rewards
local NECKLACE_OF_VALOR = 1507
local NECKLACE_OF_COURAGE = 1506

q:register_quest_items(KASHA_WOLF_FANG)
q:add_start_npc(30577) -- Rukain
q:add_talk_id(30577)
q:add_kill_id(20475) -- Kasha Wolf

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30577-03.htm" then
    st:start_quest()
    if q:has_at_least_one_quest_item(player, NECKLACE_OF_COURAGE, NECKLACE_OF_VALOR) then
      htmltext = "30577-07.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ORC then
      htmltext = "30577-00.htm"
    elseif player:level() < 4 then
      htmltext = "30577-01.htm"
    else
      htmltext = q:has_at_least_one_quest_item(player, NECKLACE_OF_COURAGE, NECKLACE_OF_VALOR) and "30577-06.htm" or "30577-02.htm"
    end
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = q:has_at_least_one_quest_item(player, NECKLACE_OF_COURAGE, NECKLACE_OF_VALOR) and "30577-07.htm" or "30577-04.htm"
    else
      htmltext = "30577-05.htm"
      q:take_items(player, KASHA_WOLF_FANG, -1)
      q:give_items(player, rnd(100) < 10 and NECKLACE_OF_VALOR or NECKLACE_OF_COURAGE, 1)
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
  q:give_items(player, KASHA_WOLF_FANG, rnd(4) == 0 and 2 or 1)
  if q:get_quest_items_count(player, KASHA_WOLF_FANG) < 50 then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  else
    st:set_cond(2, true)
  end
end)

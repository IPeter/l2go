-- Translated by questport from quests/Q00382_KailsMagicCoin (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(382, "Q00382_KailsMagicCoin", "Kail's Magic Coin")

-- Monsters
local FALLEN_ORC = 21017
local FALLEN_ORC_ARCHER = 21019
local FALLEN_ORC_SHAMAN = 21020
local FALLEN_ORC_CAPTAIN = 21022
-- Items
local ROYAL_MEMBERSHIP = 5898
local SILVER_BASILISK = 5961
local GOLD_GOLEM = 5962
local BLOOD_DRAGON = 5963

q:register_quest_items(SILVER_BASILISK, GOLD_GOLEM, BLOOD_DRAGON)
q:add_start_npc(30687) -- Vergara
q:add_talk_id(30687)
q:add_kill_id(FALLEN_ORC, FALLEN_ORC_ARCHER, FALLEN_ORC_SHAMAN, FALLEN_ORC_CAPTAIN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30687-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = (player:level() < 55 or not q:has_quest_items(player, ROYAL_MEMBERSHIP)) and "30687-01.htm" or "30687-02.htm"
  elseif sw == State.STARTED then
    htmltext = "30687-04.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  local sw = npc:id()
  if sw == FALLEN_ORC then
    if rnd(10) < 1 then
      q:give_items(player, SILVER_BASILISK, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == FALLEN_ORC_ARCHER then
    if rnd(10) < 1 then
      q:give_items(player, GOLD_GOLEM, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == FALLEN_ORC_SHAMAN then
    if rnd(10) < 1 then
      q:give_items(player, BLOOD_DRAGON, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == FALLEN_ORC_CAPTAIN then
    if rnd(10) < 1 then
      q:give_items(player, 5961 + rnd(3), 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)

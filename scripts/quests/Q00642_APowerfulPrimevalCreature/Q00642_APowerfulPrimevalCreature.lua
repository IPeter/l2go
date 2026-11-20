-- Translated by questport from quests/Q00642_APowerfulPrimevalCreature (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(642, "Q00642_APowerfulPrimevalCreature", "A Powerful Primeval Creature")

-- NPC
local ANCIENT_EGG = 18344
-- Items
local DINOSAUR_TISSUE = 8774
local DINOSAUR_EGG = 8775
-- Rewards
local REWARDS = {8690, 8692, 8694, 8696, 8698, 8700, 8702, 8704, 8706, 8708, 8710}

q:register_quest_items(DINOSAUR_TISSUE, DINOSAUR_EGG)
q:add_start_npc(32105) -- Dinn
q:add_talk_id(32105)
-- Dinosaurs + egg
q:add_kill_id(22196, 22197, 22198, 22199, 22200, 22201, 22202, 22203, 22204, 22205, 22218, 22219, 22220, 22223, 22224, 22225, ANCIENT_EGG)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32105-04.htm" then
    st:start_quest()
  elseif event == "32105-08.htm" then
    if q:get_quest_items_count(player, DINOSAUR_TISSUE) >= 150 and q:has_quest_items(player, DINOSAUR_EGG) then
      htmltext = "32105-06.htm"
    end
  elseif event == "32105-07.htm" then
    local tissues = q:get_quest_items_count(player, DINOSAUR_TISSUE)
    if tissues > 0 then
      q:take_items(player, DINOSAUR_TISSUE, -1)
      q:give_adena(player, tissues * 5000, true)
    else
      htmltext = "32105-08.htm"
    end
  elseif J.str_contains(event, "event_") then
    if q:get_quest_items_count(player, DINOSAUR_TISSUE) >= 150 and q:has_quest_items(player, DINOSAUR_EGG) then
      htmltext = "32105-07.htm"
      q:take_items(player, DINOSAUR_TISSUE, 150)
      q:take_items(player, DINOSAUR_EGG, 1)
      q:give_adena(player, 44000, true)
      q:give_items(player, REWARDS[J.parse_int(J.split(event, "_")[2]) + 1], 1)
    else
      htmltext = "32105-08.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 75 and "32105-00.htm" or "32105-01.htm"
  elseif sw == State.STARTED then
    htmltext = not q:has_quest_items(player, DINOSAUR_TISSUE) and "32105-08.htm" or "32105-05.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if npc:id() == ANCIENT_EGG then
    if rnd(100) < 1 then
      q:give_items(player, DINOSAUR_EGG, 1)
      if q:get_quest_items_count(player, DINOSAUR_TISSUE) >= 150 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif rnd(100) < 33 then
    q:reward_items(player, DINOSAUR_TISSUE, 1)
    if q:get_quest_items_count(player, DINOSAUR_TISSUE) >= 150 and q:has_quest_items(player, DINOSAUR_EGG) then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)

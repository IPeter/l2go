-- Translated by questport from quests/Q00222_TestOfTheDuelist (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(222, "Q00222_TestOfTheDuelist", "Test of the Duelist")

-- NPC
local KAIEN = 30623
-- Monsters
local PUNCHER = 20085
local NOBLE_ANT_LEADER = 20090
local MARSH_STAKATO_DRONE = 20234
local DEAD_SEEKER = 20202
local BREKA_ORC_OVERLORD = 20270
local FETTERED_SOUL = 20552
local LETO_LIZARDMAN_OVERLORD = 20582
local ENCHANTED_MONSTEREYE = 20564
local TAMLIN_ORC = 20601
local TAMLIN_ORC_ARCHER = 20602
local EXCURO = 20214
local KRATOR = 20217
local GRANDIS = 20554
local TIMAK_ORC_OVERLORD = 20588
local LAKIN = 20604
-- Items
local ORDER_GLUDIO = 2763
local ORDER_DION = 2764
local ORDER_GIRAN = 2765
local ORDER_OREN = 2766
local ORDER_ADEN = 2767
local PUNCHER_SHARD = 2768
local NOBLE_ANT_FEELER = 2769
local DRONE_CHITIN = 2770
local DEAD_SEEKER_FANG = 2771
local OVERLORD_NECKLACE = 2772
local FETTERED_SOUL_CHAIN = 2773
local CHIEF_AMULET = 2774
local ENCHANTED_EYE_MEAT = 2775
local TAMRIN_ORC_RING = 2776
local TAMRIN_ORC_ARROW = 2777
local FINAL_ORDER = 2778
local EXCURO_SKIN = 2779
local KRATOR_SHARD = 2780
local GRANDIS_SKIN = 2781
local TIMAK_ORC_BELT = 2782
local LAKIN_MACE = 2783
-- Rewards
local MARK_OF_DUELIST = 2762
local DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(ORDER_GLUDIO, ORDER_DION, ORDER_GIRAN, ORDER_OREN, ORDER_ADEN, FINAL_ORDER, PUNCHER_SHARD, NOBLE_ANT_FEELER, DRONE_CHITIN, DEAD_SEEKER_FANG, OVERLORD_NECKLACE, FETTERED_SOUL_CHAIN, CHIEF_AMULET, ENCHANTED_EYE_MEAT, TAMRIN_ORC_RING, TAMRIN_ORC_ARROW, EXCURO_SKIN, KRATOR_SHARD, GRANDIS_SKIN, TIMAK_ORC_BELT, LAKIN_MACE)
q:add_start_npc(KAIEN)
q:add_talk_id(KAIEN)
q:add_kill_id(PUNCHER, NOBLE_ANT_LEADER, MARSH_STAKATO_DRONE, DEAD_SEEKER, BREKA_ORC_OVERLORD, FETTERED_SOUL, LETO_LIZARDMAN_OVERLORD, ENCHANTED_MONSTEREYE, TAMLIN_ORC, TAMLIN_ORC_ARCHER, EXCURO, KRATOR, GRANDIS, TIMAK_ORC_OVERLORD, LAKIN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30623-04.htm" then
    if player:race() == Race.ORC then
      htmltext = "30623-05.htm"
    end
  elseif event == "30623-07.htm" then
    st:start_quest()
    st:set_cond(2)
    q:give_items(player, ORDER_GLUDIO, 1)
    q:give_items(player, ORDER_DION, 1)
    q:give_items(player, ORDER_GIRAN, 1)
    q:give_items(player, ORDER_OREN, 1)
    q:give_items(player, ORDER_ADEN, 1)
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30623-07a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30623-16.htm" then
    if st:is_cond(3) then
      st:set_cond(4, true)
      q:take_items(player, ORDER_GLUDIO, 1)
      q:take_items(player, ORDER_DION, 1)
      q:take_items(player, ORDER_GIRAN, 1)
      q:take_items(player, ORDER_OREN, 1)
      q:take_items(player, ORDER_ADEN, 1)
      q:take_items(player, PUNCHER_SHARD, -1)
      q:take_items(player, NOBLE_ANT_FEELER, -1)
      q:take_items(player, DRONE_CHITIN, -1)
      q:take_items(player, DEAD_SEEKER_FANG, -1)
      q:take_items(player, OVERLORD_NECKLACE, -1)
      q:take_items(player, FETTERED_SOUL_CHAIN, -1)
      q:take_items(player, CHIEF_AMULET, -1)
      q:take_items(player, ENCHANTED_EYE_MEAT, -1)
      q:take_items(player, TAMRIN_ORC_RING, -1)
      q:take_items(player, TAMRIN_ORC_ARROW, -1)
      q:give_items(player, FINAL_ORDER, 1)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    local classId = player:class_id()
    if classId ~= 1 and classId ~= 47 and classId ~= 19 and classId ~= 32 then
      htmltext = "30623-02.htm"
    elseif player:level() < 39 then
      htmltext = "30623-01.htm"
    else
      htmltext = "30623-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 2 then
      htmltext = "30623-07a.htm"
    elseif cond == 3 then
      htmltext = "30623-13.htm"
    elseif cond == 4 then
      htmltext = "30623-17.htm"
    elseif cond == 5 then
      htmltext = "30623-18.htm"
      q:take_items(player, FINAL_ORDER, 1)
      q:take_items(player, EXCURO_SKIN, -1)
      q:take_items(player, KRATOR_SHARD, -1)
      q:take_items(player, GRANDIS_SKIN, -1)
      q:take_items(player, TIMAK_ORC_BELT, -1)
      q:take_items(player, LAKIN_MACE, -1)
      q:give_items(player, MARK_OF_DUELIST, 1)
      q:add_exp_and_sp(player, 47015, 20000)
      player:social_action(3)
      st:exit_quest(false, true)
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if st:is_cond(2) then
    local sw = npc:id()
    if sw == PUNCHER then
      if q:get_quest_items_count(player, PUNCHER_SHARD) < 10 then
        q:give_items(player, PUNCHER_SHARD, 1)
        if q:get_quest_items_count(player, PUNCHER_SHARD) >= 10 and q:get_quest_items_count(player, NOBLE_ANT_FEELER) >= 10 and q:get_quest_items_count(player, DRONE_CHITIN) >= 10 and q:get_quest_items_count(player, DEAD_SEEKER_FANG) >= 10 and q:get_quest_items_count(player, OVERLORD_NECKLACE) >= 10 and q:get_quest_items_count(player, FETTERED_SOUL_CHAIN) >= 10 and q:get_quest_items_count(player, CHIEF_AMULET) >= 10 and q:get_quest_items_count(player, ENCHANTED_EYE_MEAT) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_RING) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_ARROW) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == NOBLE_ANT_LEADER then
      if q:get_quest_items_count(player, NOBLE_ANT_FEELER) < 10 then
        q:give_items(player, NOBLE_ANT_FEELER, 1)
        if q:get_quest_items_count(player, NOBLE_ANT_FEELER) >= 10 and q:get_quest_items_count(player, PUNCHER_SHARD) >= 10 and q:get_quest_items_count(player, DRONE_CHITIN) >= 10 and q:get_quest_items_count(player, DEAD_SEEKER_FANG) >= 10 and q:get_quest_items_count(player, OVERLORD_NECKLACE) >= 10 and q:get_quest_items_count(player, FETTERED_SOUL_CHAIN) >= 10 and q:get_quest_items_count(player, CHIEF_AMULET) >= 10 and q:get_quest_items_count(player, ENCHANTED_EYE_MEAT) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_RING) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_ARROW) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == MARSH_STAKATO_DRONE then
      if q:get_quest_items_count(player, DRONE_CHITIN) < 10 then
        q:give_items(player, DRONE_CHITIN, 1)
        if q:get_quest_items_count(player, DRONE_CHITIN) >= 10 and q:get_quest_items_count(player, PUNCHER_SHARD) >= 10 and q:get_quest_items_count(player, NOBLE_ANT_FEELER) >= 10 and q:get_quest_items_count(player, DEAD_SEEKER_FANG) >= 10 and q:get_quest_items_count(player, OVERLORD_NECKLACE) >= 10 and q:get_quest_items_count(player, FETTERED_SOUL_CHAIN) >= 10 and q:get_quest_items_count(player, CHIEF_AMULET) >= 10 and q:get_quest_items_count(player, ENCHANTED_EYE_MEAT) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_RING) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_ARROW) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == DEAD_SEEKER then
      if q:get_quest_items_count(player, DEAD_SEEKER_FANG) < 10 then
        q:give_items(player, DEAD_SEEKER_FANG, 1)
        if q:get_quest_items_count(player, DEAD_SEEKER_FANG) >= 10 and q:get_quest_items_count(player, PUNCHER_SHARD) >= 10 and q:get_quest_items_count(player, NOBLE_ANT_FEELER) >= 10 and q:get_quest_items_count(player, DRONE_CHITIN) >= 10 and q:get_quest_items_count(player, OVERLORD_NECKLACE) >= 10 and q:get_quest_items_count(player, FETTERED_SOUL_CHAIN) >= 10 and q:get_quest_items_count(player, CHIEF_AMULET) >= 10 and q:get_quest_items_count(player, ENCHANTED_EYE_MEAT) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_RING) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_ARROW) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == BREKA_ORC_OVERLORD then
      if q:get_quest_items_count(player, OVERLORD_NECKLACE) < 10 then
        q:give_items(player, OVERLORD_NECKLACE, 1)
        if q:get_quest_items_count(player, OVERLORD_NECKLACE) >= 10 and q:get_quest_items_count(player, PUNCHER_SHARD) >= 10 and q:get_quest_items_count(player, NOBLE_ANT_FEELER) >= 10 and q:get_quest_items_count(player, DRONE_CHITIN) >= 10 and q:get_quest_items_count(player, DEAD_SEEKER_FANG) >= 10 and q:get_quest_items_count(player, FETTERED_SOUL_CHAIN) >= 10 and q:get_quest_items_count(player, CHIEF_AMULET) >= 10 and q:get_quest_items_count(player, ENCHANTED_EYE_MEAT) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_RING) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_ARROW) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == FETTERED_SOUL then
      if q:get_quest_items_count(player, FETTERED_SOUL_CHAIN) < 10 then
        q:give_items(player, FETTERED_SOUL_CHAIN, 1)
        if q:get_quest_items_count(player, FETTERED_SOUL_CHAIN) >= 10 and q:get_quest_items_count(player, PUNCHER_SHARD) >= 10 and q:get_quest_items_count(player, NOBLE_ANT_FEELER) >= 10 and q:get_quest_items_count(player, DRONE_CHITIN) >= 10 and q:get_quest_items_count(player, DEAD_SEEKER_FANG) >= 10 and q:get_quest_items_count(player, OVERLORD_NECKLACE) >= 10 and q:get_quest_items_count(player, CHIEF_AMULET) >= 10 and q:get_quest_items_count(player, ENCHANTED_EYE_MEAT) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_RING) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_ARROW) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == LETO_LIZARDMAN_OVERLORD then
      if q:get_quest_items_count(player, CHIEF_AMULET) < 10 then
        q:give_items(player, CHIEF_AMULET, 1)
        if q:get_quest_items_count(player, CHIEF_AMULET) >= 10 and q:get_quest_items_count(player, PUNCHER_SHARD) >= 10 and q:get_quest_items_count(player, NOBLE_ANT_FEELER) >= 10 and q:get_quest_items_count(player, DRONE_CHITIN) >= 10 and q:get_quest_items_count(player, DEAD_SEEKER_FANG) >= 10 and q:get_quest_items_count(player, OVERLORD_NECKLACE) >= 10 and q:get_quest_items_count(player, FETTERED_SOUL_CHAIN) >= 10 and q:get_quest_items_count(player, ENCHANTED_EYE_MEAT) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_RING) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_ARROW) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == ENCHANTED_MONSTEREYE then
      if q:get_quest_items_count(player, ENCHANTED_EYE_MEAT) < 10 then
        q:give_items(player, ENCHANTED_EYE_MEAT, 1)
        if q:get_quest_items_count(player, ENCHANTED_EYE_MEAT) >= 10 and q:get_quest_items_count(player, PUNCHER_SHARD) >= 10 and q:get_quest_items_count(player, NOBLE_ANT_FEELER) >= 10 and q:get_quest_items_count(player, DRONE_CHITIN) >= 10 and q:get_quest_items_count(player, DEAD_SEEKER_FANG) >= 10 and q:get_quest_items_count(player, OVERLORD_NECKLACE) >= 10 and q:get_quest_items_count(player, FETTERED_SOUL_CHAIN) >= 10 and q:get_quest_items_count(player, CHIEF_AMULET) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_RING) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_ARROW) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == TAMLIN_ORC then
      if q:get_quest_items_count(player, TAMRIN_ORC_RING) < 10 then
        q:give_items(player, TAMRIN_ORC_RING, 1)
        if q:get_quest_items_count(player, TAMRIN_ORC_RING) >= 10 and q:get_quest_items_count(player, PUNCHER_SHARD) >= 10 and q:get_quest_items_count(player, NOBLE_ANT_FEELER) >= 10 and q:get_quest_items_count(player, DRONE_CHITIN) >= 10 and q:get_quest_items_count(player, DEAD_SEEKER_FANG) >= 10 and q:get_quest_items_count(player, OVERLORD_NECKLACE) >= 10 and q:get_quest_items_count(player, FETTERED_SOUL_CHAIN) >= 10 and q:get_quest_items_count(player, CHIEF_AMULET) >= 10 and q:get_quest_items_count(player, ENCHANTED_EYE_MEAT) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_ARROW) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == TAMLIN_ORC_ARCHER then
      if q:get_quest_items_count(player, TAMRIN_ORC_ARROW) < 10 then
        q:give_items(player, TAMRIN_ORC_ARROW, 1)
        if q:get_quest_items_count(player, TAMRIN_ORC_ARROW) >= 10 and q:get_quest_items_count(player, PUNCHER_SHARD) >= 10 and q:get_quest_items_count(player, NOBLE_ANT_FEELER) >= 10 and q:get_quest_items_count(player, DRONE_CHITIN) >= 10 and q:get_quest_items_count(player, DEAD_SEEKER_FANG) >= 10 and q:get_quest_items_count(player, OVERLORD_NECKLACE) >= 10 and q:get_quest_items_count(player, FETTERED_SOUL_CHAIN) >= 10 and q:get_quest_items_count(player, CHIEF_AMULET) >= 10 and q:get_quest_items_count(player, ENCHANTED_EYE_MEAT) >= 10 and q:get_quest_items_count(player, TAMRIN_ORC_RING) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif st:is_cond(4) then
    local sw2 = npc:id()
    if sw2 == EXCURO then
      if q:get_quest_items_count(player, EXCURO_SKIN) < 3 then
        q:give_items(player, EXCURO_SKIN, 1)
        if q:get_quest_items_count(player, EXCURO_SKIN) >= 3 and q:get_quest_items_count(player, KRATOR_SHARD) >= 3 and q:get_quest_items_count(player, LAKIN_MACE) >= 3 and q:get_quest_items_count(player, GRANDIS_SKIN) >= 3 and q:get_quest_items_count(player, TIMAK_ORC_BELT) >= 3 then
          st:set_cond(5, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw2 == KRATOR then
      if q:get_quest_items_count(player, KRATOR_SHARD) < 3 then
        q:give_items(player, KRATOR_SHARD, 1)
        if q:get_quest_items_count(player, KRATOR_SHARD) >= 3 and q:get_quest_items_count(player, EXCURO_SKIN) >= 3 and q:get_quest_items_count(player, LAKIN_MACE) >= 3 and q:get_quest_items_count(player, GRANDIS_SKIN) >= 3 and q:get_quest_items_count(player, TIMAK_ORC_BELT) >= 3 then
          st:set_cond(5, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw2 == LAKIN then
      if q:get_quest_items_count(player, LAKIN_MACE) < 3 then
        q:give_items(player, LAKIN_MACE, 1)
        if q:get_quest_items_count(player, LAKIN_MACE) >= 3 and q:get_quest_items_count(player, EXCURO_SKIN) >= 3 and q:get_quest_items_count(player, KRATOR_SHARD) >= 3 and q:get_quest_items_count(player, GRANDIS_SKIN) >= 3 and q:get_quest_items_count(player, TIMAK_ORC_BELT) >= 3 then
          st:set_cond(5, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw2 == GRANDIS then
      if q:get_quest_items_count(player, GRANDIS_SKIN) < 3 then
        q:give_items(player, GRANDIS_SKIN, 1)
        if q:get_quest_items_count(player, GRANDIS_SKIN) >= 3 and q:get_quest_items_count(player, EXCURO_SKIN) >= 3 and q:get_quest_items_count(player, KRATOR_SHARD) >= 3 and q:get_quest_items_count(player, LAKIN_MACE) >= 3 and q:get_quest_items_count(player, TIMAK_ORC_BELT) >= 3 then
          st:set_cond(5, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw2 == TIMAK_ORC_OVERLORD then
      if q:get_quest_items_count(player, TIMAK_ORC_BELT) < 3 then
        q:give_items(player, TIMAK_ORC_BELT, 1)
        if q:get_quest_items_count(player, TIMAK_ORC_BELT) >= 3 and q:get_quest_items_count(player, EXCURO_SKIN) >= 3 and q:get_quest_items_count(player, KRATOR_SHARD) >= 3 and q:get_quest_items_count(player, LAKIN_MACE) >= 3 and q:get_quest_items_count(player, GRANDIS_SKIN) >= 3 then
          st:set_cond(5, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  end
end)

-- Translated by questport from quests/Q00223_TestOfTheChampion (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(223, "Q00223_TestOfTheChampion", "Test of the Champion")
-- L2Go: the translator asked for q:requires("pets") because the attack
-- handler sends the spawned monsters after the attacker's summon. Summons do
-- not exist yet, so the quest is offered and an attacking summon is ignored
-- (see on_attack); the Warlord class change needs this mark.

-- NPCs
local ASCALON = 30624
local GROOT = 30093
local MOUEN = 30196
local MASON = 30625
-- Monsters
local HARPY = 20145
local HARPY_MATRIARCH = 27088
local MEDUSA = 20158
local WINDSUS = 20553
local ROAD_COLLECTOR = 27089
local ROAD_SCAVENGER = 20551
local LETO_LIZARDMAN = 20577
local LETO_LIZARDMAN_ARCHER = 20578
local LETO_LIZARDMAN_SOLDIER = 20579
local LETO_LIZARDMAN_WARRIOR = 20580
local LETO_LIZARDMAN_SHAMAN = 20581
local LETO_LIZARDMAN_OVERLORD = 20582
local BLOODY_AXE_ELITE = 20780
-- Items
local ASCALON_LETTER_1 = 3277
local MASON_LETTER = 3278
local IRON_ROSE_RING = 3279
local ASCALON_LETTER_2 = 3280
local WHITE_ROSE_INSIGNIA = 3281
local GROOT_LETTER = 3282
local ASCALON_LETTER_3 = 3283
local MOUEN_ORDER_1 = 3284
local MOUEN_ORDER_2 = 3285
local MOUEN_LETTER = 3286
local HARPY_EGG = 3287
local MEDUSA_VENOM = 3288
local WINDSUS_BILE = 3289
local BLOODY_AXE_HEAD = 3290
local ROAD_RATMAN_HEAD = 3291
local LETO_LIZARDMAN_FANG = 3292
-- Rewards
local MARK_OF_CHAMPION = 3276
local DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(MASON_LETTER, MEDUSA_VENOM, WINDSUS_BILE, WHITE_ROSE_INSIGNIA, HARPY_EGG, GROOT_LETTER, MOUEN_LETTER, ASCALON_LETTER_1, IRON_ROSE_RING, BLOODY_AXE_HEAD, ASCALON_LETTER_2, ASCALON_LETTER_3, MOUEN_ORDER_1, ROAD_RATMAN_HEAD, MOUEN_ORDER_2, LETO_LIZARDMAN_FANG)
q:add_start_npc(ASCALON)
q:add_talk_id(ASCALON, GROOT, MOUEN, MASON)
q:add_attack_id(HARPY, ROAD_SCAVENGER)
q:add_kill_id(HARPY, MEDUSA, HARPY_MATRIARCH, ROAD_COLLECTOR, ROAD_SCAVENGER, WINDSUS, LETO_LIZARDMAN, LETO_LIZARDMAN_ARCHER, LETO_LIZARDMAN_SOLDIER, LETO_LIZARDMAN_WARRIOR, LETO_LIZARDMAN_SHAMAN, LETO_LIZARDMAN_OVERLORD, BLOODY_AXE_ELITE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30624-06.htm" then
    st:start_quest()
    q:give_items(player, ASCALON_LETTER_1, 1)
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30624-06a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30624-10.htm" then
    st:set_cond(5, true)
    q:take_items(player, MASON_LETTER, 1)
    q:give_items(player, ASCALON_LETTER_2, 1)
  elseif event == "30624-14.htm" then
    st:set_cond(9, true)
    q:take_items(player, GROOT_LETTER, 1)
    q:give_items(player, ASCALON_LETTER_3, 1)
  elseif event == "30625-03.htm" then
    st:set_cond(2, true)
    q:take_items(player, ASCALON_LETTER_1, 1)
    q:give_items(player, IRON_ROSE_RING, 1)
  elseif event == "30093-02.htm" then
    st:set_cond(6, true)
    q:take_items(player, ASCALON_LETTER_2, 1)
    q:give_items(player, WHITE_ROSE_INSIGNIA, 1)
  elseif event == "30196-03.htm" then
    st:set_cond(10, true)
    q:take_items(player, ASCALON_LETTER_3, 1)
    q:give_items(player, MOUEN_ORDER_1, 1)
  elseif event == "30196-06.htm" then
    st:set_cond(12, true)
    q:take_items(player, MOUEN_ORDER_1, 1)
    q:take_items(player, ROAD_RATMAN_HEAD, -1)
    q:give_items(player, MOUEN_ORDER_2, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    local playerClass = player:class_id()
    if playerClass ~= ClassId.WARRIOR and playerClass ~= ClassId.ORC_RAIDER then
      htmltext = "30624-01.htm"
    elseif player:level() < 39 then
      htmltext = "30624-02.htm"
    else
      htmltext = playerClass == ClassId.WARRIOR and "30624-03.htm" or "30624-04.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ASCALON then
      if cond == 1 then
        htmltext = "30624-07.htm"
      elseif cond < 4 then
        htmltext = "30624-08.htm"
      elseif cond == 4 then
        htmltext = "30624-09.htm"
      elseif cond == 5 then
        htmltext = "30624-11.htm"
      elseif cond > 5 and cond < 8 then
        htmltext = "30624-12.htm"
      elseif cond == 8 then
        htmltext = "30624-13.htm"
      elseif cond == 9 then
        htmltext = "30624-15.htm"
      elseif cond > 9 and cond < 14 then
        htmltext = "30624-16.htm"
      elseif cond == 14 then
        htmltext = "30624-17.htm"
        q:take_items(player, MOUEN_LETTER, 1)
        q:give_items(player, MARK_OF_CHAMPION, 1)
        q:add_exp_and_sp(player, 117454, 25000)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == MASON then
      if cond == 1 then
        htmltext = "30625-01.htm"
      elseif cond == 2 then
        htmltext = "30625-04.htm"
      elseif cond == 3 then
        htmltext = "30625-05.htm"
        st:set_cond(4, true)
        q:take_items(player, BLOODY_AXE_HEAD, -1)
        q:take_items(player, IRON_ROSE_RING, 1)
        q:give_items(player, MASON_LETTER, 1)
      elseif cond == 4 then
        htmltext = "30625-06.htm"
      elseif cond > 4 then
        htmltext = "30625-07.htm"
      end
    elseif sw2 == GROOT then
      if cond == 5 then
        htmltext = "30093-01.htm"
      elseif cond == 6 then
        htmltext = "30093-03.htm"
      elseif cond == 7 then
        htmltext = "30093-04.htm"
        st:set_cond(8, true)
        q:take_items(player, WHITE_ROSE_INSIGNIA, 1)
        q:take_items(player, HARPY_EGG, -1)
        q:take_items(player, MEDUSA_VENOM, -1)
        q:take_items(player, WINDSUS_BILE, -1)
        q:give_items(player, GROOT_LETTER, 1)
      elseif cond == 8 then
        htmltext = "30093-05.htm"
      elseif cond > 8 then
        htmltext = "30093-06.htm"
      end
    elseif sw2 == MOUEN then
      if cond == 9 then
        htmltext = "30196-01.htm"
      elseif cond == 10 then
        htmltext = "30196-04.htm"
      elseif cond == 11 then
        htmltext = "30196-05.htm"
      elseif cond == 12 then
        htmltext = "30196-07.htm"
      elseif cond == 13 then
        htmltext = "30196-08.htm"
        st:set_cond(14, true)
        q:take_items(player, LETO_LIZARDMAN_FANG, -1)
        q:take_items(player, MOUEN_ORDER_2, 1)
        q:give_items(player, MOUEN_LETTER, 1)
      elseif cond > 13 then
        htmltext = "30196-09.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_attack(function(npc, attacker, damage, isPet)
  local st = q:get_quest_state(attacker, false)
  if st == nil then
    return
  end
  -- L2Go: summons do not exist yet. Mobius sends the spawned monsters after
  -- the summon (attacker.getSummon()) when a summon hits; that waits for pets.
  if isPet then
    return
  end
  -- L2Go: Mobius makes each spawned monster run (setRunning), hate
  -- (addDamageHate 999) and attack the attacker. The script API has no NPC
  -- AI calls yet, so the monsters are only spawned next to the attacker.
  local sw = npc:id()
  if sw == HARPY then
    if st:is_cond(6) and rnd.bool() and not npc:is_script_value(1) then
      -- Spawn one or two matriarchs.
      do
        local i = 1
        while i < (rnd(10) < 7 and 2 or 3) do
          q:add_spawn(HARPY_MATRIARCH, npc, true, 0)
          i = i + 1
        end
      end
      npc:set_script_value(1)
    end
  elseif sw == ROAD_SCAVENGER then
    if st:is_cond(10) and rnd.bool() and not npc:is_script_value(1) then
      -- Spawn one or two collectors.
      do
        local i = 1
        while i < (rnd(10) < 7 and 2 or 3) do
          q:add_spawn(ROAD_COLLECTOR, npc, true, 0)
          i = i + 1
        end
      end
      npc:set_script_value(1)
    end
  end
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local npcId = npc:id()
  if npcId == BLOODY_AXE_ELITE then
    if st:is_cond(2) then
      q:give_items(player, BLOODY_AXE_HEAD, 1)
      if q:get_quest_items_count(player, BLOODY_AXE_HEAD) >= 100 then
        st:set_cond(3, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == HARPY or npcId == HARPY_MATRIARCH then
    if st:is_cond(6) and rnd.bool() and q:get_quest_items_count(player, HARPY_EGG) < 30 then
      q:give_items(player, HARPY_EGG, 1)
      if q:get_quest_items_count(player, HARPY_EGG) >= 30 and q:get_quest_items_count(player, MEDUSA_VENOM) == 30 and q:get_quest_items_count(player, WINDSUS_BILE) == 30 then
        st:set_cond(7, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == MEDUSA then
    if st:is_cond(6) and rnd.bool() and q:get_quest_items_count(player, MEDUSA_VENOM) < 30 then
      q:give_items(player, MEDUSA_VENOM, 1)
      if q:get_quest_items_count(player, MEDUSA_VENOM) >= 30 and q:get_quest_items_count(player, HARPY_EGG) == 30 and q:get_quest_items_count(player, WINDSUS_BILE) == 30 then
        st:set_cond(7, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == WINDSUS then
    if st:is_cond(6) and rnd.bool() and q:get_quest_items_count(player, WINDSUS_BILE) < 30 then
      q:give_items(player, WINDSUS_BILE, 1)
      if q:get_quest_items_count(player, WINDSUS_BILE) >= 30 and q:get_quest_items_count(player, HARPY_EGG) == 30 and q:get_quest_items_count(player, MEDUSA_VENOM) == 30 then
        st:set_cond(7, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == ROAD_COLLECTOR or npcId == ROAD_SCAVENGER then
    if st:is_cond(10) then
      q:give_items(player, ROAD_RATMAN_HEAD, 1)
      if q:get_quest_items_count(player, ROAD_RATMAN_HEAD) >= 100 then
        st:set_cond(11, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == LETO_LIZARDMAN or npcId == LETO_LIZARDMAN_ARCHER or npcId == LETO_LIZARDMAN_SOLDIER or npcId == LETO_LIZARDMAN_WARRIOR or npcId == LETO_LIZARDMAN_SHAMAN or npcId == LETO_LIZARDMAN_OVERLORD then
    if st:is_cond(12) and q:get_quest_items_count(player, LETO_LIZARDMAN_FANG) < 100 and rnd(1000000) < 500000 + 100000 * (npcId - 20577) then
      q:give_items(player, LETO_LIZARDMAN_FANG, 1)
      if q:get_quest_items_count(player, LETO_LIZARDMAN_FANG) >= 100 then
        st:set_cond(13, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)

-- Translated by questport from quests/Q00224_TestOfSagittarius (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(224, "Q00224_TestOfSagittarius", "Test of Sagittarius")

-- NPCs
local BERNARD = 30702
local HAMIL = 30626
local SIR_ARON_TANFORD = 30653
local VOKIAN = 30514
local GAUEN = 30717
-- Monsters
local ANT = 20079
local ANT_CAPTAIN = 20080
local ANT_OVERSEER = 20081
local ANT_RECRUIT = 20082
local ANT_PATROL = 20084
local ANT_GUARD = 20086
local NOBLE_ANT = 20089
local NOBLE_ANT_LEADER = 20090
local BREKA_ORC_SHAMAN = 20269
local BREKA_ORC_OVERLORD = 20270
local MARSH_STAKATO_WORKER = 20230
local MARSH_STAKATO_SOLDIER = 20232
local MARSH_STAKATO_DRONE = 20234
local MARSH_SPIDER = 20233
local ROAD_SCAVENGER = 20551
local MANASHEN_GARGOYLE = 20563
local LETO_LIZARDMAN = 20577
local LETO_LIZARDMAN_ARCHER = 20578
local LETO_LIZARDMAN_SOLDIER = 20579
local LETO_LIZARDMAN_WARRIOR = 20580
local LETO_LIZARDMAN_SHAMAN = 20581
local LETO_LIZARDMAN_OVERLORD = 20582
local SERPENT_DEMON_KADESH = 27090
-- Items
local BERNARD_INTRODUCTION = 3294
local HAMIL_LETTER_1 = 3295
local HAMIL_LETTER_2 = 3296
local HAMIL_LETTER_3 = 3297
local HUNTER_RUNE_1 = 3298
local HUNTER_RUNE_2 = 3299
local TALISMAN_OF_KADESH = 3300
local TALISMAN_OF_SNAKE = 3301
local MITHRIL_CLIP = 3302
local STAKATO_CHITIN = 3303
local REINFORCED_BOWSTRING = 3304
local MANASHEN_HORN = 3305
local BLOOD_OF_LIZARDMAN = 3306
local CRESCENT_MOON_BOW = 3028
local WOODEN_ARROW = 17
-- Rewards
local MARK_OF_SAGITTARIUS = 3293
local DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(BERNARD_INTRODUCTION, HAMIL_LETTER_1, HAMIL_LETTER_2, HAMIL_LETTER_3, HUNTER_RUNE_1, HUNTER_RUNE_2, TALISMAN_OF_KADESH, TALISMAN_OF_SNAKE, MITHRIL_CLIP, STAKATO_CHITIN, REINFORCED_BOWSTRING, MANASHEN_HORN, BLOOD_OF_LIZARDMAN, CRESCENT_MOON_BOW)
q:add_start_npc(BERNARD)
q:add_talk_id(BERNARD, HAMIL, SIR_ARON_TANFORD, VOKIAN, GAUEN)
q:add_kill_id(ANT, ANT_CAPTAIN, ANT_OVERSEER, ANT_RECRUIT, ANT_PATROL, ANT_GUARD, NOBLE_ANT, NOBLE_ANT_LEADER, BREKA_ORC_SHAMAN, BREKA_ORC_OVERLORD, MARSH_STAKATO_WORKER, MARSH_STAKATO_SOLDIER, MARSH_STAKATO_DRONE, MARSH_SPIDER, ROAD_SCAVENGER, MANASHEN_GARGOYLE, LETO_LIZARDMAN, LETO_LIZARDMAN_ARCHER, LETO_LIZARDMAN_SOLDIER, LETO_LIZARDMAN_WARRIOR, LETO_LIZARDMAN_SHAMAN, LETO_LIZARDMAN_OVERLORD, SERPENT_DEMON_KADESH)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30702-04.htm" then
    st:start_quest()
    q:give_items(player, BERNARD_INTRODUCTION, 1)
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30702-04a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30626-03.htm" then
    st:set_cond(2, true)
    q:take_items(player, BERNARD_INTRODUCTION, 1)
    q:give_items(player, HAMIL_LETTER_1, 1)
  elseif event == "30626-07.htm" then
    st:set_cond(5, true)
    q:take_items(player, HUNTER_RUNE_1, 10)
    q:give_items(player, HAMIL_LETTER_2, 1)
  elseif event == "30653-02.htm" then
    st:set_cond(3, true)
    q:take_items(player, HAMIL_LETTER_1, 1)
  elseif event == "30514-02.htm" then
    st:set_cond(6, true)
    q:take_items(player, HAMIL_LETTER_2, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.ROGUE and player:class_id() ~= ClassId.ELVEN_SCOUT and player:class_id() ~= ClassId.ASSASSIN then
      htmltext = "30702-02.htm"
    elseif player:level() < 39 then
      htmltext = "30702-01.htm"
    else
      htmltext = "30702-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == BERNARD then
      htmltext = "30702-05.htm"
    elseif sw2 == HAMIL then
      if cond == 1 then
        htmltext = "30626-01.htm"
      elseif cond == 2 or cond == 3 then
        htmltext = "30626-04.htm"
      elseif cond == 4 then
        htmltext = "30626-05.htm"
      elseif cond > 4 and cond < 8 then
        htmltext = "30626-08.htm"
      elseif cond == 8 then
        htmltext = "30626-09.htm"
        st:set_cond(9, true)
        q:take_items(player, HUNTER_RUNE_2, 10)
        q:give_items(player, HAMIL_LETTER_3, 1)
      elseif cond > 8 and cond < 12 then
        htmltext = "30626-10.htm"
      elseif cond == 12 then
        htmltext = "30626-11.htm"
        st:set_cond(13, true)
      elseif cond == 13 then
        htmltext = "30626-12.htm"
      elseif cond == 14 then
        htmltext = "30626-13.htm"
        q:take_items(player, BLOOD_OF_LIZARDMAN, -1)
        q:take_items(player, CRESCENT_MOON_BOW, 1)
        q:take_items(player, TALISMAN_OF_KADESH, 1)
        q:give_items(player, MARK_OF_SAGITTARIUS, 1)
        q:add_exp_and_sp(player, 54726, 20250)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == SIR_ARON_TANFORD then
      if cond == 2 then
        htmltext = "30653-01.htm"
      elseif cond > 2 then
        htmltext = "30653-03.htm"
      end
    elseif sw2 == VOKIAN then
      if cond == 5 then
        htmltext = "30514-01.htm"
      elseif cond == 6 then
        htmltext = "30514-03.htm"
      elseif cond == 7 then
        htmltext = "30514-04.htm"
        st:set_cond(8, true)
        q:take_items(player, TALISMAN_OF_SNAKE, 1)
      elseif cond > 7 then
        htmltext = "30514-05.htm"
      end
    elseif sw2 == GAUEN then
      if cond == 9 then
        htmltext = "30717-01.htm"
        st:set_cond(10, true)
        q:take_items(player, HAMIL_LETTER_3, 1)
      elseif cond == 10 then
        htmltext = "30717-03.htm"
      elseif cond == 11 then
        htmltext = "30717-02.htm"
        st:set_cond(12, true)
        q:take_items(player, MANASHEN_HORN, 1)
        q:take_items(player, MITHRIL_CLIP, 1)
        q:take_items(player, REINFORCED_BOWSTRING, 1)
        q:take_items(player, STAKATO_CHITIN, 1)
        q:give_items(player, CRESCENT_MOON_BOW, 1)
        q:give_items(player, WOODEN_ARROW, 10)
      elseif cond > 11 then
        htmltext = "30717-04.htm"
      end
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
  local sw = npc:id()
  if sw == ANT or sw == ANT_CAPTAIN or sw == ANT_OVERSEER or sw == ANT_RECRUIT or sw == ANT_PATROL or sw == ANT_GUARD or sw == NOBLE_ANT or sw == NOBLE_ANT_LEADER then
    if st:is_cond(3) and rnd.bool() then
      q:give_items(player, HUNTER_RUNE_1, 1)
      if q:get_quest_items_count(player, HUNTER_RUNE_1) >= 10 then
        st:set_cond(4)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == BREKA_ORC_SHAMAN or sw == BREKA_ORC_OVERLORD then
    if st:is_cond(6) and rnd.bool() then
      q:give_items(player, HUNTER_RUNE_2, 1)
      if q:get_quest_items_count(player, HUNTER_RUNE_2) >= 10 then
        st:set_cond(7, true)
        q:give_items(player, TALISMAN_OF_SNAKE, 1)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == MARSH_STAKATO_WORKER or sw == MARSH_STAKATO_SOLDIER or sw == MARSH_STAKATO_DRONE then
    if st:is_cond(10) and rnd(10) < 1 then
      q:give_items(player, STAKATO_CHITIN, 1)
      if q:has_quest_items(player, MANASHEN_HORN, MITHRIL_CLIP, REINFORCED_BOWSTRING) then
        st:set_cond(11, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == MARSH_SPIDER then
    if st:is_cond(10) and rnd(10) < 1 then
      q:give_items(player, REINFORCED_BOWSTRING, 1)
      if q:has_quest_items(player, MANASHEN_HORN, MITHRIL_CLIP, STAKATO_CHITIN) then
        st:set_cond(11, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == ROAD_SCAVENGER then
    if st:is_cond(10) and rnd(10) < 1 then
      q:give_items(player, MITHRIL_CLIP, 1)
      if q:has_quest_items(player, MANASHEN_HORN, REINFORCED_BOWSTRING, STAKATO_CHITIN) then
        st:set_cond(11, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == MANASHEN_GARGOYLE then
    if st:is_cond(10) and rnd(10) < 1 then
      q:give_items(player, MANASHEN_HORN, 1)
      if q:has_quest_items(player, REINFORCED_BOWSTRING, MITHRIL_CLIP, STAKATO_CHITIN) then
        st:set_cond(11, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == LETO_LIZARDMAN or sw == LETO_LIZARDMAN_ARCHER or sw == LETO_LIZARDMAN_SOLDIER or sw == LETO_LIZARDMAN_WARRIOR or sw == LETO_LIZARDMAN_SHAMAN or sw == LETO_LIZARDMAN_OVERLORD then
    if st:is_cond(13) then
      if (q:get_quest_items_count(player, BLOOD_OF_LIZARDMAN) - 120) * 5 > rnd(100) then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_BEFORE_BATTLE)
        q:take_items(player, BLOOD_OF_LIZARDMAN, -1)
        q:add_spawn(SERPENT_DEMON_KADESH, player, false, 300000)
      else
        q:give_items(player, BLOOD_OF_LIZARDMAN, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == SERPENT_DEMON_KADESH then
    if st:is_cond(13) then
      if q:get_item_equipped(player, Paperdoll.RHAND) == CRESCENT_MOON_BOW then
        st:set_cond(14, true)
        q:give_items(player, TALISMAN_OF_KADESH, 1)
      else
        q:add_spawn(SERPENT_DEMON_KADESH, player, false, 300000)
      end
    end
  end
end)

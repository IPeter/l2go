-- Translated by questport from quests/Q00232_TestOfTheLord (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(232, "Q00232_TestOfTheLord", "Test of the Lord")

-- NPCs
local SOMAK = 30510
local MANAKIA = 30515
local JAKAL = 30558
local SUMARI = 30564
local KAKAI = 30565
local VARKEES = 30566
local TANTUS = 30567
local HATOS = 30568
local TAKUNA = 30641
local CHIANTA = 30642
local FIRST_ORC = 30643
local ANCESTOR_MARTANKUS = 30649
-- Items
local ORDEAL_NECKLACE = 3391
local VARKEES_CHARM = 3392
local TANTUS_CHARM = 3393
local HATOS_CHARM = 3394
local TAKUNA_CHARM = 3395
local CHIANTA_CHARM = 3396
local MANAKIAS_ORDERS = 3397
local BREKA_ORC_FANG = 3398
local MANAKIAS_AMULET = 3399
local HUGE_ORC_FANG = 3400
local SUMARIS_LETTER = 3401
local URUTU_BLADE = 3402
local TIMAK_ORC_SKULL = 3403
local SWORD_INTO_SKULL = 3404
local NERUGA_AXE_BLADE = 3405
local AXE_OF_CEREMONY = 3406
local MARSH_SPIDER_FEELER = 3407
local MARSH_SPIDER_FEET = 3408
local HANDIWORK_SPIDER_BROOCH = 3409
local MONSTEREYE_CORNEA = 3410
local MONSTEREYE_WOODCARVING = 3411
local BEAR_FANG_NECKLACE = 3412
local MARTANKUS_CHARM = 3413
local RAGNA_ORC_HEAD = 3414
local RAGNA_CHIEF_NOTICE = 3415
local BONE_ARROW = 1341
local IMMORTAL_FLAME = 3416
-- Rewards
local MARK_LORD = 3390
local DIMENSIONAL_DIAMOND = 7562
-- Misc
local _firstOrc -- Used to avoid to spawn multiple instances.

q:register_quest_items(VARKEES_CHARM, TANTUS_CHARM, HATOS_CHARM, TAKUNA_CHARM, CHIANTA_CHARM, MANAKIAS_ORDERS, BREKA_ORC_FANG, MANAKIAS_AMULET, HUGE_ORC_FANG, SUMARIS_LETTER, URUTU_BLADE, TIMAK_ORC_SKULL, SWORD_INTO_SKULL, NERUGA_AXE_BLADE, AXE_OF_CEREMONY, MARSH_SPIDER_FEELER, MARSH_SPIDER_FEET, HANDIWORK_SPIDER_BROOCH, MONSTEREYE_CORNEA, MONSTEREYE_WOODCARVING, BEAR_FANG_NECKLACE, MARTANKUS_CHARM, RAGNA_ORC_HEAD, RAGNA_ORC_HEAD, IMMORTAL_FLAME)
q:add_start_npc(KAKAI)
q:add_talk_id(KAKAI, CHIANTA, HATOS, SOMAK, SUMARI, TAKUNA, TANTUS, JAKAL, VARKEES, MANAKIA, ANCESTOR_MARTANKUS, FIRST_ORC)
q:add_kill_id(20233, 20269, 20270, 20564, 20583, 20584, 20585, 20586, 20587, 20588, 20778, 20779)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30565-05.htm" then
    st:start_quest()
    q:give_items(player, ORDEAL_NECKLACE, 1)
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30565-05b.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30565-08.htm" then
    st:set_cond(3, true)
    q:take_items(player, SWORD_INTO_SKULL, 1)
    q:take_items(player, AXE_OF_CEREMONY, 1)
    q:take_items(player, MONSTEREYE_WOODCARVING, 1)
    q:take_items(player, HANDIWORK_SPIDER_BROOCH, 1)
    q:take_items(player, ORDEAL_NECKLACE, 1)
    q:take_items(player, HUGE_ORC_FANG, 1)
    q:give_items(player, BEAR_FANG_NECKLACE, 1)
  elseif event == "30566-02.htm" then
    q:give_items(player, VARKEES_CHARM, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  elseif event == "30567-02.htm" then
    q:give_items(player, TANTUS_CHARM, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  elseif event == "30558-02.htm" then
    q:take_items(player, 57, 1000)
    q:give_items(player, NERUGA_AXE_BLADE, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  elseif event == "30568-02.htm" then
    q:give_items(player, HATOS_CHARM, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  elseif event == "30641-02.htm" then
    q:give_items(player, TAKUNA_CHARM, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  elseif event == "30642-02.htm" then
    q:give_items(player, CHIANTA_CHARM, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  elseif event == "30643-02.htm" then
    st:set_cond(7, true)
    q:start_quest_timer("f_orc_despawn", 10000, nil, player, false)
  elseif event == "30649-04.htm" then
    st:set_cond(4, true)
    q:take_items(player, BEAR_FANG_NECKLACE, 1)
    q:give_items(player, MARTANKUS_CHARM, 1)
  elseif event == "30649-07.htm" then
    if _firstOrc == nil then
      _firstOrc = J.oid(q:add_spawn(FIRST_ORC, {x = 21036, y = -107690, z = -3038, heading = 200000}, false, 0))
    end
  elseif event == "f_orc_despawn" then
    if _firstOrc ~= nil then
      J.obj(_firstOrc):delete()
      _firstOrc = nil
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
      htmltext = "30565-01.htm"
    elseif player:class_id() ~= ClassId.ORC_SHAMAN then
      htmltext = "30565-02.htm"
    elseif player:level() < 39 then
      htmltext = "30565-03.htm"
    else
      htmltext = "30565-04.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == VARKEES then
      if q:has_quest_items(player, HUGE_ORC_FANG) then
        htmltext = "30566-05.htm"
      elseif q:has_quest_items(player, VARKEES_CHARM) then
        if q:has_quest_items(player, MANAKIAS_AMULET) then
          htmltext = "30566-04.htm"
          q:take_items(player, VARKEES_CHARM, -1)
          q:take_items(player, MANAKIAS_AMULET, -1)
          q:give_items(player, HUGE_ORC_FANG, 1)
          if q:has_quest_items(player, SWORD_INTO_SKULL, AXE_OF_CEREMONY, MONSTEREYE_WOODCARVING, HANDIWORK_SPIDER_BROOCH, ORDEAL_NECKLACE) then
            st:set_cond(2, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        else
          htmltext = "30566-03.htm"
        end
      else
        htmltext = "30566-01.htm"
      end
    elseif sw2 == MANAKIA then
      if q:has_quest_items(player, HUGE_ORC_FANG) then
        htmltext = "30515-05.htm"
      elseif q:has_quest_items(player, MANAKIAS_AMULET) then
        htmltext = "30515-04.htm"
      elseif q:has_quest_items(player, MANAKIAS_ORDERS) then
        if q:get_quest_items_count(player, BREKA_ORC_FANG) >= 20 then
          htmltext = "30515-03.htm"
          q:take_items(player, MANAKIAS_ORDERS, -1)
          q:take_items(player, BREKA_ORC_FANG, -1)
          q:give_items(player, MANAKIAS_AMULET, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          htmltext = "30515-02.htm"
        end
      else
        htmltext = "30515-01.htm"
        q:give_items(player, MANAKIAS_ORDERS, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw2 == TANTUS then
      if q:has_quest_items(player, AXE_OF_CEREMONY) then
        htmltext = "30567-05.htm"
      elseif q:has_quest_items(player, TANTUS_CHARM) then
        if q:get_quest_items_count(player, BONE_ARROW) >= 1000 then
          htmltext = "30567-04.htm"
          q:take_items(player, BONE_ARROW, 1000)
          q:take_items(player, NERUGA_AXE_BLADE, 1)
          q:take_items(player, TANTUS_CHARM, 1)
          q:give_items(player, AXE_OF_CEREMONY, 1)
          if q:has_quest_items(player, SWORD_INTO_SKULL, MONSTEREYE_WOODCARVING, HANDIWORK_SPIDER_BROOCH, ORDEAL_NECKLACE, HUGE_ORC_FANG) then
            st:set_cond(2, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        else
          htmltext = "30567-03.htm"
        end
      else
        htmltext = "30567-01.htm"
      end
    elseif sw2 == JAKAL then
      if q:has_quest_items(player, AXE_OF_CEREMONY) then
        htmltext = "30558-05.htm"
      elseif q:has_quest_items(player, NERUGA_AXE_BLADE) then
        htmltext = "30558-04.htm"
      elseif q:has_quest_items(player, TANTUS_CHARM) then
        if q:get_quest_items_count(player, 57) >= 1000 then
          htmltext = "30558-01.htm"
        else
          htmltext = "30558-03.htm"
        end
      end
    elseif sw2 == HATOS then
      if q:has_quest_items(player, SWORD_INTO_SKULL) then
        htmltext = "30568-05.htm"
      elseif q:has_quest_items(player, HATOS_CHARM) then
        if q:has_quest_items(player, URUTU_BLADE) and q:get_quest_items_count(player, TIMAK_ORC_SKULL) >= 10 then
          htmltext = "30568-04.htm"
          q:take_items(player, HATOS_CHARM, 1)
          q:take_items(player, URUTU_BLADE, 1)
          q:take_items(player, TIMAK_ORC_SKULL, -1)
          q:give_items(player, SWORD_INTO_SKULL, 1)
          if q:has_quest_items(player, AXE_OF_CEREMONY, MONSTEREYE_WOODCARVING, HANDIWORK_SPIDER_BROOCH, ORDEAL_NECKLACE, HUGE_ORC_FANG) then
            st:set_cond(2, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        else
          htmltext = "30568-03.htm"
        end
      else
        htmltext = "30568-01.htm"
      end
    elseif sw2 == SUMARI then
      if q:has_quest_items(player, URUTU_BLADE) then
        htmltext = "30564-03.htm"
      elseif q:has_quest_items(player, SUMARIS_LETTER) then
        htmltext = "30564-02.htm"
      elseif q:has_quest_items(player, HATOS_CHARM) then
        htmltext = "30564-01.htm"
        q:give_items(player, SUMARIS_LETTER, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw2 == SOMAK then
      if q:has_quest_items(player, SWORD_INTO_SKULL) then
        htmltext = "30510-03.htm"
      elseif q:has_quest_items(player, URUTU_BLADE) then
        htmltext = "30510-02.htm"
      elseif q:has_quest_items(player, SUMARIS_LETTER) then
        htmltext = "30510-01.htm"
        q:take_items(player, SUMARIS_LETTER, 1)
        q:give_items(player, URUTU_BLADE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw2 == TAKUNA then
      if q:has_quest_items(player, HANDIWORK_SPIDER_BROOCH) then
        htmltext = "30641-05.htm"
      elseif q:has_quest_items(player, TAKUNA_CHARM) then
        if q:get_quest_items_count(player, MARSH_SPIDER_FEELER) >= 10 and q:get_quest_items_count(player, MARSH_SPIDER_FEET) >= 10 then
          htmltext = "30641-04.htm"
          q:take_items(player, MARSH_SPIDER_FEELER, -1)
          q:take_items(player, MARSH_SPIDER_FEET, -1)
          q:take_items(player, TAKUNA_CHARM, 1)
          q:give_items(player, HANDIWORK_SPIDER_BROOCH, 1)
          if q:has_quest_items(player, SWORD_INTO_SKULL, AXE_OF_CEREMONY, MONSTEREYE_WOODCARVING, ORDEAL_NECKLACE, HUGE_ORC_FANG) then
            st:set_cond(2, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        else
          htmltext = "30641-03.htm"
        end
      else
        htmltext = "30641-01.htm"
      end
    elseif sw2 == CHIANTA then
      if q:has_quest_items(player, MONSTEREYE_WOODCARVING) then
        htmltext = "30642-05.htm"
      elseif q:has_quest_items(player, CHIANTA_CHARM) then
        if q:get_quest_items_count(player, MONSTEREYE_CORNEA) >= 20 then
          htmltext = "30642-04.htm"
          q:take_items(player, MONSTEREYE_CORNEA, -1)
          q:take_items(player, CHIANTA_CHARM, 1)
          q:give_items(player, MONSTEREYE_WOODCARVING, 1)
          if q:has_quest_items(player, SWORD_INTO_SKULL, AXE_OF_CEREMONY, HANDIWORK_SPIDER_BROOCH, ORDEAL_NECKLACE, HUGE_ORC_FANG) then
            st:set_cond(2, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        else
          htmltext = "30642-03.htm"
        end
      else
        htmltext = "30642-01.htm"
      end
    elseif sw2 == KAKAI then
      if cond == 1 then
        htmltext = "30565-06.htm"
      elseif cond == 2 then
        htmltext = "30565-07.htm"
      elseif cond == 3 then
        htmltext = "30565-09.htm"
      elseif cond > 3 and cond < 7 then
        htmltext = "30565-10.htm"
      elseif cond == 7 then
        htmltext = "30565-11.htm"
        q:take_items(player, IMMORTAL_FLAME, 1)
        q:give_items(player, MARK_LORD, 1)
        q:add_exp_and_sp(player, 92955, 16250)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == ANCESTOR_MARTANKUS then
      if cond == 3 then
        htmltext = "30649-01.htm"
      elseif cond == 4 then
        htmltext = "30649-05.htm"
      elseif cond == 5 then
        htmltext = "30649-06.htm"
        st:set_cond(6, true)
        q:take_items(player, MARTANKUS_CHARM, 1)
        q:take_items(player, RAGNA_ORC_HEAD, 1)
        q:take_items(player, RAGNA_CHIEF_NOTICE, 1)
        q:give_items(player, IMMORTAL_FLAME, 1)
      elseif cond == 6 then
        htmltext = "30649-07.htm"
      elseif cond == 7 then
        htmltext = "30649-08.htm"
      end
    elseif sw2 == FIRST_ORC then
      if cond == 6 then
        htmltext = "30643-01.htm"
      elseif cond == 7 then
        htmltext = "30643-03.htm"
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
  if sw == 20564 then
    if q:has_quest_items(player, CHIANTA_CHARM) and q:get_quest_items_count(player, MONSTEREYE_CORNEA) < 20 then
      q:give_items(player, MONSTEREYE_CORNEA, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20583 or sw == 20584 or sw == 20585 then
    if q:has_quest_items(player, HATOS_CHARM) and rnd(100) < 71 and q:get_quest_items_count(player, TIMAK_ORC_SKULL) < 10 then
      q:give_items(player, TIMAK_ORC_SKULL, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20586 then
    if q:has_quest_items(player, HATOS_CHARM) and rnd(100) < 81 and q:get_quest_items_count(player, TIMAK_ORC_SKULL) < 10 then
      q:give_items(player, TIMAK_ORC_SKULL, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20587 or sw == 20588 then
    if q:has_quest_items(player, HATOS_CHARM) and q:get_quest_items_count(player, TIMAK_ORC_SKULL) < 10 then
      q:give_items(player, TIMAK_ORC_SKULL, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20233 then
    if q:has_quest_items(player, TAKUNA_CHARM) then
      if q:get_quest_items_count(player, MARSH_SPIDER_FEELER) >= 10 then
        q:give_items(player, MARSH_SPIDER_FEET, 1)
      else
        q:give_items(player, MARSH_SPIDER_FEELER, 1)
      end
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20269 then
    if q:has_quest_items(player, MANAKIAS_ORDERS) and rnd(100) < 41 and q:get_quest_items_count(player, BREKA_ORC_FANG) < 20 then
      q:give_items(player, BREKA_ORC_FANG, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20270 then
    if q:has_quest_items(player, MANAKIAS_ORDERS) and rnd(100) < 51 and q:get_quest_items_count(player, BREKA_ORC_FANG) < 20 then
      q:give_items(player, BREKA_ORC_FANG, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20778 or sw == 20779 then
    if q:has_quest_items(player, MARTANKUS_CHARM) then
      if not q:has_quest_items(player, RAGNA_CHIEF_NOTICE) then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        q:give_items(player, RAGNA_CHIEF_NOTICE, 1)
      elseif not q:has_quest_items(player, RAGNA_ORC_HEAD) then
        st:set_cond(5, true)
        q:give_items(player, RAGNA_ORC_HEAD, 1)
      end
    end
  end
end)

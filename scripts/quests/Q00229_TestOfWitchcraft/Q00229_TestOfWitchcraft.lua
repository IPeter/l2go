-- Translated by questport from quests/Q00229_TestOfWitchcraft (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(229, "Q00229_TestOfWitchcraft", "Test of Witchcraft")
local K = {}

K.LARA = 30063
K.ALEXANDRIA = 30098
K.IKER = 30110
K.VADIN = 30188
K.NESTLE = 30314
K.SIR_KLAUS_VASPER = 30417
K.LEOPOLD = 30435
K.KAIRA = 30476
K.ORIM = 30630
K.RODERIK = 30631
K.ENDRIGO = 30632
K.EVERT = 30633
K.DIRE_WYRM = 20557
K.ENCHANTED_STONE_GOLEM = 20565
K.LETO_LIZARDMAN = 20577
K.LETO_LIZARDMAN_ARCHER = 20578
K.LETO_LIZARDMAN_SOLDIER = 20579
K.LETO_LIZARDMAN_WARRIOR = 20580
K.LETO_LIZARDMAN_SHAMAN = 20581
K.LETO_LIZARDMAN_OVERLORD = 20582
K.TAMLIN_ORC = 20601
K.TAMLIN_ORC_ARCHER = 20602
K.NAMELESS_REVENANT = 27099
K.SKELETAL_MERCENARY = 27100
K.DREVANUL_PRINCE_ZERUEL = 27101
K.ORIM_DIAGRAM = 3308
K.ALEXANDRIA_BOOK = 3309
K.IKER_LIST = 3310
K.DIRE_WYRM_FANG = 3311
K.LETO_LIZARDMAN_CHARM = 3312
K.EN_GOLEM_HEARTSTONE = 3313
K.LARA_MEMO = 3314
K.NESTLE_MEMO = 3315
K.LEOPOLD_JOURNAL = 3316
K.AKLANTOTH_GEM_1 = 3317
K.AKLANTOTH_GEM_2 = 3318
K.AKLANTOTH_GEM_3 = 3319
K.AKLANTOTH_GEM_4 = 3320
K.AKLANTOTH_GEM_5 = 3321
K.AKLANTOTH_GEM_6 = 3322
K.BRIMSTONE_1 = 3323
K.ORIM_INSTRUCTIONS = 3324
K.ORIM_LETTER_1 = 3325
K.ORIM_LETTER_2 = 3326
K.SIR_VASPER_LETTER = 3327
K.VADIN_CRUCIFIX = 3328
K.TAMLIN_ORC_AMULET = 3329
K.VADIN_SANCTIONS = 3330
K.IKER_AMULET = 3331
K.SOULTRAP_CRYSTAL = 3332
K.PURGATORY_KEY = 3333
K.ZERUEL_BIND_CRYSTAL = 3334
K.BRIMSTONE_2 = 3335
K.SWORD_OF_BINDING = 3029
K.MARK_OF_WITCHCRAFT = 3307
K.DIMENSIONAL_DIAMOND = 7562
local _drevanulPrinceZeruel = false
local _swordOfBinding = false

q:register_quest_items(K.ORIM_DIAGRAM, K.ALEXANDRIA_BOOK, K.IKER_LIST, K.DIRE_WYRM_FANG, K.LETO_LIZARDMAN_CHARM, K.EN_GOLEM_HEARTSTONE, K.LARA_MEMO, K.NESTLE_MEMO, K.LEOPOLD_JOURNAL, K.AKLANTOTH_GEM_1, K.AKLANTOTH_GEM_2, K.AKLANTOTH_GEM_3, K.AKLANTOTH_GEM_4, K.AKLANTOTH_GEM_5, K.AKLANTOTH_GEM_6, K.BRIMSTONE_1, K.ORIM_INSTRUCTIONS, K.ORIM_LETTER_1, K.ORIM_LETTER_2, K.SIR_VASPER_LETTER, K.VADIN_CRUCIFIX, K.TAMLIN_ORC_AMULET, K.VADIN_SANCTIONS, K.IKER_AMULET, K.SOULTRAP_CRYSTAL, K.PURGATORY_KEY, K.ZERUEL_BIND_CRYSTAL, K.BRIMSTONE_2, K.SWORD_OF_BINDING)
q:add_start_npc(K.ORIM)
q:add_talk_id(K.LARA, K.ALEXANDRIA, K.IKER, K.VADIN, K.NESTLE, K.SIR_KLAUS_VASPER, K.LEOPOLD, K.KAIRA, K.ORIM, K.RODERIK, K.ENDRIGO, K.EVERT)
q:add_attack_id(K.NAMELESS_REVENANT, K.SKELETAL_MERCENARY, K.DREVANUL_PRINCE_ZERUEL)
q:add_kill_id(K.DIRE_WYRM, K.ENCHANTED_STONE_GOLEM, K.LETO_LIZARDMAN, K.LETO_LIZARDMAN_ARCHER, K.LETO_LIZARDMAN_SOLDIER, K.LETO_LIZARDMAN_WARRIOR, K.LETO_LIZARDMAN_SHAMAN, K.LETO_LIZARDMAN_OVERLORD, K.TAMLIN_ORC, K.TAMLIN_ORC_ARCHER, K.NAMELESS_REVENANT, K.SKELETAL_MERCENARY, K.DREVANUL_PRINCE_ZERUEL)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30630-08.htm" then
    st:start_quest()
    q:give_items(player, K.ORIM_DIAGRAM, 1)
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30630-08a.htm"
      q:give_items(player, K.DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30630-14.htm" then
    st:set_cond(4, true)
    st:unset("gem456")
    q:take_items(player, K.AKLANTOTH_GEM_1, 1)
    q:take_items(player, K.AKLANTOTH_GEM_2, 1)
    q:take_items(player, K.AKLANTOTH_GEM_3, 1)
    q:take_items(player, K.AKLANTOTH_GEM_4, 1)
    q:take_items(player, K.AKLANTOTH_GEM_5, 1)
    q:take_items(player, K.AKLANTOTH_GEM_6, 1)
    q:take_items(player, K.ALEXANDRIA_BOOK, 1)
    q:give_items(player, K.BRIMSTONE_1, 1)
    q:add_spawn(K.DREVANUL_PRINCE_ZERUEL, {x = 70381, y = 109638, z = -3726, heading = 0}, false, 120000)
  elseif event == "30630-16.htm" then
    st:set_cond(6, true)
    q:take_items(player, K.BRIMSTONE_1, 1)
    q:give_items(player, K.ORIM_INSTRUCTIONS, 1)
    q:give_items(player, K.ORIM_LETTER_1, 1)
    q:give_items(player, K.ORIM_LETTER_2, 1)
  elseif event == "30630-22.htm" then
    q:take_items(player, K.IKER_AMULET, 1)
    q:take_items(player, K.ORIM_INSTRUCTIONS, 1)
    q:take_items(player, K.PURGATORY_KEY, 1)
    q:take_items(player, K.SWORD_OF_BINDING, 1)
    q:take_items(player, K.ZERUEL_BIND_CRYSTAL, 1)
    q:give_items(player, K.MARK_OF_WITCHCRAFT, 1)
    q:add_exp_and_sp(player, 139796, 40000)
    player:social_action(3)
    st:exit_quest(false, true)
  elseif event == "30098-03.htm" then
    st:set_cond(2, true)
    st:set("gem456", "1")
    q:take_items(player, K.ORIM_DIAGRAM, 1)
    q:give_items(player, K.ALEXANDRIA_BOOK, 1)
  elseif event == "30110-03.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.IKER_LIST, 1)
  elseif event == "30110-08.htm" then
    q:take_items(player, K.ORIM_LETTER_2, 1)
    q:give_items(player, K.IKER_AMULET, 1)
    q:give_items(player, K.SOULTRAP_CRYSTAL, 1)
    if q:has_quest_items(player, K.SWORD_OF_BINDING) then
      st:set_cond(7, true)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30476-02.htm" then
    q:give_items(player, K.AKLANTOTH_GEM_2, 1)
    if q:has_quest_items(player, K.AKLANTOTH_GEM_1, K.AKLANTOTH_GEM_3) and st:get_int("gem456") == 6 then
      st:set_cond(3, true)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30063-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.LARA_MEMO, 1)
  elseif event == "30314-02.htm" then
    st:set("gem456", "2")
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.NESTLE_MEMO, 1)
  elseif event == "30435-02.htm" then
    st:set("gem456", "3")
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.NESTLE_MEMO, 1)
    q:give_items(player, K.LEOPOLD_JOURNAL, 1)
  elseif event == "30417-03.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.ORIM_LETTER_1, 1)
    q:give_items(player, K.SIR_VASPER_LETTER, 1)
  elseif event == "30633-02.htm" then
    st:set_cond(9, true)
    q:give_items(player, K.BRIMSTONE_2, 1)
    if not _drevanulPrinceZeruel then
      q:add_spawn(K.DREVANUL_PRINCE_ZERUEL, {x = 13395, y = 169807, z = -3708, heading = 0}, false, 299000)
      _drevanulPrinceZeruel = true
      -- Resets Drevanul Prince Zeruel
      q:start_quest_timer("zeruel_cleanup", 300000, nil, player, false)
    end
  elseif event == "zeruel_despawn" then
    npc:abort_attack()
    npc:delete()
    return nil
  elseif event == "zeruel_cleanup" then
    _drevanulPrinceZeruel = false
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.KNIGHT and player:class_id() ~= ClassId.WIZARD and player:class_id() ~= ClassId.PALUS_KNIGHT then
      htmltext = "30630-01.htm"
    elseif player:level() < 39 then
      htmltext = "30630-02.htm"
    else
      htmltext = player:class_id() == ClassId.WIZARD and "30630-03.htm" or "30630-05.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local gem456 = st:get_int("gem456")
    local sw2 = npc:id()
    if sw2 == K.ORIM then
      if cond == 1 then
        htmltext = "30630-09.htm"
      elseif cond == 2 then
        htmltext = "30630-10.htm"
      elseif cond == 3 then
        htmltext = "30630-11.htm"
      elseif cond == 4 then
        htmltext = "30630-14.htm"
      elseif cond == 5 then
        htmltext = "30630-15.htm"
      elseif cond == 6 then
        htmltext = "30630-17.htm"
      elseif cond == 7 then
        htmltext = "30630-18.htm"
        st:set_cond(8, true)
      elseif cond == 8 or cond == 9 then
        htmltext = "30630-18.htm"
      elseif cond == 10 then
        htmltext = "30630-19.htm"
      end
    elseif sw2 == K.ALEXANDRIA then
      if cond == 1 then
        htmltext = "30098-01.htm"
      elseif cond == 2 then
        htmltext = "30098-04.htm"
      else
        htmltext = "30098-05.htm"
      end
    elseif sw2 == K.KAIRA then
      if q:has_quest_items(player, K.AKLANTOTH_GEM_2) then
        htmltext = "30476-03.htm"
      elseif cond == 2 then
        htmltext = "30476-01.htm"
      elseif cond > 3 then
        htmltext = "30476-04.htm"
      end
    elseif sw2 == K.IKER then
      if q:has_quest_items(player, K.AKLANTOTH_GEM_1) then
        htmltext = "30110-06.htm"
      elseif q:has_quest_items(player, K.IKER_LIST) then
        if q:get_quest_items_count(player, K.DIRE_WYRM_FANG) + q:get_quest_items_count(player, K.LETO_LIZARDMAN_CHARM) + q:get_quest_items_count(player, K.EN_GOLEM_HEARTSTONE) < 60 then
          htmltext = "30110-04.htm"
        else
          htmltext = "30110-05.htm"
          q:take_items(player, K.IKER_LIST, 1)
          q:take_items(player, K.DIRE_WYRM_FANG, -1)
          q:take_items(player, K.EN_GOLEM_HEARTSTONE, -1)
          q:take_items(player, K.LETO_LIZARDMAN_CHARM, -1)
          q:give_items(player, K.AKLANTOTH_GEM_1, 1)
          if q:has_quest_items(player, K.AKLANTOTH_GEM_2, K.AKLANTOTH_GEM_3) and gem456 == 6 then
            st:set_cond(3, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        end
      elseif cond == 2 then
        htmltext = "30110-01.htm"
      elseif cond == 6 and not q:has_quest_items(player, K.SOULTRAP_CRYSTAL) then
        htmltext = "30110-07.htm"
      elseif cond >= 6 and cond < 10 then
        htmltext = "30110-09.htm"
      elseif cond == 10 then
        htmltext = "30110-10.htm"
      end
    elseif sw2 == K.LARA then
      if q:has_quest_items(player, K.AKLANTOTH_GEM_3) then
        htmltext = "30063-04.htm"
      elseif q:has_quest_items(player, K.LARA_MEMO) then
        htmltext = "30063-03.htm"
      elseif cond == 2 then
        htmltext = "30063-01.htm"
      elseif cond > 2 then
        htmltext = "30063-05.htm"
      end
    elseif sw2 == K.RODERIK or sw2 == K.ENDRIGO then
      if q:has_at_least_one_quest_item(player, K.LARA_MEMO, K.AKLANTOTH_GEM_3) then
        htmltext = npc:id() .. "-01.htm"
      end
    elseif sw2 == K.NESTLE then
      if gem456 == 1 then
        htmltext = "30314-01.htm"
      elseif gem456 == 2 then
        htmltext = "30314-03.htm"
      elseif gem456 > 2 then
        htmltext = "30314-04.htm"
      end
    elseif sw2 == K.LEOPOLD then
      if gem456 == 2 then
        htmltext = "30435-01.htm"
      elseif gem456 > 2 and gem456 < 6 then
        htmltext = "30435-03.htm"
      elseif gem456 == 6 then
        htmltext = "30435-04.htm"
      elseif cond > 3 then
        htmltext = "30435-05.htm"
      end
    elseif sw2 == K.SIR_KLAUS_VASPER then
      if q:has_at_least_one_quest_item(player, K.SIR_VASPER_LETTER, K.VADIN_CRUCIFIX) then
        htmltext = "30417-04.htm"
      elseif q:has_quest_items(player, K.VADIN_SANCTIONS) then
        htmltext = "30417-05.htm"
        q:take_items(player, K.VADIN_SANCTIONS, 1)
        q:give_items(player, K.SWORD_OF_BINDING, 1)
        if q:has_quest_items(player, K.SOULTRAP_CRYSTAL) then
          st:set_cond(7, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif cond == 6 then
        htmltext = "30417-01.htm"
      elseif cond > 6 then
        htmltext = "30417-06.htm"
      end
    elseif sw2 == K.VADIN then
      if q:has_quest_items(player, K.SIR_VASPER_LETTER) then
        htmltext = "30188-01.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:take_items(player, K.SIR_VASPER_LETTER, 1)
        q:give_items(player, K.VADIN_CRUCIFIX, 1)
      elseif q:has_quest_items(player, K.VADIN_CRUCIFIX) then
        if q:get_quest_items_count(player, K.TAMLIN_ORC_AMULET) < 20 then
          htmltext = "30188-02.htm"
        else
          htmltext = "30188-03.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.TAMLIN_ORC_AMULET, -1)
          q:take_items(player, K.VADIN_CRUCIFIX, -1)
          q:give_items(player, K.VADIN_SANCTIONS, 1)
        end
      elseif q:has_quest_items(player, K.VADIN_SANCTIONS) then
        htmltext = "30188-04.htm"
      elseif cond > 6 then
        htmltext = "30188-05.htm"
      end
    elseif sw2 == K.EVERT then
      if cond == 7 or cond == 8 then
        htmltext = "30633-01.htm"
      elseif cond == 9 then
        htmltext = "30633-02.htm"
        if not _drevanulPrinceZeruel then
          q:add_spawn(K.DREVANUL_PRINCE_ZERUEL, {x = 13395, y = 169807, z = -3708, heading = 0}, false, 299000)
          _drevanulPrinceZeruel = true
          -- Resets Drevanul Prince Zeruel
          q:start_quest_timer("zeruel_cleanup", 300000, nil, player, false)
        end
      elseif cond == 10 then
        htmltext = "30633-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_attack(function(npc, attacker, damage, isPet)
  local st = q:get_quest_state(attacker, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == K.NAMELESS_REVENANT then
    if q:has_quest_items(attacker, K.LARA_MEMO) and not npc:is_script_value(1) then
      npc:set_script_value(1)
      npc:broadcast_say(ChatType.GENERAL, "I absolutely cannot give it to you! It is my precious jewel!")
    end
  elseif sw == K.SKELETAL_MERCENARY then
    if st:get_int("gem456") > 2 and st:get_int("gem456") < 6 and not npc:is_script_value(1) then
      npc:set_script_value(1)
      npc:broadcast_say(ChatType.GENERAL, "I absolutely cannot give it to you! It is my precious jewel!")
    end
  elseif sw == K.DREVANUL_PRINCE_ZERUEL then
    if st:is_cond(4) and not npc:is_script_value(1) then
      st:set_cond(5, true)
      npc:set_script_value(1)
      npc:broadcast_say(ChatType.GENERAL, "I'll take your lives later!!")
      q:start_quest_timer("zeruel_despawn", 1000, npc, attacker, false)
    elseif st:is_cond(9) and _drevanulPrinceZeruel then
      if q:get_item_equipped(attacker, 7) == K.SWORD_OF_BINDING then
        _swordOfBinding = true
        if not npc:is_script_value(1) then
          npc:set_script_value(1)
          npc:broadcast_say(ChatType.GENERAL, "That sword is really...!")
        end
      else
        _swordOfBinding = false
      end
    end
  end
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == K.DIRE_WYRM then
    if q:has_quest_items(player, K.IKER_LIST) and q:get_quest_items_count(player, K.DIRE_WYRM_FANG) < 20 then
      q:give_items(player, K.DIRE_WYRM_FANG, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.ENCHANTED_STONE_GOLEM then
    if q:has_quest_items(player, K.IKER_LIST) and q:get_quest_items_count(player, K.EN_GOLEM_HEARTSTONE) < 20 then
      q:give_items(player, K.EN_GOLEM_HEARTSTONE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.LETO_LIZARDMAN or sw == K.LETO_LIZARDMAN_ARCHER then
    if q:has_quest_items(player, K.IKER_LIST) and rnd(10) < 5 and q:get_quest_items_count(player, K.LETO_LIZARDMAN_CHARM) < 20 then
      q:give_items(player, K.LETO_LIZARDMAN_CHARM, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.LETO_LIZARDMAN_SOLDIER or sw == K.LETO_LIZARDMAN_WARRIOR then
    if q:has_quest_items(player, K.IKER_LIST) and rnd(10) < 6 and q:get_quest_items_count(player, K.LETO_LIZARDMAN_CHARM) < 20 then
      q:give_items(player, K.LETO_LIZARDMAN_CHARM, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.LETO_LIZARDMAN_SHAMAN or sw == K.LETO_LIZARDMAN_OVERLORD then
    if q:has_quest_items(player, K.IKER_LIST) and rnd(10) < 7 and q:get_quest_items_count(player, K.LETO_LIZARDMAN_CHARM) < 20 then
      q:give_items(player, K.LETO_LIZARDMAN_CHARM, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.NAMELESS_REVENANT then
    if q:has_quest_items(player, K.LARA_MEMO) then
      q:take_items(player, K.LARA_MEMO, 1)
      q:give_items(player, K.AKLANTOTH_GEM_3, 1)
      if q:has_quest_items(player, K.AKLANTOTH_GEM_1, K.AKLANTOTH_GEM_2) and st:get_int("gem456") == 6 then
        st:set_cond(3, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.SKELETAL_MERCENARY then
    local gem456 = st:get_int("gem456")
    if gem456 == 3 then
      st:set("gem456", "4")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(player, K.AKLANTOTH_GEM_4, 1)
    elseif gem456 == 4 then
      st:set("gem456", "5")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(player, K.AKLANTOTH_GEM_5, 1)
    elseif gem456 == 5 then
      st:set("gem456", "6")
      q:take_items(player, K.LEOPOLD_JOURNAL, 1)
      q:give_items(player, K.AKLANTOTH_GEM_6, 1)
      if q:has_quest_items(player, K.AKLANTOTH_GEM_1, K.AKLANTOTH_GEM_2, K.AKLANTOTH_GEM_3) then
        st:set_cond(3, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.TAMLIN_ORC or sw == K.TAMLIN_ORC_ARCHER then
    if q:has_quest_items(player, K.VADIN_CRUCIFIX) and rnd(10) < 5 and q:get_quest_items_count(player, K.TAMLIN_ORC_AMULET) < 20 then
      q:give_items(player, K.TAMLIN_ORC_AMULET, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.DREVANUL_PRINCE_ZERUEL then
    if st:is_cond(9) and _drevanulPrinceZeruel then
      if _swordOfBinding then
        st:set_cond(10)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:take_items(player, K.BRIMSTONE_2, 1)
        q:take_items(player, K.SOULTRAP_CRYSTAL, 1)
        q:give_items(player, K.PURGATORY_KEY, 1)
        q:give_items(player, K.ZERUEL_BIND_CRYSTAL, 1)
        npc:broadcast_say(ChatType.GENERAL, "No! I haven't completely finished the command for destruction and slaughter yet!!!")
      end
      q:cancel_quest_timer("zeruel_cleanup", nil, player)
      _drevanulPrinceZeruel = false
    end
  end
end)

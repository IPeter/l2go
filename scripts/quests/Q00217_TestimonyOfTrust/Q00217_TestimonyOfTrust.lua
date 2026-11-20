-- Translated by questport from quests/Q00217_TestimonyOfTrust (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(217, "Q00217_TestimonyOfTrust", "Testimony of Trust")
local K = {}

K.HOLLINT = 30191
K.ASTERIOS = 30154
K.THIFIELL = 30358
K.CLAYTON = 30464
K.SERESIN = 30657
K.KAKAI = 30565
K.MANAKIA = 30515
K.LOCKIRIN = 30531
K.NIKOLA = 30621
K.BIOTIN = 30031
K.DRYAD = 20013
K.DRYAD_ELDER = 20019
K.LIREIN = 20036
K.LIREIN_ELDER = 20044
K.ACTEA_OF_VERDANT_WILDS = 27121
K.LUELL_OF_ZEPHYR_WINDS = 27120
K.GUARDIAN_BASILIK = 20550
K.ANT_RECRUIT = 20082
K.ANT_PATROL = 20084
K.ANT_GUARD = 20086
K.ANT_SOLDIER = 20087
K.ANT_WARRIOR_CAPTAIN = 20088
K.MARSH_STAKATO = 20157
K.MARSH_STAKATO_WORKER = 20230
K.MARSH_STAKATO_SOLDIER = 20232
K.MARSH_STAKATO_DRONE = 20234
K.WINDSUS = 20553
K.PORTA = 20213
K.LETTER_TO_ELF = 2735
K.LETTER_TO_DARK_ELF = 2736
K.LETTER_TO_DWARF = 2737
K.LETTER_TO_ORC = 2738
K.LETTER_TO_SERESIN = 2739
K.SCROLL_OF_DARK_ELF_TRUST = 2740
K.SCROLL_OF_ELF_TRUST = 2741
K.SCROLL_OF_DWARF_TRUST = 2742
K.SCROLL_OF_ORC_TRUST = 2743
K.RECOMMENDATION_OF_HOLLINT = 2744
K.ORDER_OF_ASTERIOS = 2745
K.BREATH_OF_WINDS = 2746
K.SEED_OF_VERDURE = 2747
K.LETTER_FROM_THIFIELL = 2748
K.BLOOD_GUARDIAN_BASILIK = 2749
K.GIANT_APHID = 2750
K.STAKATO_FLUIDS = 2751
K.BASILIK_PLASMA = 2752
K.HONEY_DEW = 2753
K.STAKATO_ICHOR = 2754
K.ORDER_OF_CLAYTON = 2755
K.PARASITE_OF_LOTA = 2756
K.LETTER_TO_MANAKIA = 2757
K.LETTER_OF_MANAKIA = 2758
K.LETTER_TO_NIKOLA = 2759
K.ORDER_OF_NIKOLA = 2760
K.HEARTSTONE_OF_PORTA = 2761
K.MARK_OF_TRUST = 2734
K.DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(K.LETTER_TO_ELF, K.LETTER_TO_DARK_ELF, K.LETTER_TO_DWARF, K.LETTER_TO_ORC, K.LETTER_TO_SERESIN, K.SCROLL_OF_DARK_ELF_TRUST, K.SCROLL_OF_ELF_TRUST, K.SCROLL_OF_DWARF_TRUST, K.SCROLL_OF_ORC_TRUST, K.RECOMMENDATION_OF_HOLLINT, K.ORDER_OF_ASTERIOS, K.BREATH_OF_WINDS, K.SEED_OF_VERDURE, K.LETTER_FROM_THIFIELL, K.BLOOD_GUARDIAN_BASILIK, K.GIANT_APHID, K.STAKATO_FLUIDS, K.BASILIK_PLASMA, K.HONEY_DEW, K.STAKATO_ICHOR, K.ORDER_OF_CLAYTON, K.PARASITE_OF_LOTA, K.LETTER_TO_MANAKIA, K.LETTER_OF_MANAKIA, K.LETTER_TO_NIKOLA, K.ORDER_OF_NIKOLA, K.HEARTSTONE_OF_PORTA)
q:add_start_npc(K.HOLLINT)
q:add_talk_id(K.HOLLINT, K.ASTERIOS, K.THIFIELL, K.CLAYTON, K.SERESIN, K.KAKAI, K.MANAKIA, K.LOCKIRIN, K.NIKOLA, K.BIOTIN)
q:add_kill_id(K.DRYAD, K.DRYAD_ELDER, K.LIREIN, K.LIREIN_ELDER, K.ACTEA_OF_VERDANT_WILDS, K.LUELL_OF_ZEPHYR_WINDS, K.GUARDIAN_BASILIK, K.ANT_RECRUIT, K.ANT_PATROL, K.ANT_GUARD, K.ANT_SOLDIER, K.ANT_WARRIOR_CAPTAIN, K.MARSH_STAKATO, K.MARSH_STAKATO_WORKER, K.MARSH_STAKATO_SOLDIER, K.MARSH_STAKATO_DRONE, K.WINDSUS, K.PORTA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30191-04.htm" then
    st:start_quest()
    q:give_items(player, K.LETTER_TO_ELF, 1)
    q:give_items(player, K.LETTER_TO_DARK_ELF, 1)
    if not player:var_bool("secondClassChange37", false) then
      htmltext = "30191-04a.htm"
      q:give_items(player, K.DIMENSIONAL_DIAMOND, quest.DF_REWARD_37[player:race()])
      player:set_var("secondClassChange37", true)
    end
  elseif event == "30154-03.htm" then
    st:set_cond(2, true)
    q:take_items(player, K.LETTER_TO_ELF, 1)
    q:give_items(player, K.ORDER_OF_ASTERIOS, 1)
  elseif event == "30358-02.htm" then
    st:set_cond(5, true)
    q:take_items(player, K.LETTER_TO_DARK_ELF, 1)
    q:give_items(player, K.LETTER_FROM_THIFIELL, 1)
  elseif event == "30515-02.htm" then
    st:set_cond(14, true)
    q:take_items(player, K.LETTER_TO_MANAKIA, 1)
  elseif event == "30531-02.htm" then
    st:set_cond(18, true)
    q:take_items(player, K.LETTER_TO_DWARF, 1)
    q:give_items(player, K.LETTER_TO_NIKOLA, 1)
  elseif event == "30565-02.htm" then
    st:set_cond(13, true)
    q:take_items(player, K.LETTER_TO_ORC, 1)
    q:give_items(player, K.LETTER_TO_MANAKIA, 1)
  elseif event == "30621-02.htm" then
    st:set_cond(19, true)
    q:take_items(player, K.LETTER_TO_NIKOLA, 1)
    q:give_items(player, K.ORDER_OF_NIKOLA, 1)
  elseif event == "30657-03.htm" then
    if player:level() < 38 then
      htmltext = "30657-02.htm"
      if st:is_cond(10) then
        st:set_cond(11, true)
      end
    else
      st:set_cond(12, true)
      q:take_items(player, K.LETTER_TO_SERESIN, 1)
      q:give_items(player, K.LETTER_TO_DWARF, 1)
      q:give_items(player, K.LETTER_TO_ORC, 1)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if ClassId.level(player:class_id()) ~= 1 then
      htmltext = "30191-01a.htm"
    elseif player:race() ~= Race.HUMAN then
      htmltext = "30191-02.htm"
    elseif player:level() < 37 then
      htmltext = "30191-01.htm"
    else
      htmltext = "30191-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == K.HOLLINT then
      if cond < 9 then
        htmltext = "30191-08.htm"
      elseif cond == 9 then
        htmltext = "30191-05.htm"
        st:set_cond(10, true)
        q:take_items(player, K.SCROLL_OF_DARK_ELF_TRUST, 1)
        q:take_items(player, K.SCROLL_OF_ELF_TRUST, 1)
        q:give_items(player, K.LETTER_TO_SERESIN, 1)
      elseif cond > 9 and cond < 22 then
        htmltext = "30191-09.htm"
      elseif cond == 22 then
        htmltext = "30191-06.htm"
        st:set_cond(23, true)
        q:take_items(player, K.SCROLL_OF_DWARF_TRUST, 1)
        q:take_items(player, K.SCROLL_OF_ORC_TRUST, 1)
        q:give_items(player, K.RECOMMENDATION_OF_HOLLINT, 1)
      elseif cond == 23 then
        htmltext = "30191-07.htm"
      end
    elseif sw2 == K.ASTERIOS then
      if cond == 1 then
        htmltext = "30154-01.htm"
      elseif cond == 2 then
        htmltext = "30154-04.htm"
      elseif cond == 3 then
        htmltext = "30154-05.htm"
        st:set_cond(4, true)
        q:take_items(player, K.BREATH_OF_WINDS, 1)
        q:take_items(player, K.SEED_OF_VERDURE, 1)
        q:take_items(player, K.ORDER_OF_ASTERIOS, 1)
        q:give_items(player, K.SCROLL_OF_ELF_TRUST, 1)
      elseif cond > 3 then
        htmltext = "30154-06.htm"
      end
    elseif sw2 == K.THIFIELL then
      if cond == 4 then
        htmltext = "30358-01.htm"
      elseif cond > 4 and cond < 8 then
        htmltext = "30358-05.htm"
      elseif cond == 8 then
        htmltext = "30358-03.htm"
        st:set_cond(9, true)
        q:take_items(player, K.BASILIK_PLASMA, 1)
        q:take_items(player, K.HONEY_DEW, 1)
        q:take_items(player, K.STAKATO_ICHOR, 1)
        q:give_items(player, K.SCROLL_OF_DARK_ELF_TRUST, 1)
      elseif cond > 8 then
        htmltext = "30358-04.htm"
      end
    elseif sw2 == K.CLAYTON then
      if cond == 5 then
        htmltext = "30464-01.htm"
        st:set_cond(6, true)
        q:take_items(player, K.LETTER_FROM_THIFIELL, 1)
        q:give_items(player, K.ORDER_OF_CLAYTON, 1)
      elseif cond == 6 then
        htmltext = "30464-02.htm"
      elseif cond > 6 then
        htmltext = "30464-03.htm"
        if cond == 7 then
          st:set_cond(8, true)
          q:take_items(player, K.ORDER_OF_CLAYTON, 1)
        end
      end
    elseif sw2 == K.SERESIN then
      if cond == 10 or cond == 11 then
        htmltext = "30657-01.htm"
      elseif cond > 11 and cond < 22 then
        htmltext = "30657-04.htm"
      elseif cond == 22 then
        htmltext = "30657-05.htm"
      end
    elseif sw2 == K.KAKAI then
      if cond == 12 then
        htmltext = "30565-01.htm"
      elseif cond > 12 and cond < 16 then
        htmltext = "30565-03.htm"
      elseif cond == 16 then
        htmltext = "30565-04.htm"
        st:set_cond(17, true)
        q:take_items(player, K.LETTER_OF_MANAKIA, 1)
        q:give_items(player, K.SCROLL_OF_ORC_TRUST, 1)
      elseif cond > 16 then
        htmltext = "30565-05.htm"
      end
    elseif sw2 == K.MANAKIA then
      if cond == 13 then
        htmltext = "30515-01.htm"
      elseif cond == 14 then
        htmltext = "30515-03.htm"
      elseif cond == 15 then
        htmltext = "30515-04.htm"
        st:set_cond(16, true)
        q:take_items(player, K.PARASITE_OF_LOTA, -1)
        q:give_items(player, K.LETTER_OF_MANAKIA, 1)
      elseif cond > 15 then
        htmltext = "30515-05.htm"
      end
    elseif sw2 == K.LOCKIRIN then
      if cond == 17 then
        htmltext = "30531-01.htm"
      elseif cond > 17 and cond < 21 then
        htmltext = "30531-03.htm"
      elseif cond == 21 then
        htmltext = "30531-04.htm"
        st:set_cond(22, true)
        q:give_items(player, K.SCROLL_OF_DWARF_TRUST, 1)
      elseif cond == 22 then
        htmltext = "30531-05.htm"
      end
    elseif sw2 == K.NIKOLA then
      if cond == 18 then
        htmltext = "30621-01.htm"
      elseif cond == 19 then
        htmltext = "30621-03.htm"
      elseif cond == 20 then
        htmltext = "30621-04.htm"
        st:set_cond(21, true)
        q:take_items(player, K.HEARTSTONE_OF_PORTA, -1)
        q:take_items(player, K.ORDER_OF_NIKOLA, 1)
      elseif cond > 20 then
        htmltext = "30621-05.htm"
      end
    elseif sw2 == K.BIOTIN then
      if cond == 23 then
        htmltext = "30031-01.htm"
        q:take_items(player, K.RECOMMENDATION_OF_HOLLINT, 1)
        q:give_items(player, K.MARK_OF_TRUST, 1)
        q:add_exp_and_sp(player, 39571, 2500)
        player:social_action(3)
        st:exit_quest(false, true)
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
  if sw == K.DRYAD or sw == K.DRYAD_ELDER then
    if st:is_cond(2) and not q:has_quest_items(player, K.SEED_OF_VERDURE) and rnd(100) < 33 then
      q:add_spawn(K.ACTEA_OF_VERDANT_WILDS, npc, true, 200000)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_BEFORE_BATTLE)
    end
  elseif sw == K.LIREIN or sw == K.LIREIN_ELDER then
    if st:is_cond(2) and not q:has_quest_items(player, K.BREATH_OF_WINDS) and rnd(100) < 33 then
      q:add_spawn(K.LUELL_OF_ZEPHYR_WINDS, npc, true, 200000)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_BEFORE_BATTLE)
    end
  elseif sw == K.ACTEA_OF_VERDANT_WILDS then
    if st:is_cond(2) and not q:has_quest_items(player, K.SEED_OF_VERDURE) then
      q:give_items(player, K.SEED_OF_VERDURE, 1)
      if q:has_quest_items(player, K.BREATH_OF_WINDS) then
        st:set_cond(3, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.LUELL_OF_ZEPHYR_WINDS then
    if st:is_cond(2) and not q:has_quest_items(player, K.BREATH_OF_WINDS) then
      q:give_items(player, K.BREATH_OF_WINDS, 1)
      if q:has_quest_items(player, K.SEED_OF_VERDURE) then
        st:set_cond(3, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.MARSH_STAKATO or sw == K.MARSH_STAKATO_WORKER or sw == K.MARSH_STAKATO_SOLDIER or sw == K.MARSH_STAKATO_DRONE then
    if st:is_cond(6) and not q:has_quest_items(player, K.STAKATO_ICHOR) then
      q:give_items(player, K.STAKATO_FLUIDS, 1)
      if q:get_quest_items_count(player, K.STAKATO_FLUIDS) >= 10 then
        q:take_items(player, K.STAKATO_FLUIDS, -1)
        q:give_items(player, K.STAKATO_ICHOR, 1)
        if q:has_quest_items(player, K.BASILIK_PLASMA, K.HONEY_DEW) then
          st:set_cond(7, true)
        end
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.ANT_RECRUIT or sw == K.ANT_PATROL or sw == K.ANT_GUARD or sw == K.ANT_SOLDIER or sw == K.ANT_WARRIOR_CAPTAIN then
    if st:is_cond(6) and not q:has_quest_items(player, K.HONEY_DEW) then
      q:give_items(player, K.GIANT_APHID, 1)
      if q:get_quest_items_count(player, K.GIANT_APHID) >= 10 then
        q:take_items(player, K.GIANT_APHID, -1)
        q:give_items(player, K.HONEY_DEW, 1)
        if q:has_quest_items(player, K.BASILIK_PLASMA, K.STAKATO_ICHOR) then
          st:set_cond(7, true)
        end
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.GUARDIAN_BASILIK then
    if st:is_cond(6) and not q:has_quest_items(player, K.BASILIK_PLASMA) then
      q:give_items(player, K.BLOOD_GUARDIAN_BASILIK, 1)
      if q:get_quest_items_count(player, K.BLOOD_GUARDIAN_BASILIK) >= 10 then
        q:take_items(player, K.BLOOD_GUARDIAN_BASILIK, -1)
        q:give_items(player, K.BASILIK_PLASMA, 1)
        if q:has_quest_items(player, K.HONEY_DEW, K.STAKATO_ICHOR) then
          st:set_cond(7, true)
        end
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.WINDSUS then
    if st:is_cond(14) and rnd.bool() then
      q:give_items(player, K.PARASITE_OF_LOTA, 1)
      if q:get_quest_items_count(player, K.PARASITE_OF_LOTA) >= 10 then
        st:set_cond(15, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.PORTA then
    if st:is_cond(19) then
      q:give_items(player, K.HEARTSTONE_OF_PORTA, 1)
      if q:get_quest_items_count(player, K.HEARTSTONE_OF_PORTA) >= 10 then
        st:set_cond(20, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)

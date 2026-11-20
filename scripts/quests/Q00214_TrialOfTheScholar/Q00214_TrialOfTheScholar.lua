-- Translated by questport from quests/Q00214_TrialOfTheScholar (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(214, "Q00214_TrialOfTheScholar", "Trial of the Scholar")
local K = {}

K.SYLVAIN = 30070
K.LUCAS = 30071
K.VALKON = 30103
K.DIETER = 30111
K.JUREK = 30115
K.EDROC = 30230
K.RAUT = 30316
K.POITAN = 30458
K.MIRIEN = 30461
K.MARIA = 30608
K.CRETA = 30609
K.CRONOS = 30610
K.TRIFF = 30611
K.CASIAN = 30612
K.MONSTER_EYE_DESTROYER = 20068
K.MEDUSA = 20158
K.GHOUL = 20201
K.SHACKLE_1 = 20235
K.SHACKLE_2 = 20279
K.BREKA_ORC_SHAMAN = 20269
K.FETTERED_SOUL = 20552
K.GRANDIS = 20554
K.ENCHANTED_GARGOYLE = 20567
K.LETO_LIZARDMAN_WARRIOR = 20580
K.MIRIEN_SIGIL_1 = 2675
K.MIRIEN_SIGIL_2 = 2676
K.MIRIEN_SIGIL_3 = 2677
K.MIRIEN_INSTRUCTION = 2678
K.MARIA_LETTER_1 = 2679
K.MARIA_LETTER_2 = 2680
K.LUCAS_LETTER = 2681
K.LUCILLA_HANDBAG = 2682
K.CRETA_LETTER_1 = 2683
K.CRETA_PAINTING_1 = 2684
K.CRETA_PAINTING_2 = 2685
K.CRETA_PAINTING_3 = 2686
K.BROWN_SCROLL_SCRAP = 2687
K.CRYSTAL_OF_PURITY_1 = 2688
K.HIGH_PRIEST_SIGIL = 2689
K.GRAND_MAGISTER_SIGIL = 2690
K.CRONOS_SIGIL = 2691
K.SYLVAIN_LETTER = 2692
K.SYMBOL_OF_SYLVAIN = 2693
K.JUREK_LIST = 2694
K.MONSTER_EYE_DESTROYER_SKIN = 2695
K.SHAMAN_NECKLACE = 2696
K.SHACKLE_SCALP = 2697
K.SYMBOL_OF_JUREK = 2698
K.CRONOS_LETTER = 2699
K.DIETER_KEY = 2700
K.CRETA_LETTER_2 = 2701
K.DIETER_LETTER = 2702
K.DIETER_DIARY = 2703
K.RAUT_LETTER_ENVELOPE = 2704
K.TRIFF_RING = 2705
K.SCRIPTURE_CHAPTER_1 = 2706
K.SCRIPTURE_CHAPTER_2 = 2707
K.SCRIPTURE_CHAPTER_3 = 2708
K.SCRIPTURE_CHAPTER_4 = 2709
K.VALKON_REQUEST = 2710
K.POITAN_NOTES = 2711
K.STRONG_LIQUOR = 2713
K.CRYSTAL_OF_PURITY_2 = 2714
K.CASIAN_LIST = 2715
K.GHOUL_SKIN = 2716
K.MEDUSA_BLOOD = 2717
K.FETTERED_SOUL_ICHOR = 2718
K.ENCHANTED_GARGOYLE_NAIL = 2719
K.SYMBOL_OF_CRONOS = 2720
K.MARK_OF_SCHOLAR = 2674
K.DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(K.MIRIEN_SIGIL_1, K.MIRIEN_SIGIL_2, K.MIRIEN_SIGIL_3, K.MIRIEN_INSTRUCTION, K.MARIA_LETTER_1, K.MARIA_LETTER_2, K.LUCAS_LETTER, K.LUCILLA_HANDBAG, K.CRETA_LETTER_1, K.CRETA_PAINTING_1, K.CRETA_PAINTING_2, K.CRETA_PAINTING_3, K.BROWN_SCROLL_SCRAP, K.CRYSTAL_OF_PURITY_1, K.HIGH_PRIEST_SIGIL, K.GRAND_MAGISTER_SIGIL, K.CRONOS_SIGIL, K.SYLVAIN_LETTER, K.SYMBOL_OF_SYLVAIN, K.JUREK_LIST, K.MONSTER_EYE_DESTROYER_SKIN, K.SHAMAN_NECKLACE, K.SHACKLE_SCALP, K.SYMBOL_OF_JUREK, K.CRONOS_LETTER, K.DIETER_KEY, K.CRETA_LETTER_2, K.DIETER_LETTER, K.DIETER_DIARY, K.RAUT_LETTER_ENVELOPE, K.TRIFF_RING, K.SCRIPTURE_CHAPTER_1, K.SCRIPTURE_CHAPTER_2, K.SCRIPTURE_CHAPTER_3, K.SCRIPTURE_CHAPTER_4, K.VALKON_REQUEST, K.POITAN_NOTES, K.STRONG_LIQUOR, K.CRYSTAL_OF_PURITY_2, K.CASIAN_LIST, K.GHOUL_SKIN, K.MEDUSA_BLOOD, K.FETTERED_SOUL_ICHOR, K.ENCHANTED_GARGOYLE_NAIL, K.SYMBOL_OF_CRONOS)
q:add_start_npc(K.MIRIEN)
q:add_talk_id(K.MIRIEN, K.SYLVAIN, K.LUCAS, K.VALKON, K.DIETER, K.JUREK, K.EDROC, K.RAUT, K.POITAN, K.MARIA, K.CRETA, K.CRONOS, K.TRIFF, K.CASIAN)
q:add_kill_id(K.MONSTER_EYE_DESTROYER, K.MEDUSA, K.GHOUL, K.SHACKLE_1, K.SHACKLE_2, K.BREKA_ORC_SHAMAN, K.FETTERED_SOUL, K.GRANDIS, K.ENCHANTED_GARGOYLE, K.LETO_LIZARDMAN_WARRIOR)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30461-04.htm" then
    st:start_quest()
    q:give_items(player, K.MIRIEN_SIGIL_1, 1)
    if not player:var_bool("secondClassChange35", false) then
      htmltext = "30461-04a.htm"
      q:give_items(player, K.DIMENSIONAL_DIAMOND, quest.DF_REWARD_35[player:class_id()])
      player:set_var("secondClassChange35", true)
    end
  elseif event == "30461-09.htm" then
    if player:level() < 36 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(player, K.MIRIEN_INSTRUCTION, 1)
    else
      htmltext = "30461-10.htm"
      st:set_cond(19, true)
      q:take_items(player, K.MIRIEN_SIGIL_2, 1)
      q:take_items(player, K.SYMBOL_OF_JUREK, 1)
      q:give_items(player, K.MIRIEN_SIGIL_3, 1)
    end
  elseif event == "30070-02.htm" then
    st:set_cond(2, true)
    q:give_items(player, K.HIGH_PRIEST_SIGIL, 1)
    q:give_items(player, K.SYLVAIN_LETTER, 1)
  elseif event == "30608-02.htm" then
    st:set_cond(3, true)
    q:take_items(player, K.SYLVAIN_LETTER, 1)
    q:give_items(player, K.MARIA_LETTER_1, 1)
  elseif event == "30608-08.htm" then
    st:set_cond(7, true)
    q:take_items(player, K.CRETA_LETTER_1, 1)
    q:give_items(player, K.LUCILLA_HANDBAG, 1)
  elseif event == "30608-14.htm" then
    st:set_cond(13, true)
    q:take_items(player, K.BROWN_SCROLL_SCRAP, -1)
    q:take_items(player, K.CRETA_PAINTING_3, 1)
    q:give_items(player, K.CRYSTAL_OF_PURITY_1, 1)
  elseif event == "30115-03.htm" then
    st:set_cond(16, true)
    q:give_items(player, K.GRAND_MAGISTER_SIGIL, 1)
    q:give_items(player, K.JUREK_LIST, 1)
  elseif event == "30071-04.htm" then
    st:set_cond(10, true)
    q:take_items(player, K.CRETA_PAINTING_2, 1)
    q:give_items(player, K.CRETA_PAINTING_3, 1)
  elseif event == "30609-05.htm" then
    st:set_cond(6, true)
    q:take_items(player, K.MARIA_LETTER_2, 1)
    q:give_items(player, K.CRETA_LETTER_1, 1)
  elseif event == "30609-09.htm" then
    st:set_cond(8, true)
    q:take_items(player, K.LUCILLA_HANDBAG, 1)
    q:give_items(player, K.CRETA_PAINTING_1, 1)
  elseif event == "30609-14.htm" then
    st:set_cond(22, true)
    q:take_items(player, K.DIETER_KEY, 1)
    q:give_items(player, K.CRETA_LETTER_2, 1)
  elseif event == "30610-10.htm" then
    st:set_cond(20, true)
    q:give_items(player, K.CRONOS_LETTER, 1)
    q:give_items(player, K.CRONOS_SIGIL, 1)
  elseif event == "30610-14.htm" then
    st:set_cond(31, true)
    q:take_items(player, K.CRONOS_SIGIL, 1)
    q:take_items(player, K.DIETER_DIARY, 1)
    q:take_items(player, K.SCRIPTURE_CHAPTER_1, 1)
    q:take_items(player, K.SCRIPTURE_CHAPTER_2, 1)
    q:take_items(player, K.SCRIPTURE_CHAPTER_3, 1)
    q:take_items(player, K.SCRIPTURE_CHAPTER_4, 1)
    q:take_items(player, K.TRIFF_RING, 1)
    q:give_items(player, K.SYMBOL_OF_CRONOS, 1)
  elseif event == "30111-05.htm" then
    st:set_cond(21, true)
    q:take_items(player, K.CRONOS_LETTER, 1)
    q:give_items(player, K.DIETER_KEY, 1)
  elseif event == "30111-09.htm" then
    st:set_cond(23, true)
    q:take_items(player, K.CRETA_LETTER_2, 1)
    q:give_items(player, K.DIETER_DIARY, 1)
    q:give_items(player, K.DIETER_LETTER, 1)
  elseif event == "30230-02.htm" then
    st:set_cond(24, true)
    q:take_items(player, K.DIETER_LETTER, 1)
    q:give_items(player, K.RAUT_LETTER_ENVELOPE, 1)
  elseif event == "30316-02.htm" then
    st:set_cond(25, true)
    q:take_items(player, K.RAUT_LETTER_ENVELOPE, 1)
    q:give_items(player, K.SCRIPTURE_CHAPTER_1, 1)
    q:give_items(player, K.STRONG_LIQUOR, 1)
  elseif event == "30611-04.htm" then
    st:set_cond(26, true)
    q:take_items(player, K.STRONG_LIQUOR, 1)
    q:give_items(player, K.TRIFF_RING, 1)
  elseif event == "30103-04.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.VALKON_REQUEST, 1)
  elseif event == "30612-04.htm" then
    st:set_cond(28, true)
    q:give_items(player, K.CASIAN_LIST, 1)
  elseif event == "30612-07.htm" then
    st:set_cond(30, true)
    q:take_items(player, K.CASIAN_LIST, 1)
    q:take_items(player, K.ENCHANTED_GARGOYLE_NAIL, -1)
    q:take_items(player, K.FETTERED_SOUL_ICHOR, -1)
    q:take_items(player, K.GHOUL_SKIN, -1)
    q:take_items(player, K.MEDUSA_BLOOD, -1)
    q:take_items(player, K.POITAN_NOTES, 1)
    q:give_items(player, K.SCRIPTURE_CHAPTER_4, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.WIZARD and player:class_id() ~= ClassId.ELVEN_WIZARD and player:class_id() ~= ClassId.DARK_WIZARD then
      htmltext = "30461-01.htm"
    elseif player:level() < 35 then
      htmltext = "30461-02.htm"
    else
      htmltext = "30461-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == K.MIRIEN then
      if cond < 14 then
        htmltext = "30461-05.htm"
      elseif cond == 14 then
        htmltext = "30461-06.htm"
        st:set_cond(15, true)
        q:take_items(player, K.MIRIEN_SIGIL_1, 1)
        q:take_items(player, K.SYMBOL_OF_SYLVAIN, 1)
        q:give_items(player, K.MIRIEN_SIGIL_2, 1)
      elseif cond > 14 and cond < 18 then
        htmltext = "30461-07.htm"
      elseif cond == 18 then
        if not q:has_quest_items(player, K.MIRIEN_INSTRUCTION) then
          htmltext = "30461-08.htm"
        else
          if player:level() < 36 then
            htmltext = "30461-11.htm"
          else
            htmltext = "30461-12.htm"
            st:set_cond(19, true)
            q:take_items(player, K.MIRIEN_INSTRUCTION, 1)
            q:take_items(player, K.MIRIEN_SIGIL_2, 1)
            q:take_items(player, K.SYMBOL_OF_JUREK, 1)
            q:give_items(player, K.MIRIEN_SIGIL_3, 1)
          end
        end
      elseif cond > 18 and cond < 31 then
        htmltext = "30461-13.htm"
      elseif cond == 31 then
        htmltext = "30461-14.htm"
        q:take_items(player, K.MIRIEN_SIGIL_3, 1)
        q:take_items(player, K.SYMBOL_OF_CRONOS, 1)
        q:give_items(player, K.MARK_OF_SCHOLAR, 1)
        q:add_exp_and_sp(player, 80265, 30000)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == K.SYLVAIN then
      if cond == 1 then
        htmltext = "30070-01.htm"
      elseif cond < 13 then
        htmltext = "30070-03.htm"
      elseif cond == 13 then
        htmltext = "30070-04.htm"
        st:set_cond(14, true)
        q:take_items(player, K.CRYSTAL_OF_PURITY_1, 1)
        q:take_items(player, K.HIGH_PRIEST_SIGIL, 1)
        q:give_items(player, K.SYMBOL_OF_SYLVAIN, 1)
      elseif cond == 14 then
        htmltext = "30070-05.htm"
      elseif cond > 14 then
        htmltext = "30070-06.htm"
      end
    elseif sw2 == K.MARIA then
      if cond == 2 then
        htmltext = "30608-01.htm"
      elseif cond == 3 then
        htmltext = "30608-03.htm"
      elseif cond == 4 then
        htmltext = "30608-04.htm"
        st:set_cond(5, true)
        q:take_items(player, K.LUCAS_LETTER, 1)
        q:give_items(player, K.MARIA_LETTER_2, 1)
      elseif cond == 5 then
        htmltext = "30608-05.htm"
      elseif cond == 6 then
        htmltext = "30608-06.htm"
      elseif cond == 7 then
        htmltext = "30608-09.htm"
      elseif cond == 8 then
        htmltext = "30608-10.htm"
        st:set_cond(9, true)
        q:take_items(player, K.CRETA_PAINTING_1, 1)
        q:give_items(player, K.CRETA_PAINTING_2, 1)
      elseif cond == 9 then
        htmltext = "30608-11.htm"
      elseif cond == 10 then
        htmltext = "30608-12.htm"
        st:set_cond(11, true)
      elseif cond == 11 then
        htmltext = "30608-12.htm"
      elseif cond == 12 then
        htmltext = "30608-13.htm"
      elseif cond == 13 then
        htmltext = "30608-15.htm"
      elseif q:has_at_least_one_quest_item(player, K.SYMBOL_OF_SYLVAIN, K.MIRIEN_SIGIL_2) then
        htmltext = "30608-16.htm"
      elseif cond > 18 then
        if not q:has_quest_items(player, K.VALKON_REQUEST) then
          htmltext = "30608-17.htm"
        else
          htmltext = "30608-18.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.VALKON_REQUEST, 1)
          q:give_items(player, K.CRYSTAL_OF_PURITY_2, 1)
        end
      end
    elseif sw2 == K.JUREK then
      if cond == 15 then
        htmltext = "30115-01.htm"
      elseif cond == 16 then
        htmltext = "30115-04.htm"
      elseif cond == 17 then
        htmltext = "30115-05.htm"
        st:set_cond(18, true)
        q:take_items(player, K.GRAND_MAGISTER_SIGIL, 1)
        q:take_items(player, K.JUREK_LIST, 1)
        q:take_items(player, K.MONSTER_EYE_DESTROYER_SKIN, -1)
        q:take_items(player, K.SHACKLE_SCALP, -1)
        q:take_items(player, K.SHAMAN_NECKLACE, -1)
        q:give_items(player, K.SYMBOL_OF_JUREK, 1)
      elseif cond == 18 then
        htmltext = "30115-06.htm"
      elseif cond > 18 then
        htmltext = "30115-07.htm"
      end
    elseif sw2 == K.LUCAS then
      if cond == 3 then
        htmltext = "30071-01.htm"
        st:set_cond(4, true)
        q:take_items(player, K.MARIA_LETTER_1, 1)
        q:give_items(player, K.LUCAS_LETTER, 1)
      elseif cond > 3 and cond < 9 then
        htmltext = "30071-02.htm"
      elseif cond == 9 then
        htmltext = "30071-03.htm"
      elseif cond == 10 or cond == 11 then
        htmltext = "30071-05.htm"
      elseif cond == 12 then
        htmltext = "30071-06.htm"
      elseif cond > 12 then
        htmltext = "30071-07.htm"
      end
    elseif sw2 == K.CRETA then
      if cond == 5 then
        htmltext = "30609-01.htm"
      elseif cond == 6 then
        htmltext = "30609-06.htm"
      elseif cond == 7 then
        htmltext = "30609-07.htm"
      elseif cond > 7 and cond < 13 then
        htmltext = "30609-10.htm"
      elseif cond >= 13 and cond < 19 then
        htmltext = "30609-11.htm"
      elseif cond == 21 then
        htmltext = "30609-12.htm"
      elseif cond > 21 then
        htmltext = "30609-15.htm"
      end
    elseif sw2 == K.CRONOS then
      if cond == 19 then
        htmltext = "30610-01.htm"
      elseif cond > 19 and cond < 30 then
        htmltext = "30610-11.htm"
      elseif cond == 30 then
        htmltext = "30610-12.htm"
      elseif cond == 31 then
        htmltext = "30610-15.htm"
      end
    elseif sw2 == K.DIETER then
      if cond == 20 then
        htmltext = "30111-01.htm"
      elseif cond == 21 then
        htmltext = "30111-06.htm"
      elseif cond == 22 then
        htmltext = "30111-07.htm"
      elseif cond == 23 then
        htmltext = "30111-10.htm"
      elseif cond == 24 then
        htmltext = "30111-11.htm"
      elseif cond > 24 and cond < 31 then
        htmltext = not q:has_quest_items(player, K.SCRIPTURE_CHAPTER_1, K.SCRIPTURE_CHAPTER_2, K.SCRIPTURE_CHAPTER_3, K.SCRIPTURE_CHAPTER_4) and "30111-12.htm" or "30111-13.htm"
      elseif cond == 31 then
        htmltext = "30111-15.htm"
      end
    elseif sw2 == K.EDROC then
      if cond == 23 then
        htmltext = "30230-01.htm"
      elseif cond == 24 then
        htmltext = "30230-03.htm"
      elseif cond > 24 then
        htmltext = "30230-04.htm"
      end
    elseif sw2 == K.RAUT then
      if cond == 24 then
        htmltext = "30316-01.htm"
      elseif cond == 25 then
        htmltext = "30316-04.htm"
      elseif cond > 25 then
        htmltext = "30316-05.htm"
      end
    elseif sw2 == K.TRIFF then
      if cond == 25 then
        htmltext = "30611-01.htm"
      elseif cond > 25 then
        htmltext = "30611-05.htm"
      end
    elseif sw2 == K.VALKON then
      if q:has_quest_items(player, K.TRIFF_RING) then
        if not q:has_quest_items(player, K.SCRIPTURE_CHAPTER_2) then
          if not q:has_quest_items(player, K.VALKON_REQUEST) then
            if not q:has_quest_items(player, K.CRYSTAL_OF_PURITY_2) then
              htmltext = "30103-01.htm"
            else
              htmltext = "30103-06.htm"
              q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
              q:take_items(player, K.CRYSTAL_OF_PURITY_2, 1)
              q:give_items(player, K.SCRIPTURE_CHAPTER_2, 1)
            end
          else
            htmltext = "30103-05.htm"
          end
        else
          htmltext = "30103-07.htm"
        end
      end
    elseif sw2 == K.POITAN then
      if cond == 26 or cond == 27 then
        if not q:has_quest_items(player, K.POITAN_NOTES) then
          htmltext = "30458-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:give_items(player, K.POITAN_NOTES, 1)
        else
          htmltext = "30458-02.htm"
        end
      elseif cond == 28 or cond == 29 then
        htmltext = "30458-03.htm"
      elseif cond == 30 then
        htmltext = "30458-04.htm"
      end
    elseif sw2 == K.CASIAN then
      if (cond == 26 or cond == 27) and q:has_quest_items(player, K.POITAN_NOTES) then
        if q:has_quest_items(player, K.SCRIPTURE_CHAPTER_1, K.SCRIPTURE_CHAPTER_2, K.SCRIPTURE_CHAPTER_3) then
          htmltext = "30612-02.htm"
        else
          htmltext = "30612-01.htm"
          if cond == 26 then
            st:set_cond(27, true)
          end
        end
      elseif cond == 28 then
        htmltext = "30612-05.htm"
      elseif cond == 29 then
        htmltext = "30612-06.htm"
      elseif cond == 30 then
        htmltext = "30612-08.htm"
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
  if sw == K.LETO_LIZARDMAN_WARRIOR then
    if st:is_cond(11) and rnd.bool() then
      q:give_items(player, K.BROWN_SCROLL_SCRAP, 1)
      if q:get_quest_items_count(player, K.BROWN_SCROLL_SCRAP) >= 5 then
        st:set_cond(12, true)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.SHACKLE_1 or sw == K.SHACKLE_2 then
    if st:is_cond(16) and q:get_quest_items_count(player, K.SHACKLE_SCALP) < 2 and rnd.bool() then
      q:give_items(player, K.SHACKLE_SCALP, 1)
      if q:get_quest_items_count(player, K.SHACKLE_SCALP) >= 2 and q:get_quest_items_count(player, K.MONSTER_EYE_DESTROYER_SKIN) == 5 and q:get_quest_items_count(player, K.SHAMAN_NECKLACE) == 5 then
        st:set_cond(17, true)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.MONSTER_EYE_DESTROYER then
    if st:is_cond(16) and q:get_quest_items_count(player, K.MONSTER_EYE_DESTROYER_SKIN) < 5 and rnd.bool() then
      q:give_items(player, K.MONSTER_EYE_DESTROYER_SKIN, 1)
      if q:get_quest_items_count(player, K.MONSTER_EYE_DESTROYER_SKIN) >= 5 and q:get_quest_items_count(player, K.SHACKLE_SCALP) == 2 and q:get_quest_items_count(player, K.SHAMAN_NECKLACE) == 5 then
        st:set_cond(17, true)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.BREKA_ORC_SHAMAN then
    if st:is_cond(16) and q:get_quest_items_count(player, K.SHAMAN_NECKLACE) < 5 and rnd.bool() then
      q:give_items(player, K.SHAMAN_NECKLACE, 1)
      if q:get_quest_items_count(player, K.SHAMAN_NECKLACE) >= 5 and q:get_quest_items_count(player, K.SHACKLE_SCALP) == 2 and q:get_quest_items_count(player, K.MONSTER_EYE_DESTROYER_SKIN) == 5 then
        st:set_cond(17, true)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.GRANDIS then
    if q:has_quest_items(player, K.TRIFF_RING) and not q:has_quest_items(player, K.SCRIPTURE_CHAPTER_3) and rnd(100) < 30 then
      q:give_items(player, K.SCRIPTURE_CHAPTER_3, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
  elseif sw == K.GHOUL then
    if st:is_cond(28) and q:get_quest_items_count(player, K.GHOUL_SKIN) < 10 then
      q:give_items(player, K.GHOUL_SKIN, 1)
      if q:get_quest_items_count(player, K.GHOUL_SKIN) >= 10 and q:get_quest_items_count(player, K.MEDUSA_BLOOD) == 12 and q:get_quest_items_count(player, K.FETTERED_SOUL_ICHOR) == 5 and q:get_quest_items_count(player, K.ENCHANTED_GARGOYLE_NAIL) == 5 then
        st:set_cond(29, true)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.MEDUSA then
    if st:is_cond(28) and q:get_quest_items_count(player, K.MEDUSA_BLOOD) < 12 then
      q:give_items(player, K.MEDUSA_BLOOD, 1)
      if q:get_quest_items_count(player, K.MEDUSA_BLOOD) >= 12 and q:get_quest_items_count(player, K.GHOUL_SKIN) == 10 and q:get_quest_items_count(player, K.FETTERED_SOUL_ICHOR) == 5 and q:get_quest_items_count(player, K.ENCHANTED_GARGOYLE_NAIL) == 5 then
        st:set_cond(29, true)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.FETTERED_SOUL then
    if st:is_cond(28) and q:get_quest_items_count(player, K.FETTERED_SOUL_ICHOR) < 5 then
      q:give_items(player, K.FETTERED_SOUL_ICHOR, 1)
      if q:get_quest_items_count(player, K.FETTERED_SOUL_ICHOR) >= 5 and q:get_quest_items_count(player, K.MEDUSA_BLOOD) == 12 and q:get_quest_items_count(player, K.GHOUL_SKIN) == 10 and q:get_quest_items_count(player, K.ENCHANTED_GARGOYLE_NAIL) == 5 then
        st:set_cond(29, true)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.ENCHANTED_GARGOYLE then
    if st:is_cond(28) and q:get_quest_items_count(player, K.ENCHANTED_GARGOYLE_NAIL) < 5 then
      q:give_items(player, K.ENCHANTED_GARGOYLE_NAIL, 1)
      if q:get_quest_items_count(player, K.ENCHANTED_GARGOYLE_NAIL) >= 5 and q:get_quest_items_count(player, K.MEDUSA_BLOOD) == 12 and q:get_quest_items_count(player, K.GHOUL_SKIN) == 10 and q:get_quest_items_count(player, K.FETTERED_SOUL_ICHOR) == 5 then
        st:set_cond(29, true)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)

-- Translated by questport from quests/Q00213_TrialOfTheSeeker (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(213, "Q00213_TrialOfTheSeeker", "Trial of the Seeker")

-- NPCs
local TERRY = 30064
local DUFNER = 30106
local BRUNON = 30526
local VIKTOR = 30684
local MARINA = 30715
-- Monsters
local NEER_GHOUL_BERSERKER = 20198
local ANT_CAPTAIN = 20080
local OL_MAHUM_CAPTAIN = 20211
local TURAK_BUGBEAR_WARRIOR = 20249
local TUREK_ORC_WARLORD = 20495
local MEDUSA = 20158
local ANT_WARRIOR_CAPTAIN = 20088
local MARSH_STAKATO_DRONE = 20234
local BREKA_ORC_OVERLORD = 20270
local LETO_LIZARDMAN_WARRIOR = 20580
-- Items
local DUFNER_LETTER = 2647
local TERRY_ORDER_1 = 2648
local TERRY_ORDER_2 = 2649
local TERRY_LETTER = 2650
local VIKTOR_LETTER = 2651
local HAWKEYE_LETTER = 2652
local MYSTERIOUS_RUNESTONE = 2653
local OL_MAHUM_RUNESTONE = 2654
local TUREK_RUNESTONE = 2655
local ANT_RUNESTONE = 2656
local TURAK_BUGBEAR_RUNESTONE = 2657
local TERRY_BOX = 2658
local VIKTOR_REQUEST = 2659
local MEDUSA_SCALES = 2660
local SHILEN_RUNESTONE = 2661
local ANALYSIS_REQUEST = 2662
local MARINA_LETTER = 2663
local EXPERIMENT_TOOLS = 2664
local ANALYSIS_RESULT = 2665
local TERRY_ORDER_3 = 2666
local LIST_OF_HOST = 2667
local ABYSS_RUNESTONE_1 = 2668
local ABYSS_RUNESTONE_2 = 2669
local ABYSS_RUNESTONE_3 = 2670
local ABYSS_RUNESTONE_4 = 2671
local TERRY_REPORT = 2672
-- Rewards
local MARK_OF_SEEKER = 2673
local DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(DUFNER_LETTER, TERRY_ORDER_1, TERRY_ORDER_2, TERRY_LETTER, VIKTOR_LETTER, HAWKEYE_LETTER, MYSTERIOUS_RUNESTONE, OL_MAHUM_RUNESTONE, TUREK_RUNESTONE, ANT_RUNESTONE, TURAK_BUGBEAR_RUNESTONE, TERRY_BOX, VIKTOR_REQUEST, MEDUSA_SCALES, SHILEN_RUNESTONE, ANALYSIS_REQUEST, MARINA_LETTER, EXPERIMENT_TOOLS, ANALYSIS_RESULT, TERRY_ORDER_3, LIST_OF_HOST, ABYSS_RUNESTONE_1, ABYSS_RUNESTONE_2, ABYSS_RUNESTONE_3, ABYSS_RUNESTONE_4, TERRY_REPORT)
q:add_start_npc(DUFNER)
q:add_talk_id(TERRY, DUFNER, BRUNON, VIKTOR, MARINA)
q:add_kill_id(NEER_GHOUL_BERSERKER, ANT_CAPTAIN, OL_MAHUM_CAPTAIN, TURAK_BUGBEAR_WARRIOR, TUREK_ORC_WARLORD, ANT_WARRIOR_CAPTAIN, MARSH_STAKATO_DRONE, BREKA_ORC_OVERLORD, LETO_LIZARDMAN_WARRIOR, MEDUSA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30106-05.htm" then
    st:start_quest()
    q:give_items(player, DUFNER_LETTER, 1)
    if not player:var_bool("secondClassChange35", false) then
      htmltext = "30106-05a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_35[player:class_id()])
      player:set_var("secondClassChange35", true)
    end
  elseif event == "30064-03.htm" then
    st:set_cond(2, true)
    q:take_items(player, DUFNER_LETTER, 1)
    q:give_items(player, TERRY_ORDER_1, 1)
  elseif event == "30064-06.htm" then
    st:set_cond(4, true)
    q:take_items(player, MYSTERIOUS_RUNESTONE, 1)
    q:take_items(player, TERRY_ORDER_1, 1)
    q:give_items(player, TERRY_ORDER_2, 1)
  elseif event == "30064-10.htm" then
    st:set_cond(6, true)
    q:take_items(player, ANT_RUNESTONE, 1)
    q:take_items(player, OL_MAHUM_RUNESTONE, 1)
    q:take_items(player, TURAK_BUGBEAR_RUNESTONE, 1)
    q:take_items(player, TUREK_RUNESTONE, 1)
    q:take_items(player, TERRY_ORDER_2, 1)
    q:give_items(player, TERRY_BOX, 1)
    q:give_items(player, TERRY_LETTER, 1)
  elseif event == "30064-18.htm" then
    if player:level() < 36 then
      htmltext = "30064-17.htm"
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, ANALYSIS_RESULT, 1)
      q:give_items(player, TERRY_ORDER_3, 1)
    else
      st:set_cond(16, true)
      q:take_items(player, ANALYSIS_RESULT, 1)
      q:give_items(player, LIST_OF_HOST, 1)
    end
  elseif event == "30684-05.htm" then
    st:set_cond(7, true)
    q:take_items(player, TERRY_LETTER, 1)
    q:give_items(player, VIKTOR_LETTER, 1)
  elseif event == "30684-11.htm" then
    st:set_cond(9, true)
    q:take_items(player, TERRY_LETTER, 1)
    q:take_items(player, TERRY_BOX, 1)
    q:take_items(player, HAWKEYE_LETTER, 1)
    q:take_items(player, VIKTOR_LETTER, 1)
    q:give_items(player, VIKTOR_REQUEST, 1)
  elseif event == "30684-15.htm" then
    st:set_cond(11, true)
    q:take_items(player, VIKTOR_REQUEST, 1)
    q:take_items(player, MEDUSA_SCALES, 10)
    q:give_items(player, ANALYSIS_REQUEST, 1)
    q:give_items(player, SHILEN_RUNESTONE, 1)
  elseif event == "30715-02.htm" then
    st:set_cond(12, true)
    q:take_items(player, SHILEN_RUNESTONE, 1)
    q:take_items(player, ANALYSIS_REQUEST, 1)
    q:give_items(player, MARINA_LETTER, 1)
  elseif event == "30715-05.htm" then
    st:set_cond(14, true)
    q:take_items(player, EXPERIMENT_TOOLS, 1)
    q:give_items(player, ANALYSIS_RESULT, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() == ClassId.ROGUE or player:class_id() == ClassId.ELVEN_SCOUT or player:class_id() == ClassId.ASSASSIN then
      htmltext = player:level() < 35 and "30106-02.htm" or "30106-03.htm"
    else
      htmltext = "30106-00.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == DUFNER then
      if cond == 1 then
        htmltext = "30106-06.htm"
      elseif cond > 1 then
        if not q:has_quest_items(player, TERRY_REPORT) then
          htmltext = "30106-07.htm"
        else
          htmltext = "30106-08.htm"
          q:take_items(player, TERRY_REPORT, 1)
          q:give_items(player, MARK_OF_SEEKER, 1)
          q:add_exp_and_sp(player, 72126, 11000)
          player:social_action(3)
          st:exit_quest(false, true)
        end
      end
    elseif sw2 == TERRY then
      if cond == 1 then
        htmltext = "30064-01.htm"
      elseif cond == 2 then
        htmltext = "30064-04.htm"
      elseif cond == 3 then
        htmltext = "30064-05.htm"
      elseif cond == 4 then
        htmltext = "30064-08.htm"
      elseif cond == 5 then
        htmltext = "30064-09.htm"
      elseif cond == 6 then
        htmltext = "30064-11.htm"
      elseif cond == 7 then
        htmltext = "30064-12.htm"
        st:set_cond(8, true)
        q:take_items(player, VIKTOR_LETTER, 1)
        q:give_items(player, HAWKEYE_LETTER, 1)
      elseif cond == 8 then
        htmltext = "30064-13.htm"
      elseif cond > 8 and cond < 14 then
        htmltext = "30064-14.htm"
      elseif cond == 14 then
        if not q:has_quest_items(player, TERRY_ORDER_3) then
          htmltext = "30064-15.htm"
        elseif player:level() < 36 then
          htmltext = "30064-20.htm"
        else
          htmltext = "30064-21.htm"
          st:set_cond(15, true)
          q:take_items(player, TERRY_ORDER_3, 1)
          q:give_items(player, LIST_OF_HOST, 1)
        end
      elseif cond == 15 or cond == 16 then
        htmltext = "30064-22.htm"
      elseif cond == 17 then
        if not q:has_quest_items(player, TERRY_REPORT) then
          htmltext = "30064-23.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
          q:take_items(player, LIST_OF_HOST, 1)
          q:take_items(player, ABYSS_RUNESTONE_1, 1)
          q:take_items(player, ABYSS_RUNESTONE_2, 1)
          q:take_items(player, ABYSS_RUNESTONE_3, 1)
          q:take_items(player, ABYSS_RUNESTONE_4, 1)
          q:give_items(player, TERRY_REPORT, 1)
        else
          htmltext = "30064-24.htm"
        end
      end
    elseif sw2 == VIKTOR then
      if cond == 6 then
        htmltext = "30684-01.htm"
      elseif cond == 7 then
        htmltext = "30684-05.htm"
      elseif cond == 8 then
        htmltext = "30684-12.htm"
      elseif cond == 9 then
        htmltext = "30684-13.htm"
      elseif cond == 10 then
        htmltext = "30684-14.htm"
      elseif cond == 11 then
        htmltext = "30684-16.htm"
      elseif cond > 11 then
        htmltext = "30684-17.htm"
      end
    elseif sw2 == MARINA then
      if cond == 11 then
        htmltext = "30715-01.htm"
      elseif cond == 12 then
        htmltext = "30715-03.htm"
      elseif cond == 13 then
        htmltext = "30715-04.htm"
      elseif q:has_quest_items(player, ANALYSIS_RESULT) then
        htmltext = "30715-06.htm"
      end
    elseif sw2 == BRUNON then
      if cond == 12 then
        htmltext = "30526-01.htm"
        st:set_cond(13, true)
        q:take_items(player, MARINA_LETTER, 1)
        q:give_items(player, EXPERIMENT_TOOLS, 1)
      elseif cond == 13 then
        htmltext = "30526-02.htm"
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
  if sw == NEER_GHOUL_BERSERKER then
    if st:is_cond(2) and rnd(100) < 10 then
      q:give_items(player, MYSTERIOUS_RUNESTONE, 1)
      st:set_cond(3, true)
    end
  elseif sw == ANT_CAPTAIN then
    if st:is_cond(4) and not q:has_quest_items(player, ANT_RUNESTONE) and rnd(100) < 25 then
      q:give_items(player, ANT_RUNESTONE, 1)
      if q:has_quest_items(player, OL_MAHUM_RUNESTONE, TURAK_BUGBEAR_RUNESTONE, TUREK_RUNESTONE) then
        st:set_cond(5, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == OL_MAHUM_CAPTAIN then
    if st:is_cond(4) and not q:has_quest_items(player, OL_MAHUM_RUNESTONE) and rnd(100) < 25 then
      q:give_items(player, OL_MAHUM_RUNESTONE, 1)
      if q:has_quest_items(player, ANT_RUNESTONE, TURAK_BUGBEAR_RUNESTONE, TUREK_RUNESTONE) then
        st:set_cond(5, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == TURAK_BUGBEAR_WARRIOR then
    if st:is_cond(4) and not q:has_quest_items(player, TURAK_BUGBEAR_RUNESTONE) and rnd(100) < 25 then
      q:give_items(player, TURAK_BUGBEAR_RUNESTONE, 1)
      if q:has_quest_items(player, ANT_RUNESTONE, OL_MAHUM_RUNESTONE, TUREK_RUNESTONE) then
        st:set_cond(5, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == TUREK_ORC_WARLORD then
    if st:is_cond(4) and not q:has_quest_items(player, TUREK_RUNESTONE) and rnd(100) < 25 then
      q:give_items(player, TUREK_RUNESTONE, 1)
      if q:has_quest_items(player, ANT_RUNESTONE, OL_MAHUM_RUNESTONE, TURAK_BUGBEAR_RUNESTONE) then
        st:set_cond(5, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == MEDUSA then
    if st:is_cond(9) and rnd(100) < 30 then
      q:give_items(player, MEDUSA_SCALES, 1)
      if q:get_quest_items_count(player, MEDUSA_SCALES) >= 10 then
        st:set_cond(10, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == MARSH_STAKATO_DRONE then
    if (st:is_cond(15) or st:is_cond(16)) and not q:has_quest_items(player, ABYSS_RUNESTONE_1) and rnd(100) < 25 then
      q:give_items(player, ABYSS_RUNESTONE_1, 1)
      if q:has_quest_items(player, ABYSS_RUNESTONE_2, ABYSS_RUNESTONE_3, ABYSS_RUNESTONE_4) then
        st:set_cond(17, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == BREKA_ORC_OVERLORD then
    if (st:is_cond(15) or st:is_cond(16)) and not q:has_quest_items(player, ABYSS_RUNESTONE_2) and rnd(100) < 25 then
      q:give_items(player, ABYSS_RUNESTONE_2, 1)
      if q:has_quest_items(player, ABYSS_RUNESTONE_1, ABYSS_RUNESTONE_3, ABYSS_RUNESTONE_4) then
        st:set_cond(17, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == ANT_WARRIOR_CAPTAIN then
    if (st:is_cond(15) or st:is_cond(16)) and not q:has_quest_items(player, ABYSS_RUNESTONE_3) and rnd(100) < 25 then
      q:give_items(player, ABYSS_RUNESTONE_3, 1)
      if q:has_quest_items(player, ABYSS_RUNESTONE_1, ABYSS_RUNESTONE_2, ABYSS_RUNESTONE_4) then
        st:set_cond(17, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == LETO_LIZARDMAN_WARRIOR then
    if (st:is_cond(15) or st:is_cond(16)) and not q:has_quest_items(player, ABYSS_RUNESTONE_4) and rnd(100) < 25 then
      q:give_items(player, ABYSS_RUNESTONE_4, 1)
      if q:has_quest_items(player, ABYSS_RUNESTONE_1, ABYSS_RUNESTONE_2, ABYSS_RUNESTONE_3) then
        st:set_cond(17, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)

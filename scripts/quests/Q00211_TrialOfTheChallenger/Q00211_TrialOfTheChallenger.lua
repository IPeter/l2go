-- Translated by questport from quests/Q00211_TrialOfTheChallenger (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(211, "Q00211_TrialOfTheChallenger", "Trial of the Challenger")

-- NPCs
local FILAUR = 30535
local KASH = 30644
local MARTIEN = 30645
local RALDO = 30646
local CHEST_OF_SHYSLASSYS = 30647
-- Monsters
local SHYSLASSYS = 27110
local GORR = 27112
local BARAHAM = 27113
local SUCCUBUS_QUEEN = 27114
-- Items
local LETTER_OF_KASH = 2628
local WATCHER_EYE_1 = 2629
local WATCHER_EYE_2 = 2630
local SCROLL_OF_SHYSLASSYS = 2631
local BROKEN_KEY = 2632
-- Rewards
local ADENA = 57
local ELVEN_NECKLACE_BEADS = 1904
local WHITE_TUNIC_PATTERN = 1936
local IRON_BOOTS_DESIGN = 1940
local MANTICOR_SKIN_GAITERS_PATTERN = 1943
local RIP_GAUNTLETS_PATTERN = 1946
local TOME_OF_BLOOD_PAGE = 2030
local MITHRIL_SCALE_GAITERS_MATERIAL = 2918
local BRIGANDINE_GAUNTLETS_PATTERN = 2927
local MARK_OF_CHALLENGER = 2627
local DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(LETTER_OF_KASH, WATCHER_EYE_1, WATCHER_EYE_2, SCROLL_OF_SHYSLASSYS, BROKEN_KEY)
q:add_start_npc(KASH)
q:add_talk_id(FILAUR, KASH, MARTIEN, RALDO, CHEST_OF_SHYSLASSYS)
q:add_kill_id(SHYSLASSYS, GORR, BARAHAM, SUCCUBUS_QUEEN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30644-05.htm" then
    st:start_quest()
    if not player:var_bool("secondClassChange35", false) then
      htmltext = "30644-05a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_35[player:class_id()])
      player:set_var("secondClassChange35", true)
    end
  elseif event == "30645-02.htm" then
    st:set_cond(4, true)
    q:take_items(player, LETTER_OF_KASH, 1)
  elseif event == "30646-04.htm" or event == "30646-06.htm" then
    st:set_cond(8, true)
    q:take_items(player, WATCHER_EYE_2, 1)
  elseif event == "30647-04.htm" then
    if q:has_quest_items(player, BROKEN_KEY) then
      if rnd(10) < 2 then
        htmltext = "30647-03.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_JACKPOT)
        q:take_items(player, BROKEN_KEY, 1)
        local chance = rnd(100)
        if chance > 90 then
          q:reward_items(player, BRIGANDINE_GAUNTLETS_PATTERN, 1)
          q:reward_items(player, IRON_BOOTS_DESIGN, 1)
          q:reward_items(player, MANTICOR_SKIN_GAITERS_PATTERN, 1)
          q:reward_items(player, MITHRIL_SCALE_GAITERS_MATERIAL, 1)
          q:reward_items(player, RIP_GAUNTLETS_PATTERN, 1)
        elseif chance > 70 then
          q:reward_items(player, ELVEN_NECKLACE_BEADS, 1)
          q:reward_items(player, TOME_OF_BLOOD_PAGE, 1)
        elseif chance > 40 then
          q:reward_items(player, WHITE_TUNIC_PATTERN, 1)
        else
          q:reward_items(player, IRON_BOOTS_DESIGN, 1)
        end
      else
        htmltext = "30647-02.htm"
        q:take_items(player, BROKEN_KEY, 1)
        q:reward_items(player, ADENA, rnd(1, 1000))
      end
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.WARRIOR and player:class_id() ~= ClassId.ELVEN_KNIGHT and player:class_id() ~= ClassId.PALUS_KNIGHT and player:class_id() ~= ClassId.ORC_RAIDER and player:class_id() ~= ClassId.ORC_MONK then
      htmltext = "30644-02.htm"
    elseif player:level() < 35 then
      htmltext = "30644-01.htm"
    else
      htmltext = "30644-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == KASH then
      if cond == 1 then
        htmltext = "30644-06.htm"
      elseif cond == 2 then
        htmltext = "30644-07.htm"
        st:set_cond(3, true)
        q:take_items(player, SCROLL_OF_SHYSLASSYS, 1)
        q:give_items(player, LETTER_OF_KASH, 1)
      elseif cond == 3 then
        htmltext = "30644-08.htm"
      elseif cond > 3 then
        htmltext = "30644-09.htm"
      end
    elseif sw2 == CHEST_OF_SHYSLASSYS then
      htmltext = "30647-01.htm"
    elseif sw2 == MARTIEN then
      if cond == 3 then
        htmltext = "30645-01.htm"
      elseif cond == 4 then
        htmltext = "30645-03.htm"
      elseif cond == 5 then
        htmltext = "30645-04.htm"
        st:set_cond(6, true)
        q:take_items(player, WATCHER_EYE_1, 1)
      elseif cond == 6 then
        htmltext = "30645-05.htm"
      elseif cond == 7 then
        htmltext = "30645-07.htm"
      elseif cond > 7 then
        htmltext = "30645-06.htm"
      end
    elseif sw2 == RALDO then
      if cond == 7 then
        htmltext = "30646-01.htm"
      elseif cond == 8 then
        htmltext = "30646-06a.htm"
      elseif cond == 10 then
        htmltext = "30646-07.htm"
        q:take_items(player, BROKEN_KEY, 1)
        q:give_items(player, MARK_OF_CHALLENGER, 1)
        q:add_exp_and_sp(player, 72394, 11250)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == FILAUR then
      if cond == 8 then
        if player:level() >= 36 then
          htmltext = "30535-01.htm"
          st:set_cond(9, true)
        else
          htmltext = "30535-03.htm"
        end
      elseif cond == 9 then
        htmltext = "30535-02.htm"
        q:add_radar(player, 176560, -184969, -3729)
      elseif cond == 10 then
        htmltext = "30535-04.htm"
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
  if sw == SHYSLASSYS then
    if st:is_cond(1) then
      st:set_cond(2, true)
      q:give_items(player, BROKEN_KEY, 1)
      q:give_items(player, SCROLL_OF_SHYSLASSYS, 1)
      q:add_spawn(CHEST_OF_SHYSLASSYS, npc, false, 200000)
    end
  elseif sw == GORR then
    if st:is_cond(4) then
      q:give_items(player, WATCHER_EYE_1, 1)
      st:set_cond(5, true)
    end
  elseif sw == BARAHAM then
    if st:is_cond(6) then
      q:give_items(player, WATCHER_EYE_2, 1)
      st:set_cond(7, true)
    end
    q:add_spawn(RALDO, npc, false, 100000)
  elseif sw == SUCCUBUS_QUEEN then
    if st:is_cond(9) then
      st:set_cond(10, true)
    end
    q:add_spawn(RALDO, npc, false, 100000)
  end
end)

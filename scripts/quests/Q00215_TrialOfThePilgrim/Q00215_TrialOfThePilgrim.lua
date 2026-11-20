-- Translated by questport from quests/Q00215_TrialOfThePilgrim (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(215, "Q00215_TrialOfThePilgrim", "Trial of the Pilgrim")

-- NPCs
local SANTIAGO = 30648
local TANAPI = 30571
local ANCESTOR_MARTANKUS = 30649
local GAURI_TWINKLEROCK = 30550
local DORF = 30651
local GERALD = 30650
local PRIMOS = 30117
local PETRON = 30036
local ANDELLIA = 30362
local URUHA = 30652
local CASIAN = 30612
-- Monsters
local LAVA_SALAMANDER = 27116
local NAHIR = 27117
local BLACK_WILLOW = 27118
-- Items
local BOOK_OF_SAGE = 2722
local VOUCHER_OF_TRIAL = 2723
local SPIRIT_OF_FLAME = 2724
local ESSENCE_OF_FLAME = 2725
local BOOK_OF_GERALD = 2726
local GRAY_BADGE = 2727
local PICTURE_OF_NAHIR = 2728
local HAIR_OF_NAHIR = 2729
local STATUE_OF_EINHASAD = 2730
local BOOK_OF_DARKNESS = 2731
local DEBRIS_OF_WILLOW = 2732
local TAG_OF_RUMOR = 2733
-- Rewards
local MARK_OF_PILGRIM = 2721
local DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(BOOK_OF_SAGE, VOUCHER_OF_TRIAL, SPIRIT_OF_FLAME, ESSENCE_OF_FLAME, BOOK_OF_GERALD, GRAY_BADGE, PICTURE_OF_NAHIR, HAIR_OF_NAHIR, STATUE_OF_EINHASAD, BOOK_OF_DARKNESS, DEBRIS_OF_WILLOW, TAG_OF_RUMOR)
q:add_start_npc(SANTIAGO)
q:add_talk_id(SANTIAGO, TANAPI, ANCESTOR_MARTANKUS, GAURI_TWINKLEROCK, DORF, GERALD, PRIMOS, PETRON, ANDELLIA, URUHA, CASIAN)
q:add_kill_id(LAVA_SALAMANDER, NAHIR, BLACK_WILLOW)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30648-04.htm" then
    st:start_quest()
    q:give_items(player, VOUCHER_OF_TRIAL, 1)
    if not player:var_bool("secondClassChange35", false) then
      htmltext = "30648-04a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_35[player:class_id()])
      player:set_var("secondClassChange35", true)
    end
  elseif event == "30649-04.htm" then
    st:set_cond(5, true)
    q:take_items(player, ESSENCE_OF_FLAME, 1)
    q:give_items(player, SPIRIT_OF_FLAME, 1)
  elseif event == "30650-02.htm" then
    if q:get_quest_items_count(player, 57) >= 100000 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, 57, 100000)
      q:give_items(player, BOOK_OF_GERALD, 1)
    else
      htmltext = "30650-03.htm"
    end
  elseif event == "30652-02.htm" then
    st:set_cond(15, true)
    q:take_items(player, DEBRIS_OF_WILLOW, 1)
    q:give_items(player, BOOK_OF_DARKNESS, 1)
  elseif event == "30362-04.htm" then
    st:set_cond(16, true)
  elseif event == "30362-05.htm" then
    st:set_cond(16, true)
    q:take_items(player, BOOK_OF_DARKNESS, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.CLERIC and player:class_id() ~= ClassId.ORACLE and player:class_id() ~= ClassId.SHILLIEN_ORACLE and player:class_id() ~= ClassId.ORC_SHAMAN then
      htmltext = "30648-02.htm"
    elseif player:level() < 35 then
      htmltext = "30648-01.htm"
    else
      htmltext = "30648-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == SANTIAGO then
      if cond < 17 then
        htmltext = "30648-09.htm"
      elseif cond == 17 then
        htmltext = "30648-10.htm"
        q:take_items(player, BOOK_OF_SAGE, 1)
        q:give_items(player, MARK_OF_PILGRIM, 1)
        q:add_exp_and_sp(player, 77382, 16000)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == TANAPI then
      if cond == 1 then
        htmltext = "30571-01.htm"
        st:set_cond(2, true)
        q:take_items(player, VOUCHER_OF_TRIAL, 1)
      elseif cond < 5 then
        htmltext = "30571-02.htm"
      elseif cond >= 5 then
        htmltext = "30571-03.htm"
        if cond == 5 then
          st:set_cond(6, true)
        end
      end
    elseif sw2 == ANCESTOR_MARTANKUS then
      if cond == 2 then
        htmltext = "30649-01.htm"
        st:set_cond(3, true)
      elseif cond == 3 then
        htmltext = "30649-02.htm"
      elseif cond == 4 then
        htmltext = "30649-03.htm"
      end
    elseif sw2 == GAURI_TWINKLEROCK then
      if cond == 6 then
        htmltext = "30550-01.htm"
        st:set_cond(7, true)
        q:give_items(player, TAG_OF_RUMOR, 1)
      elseif cond > 6 then
        htmltext = "30550-02.htm"
      end
    elseif sw2 == DORF then
      if cond == 7 then
        htmltext = not q:has_quest_items(player, BOOK_OF_GERALD) and "30651-01.htm" or "30651-02.htm"
        st:set_cond(8, true)
        q:take_items(player, TAG_OF_RUMOR, 1)
        q:give_items(player, GRAY_BADGE, 1)
      elseif cond > 7 then
        htmltext = "30651-03.htm"
      end
    elseif sw2 == GERALD then
      if cond == 7 and not q:has_quest_items(player, BOOK_OF_GERALD) then
        htmltext = "30650-01.htm"
      elseif cond == 8 and q:has_quest_items(player, BOOK_OF_GERALD) then
        htmltext = "30650-04.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:take_items(player, BOOK_OF_GERALD, 1)
        q:give_adena(player, 100000, true)
      end
    elseif sw2 == PRIMOS then
      if cond == 8 then
        htmltext = "30117-01.htm"
        st:set_cond(9, true)
      elseif cond > 8 then
        htmltext = "30117-02.htm"
      end
    elseif sw2 == PETRON then
      if cond == 9 then
        htmltext = "30036-01.htm"
        st:set_cond(10, true)
        q:give_items(player, PICTURE_OF_NAHIR, 1)
      elseif cond == 10 then
        htmltext = "30036-02.htm"
      elseif cond == 11 then
        htmltext = "30036-03.htm"
        st:set_cond(12, true)
        q:take_items(player, HAIR_OF_NAHIR, 1)
        q:take_items(player, PICTURE_OF_NAHIR, 1)
        q:give_items(player, STATUE_OF_EINHASAD, 1)
      elseif cond > 11 then
        htmltext = "30036-04.htm"
      end
    elseif sw2 == ANDELLIA then
      if cond == 12 then
        if player:level() < 36 then
          htmltext = "30362-01a.htm"
        else
          htmltext = "30362-01.htm"
          st:set_cond(13, true)
        end
      elseif cond == 13 then
        htmltext = rnd.bool() and "30362-02.htm" or "30362-02a.htm"
      elseif cond == 14 then
        htmltext = "30362-07.htm"
      elseif cond == 15 then
        htmltext = "30362-03.htm"
      elseif cond == 16 then
        htmltext = "30362-06.htm"
      end
    elseif sw2 == URUHA then
      if cond == 14 then
        htmltext = "30652-01.htm"
      elseif cond == 15 then
        htmltext = "30652-03.htm"
      end
    elseif sw2 == CASIAN then
      if cond == 16 then
        htmltext = "30612-01.htm"
        st:set_cond(17, true)
        q:take_items(player, BOOK_OF_DARKNESS, 1)
        q:take_items(player, GRAY_BADGE, 1)
        q:take_items(player, SPIRIT_OF_FLAME, 1)
        q:take_items(player, STATUE_OF_EINHASAD, 1)
        q:give_items(player, BOOK_OF_SAGE, 1)
      elseif cond == 17 then
        htmltext = "30612-02.htm"
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
  if sw == LAVA_SALAMANDER then
    if st:is_cond(3) and rnd(10) < 2 then
      q:give_items(player, ESSENCE_OF_FLAME, 1)
      st:set_cond(4, true)
    end
  elseif sw == NAHIR then
    if st:is_cond(10) and rnd(10) < 2 then
      q:give_items(player, HAIR_OF_NAHIR, 1)
      st:set_cond(11, true)
    end
  elseif sw == BLACK_WILLOW then
    if st:is_cond(13) and rnd(10) < 2 then
      q:give_items(player, DEBRIS_OF_WILLOW, 1)
      st:set_cond(14, true)
    end
  end
end)

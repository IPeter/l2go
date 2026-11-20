-- Translated by questport from quests/Q00212_TrialOfDuty (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(212, "Q00212_TrialOfDuty", "Trial of Duty")

-- NPCs
local HANNAVALT = 30109
local DUSTIN = 30116
local SIR_COLLIN = 30311
local SIR_ARON = 30653
local SIR_KIEL = 30654
local SILVERSHADOW = 30655
local SPIRIT_TALIANUS = 30656
-- Items
local LETTER_OF_DUSTIN = 2634
local KNIGHTS_TEAR = 2635
local MIRROR_OF_ORPIC = 2636
local TEAR_OF_CONFESSION = 2637
local REPORT_PIECE_1 = 2638
local REPORT_PIECE_2 = 2639
local TEAR_OF_LOYALTY = 2640
local MILITAS_ARTICLE = 2641
local SAINTS_ASHES_URN = 2642
local ATHEBALDT_SKULL = 2643
local ATHEBALDT_RIBS = 2644
local ATHEBALDT_SHIN = 2645
local LETTER_OF_WINDAWOOD = 2646
local OLD_KNIGHT_SWORD = 3027
-- Rewards
local MARK_OF_DUTY = 2633
local DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(LETTER_OF_DUSTIN, KNIGHTS_TEAR, MIRROR_OF_ORPIC, TEAR_OF_CONFESSION, REPORT_PIECE_1, REPORT_PIECE_2, TEAR_OF_LOYALTY, MILITAS_ARTICLE, SAINTS_ASHES_URN, ATHEBALDT_SKULL, ATHEBALDT_RIBS, ATHEBALDT_SHIN, LETTER_OF_WINDAWOOD, OLD_KNIGHT_SWORD)
q:add_start_npc(HANNAVALT)
q:add_talk_id(HANNAVALT, DUSTIN, SIR_COLLIN, SIR_ARON, SIR_KIEL, SILVERSHADOW, SPIRIT_TALIANUS)
q:add_kill_id(20144, 20190, 20191, 20200, 20201, 20270, 27119, 20577, 20578, 20579, 20580, 20581, 20582)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30109-04.htm" then
    st:start_quest()
    if not player:var_bool("secondClassChange35", false) then
      htmltext = "30109-04a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_35[player:class_id()])
      player:set_var("secondClassChange35", true)
    end
  elseif event == "30116-05.htm" then
    st:set_cond(14, true)
    q:take_items(player, TEAR_OF_LOYALTY, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.KNIGHT and player:class_id() ~= ClassId.ELVEN_KNIGHT and player:class_id() ~= ClassId.PALUS_KNIGHT then
      htmltext = "30109-02.htm"
    elseif player:level() < 35 then
      htmltext = "30109-01.htm"
    else
      htmltext = "30109-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == HANNAVALT then
      if cond == 18 then
        htmltext = "30109-05.htm"
        q:take_items(player, LETTER_OF_DUSTIN, 1)
        q:give_items(player, MARK_OF_DUTY, 1)
        q:add_exp_and_sp(player, 79832, 3750)
        player:social_action(3)
        st:exit_quest(false, true)
      else
        htmltext = "30109-04a.htm"
      end
    elseif sw2 == SIR_ARON then
      if cond == 1 then
        htmltext = "30653-01.htm"
        st:set_cond(2, true)
        q:give_items(player, OLD_KNIGHT_SWORD, 1)
      elseif cond == 2 then
        htmltext = "30653-02.htm"
      elseif cond == 3 then
        htmltext = "30653-03.htm"
        st:set_cond(4, true)
        q:take_items(player, KNIGHTS_TEAR, 1)
        q:take_items(player, OLD_KNIGHT_SWORD, 1)
      elseif cond > 3 then
        htmltext = "30653-04.htm"
      end
    elseif sw2 == SIR_KIEL then
      if cond == 4 then
        htmltext = "30654-01.htm"
        st:set_cond(5, true)
      elseif cond == 5 then
        htmltext = "30654-02.htm"
      elseif cond == 6 then
        htmltext = "30654-03.htm"
        st:set_cond(7, true)
        q:give_items(player, MIRROR_OF_ORPIC, 1)
      elseif cond == 7 then
        htmltext = "30654-04.htm"
      elseif cond == 9 then
        htmltext = "30654-05.htm"
        st:set_cond(10, true)
        q:take_items(player, TEAR_OF_CONFESSION, 1)
      elseif cond > 9 then
        htmltext = "30654-06.htm"
      end
    elseif sw2 == SPIRIT_TALIANUS then
      if cond == 8 then
        htmltext = "30656-01.htm"
        st:set_cond(9, true)
        q:take_items(player, MIRROR_OF_ORPIC, 1)
        q:take_items(player, REPORT_PIECE_2, 1)
        q:give_items(player, TEAR_OF_CONFESSION, 1)
        -- Despawn the spirit.
        npc:delete()
      end
    elseif sw2 == SILVERSHADOW then
      if cond == 10 then
        if player:level() < 35 then
          htmltext = "30655-01.htm"
        else
          htmltext = "30655-02.htm"
          st:set_cond(11, true)
        end
      elseif cond == 11 then
        htmltext = "30655-03.htm"
      elseif cond == 12 then
        htmltext = "30655-04.htm"
        st:set_cond(13, true)
        q:take_items(player, MILITAS_ARTICLE, -1)
        q:give_items(player, TEAR_OF_LOYALTY, 1)
      elseif cond == 13 then
        htmltext = "30655-05.htm"
      end
    elseif sw2 == DUSTIN then
      if cond == 13 then
        htmltext = "30116-01.htm"
      elseif cond == 14 then
        htmltext = "30116-06.htm"
      elseif cond == 15 then
        htmltext = "30116-07.htm"
        st:set_cond(16, true)
        q:take_items(player, ATHEBALDT_SKULL, 1)
        q:take_items(player, ATHEBALDT_RIBS, 1)
        q:take_items(player, ATHEBALDT_SHIN, 1)
        q:give_items(player, SAINTS_ASHES_URN, 1)
      elseif cond == 16 then
        htmltext = "30116-09.htm"
      elseif cond == 17 then
        htmltext = "30116-08.htm"
        st:set_cond(18, true)
        q:take_items(player, LETTER_OF_WINDAWOOD, 1)
        q:give_items(player, LETTER_OF_DUSTIN, 1)
      elseif cond == 18 then
        htmltext = "30116-10.htm"
      end
    elseif sw2 == SIR_COLLIN then
      if cond == 16 then
        htmltext = "30311-01.htm"
        st:set_cond(17, true)
        q:take_items(player, SAINTS_ASHES_URN, 1)
        q:give_items(player, LETTER_OF_WINDAWOOD, 1)
      elseif cond > 16 then
        htmltext = "30311-02.htm"
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
  if sw == 20190 or sw == 20191 then
    if st:is_cond(2) and rnd(10) < 1 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_BEFORE_BATTLE)
      q:add_spawn(27119, npc, false, 120000)
    end
  elseif sw == 27119 then
    if st:is_cond(2) and q:get_item_equipped(player, Paperdoll.RHAND) == OLD_KNIGHT_SWORD then
      st:set_cond(3, true)
      q:give_items(player, KNIGHTS_TEAR, 1)
    end
  elseif sw == 20201 or sw == 20200 then
    if st:is_cond(5) then
      q:give_items(player, REPORT_PIECE_1, 1)
      if q:get_quest_items_count(player, REPORT_PIECE_1) >= 10 then
        st:set_cond(6)
        q:take_items(player, REPORT_PIECE_1, -1)
        q:give_items(player, REPORT_PIECE_2, 1)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 20144 then
    if (st:is_cond(7) or st:is_cond(8)) and rnd(100) < 33 then
      if st:is_cond(7) then
        st:set_cond(8, true)
      end
      q:add_spawn(30656, npc, false, 300000)
    end
  elseif sw == 20577 or sw == 20578 or sw == 20579 or sw == 20580 or sw == 20581 or sw == 20582 then
    if st:is_cond(11) then
      q:give_items(player, MILITAS_ARTICLE, 1)
      if q:get_quest_items_count(player, MILITAS_ARTICLE) >= 20 then
        st:set_cond(12, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 20270 then
    if st:is_cond(14) and rnd.bool() then
      if not q:has_quest_items(player, ATHEBALDT_SKULL) then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, ATHEBALDT_SKULL, 1)
      elseif not q:has_quest_items(player, ATHEBALDT_RIBS) then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, ATHEBALDT_RIBS, 1)
      elseif not q:has_quest_items(player, ATHEBALDT_SHIN) then
        st:set_cond(15, true)
        q:give_items(player, ATHEBALDT_SHIN, 1)
      end
    end
  end
end)

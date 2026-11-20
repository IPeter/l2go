-- Translated by questport from quests/Q00414_PathOfTheOrcRaider (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(414, "Q00414_PathOfTheOrcRaider", "Path to an Orc Raider")

-- NPCs
local KARUKIA = 30570
local KASMAN = 30501
local TAZEER = 31978
-- Monsters
local GOBLIN_TOMB_RAIDER_LEADER = 20320
local KURUKA_RATMAN_LEADER = 27045
local UMBAR_ORC = 27054
local TIMORA_ORC = 27320
-- Items
local GREEN_BLOOD = 1578
local GOBLIN_DWELLING_MAP = 1579
local KURUKA_RATMAN_TOOTH = 1580
local BETRAYER_REPORT_1 = 1589
local BETRAYER_REPORT_2 = 1590
local HEAD_OF_BETRAYER = 1591
local MARK_OF_RAIDER = 1592
local TIMORA_ORC_HEAD = 8544

q:register_quest_items(GREEN_BLOOD, GOBLIN_DWELLING_MAP, KURUKA_RATMAN_TOOTH, BETRAYER_REPORT_1, BETRAYER_REPORT_2, HEAD_OF_BETRAYER, TIMORA_ORC_HEAD)
q:add_start_npc(KARUKIA)
q:add_talk_id(KARUKIA, KASMAN, TAZEER)
q:add_kill_id(GOBLIN_TOMB_RAIDER_LEADER, KURUKA_RATMAN_LEADER, UMBAR_ORC, TIMORA_ORC)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30570-05.htm" then
    if player:class_id() ~= ClassId.ORC_FIGHTER then
      htmltext = player:class_id() == ClassId.ORC_RAIDER and "30570-02a.htm" or "30570-03.htm"
    elseif player:level() < 19 then
      htmltext = "30570-02.htm"
    elseif q:has_quest_items(player, MARK_OF_RAIDER) then
      htmltext = "30570-04.htm"
    else
      st:start_quest()
      q:give_items(player, GOBLIN_DWELLING_MAP, 1)
    end
  elseif event == "30570-07a.htm" then
    st:set_cond(3, true)
    q:take_items(player, GOBLIN_DWELLING_MAP, 1)
    q:take_items(player, KURUKA_RATMAN_TOOTH, -1)
    q:give_items(player, BETRAYER_REPORT_1, 1)
    q:give_items(player, BETRAYER_REPORT_2, 1)
  elseif event == "30570-07b.htm" then
    st:set_cond(5, true)
    q:take_items(player, GOBLIN_DWELLING_MAP, 1)
    q:take_items(player, KURUKA_RATMAN_TOOTH, -1)
  elseif event == "31978-03.htm" then
    st:set_cond(6, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30570-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == KARUKIA then
      if cond == 1 then
        htmltext = "30570-06.htm"
      elseif cond == 2 then
        htmltext = "30570-07.htm"
      elseif cond == 3 or cond == 4 then
        htmltext = "30570-08.htm"
      elseif cond == 5 then
        htmltext = "30570-07b.htm"
      end
    elseif sw2 == KASMAN then
      if cond == 3 then
        htmltext = q:has_quest_items(player, HEAD_OF_BETRAYER) and "30501-02.htm" or "30501-01.htm"
      elseif cond == 4 then
        htmltext = "30501-03.htm"
        q:take_items(player, BETRAYER_REPORT_1, 1)
        q:take_items(player, BETRAYER_REPORT_2, 1)
        q:take_items(player, HEAD_OF_BETRAYER, -1)
        q:give_items(player, MARK_OF_RAIDER, 1)
        q:add_exp_and_sp(player, 3200, 2360)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == TAZEER then
      if cond == 5 then
        htmltext = "31978-01.htm"
      elseif cond == 6 then
        htmltext = "31978-04.htm"
      elseif cond == 7 then
        htmltext = "31978-05.htm"
        q:take_items(player, TIMORA_ORC_HEAD, 1)
        q:give_items(player, MARK_OF_RAIDER, 1)
        q:add_exp_and_sp(player, 3200, 2360)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == GOBLIN_TOMB_RAIDER_LEADER then
    if st:is_cond(1) then
      if q:get_quest_items_count(player, GREEN_BLOOD) <= rnd(20) then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, GREEN_BLOOD, 1)
      else
        q:take_items(player, GREEN_BLOOD, -1)
        q:add_spawn(KURUKA_RATMAN_LEADER, npc, false, 300000)
      end
    end
  elseif sw == KURUKA_RATMAN_LEADER then
    if st:is_cond(1) then
      if q:get_quest_items_count(player, KURUKA_RATMAN_TOOTH) < 10 then
        q:give_items(player, KURUKA_RATMAN_TOOTH, 1)
        if q:get_quest_items_count(player, KURUKA_RATMAN_TOOTH) >= 10 then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == UMBAR_ORC then
    if st:is_cond(3) and q:get_quest_items_count(player, HEAD_OF_BETRAYER) < 2 and rnd(10) < 2 then
      q:give_items(player, HEAD_OF_BETRAYER, 1)
      if q:get_quest_items_count(player, HEAD_OF_BETRAYER) >= 2 then
        st:set_cond(4, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == TIMORA_ORC then
    if st:is_cond(6) and rnd(10) < 6 then
      q:give_items(player, TIMORA_ORC_HEAD, 1)
      st:set_cond(7, true)
    end
  end
end)

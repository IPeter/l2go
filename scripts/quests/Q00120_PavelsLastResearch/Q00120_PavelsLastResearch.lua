-- Translated by questport from quests/Q00120_PavelsLastResearch (Mobius CT_0 Interlude); owned by L2Go from here on.
-- @author Mobius
-- @note Based on python script
local J = require("lib.java")

local q = quest.new(120, "Q00120_PavelsLastResearch", "Pavel's Research")

-- NPCs
local YUMI = 32041
local WEATHER1 = 32042 -- north
local WEATHER2 = 32043 -- east
local WEATHER3 = 32044 -- west
local BOOKSHELF = 32045
local STONES = 32046
local WENDY = 32047
-- Items
local EAR_BINDING = 854
local REPORT = 8058
local REPORT2 = 8059
local ENIGMA = 8060
local FLOWER = 8290
local HEART = 8291
local NECKLACE = 8292

q:add_start_npc(STONES)
q:add_talk_id(BOOKSHELF, STONES, WEATHER1, WEATHER2, WEATHER3, WENDY, YUMI)
q:register_quest_items(FLOWER, REPORT, REPORT2, ENIGMA, HEART, NECKLACE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return htmltext
  end
  if event == "32041-03.htm" then
    qs:set_cond(3, true)
  elseif event == "32041-04.htm" then
    qs:set_cond(4, true)
  elseif event == "32041-12.htm" then
    qs:set_cond(8, true)
  elseif event == "32041-16.htm" then
    qs:set_cond(16, true)
    q:give_items(player, ENIGMA, 1)
  elseif event == "32041-22.htm" then
    qs:set_cond(17, true)
    q:take_items(player, ENIGMA, 1)
  elseif event == "32041-32.htm" then
    q:take_items(player, NECKLACE, 1)
    q:give_items(player, EAR_BINDING, 1)
    qs:exit_quest(true, true)
  elseif event == "32042-06.htm" then
    if qs:is_cond(10) then
      if qs:get_int("talk") + qs:get_int("talk1") == 2 then
        qs:set_cond(11, true)
        qs:set("talk", "0")
        qs:set("talk1", "0")
      else
        htmltext = "32042-03.htm"
      end
    end
  elseif event == "32042-10.htm" then
    if qs:get_int("talk") + qs:get_int("talk1") + qs:get_int("talk2") == 3 then
      htmltext = "32042-14.htm"
    end
  elseif event == "32042-11.htm" then
    if qs:get_int("talk") == 0 then
      qs:set("talk", "1")
    end
  elseif event == "32042-12.htm" then
    if qs:get_int("talk1") == 0 then
      qs:set("talk1", "1")
    end
  elseif event == "32042-13.htm" then
    if qs:get_int("talk2") == 0 then
      qs:set("talk2", "1")
    end
  elseif event == "32042-15.htm" then
    qs:set_cond(12, true)
    qs:set("talk", "0")
    qs:set("talk1", "0")
    qs:set("talk2", "0")
  elseif event == "32043-06.htm" then
    if qs:is_cond(17) then
      if qs:get_int("talk") + qs:get_int("talk1") == 2 then
        qs:set_cond(18, true)
        qs:set("talk", "0")
        qs:set("talk1", "0")
      else
        htmltext = "32043-03.htm"
      end
    end
  elseif event == "32043-15.htm" then
    if qs:get_int("talk") + qs:get_int("talk1") == 2 then
      htmltext = "32043-29.htm"
    end
  elseif event == "32043-18.htm" then
    if qs:get_int("talk") == 1 then
      htmltext = "32043-21.htm"
    end
  elseif event == "32043-20.htm" then
    qs:set("talk", "1")
    q:play_sound(player, QuestSound.AMBSOUND_DRONE)
  elseif event == "32043-28.htm" then
    qs:set("talk1", "1")
  elseif event == "32043-30.htm" then
    qs:set_cond(19)
    qs:set("talk", "0")
    qs:set("talk1", "0")
  elseif event == "32044-06.htm" then
    if qs:is_cond(20) then
      if qs:get_int("talk") + qs:get_int("talk1") == 2 then
        qs:set_cond(21, true)
        qs:set("talk", "0")
        qs:set("talk1", "0")
      else
        htmltext = "32044-03.htm"
      end
    end
  elseif event == "32044-08.htm" then
    if qs:get_int("talk") + qs:get_int("talk1") == 2 then
      htmltext = "32044-11.htm"
    end
  elseif event == "32044-09.htm" then
    if qs:get_int("talk") == 0 then
      qs:set("talk", "1")
    end
  elseif event == "32044-10.htm" then
    if qs:get_int("talk1") == 0 then
      qs:set("talk1", "1")
    end
  elseif event == "32044-17.htm" then
    qs:set_cond(22, true)
    qs:set("talk", "0")
    qs:set("talk1", "0")
  elseif event == "32045-02.htm" then
    qs:set_cond(15, true)
    q:give_items(player, REPORT2, 1)
    npc:magic_skill_use(qs:get_player(), 5073, 5, 1500, 0)
  elseif event == "32046-04.htm" or event == "32046-05.htm" then
    qs:exit_quest(true)
  elseif event == "32046-06.htm" then
    if qs:get_player():level() >= 50 then
      qs:start_quest()
    else
      htmltext = "32046-00.htm"
      qs:exit_quest(true)
    end
  elseif event == "32046-08.htm" then
    qs:set_cond(2)
  elseif event == "32046-12.htm" then
    qs:set_cond(6, true)
    q:give_items(player, FLOWER, 1)
  elseif event == "32046-22.htm" then
    qs:set_cond(10, true)
  elseif event == "32046-29.htm" then
    qs:set_cond(13, true)
  elseif event == "32046-35.htm" then
    qs:set_cond(20, true)
  elseif event == "32046-38.htm" then
    qs:set_cond(23, true)
    q:give_items(player, HEART, 1)
  elseif event == "32047-06.htm" then
    qs:set_cond(5, true)
  elseif event == "32047-10.htm" then
    qs:set_cond(7, true)
    q:take_items(player, FLOWER, 1)
  elseif event == "32047-15.htm" then
    qs:set_cond(9, true)
  elseif event == "32047-18.htm" then
    qs:set_cond(14, true)
  elseif event == "32047-26.htm" then
    qs:set_cond(24, true)
    q:take_items(player, HEART, 1)
  elseif event == "32047-32.htm" then
    qs:set_cond(25, true)
    q:give_items(player, NECKLACE, 1)
  elseif event == "w1_1" then
    qs:set("talk", "1")
    htmltext = "32042-04.htm"
  elseif event == "w1_2" then
    qs:set("talk1", "1")
    htmltext = "32042-05.htm"
  elseif event == "w2_1" then
    qs:set("talk", "1")
    htmltext = "32043-04.htm"
  elseif event == "w2_2" then
    qs:set("talk1", "1")
    htmltext = "32043-05.htm"
  elseif event == "w3_1" then
    qs:set("talk", "1")
    htmltext = "32044-04.htm"
  elseif event == "w3_2" then
    qs:set("talk1", "1")
    htmltext = "32044-05.htm"
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  if qs == nil then
    return q:get_no_quest_msg(player)
  end
  local htmltext = q:get_no_quest_msg(player)
  local state = qs:get_state()
  local npcId = npc:id()
  local cond = qs:get_cond()
  if state == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  elseif npcId == STONES then
    if state == State.CREATED then
      local qs2 = player:get_quest_state("Q00114_ResurrectionOfAnOldManager")
      if qs2 ~= nil then
        if player:level() >= 49 and qs2:get_state() == State.COMPLETED then
          htmltext = "32046-01.htm"
        else
          htmltext = "32046-00.htm"
          qs:exit_quest(true)
        end
      else
        htmltext = "32046-00.htm"
        qs:exit_quest(true)
      end
    elseif cond == 1 then
      htmltext = "32046-06.htm"
    elseif cond == 2 then
      htmltext = "32046-09.htm"
    elseif cond == 5 then
      htmltext = "32046-10.htm"
    elseif cond == 6 then
      htmltext = "32046-13.htm"
    elseif cond == 9 then
      htmltext = "32046-14.htm"
    elseif cond == 10 then
      htmltext = "32046-23.htm"
    elseif cond == 12 then
      htmltext = "32046-26.htm"
    elseif cond == 13 then
      htmltext = "32046-30.htm"
    elseif cond == 19 then
      htmltext = "32046-31.htm"
    elseif cond == 20 then
      htmltext = "32046-36.htm"
    elseif cond == 22 then
      htmltext = "32046-37.htm"
    elseif cond == 23 then
      htmltext = "32046-39.htm"
    end
  elseif npcId == WENDY then
    if cond >= 2 and cond <= 4 then
      htmltext = "32047-01.htm"
    elseif cond == 5 then
      htmltext = "32047-07.htm"
    elseif cond == 6 then
      htmltext = "32047-08.htm"
    elseif cond == 7 then
      htmltext = "32047-11.htm"
    elseif cond == 8 then
      htmltext = "32047-12.htm"
    elseif cond == 9 then
      htmltext = "32047-15.htm"
    elseif cond == 13 then
      htmltext = "32047-16.htm"
    elseif cond == 14 then
      htmltext = "32047-19.htm"
    elseif cond == 15 then
      htmltext = "32047-20.htm"
    elseif cond == 23 then
      htmltext = "32047-21.htm"
    elseif cond == 24 then
      htmltext = "32047-26.htm"
    elseif cond == 25 then
      htmltext = "32047-33.htm"
    end
  elseif npcId == YUMI then
    if cond == 2 then
      htmltext = "32041-01.htm"
    elseif cond == 3 then
      htmltext = "32041-05.htm"
    elseif cond == 4 then
      htmltext = "32041-06.htm"
    elseif cond == 7 then
      htmltext = "32041-07.htm"
    elseif cond == 8 then
      htmltext = "32041-13.htm"
    elseif cond == 15 then
      htmltext = "32041-14.htm"
    elseif cond == 16 then
      if q:get_quest_items_count(player, REPORT2) == 0 then
        htmltext = "32041-17.htm"
      else
        htmltext = "32041-18.htm"
      end
    elseif cond == 17 then
      htmltext = "32041-22.htm"
    elseif cond == 25 then
      htmltext = "32041-26.htm"
    end
  elseif npcId == WEATHER1 then
    if cond == 10 then
      htmltext = "32042-01.htm"
    elseif cond == 11 then
      if qs:get_int("talk") + qs:get_int("talk1") + qs:get_int("talk2") == 3 then
        htmltext = "32042-14.htm"
      else
        htmltext = "32042-06.htm"
      end
    elseif cond == 12 then
      htmltext = "32042-15.htm"
    end
  elseif npcId == WEATHER2 then
    if cond == 17 then
      htmltext = "32043-01.htm"
    elseif cond == 18 then
      if qs:get_int("talk") + qs:get_int("talk1") == 2 then
        htmltext = "32043-29.htm"
      else
        htmltext = "32043-06.htm"
      end
    elseif cond == 19 then
      htmltext = "32043-30.htm"
    end
  elseif npcId == WEATHER3 then
    if cond == 20 then
      htmltext = "32044-01.htm"
    elseif cond == 21 then
      htmltext = "32044-06.htm"
    elseif cond > 21 then
      htmltext = "32044-18.htm"
    end
  elseif npcId == BOOKSHELF then
    if cond == 14 then
      htmltext = "32045-01.htm"
    elseif cond == 15 then
      htmltext = "32045-03.htm"
    end
  end
  return htmltext
end)

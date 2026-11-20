-- Translated by questport from quests/Q00115_TheOtherSideOfTruth (Mobius CT_0 Interlude); owned by L2Go from here on.
-- @author Mobius
-- @note Based on python script
local J = require("lib.java")

local q = quest.new(115, "Q00115_TheOtherSideOfTruth", "The Other Side of Truth")

-- NPCs
local MISA = 32018
local SUSPICIOUS = 32019
local RAFFORTY = 32020
local SCULPTURE1 = 32021
local KIERRE = 32022
local SCULPTURE2 = 32077
local SCULPTURE3 = 32078
local SCULPTURE4 = 32079
-- Items
local LETTER = 8079
local LETTER2 = 8080
local TABLET = 8081
local REPORT = 8082

q:add_start_npc(RAFFORTY)
q:add_talk_id(RAFFORTY, MISA, SCULPTURE1, SCULPTURE2, SCULPTURE3, SCULPTURE4, KIERRE)
q:register_quest_items(LETTER, LETTER2, TABLET, REPORT)

q:on_event(function(event, npc, player)
  local htmltext = event
  local qs = player:get_quest_state(q:name())
  if qs == nil then
    return htmltext
  end
  if event == "32018-04.htm" then
    qs:set_cond(7, true)
    q:take_items(player, LETTER2, 1)
  elseif event == "32020-02.htm" then
    qs:start_quest()
  elseif event == "32020-05.htm" then
    qs:set_cond(3, true)
    q:take_items(player, LETTER, 1)
  elseif event == "32020-06.htm" or event == "32020-08a.htm" then
    qs:exit_quest(true, true)
  elseif event == "32020-08.htm" or event == "32020-07a.htm" then
    qs:set_cond(4, true)
  elseif event == "32020-12.htm" then
    qs:set_cond(5, true)
  elseif event == "32020-16.htm" then
    qs:set_cond(10, true)
    q:take_items(player, REPORT, 1)
  elseif event == "32020-18.htm" then
    if q:get_quest_items_count(player, TABLET) == 0 then
      qs:set_cond(11, true)
      htmltext = "32020-19.htm"
    else
      qs:exit_quest(false, true)
      q:give_adena(player, 115673, true)
      q:add_exp_and_sp(player, 493595, 40442)
    end
  elseif event == "32020-19.htm" then
    qs:set_cond(11, true)
  elseif event == "32022-02.htm" then
    qs:set_cond(9, true)
    local man = q:add_spawn(SUSPICIOUS, {x = 104562, y = -107598, z = -3688, heading = 0}, false, 4000)
    man:broadcast_say(ChatType.GENERAL, "We meet again.")
    q:start_quest_timer("2", 3700, man, player)
    q:give_items(player, REPORT, 1)
  elseif event == "Sculpture-04.htm" then
    qs:set("talk", "1")
    htmltext = "Sculpture-05.htm"
    qs:set(tostring(npc:id()), "1")
  elseif event == "Sculpture-04a.htm" then
    qs:set_cond(8, true)
    local man = q:add_spawn(SUSPICIOUS, {x = 117890, y = -126478, z = -2584, heading = 0}, false, 4000)
    man:broadcast_say(ChatType.GENERAL, "This looks like the right place...")
    q:start_quest_timer("1", 3700, man, player)
    htmltext = "Sculpture-04.htm"
    if qs:get_int(tostring(SCULPTURE1)) == 0 and qs:get_int(tostring(SCULPTURE2)) == 0 then
      q:give_items(player, TABLET, 1)
    end
  elseif event == "Sculpture-05.htm" then
    qs:set(tostring(npc:id()), "1")
  elseif event == "1" then
    npc:broadcast_say(ChatType.GENERAL, "I see someone. Is this fate?")
  elseif event == "2" then
    npc:broadcast_say(ChatType.GENERAL, "Don't bother trying to find out more about me. Follow your own destiny.")
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local state = qs:get_state()
  local npcId = npc:id()
  local cond = qs:get_cond()
  if state == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  elseif npcId == RAFFORTY then
    if state == State.CREATED then
      if qs:get_player():level() >= 53 then
        htmltext = "32020-01.htm"
      else
        htmltext = "32020-00.htm"
        qs:exit_quest(true)
      end
    elseif cond == 1 then
      htmltext = "32020-03.htm"
    elseif cond == 2 then
      htmltext = "32020-04.htm"
    elseif cond == 3 then
      htmltext = "32020-05.htm"
    elseif cond == 4 then
      htmltext = "32020-11.htm"
    elseif cond == 5 then
      htmltext = "32020-13.htm"
      q:give_items(player, LETTER2, 1)
      qs:set_cond(6, true)
    elseif cond == 6 then
      htmltext = "32020-14.htm"
    elseif cond == 9 then
      htmltext = "32020-15.htm"
    elseif cond == 10 then
      htmltext = "32020-17.htm"
    elseif cond == 11 then
      htmltext = "32020-20.htm"
    elseif cond == 12 then
      htmltext = "32020-18.htm"
      qs:exit_quest(false, true)
      q:give_adena(player, 60044, true)
    end
  elseif npcId == MISA then
    if cond == 1 then
      htmltext = "32018-01.htm"
      q:give_items(player, LETTER, 1)
      qs:set_cond(2, true)
    elseif cond == 2 then
      htmltext = "32018-02.htm"
    elseif cond == 6 then
      htmltext = "32018-03.htm"
    elseif cond == 7 then
      htmltext = "32018-05.htm"
    end
  elseif npcId == SCULPTURE1 then
    if cond == 7 then
      if qs:get_int(tostring(npcId)) == 1 then
        htmltext = "Sculpture-02.htm"
      elseif qs:get_int("talk") == 1 then
        htmltext = "Sculpture-06.htm"
      else
        htmltext = "Sculpture-03.htm"
      end
    elseif cond == 8 then
      htmltext = "Sculpture-04.htm"
    elseif cond == 11 then
      q:give_items(player, TABLET, 1)
      qs:set_cond(12, true)
      htmltext = "Sculpture-07.htm"
    elseif cond == 12 then
      htmltext = "Sculpture-08.htm"
    end
  elseif npcId == SCULPTURE2 then
    if cond == 7 then
      if qs:get_int(tostring(npcId)) == 1 then
        htmltext = "Sculpture-02.htm"
      elseif qs:get_int("talk") == 1 then
        htmltext = "Sculpture-06.htm"
      else
        htmltext = "Sculpture-03.htm"
      end
    elseif cond == 8 then
      htmltext = "Sculpture-04.htm"
    elseif cond == 11 then
      q:give_items(player, TABLET, 1)
      qs:set_cond(12, true)
      htmltext = "Sculpture-07.htm"
    elseif cond == 12 then
      htmltext = "Sculpture-08.htm"
    end
  elseif npcId == SCULPTURE3 then
    if cond == 7 then
      if qs:get_int(tostring(npcId)) == 1 then
        htmltext = "Sculpture-02.htm"
      else
        htmltext = "Sculpture-01.htm"
        qs:set(tostring(npcId), "1")
      end
    elseif cond == 8 then
      htmltext = "Sculpture-04.htm"
    elseif cond == 11 then
      q:give_items(player, TABLET, 1)
      qs:set_cond(12, true)
      htmltext = "Sculpture-07.htm"
    elseif cond == 12 then
      htmltext = "Sculpture-08.htm"
    end
  elseif npcId == SCULPTURE4 then
    if cond == 7 then
      if qs:get_int(tostring(npcId)) == 1 then
        htmltext = "Sculpture-02.htm"
      else
        htmltext = "Sculpture-01.htm"
        qs:set(tostring(npcId), "1")
      end
    elseif cond == 8 then
      htmltext = "Sculpture-04.htm"
    elseif cond == 11 then
      q:give_items(player, TABLET, 1)
      qs:set_cond(12, true)
      htmltext = "Sculpture-07.htm"
    elseif cond == 12 then
      htmltext = "Sculpture-08.htm"
    end
  elseif npcId == KIERRE then
    if cond == 8 then
      htmltext = "32022-01.htm"
    elseif cond == 9 then
      htmltext = "32022-03.htm"
    end
  end
  return htmltext
end)

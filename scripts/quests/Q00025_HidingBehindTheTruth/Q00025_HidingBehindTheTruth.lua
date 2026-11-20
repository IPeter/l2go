-- Translated by questport from quests/Q00025_HidingBehindTheTruth (Mobius CT_0 Interlude); owned by L2Go from here on.
-- @author Mobius
-- @note Based on python script
local J = require("lib.java")

local q = quest.new(25, "Q00025_HidingBehindTheTruth", "Hiding Behind the Truth")

-- NPCs
local AGRIPEL = 31348
local BENEDICT = 31349
local WIZARD = 31522
local TOMBSTONE = 31531
local LIDIA = 31532
local BOOKSHELF = 31533
local BOOKSHELF2 = 31534
local BOOKSHELF3 = 31535
local COFFIN = 31536
local TRIOL = 27218
-- Items
local CONTRACT = 7066
local DRESS = 7155
local SUSPICIOUS_TOTEM = 7156
local GEMSTONE_KEY = 7157
local TOTEM_DOLL = 7158

q:add_start_npc(BENEDICT)
q:add_talk_id(AGRIPEL, BENEDICT, BOOKSHELF, BOOKSHELF2, BOOKSHELF3, WIZARD, LIDIA, TOMBSTONE, COFFIN)
q:add_kill_id(TRIOL)
q:register_quest_items(SUSPICIOUS_TOTEM, GEMSTONE_KEY, TOTEM_DOLL, DRESS)

q:on_event(function(event, npc, player)
  local htmltext = event
  local qs = player:get_quest_state(q:name())
  if qs == nil then
    return htmltext
  end
  if event == "31349-02.htm" then
    qs:start_quest()
  elseif event == "31349-03.htm" then
    if q:get_quest_items_count(player, SUSPICIOUS_TOTEM) > 0 then
      htmltext = "31349-05.htm"
    else
      qs:set_cond(2, true)
    end
  elseif event == "31349-10.htm" then
    qs:set_cond(4, true)
  elseif event == "31348-02.htm" then
    q:take_items(player, SUSPICIOUS_TOTEM, -1)
  elseif event == "31348-07.htm" then
    qs:set_cond(5, true)
    q:give_items(player, GEMSTONE_KEY, 1)
  elseif event == "31522-04.htm" then
    qs:set_cond(6, true)
  elseif event == "31535-03.htm" then
    if qs:get_int("step") == 0 then
      qs:set("step", "1")
      local triol = q:add_spawn(TRIOL, {x = 59712, y = -47568, z = -2712, heading = -1}, false, 300000)
      triol:broadcast_say(ChatType.GENERAL, "That box was sealed by my master. Don't touch it!")
      triol:set_running()
      triol:add_damage_hate(player, 0, 999)
      triol:attack(player)
      qs:set_cond(7, true)
    elseif qs:get_int("step") == 2 then
      htmltext = "31535-04.htm"
    end
  elseif event == "31535-05.htm" then
    q:give_items(player, CONTRACT, 1)
    q:take_items(player, GEMSTONE_KEY, -1)
    qs:set_cond(9, true)
  elseif event == "31532-02.htm" then
    q:take_items(player, CONTRACT, -1)
  elseif event == "31532-06.htm" then
    qs:set_cond(11, true)
  elseif event == "31531-02.htm" then
    qs:set_cond(12, true)
    q:add_spawn(COFFIN, {x = 60104, y = -35820, z = -664, heading = -1}, false, 20000)
  elseif event == "31532-18.htm" then
    qs:set_cond(15, true)
  elseif event == "31522-12.htm" then
    qs:set_cond(16, true)
  elseif event == "31348-10.htm" then
    q:take_items(player, TOTEM_DOLL, -1)
  elseif event == "31348-15.htm" then
    qs:set_cond(17, true)
  elseif event == "31348-16.htm" then
    qs:set_cond(18, true)
  elseif event == "31532-20.htm" then
    q:give_items(player, 905, 2)
    q:give_items(player, 874, 1)
    q:take_items(player, 7063, -1)
    q:add_exp_and_sp(player, 572277, 53750)
    qs:unset("cond")
    qs:exit_quest(true, true)
  elseif event == "31522-15.htm" then
    q:give_items(player, 936, 1)
    q:give_items(player, 874, 1)
    q:take_items(player, 7063, -1)
    q:add_exp_and_sp(player, 572277, 53750)
    qs:unset("cond")
    qs:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local qs = q:get_quest_state(player, true)
  local npcId = npc:id()
  local id = qs:get_state()
  local cond = qs:get_cond()
  if id == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  elseif id == State.CREATED then
    if npcId == BENEDICT then
      local qs2 = player:get_quest_state("Q00024_InhabitantsOfTheForestOfTheDead")
      if qs2 ~= nil then
        if qs2:get_state() == State.COMPLETED and player:level() >= 66 then
          htmltext = "31349-01.htm"
        else
          htmltext = "31349-00.htm"
        end
      end
    end
  elseif id == State.STARTED then
    if npcId == BENEDICT then
      if cond == 1 then
        htmltext = "31349-02.htm"
      elseif cond == 2 or cond == 3 then
        htmltext = "31349-04.htm"
      elseif cond == 4 then
        htmltext = "31349-10.htm"
      end
    elseif npcId == WIZARD then
      if cond == 2 then
        htmltext = "31522-01.htm"
        qs:set_cond(3, true)
        q:give_items(player, SUSPICIOUS_TOTEM, 1)
      elseif cond == 3 then
        htmltext = "31522-02.htm"
      elseif cond == 5 then
        htmltext = "31522-03.htm"
      elseif cond == 6 then
        htmltext = "31522-04.htm"
      elseif cond == 9 then
        htmltext = "31522-05.htm"
        qs:set_cond(10, true)
      elseif cond == 10 then
        htmltext = "31522-05.htm"
      elseif cond == 15 then
        htmltext = "31522-06.htm"
      elseif cond == 16 then
        htmltext = "31522-13.htm"
      elseif cond == 17 then
        htmltext = "31522-16.htm"
      elseif cond == 18 then
        htmltext = "31522-14.htm"
      end
    elseif npcId == AGRIPEL then
      if cond == 4 then
        htmltext = "31348-01.htm"
      elseif cond == 5 then
        htmltext = "31348-08.htm"
      elseif cond == 16 then
        htmltext = "31348-09.htm"
      elseif cond == 17 then
        htmltext = "31348-17.htm"
      elseif cond == 18 then
        htmltext = "31348-18.htm"
      end
    elseif npcId == BOOKSHELF then
      if cond == 6 then
        htmltext = "31533-01.htm"
      end
    elseif npcId == BOOKSHELF2 then
      if cond == 6 then
        htmltext = "31534-01.htm"
      end
    elseif npcId == BOOKSHELF3 then
      if cond >= 6 and cond <= 8 then
        htmltext = "31535-01.htm"
      elseif cond == 9 then
        htmltext = "31535-06.htm"
      end
    elseif npcId == LIDIA then
      if cond == 10 then
        htmltext = "31532-01.htm"
      elseif cond == 11 or cond == 12 then
        htmltext = "31532-06.htm"
      elseif cond == 13 then
        htmltext = "31532-07.htm"
        qs:set_cond(14)
        q:take_items(player, DRESS, -1)
      elseif cond == 14 then
        htmltext = "31532-08.htm"
      elseif cond == 15 then
        htmltext = "31532-18.htm"
      elseif cond == 17 then
        htmltext = "31532-19.htm"
      elseif cond == 18 then
        htmltext = "31532-21.htm"
      end
    elseif npcId == TOMBSTONE then
      if cond == 11 or cond == 12 then
        htmltext = "31531-01.htm"
      elseif cond == 13 then
        htmltext = "31531-03.htm"
      end
    elseif npcId == COFFIN then
      if cond == 12 then
        htmltext = "31536-01.htm"
        q:give_items(player, DRESS, 1)
        qs:set_cond(13, true)
        npc:delete()
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local qs = q:get_quest_state(player, false)
  if qs == nil or not qs:is_started() then
    return
  end
  if qs:is_cond(7) then
    qs:set_cond(8, true)
    npc:broadcast_say(ChatType.GENERAL, "You've ended my immortal life! You've protected by the feudal lord, aren't you?")
    q:give_items(player, TOTEM_DOLL, 1)
    qs:set("step", "2")
  end
end)

-- Translated by questport from quests/Q00114_ResurrectionOfAnOldManager (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(114, "Q00114_ResurrectionOfAnOldManager", "Resurrection of an Old Manager")

-- NPCs
local NEWYEAR = 31961
local YUMI = 32041
local STONE = 32046
local WENDY = 32047
local BOX = 32050
-- Monsters
local GOLEM = 27318
-- Items
local LETTER = 8288
local DETECTOR = 8090
local DETECTOR_2 = 8091
local STARSTONE = 8287
local STARSTONE_2 = 8289

q:register_quest_items(LETTER, DETECTOR, DETECTOR_2, STARSTONE, STARSTONE_2)
q:add_start_npc(YUMI)
q:add_talk_id(YUMI, WENDY, BOX, STONE, NEWYEAR)
q:add_kill_id(GOLEM)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32041-02.htm" then
    st:start_quest()
    st:set("golemSpawned", "0")
  elseif event == "32041-06.htm" then
    st:set("talk", "1")
  elseif event == "32041-07.htm" then
    st:set_cond(2, true)
    st:set("talk", "0")
  elseif event == "32041-10.htm" then
    local choice = st:get_int("choice")
    if choice == 1 then
      htmltext = "32041-10.htm"
    elseif choice == 2 then
      htmltext = "32041-10a.htm"
    elseif choice == 3 then
      htmltext = "32041-10b.htm"
    end
  elseif event == "32041-11.htm" then
    st:set("talk", "1")
  elseif event == "32041-18.htm" then
    st:set("talk", "2")
  elseif event == "32041-20.htm" then
    st:set_cond(6, true)
    st:set("talk", "0")
  elseif event == "32041-25.htm" then
    st:set_cond(17, true)
    q:give_items(player, DETECTOR, 1)
  elseif event == "32041-28.htm" then
    st:set("talk", "1")
    q:take_items(player, DETECTOR_2, 1)
  elseif event == "32041-31.htm" then
    if st:get_int("choice") > 1 then
      htmltext = "32041-37.htm"
    end
  elseif event == "32041-32.htm" then
    st:set_cond(21, true)
    q:give_items(player, LETTER, 1)
  elseif event == "32041-36.htm" then
    st:set_cond(20, true)
  elseif event == "32046-02.htm" then
    st:set_cond(19, true)
  elseif event == "32046-06.htm" then
    st:exit_quest(false, true)
  elseif event == "32047-01.htm" then
    local talk = st:get_int("talk")
    local talk1 = st:get_int("talk1")
    if talk == 1 and talk1 == 1 then
      htmltext = "32047-04.htm"
    elseif talk == 2 and talk1 == 2 and st:get_int("talk2") == 2 then
      htmltext = "32047-08.htm"
    end
  elseif event == "32047-02.htm" then
    if st:get_int("talk") == 0 then
      st:set("talk", "1")
    end
  elseif event == "32047-03.htm" then
    if st:get_int("talk1") == 0 then
      st:set("talk1", "1")
    end
  elseif event == "32047-05.htm" then
    st:set_cond(3, true)
    st:set("talk", "0")
    st:set("choice", "1")
    st:unset("talk1")
  elseif event == "32047-06.htm" then
    st:set_cond(4, true)
    st:set("talk", "0")
    st:set("choice", "2")
    st:unset("talk1")
  elseif event == "32047-07.htm" then
    st:set_cond(5, true)
    st:set("talk", "0")
    st:set("choice", "3")
    st:unset("talk1")
  elseif event == "32047-13.htm" then
    st:set_cond(7, true)
  elseif event == "32047-13a.htm" then
    st:set_cond(10, true)
  elseif event == "32047-15.htm" then
    if st:get_int("talk") == 0 then
      st:set("talk", "1")
    end
  elseif event == "32047-15a.htm" then
    if st:get_int("golemSpawned") == 0 then
      local golem = q:add_spawn(GOLEM, {x = 96977, y = -110625, z = -3322, heading = 0}, true, 0)
      golem:broadcast_say(ChatType.GENERAL, "You, " .. player:name() .. ", you attacked Wendy. Prepare to die!")
      -- Mobius also makes the golem attack the player (addDamageHate(player,
      -- 0, 999) and an attack intention); the Lua API has no NPC attack yet.
      st:set("golemSpawned", "1")
      q:start_quest_timer("golemDespawn", 900000, golem, player, false)
    else
      htmltext = "32047-19a.htm"
    end
  elseif event == "32047-17a.htm" then
    st:set_cond(12, true)
  elseif event == "32047-20.htm" then
    st:set("talk", "2")
  elseif event == "32047-23.htm" then
    st:set_cond(13, true)
    st:set("talk", "0")
  elseif event == "32047-25.htm" then
    st:set_cond(15, true)
    q:take_items(player, STARSTONE, 1)
  elseif event == "32047-30.htm" then
    st:set("talk", "2")
  elseif event == "32047-33.htm" then
    local cond = st:get_cond()
    if cond == 7 then
      st:set_cond(8, true)
      st:set("talk", "0")
    elseif cond == 8 then
      st:set_cond(9, true)
      htmltext = "32047-34.htm"
    end
  elseif event == "32047-34.htm" then
    st:set_cond(9, true)
  elseif event == "32047-38.htm" then
    if q:get_quest_items_count(player, 57) >= 3000 then
      st:set_cond(26, true)
      q:take_items(player, 57, 3000)
      q:give_items(player, STARSTONE_2, 1)
    else
      htmltext = "32047-39.htm"
    end
  elseif event == "32050-02.htm" then
    st:set("talk", "1")
    q:play_sound(player, QuestSound.ITEMSOUND_ARMOR_WOOD)
  elseif event == "32050-04.htm" then
    st:set_cond(14, true)
    st:set("talk", "0")
    q:give_items(player, STARSTONE, 1)
  elseif event == "31961-02.htm" then
    st:set_cond(22, true)
    q:take_items(player, LETTER, 1)
    q:give_items(player, STARSTONE_2, 1)
  elseif event == "golemDespawn" then
    st:unset("golemSpawned")
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    local pavelReq = player:get_quest_state("Q00121_PavelTheGiant")
    htmltext = (pavelReq == nil or not pavelReq:is_completed() or player:level() < 49) and "32041-00.htm" or "32041-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local talk = st:get_int("talk")
    local sw2 = npc:id()
    if sw2 == YUMI then
      if cond == 1 then
        if talk == 0 then
          htmltext = "32041-02.htm"
        else
          htmltext = "32041-06.htm"
        end
      elseif cond == 2 then
        htmltext = "32041-08.htm"
      elseif cond > 2 and cond < 6 then
        if talk == 0 then
          htmltext = "32041-09.htm"
        elseif talk == 1 then
          htmltext = "32041-11.htm"
        else
          htmltext = "32041-18.htm"
        end
      elseif cond == 6 then
        htmltext = "32041-21.htm"
      elseif cond == 9 or cond == 12 or cond == 16 then
        htmltext = "32041-22.htm"
      elseif cond == 17 then
        htmltext = "32041-26.htm"
      elseif cond == 19 then
        if talk == 0 then
          htmltext = "32041-27.htm"
        else
          htmltext = "32041-28.htm"
        end
      elseif cond == 20 then
        htmltext = "32041-36.htm"
      elseif cond == 21 then
        htmltext = "32041-33.htm"
      elseif cond == 22 or cond == 26 then
        htmltext = "32041-34.htm"
        st:set_cond(27, true)
      elseif cond == 27 then
        htmltext = "32041-35.htm"
      end
    elseif sw2 == WENDY then
      if cond == 2 then
        if talk == 0 and st:get_int("talk1") == 0 then
          htmltext = "32047-01.htm"
        elseif talk == 1 and st:get_int("talk1") == 1 then
          htmltext = "32047-04.htm"
        end
      elseif cond == 3 then
        htmltext = "32047-09.htm"
      elseif cond == 4 or cond == 5 then
        htmltext = "32047-09a.htm"
      elseif cond == 6 then
        local choice = st:get_int("choice")
        if choice == 1 then
          if talk == 0 then
            htmltext = "32047-10.htm"
          elseif talk == 1 then
            htmltext = "32047-20.htm"
          end
        elseif choice == 2 then
          htmltext = "32047-10a.htm"
        elseif choice == 3 then
          if talk == 0 then
            htmltext = "32047-14.htm"
          elseif talk == 1 then
            htmltext = "32047-15.htm"
          else
            htmltext = "32047-20.htm"
          end
        end
      elseif cond == 7 then
        if talk == 0 then
          htmltext = "32047-14.htm"
        elseif talk == 1 then
          htmltext = "32047-15.htm"
        else
          htmltext = "32047-20.htm"
        end
      elseif cond == 8 then
        htmltext = "32047-30.htm"
      elseif cond == 9 then
        htmltext = "32047-27.htm"
      elseif cond == 10 then
        htmltext = "32047-14a.htm"
      elseif cond == 11 then
        htmltext = "32047-16a.htm"
      elseif cond == 12 then
        htmltext = "32047-18a.htm"
      elseif cond == 13 then
        htmltext = "32047-23.htm"
      elseif cond == 14 then
        htmltext = "32047-24.htm"
      elseif cond == 15 then
        htmltext = "32047-26.htm"
        st:set_cond(16, true)
      elseif cond == 16 then
        htmltext = "32047-27.htm"
      elseif cond == 20 then
        htmltext = "32047-35.htm"
      elseif cond == 26 then
        htmltext = "32047-40.htm"
      end
    elseif sw2 == BOX then
      if cond == 13 then
        if talk == 0 then
          htmltext = "32050-01.htm"
        else
          htmltext = "32050-03.htm"
        end
      elseif cond == 14 then
        htmltext = "32050-05.htm"
      end
    elseif sw2 == STONE then
      if cond == 17 then
        st:set_cond(18, true)
        q:take_items(player, DETECTOR, 1)
        q:give_items(player, DETECTOR_2, 1)
        q:show_on_screen_msg(player, "The radio signal detector is responding. # A suspicious pile of stones catches your eye.", 4500)
        return nil
      elseif cond == 18 then
        htmltext = "32046-01.htm"
      elseif cond == 19 then
        htmltext = "32046-02.htm"
      elseif cond == 27 then
        htmltext = "32046-03.htm"
      end
    elseif sw2 == NEWYEAR then
      if cond == 21 then
        htmltext = "31961-01.htm"
      elseif cond == 22 then
        htmltext = "31961-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(10) then
    return
  end
  npc:broadcast_say(ChatType.GENERAL, "This enemy is far too powerful for me to fight. I must withdraw!")
  st:set_cond(11, true)
  st:unset("golemSpawned")
end)

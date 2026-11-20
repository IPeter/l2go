-- Translated by questport from quests/Q00343_UnderTheShadowOfTheIvoryTower (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Adapted from FirstTeam Interlude
local J = require("lib.java")

local q = quest.new(343, "Q00343_UnderTheShadowOfTheIvoryTower", "Under the Shadow of the Ivory Tower")

-- NPCs
local CEMA = 30834
local ICARUS = 30835
local MARSHA = 30934
local TRUMPIN = 30935
local MONSTERS = {20563, 20564, 20565, 20566}
-- Items
local ORB = 4364
local ECTOPLASM = 4365
-- Misc
local CHANCE = 50
local ALLOWED_CLASSES = {11, 12, 13, 14, 26, 27, 28, 39, 40, 41}

q:add_start_npc(CEMA)
q:add_talk_id(CEMA, ICARUS, MARSHA, TRUMPIN)
q:add_kill_id(MONSTERS)
q:register_quest_items(ORB)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  local random1 = rnd(3)
  local random2 = rnd(2)
  local orbs = q:get_quest_items_count(player, ORB)
  if event == "30834-03.htm" then
    st:start_quest()
  elseif event == "30834-08.htm" then
    if orbs > 0 then
      q:give_adena(player, orbs * 125, true)
      q:take_items(player, ORB, -1)
    else
      htmltext = "30834-08.htm"
    end
  elseif event == "30834-09.htm" then
    st:exit_quest(true, true)
  elseif event == "30934-02.htm" or event == "30934-03.htm" then
    if orbs < 10 then
      htmltext = "noorbs.htm"
    elseif "30934-03.htm" == event then
      if orbs >= 10 then
        q:take_items(player, ORB, 10)
        st:set("playing", "1")
      else
        htmltext = "noorbs.htm"
      end
    end
  elseif event == "30934-04.htm" then
    if st:get_int("playing") > 0 then
      if random1 == 0 then
        htmltext = "30934-05.htm"
        q:give_items(player, ORB, 10)
      elseif random1 == 1 then
        htmltext = "30934-06.htm"
      else
        htmltext = "30934-04.htm"
        q:give_items(player, ORB, 20)
      end
      st:unset("playing")
    else
      htmltext = "Player is cheating"
      q:take_items(player, ORB, -1)
      st:exit_quest(true)
    end
  elseif event == "30934-05.htm" then
    if st:get_int("playing") > 0 then
      if random1 == 0 then
        htmltext = "30934-04.htm"
        q:give_items(player, ORB, 20)
      elseif random1 == 1 then
        htmltext = "30934-05.htm"
        q:give_items(player, ORB, 10)
      else
        htmltext = "30934-06.htm"
      end
      st:unset("playing")
    else
      htmltext = "Player is cheating"
      q:take_items(player, ORB, -1)
      st:exit_quest(true)
    end
  elseif event == "30934-06.htm" then
    if st:get_int("playing") > 0 then
      if random1 == 0 then
        htmltext = "30934-04.htm"
        q:give_items(player, ORB, 20)
      elseif random1 == 1 then
        htmltext = "30934-06.htm"
      else
        htmltext = "30934-05.htm"
        q:give_items(player, ORB, 10)
      end
      st:unset("playing")
    else
      htmltext = "Player is cheating"
      q:take_items(player, ORB, -1)
      st:exit_quest(true)
    end
  elseif event == "30935-02.htm" or event == "30935-03.htm" then
    st:unset("toss")
    if orbs < 10 then
      htmltext = "noorbs.htm"
    end
  elseif event == "30935-05.htm" then
    if orbs >= 10 then
      if random2 == 0 then
        local toss = st:get_int("toss")
        if toss == 4 then
          st:unset("toss")
          q:give_items(player, ORB, 150)
          htmltext = "30935-07.htm"
        else
          st:set("toss", tostring(toss + 1))
          htmltext = "30935-04.htm"
        end
      else
        st:unset("toss")
        q:take_items(player, ORB, 10)
      end
    else
      htmltext = "noorbs.htm"
    end
  elseif event == "30935-06.htm" then
    if orbs >= 10 then
      local toss = st:get_int("toss")
      st:unset("toss")
      if toss == 1 then
        q:give_items(player, ORB, 10)
      elseif toss == 2 then
        q:give_items(player, ORB, 30)
      elseif toss == 3 then
        q:give_items(player, ORB, 70)
      elseif toss == 4 then
        q:give_items(player, ORB, 150)
      end
    else
      htmltext = "noorbs.htm"
    end
  elseif event == "30835-02.htm" then
    if q:get_quest_items_count(player, ECTOPLASM) > 0 then
      q:take_items(player, ECTOPLASM, 1)
      local random3 = rnd(1000)
      if random3 <= 119 then
        q:give_items(player, 955, 1)
      elseif random3 <= 169 then
        q:give_items(player, 951, 1)
      elseif random3 <= 329 then
        q:give_items(player, 2511, rnd(200) + 401)
      elseif random3 <= 559 then
        q:give_items(player, 2510, rnd(200) + 401)
      elseif random3 <= 561 then
        q:give_items(player, 316, 1)
      elseif random3 <= 578 then
        q:give_items(player, 630, 1)
      elseif random3 <= 579 then
        q:give_items(player, 188, 1)
      elseif random3 <= 581 then
        q:give_items(player, 885, 1)
      elseif random3 <= 582 then
        q:give_items(player, 103, 1)
      elseif random3 <= 584 then
        q:give_items(player, 917, 1)
      else
        q:give_items(player, 736, 1)
      end
    else
      htmltext = "30835-03.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = npc:id()
  if sw == CEMA then
    if not st:is_started() then
      for _, classId in ipairs(ALLOWED_CLASSES) do
        if st:get_player():class_id() == classId and st:get_player():level() >= 40 then
          htmltext = "30834-01.htm"
        end
      end
      if not ("30834-01.htm" == htmltext) then
        htmltext = "30834-07.htm"
        st:exit_quest(true)
      end
    elseif q:get_quest_items_count(player, ORB) > 0 then
      htmltext = "30834-06.htm"
    else
      htmltext = "30834-05.htm"
    end
  elseif sw == ICARUS then
    htmltext = "30835-01.htm"
  elseif sw == MARSHA then
    htmltext = "30934-01.htm"
  elseif sw == TRUMPIN then
    htmltext = "30935-01.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if rnd(100) < CHANCE then
    q:give_items(player, ORB, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)

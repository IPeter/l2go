-- Translated by questport from quests/Q00347_GoGetTheCalculator (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(347, "Q00347_GoGetTheCalculator", "Go Get the Calculator")

-- NPCs
local BRUNON = 30526
local SILVERA = 30527
local SPIRON = 30532
local BALANKI = 30533
-- Items
local GEMSTONE_BEAST_CRYSTAL = 4286
local CALCULATOR_QUEST = 4285
local CALCULATOR_REAL = 4393

q:register_quest_items(GEMSTONE_BEAST_CRYSTAL, CALCULATOR_QUEST)
q:add_start_npc(BRUNON)
q:add_talk_id(BRUNON, SILVERA, SPIRON, BALANKI)
q:add_kill_id(20540)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30526-05.htm" then
    st:start_quest()
  elseif event == "30533-03.htm" then
    if q:get_quest_items_count(player, 57) >= 100 then
      htmltext = "30533-02.htm"
      q:take_items(player, 57, 100)
      if st:is_cond(3) then
        st:set_cond(4, true)
      else
        st:set_cond(2, true)
      end
    end
  elseif event == "30532-02.htm" then
    if st:is_cond(2) then
      st:set_cond(4, true)
    else
      st:set_cond(3, true)
    end
  elseif event == "30526-08.htm" then
    q:take_items(player, CALCULATOR_QUEST, -1)
    q:give_items(player, CALCULATOR_REAL, 1)
    st:exit_quest(true, true)
  elseif event == "30526-09.htm" then
    q:take_items(player, CALCULATOR_QUEST, -1)
    q:give_adena(player, 1000, true)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 12 and "30526-00.htm" or "30526-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == BRUNON then
      htmltext = not q:has_quest_items(player, CALCULATOR_QUEST) and "30526-06.htm" or "30526-07.htm"
    elseif sw2 == SPIRON then
      htmltext = cond < 4 and "30532-01.htm" or "30532-05.htm"
    elseif sw2 == BALANKI then
      htmltext = cond < 4 and "30533-01.htm" or "30533-04.htm"
    elseif sw2 == SILVERA then
      if cond < 4 then
        htmltext = "30527-00.htm"
      elseif cond == 4 then
        htmltext = "30527-01.htm"
        st:set_cond(5, true)
      elseif cond == 5 then
        if q:get_quest_items_count(player, GEMSTONE_BEAST_CRYSTAL) < 10 then
          htmltext = "30527-02.htm"
        else
          htmltext = "30527-03.htm"
          st:set_cond(6, true)
          q:take_items(player, GEMSTONE_BEAST_CRYSTAL, -1)
          q:give_items(player, CALCULATOR_QUEST, 1)
        end
      elseif cond == 6 then
        htmltext = "30527-04.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(5) then
    return
  end
  if rnd.bool() and q:get_quest_items_count(player, GEMSTONE_BEAST_CRYSTAL) < 10 then
    q:give_items(player, GEMSTONE_BEAST_CRYSTAL, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)

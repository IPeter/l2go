-- Translated by questport from quests/Q00126_TheNameOfEvil2 (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(126, "Q00126_TheNameOfEvil2", "The Name of Evil - 2")

local MUSHIKA = 32114
local ASAMANAH = 32115
local ULU_KAIMU = 32119
local BALU_KAIMU = 32120
local CHUTA_KAIMU = 32121
local WARRIOR_GRAVE = 32122
local SHILEN_STONE_STATUE = 32109
local BONEPOWDER = 8783
local EWA = 729

local function getSongOne(st)
  local htmltext = "32122-24.htm"
  if st:is_cond(14) and st:get_int("DO") > 0 and st:get_int("MI") > 0 and st:get_int("FA") > 0 and st:get_int("SOL") > 0 and st:get_int("FA_2") > 0 then
    htmltext = "32122-42.htm"
    st:set_cond(15, true)
    st:unset("DO")
    st:unset("MI")
    st:unset("FA")
    st:unset("SOL")
    st:unset("FA_2")
  end
  return htmltext
end

local function getSongTwo(st)
  local htmltext = "32122-45.htm"
  if st:is_cond(15) and st:get_int("FA") > 0 and st:get_int("SOL") > 0 and st:get_int("TI") > 0 and st:get_int("SOL_2") > 0 and st:get_int("FA_2") > 0 then
    htmltext = "32122-63.htm"
    st:set_cond(16, true)
    st:unset("FA")
    st:unset("SOL")
    st:unset("TI")
    st:unset("SOL_2")
    st:unset("FA3_2") -- sic (Mobius): FA_2 stays set for the third song
  end
  return htmltext
end

local function getSongTri(st)
  local htmltext = "32122-66.htm"
  if st:is_cond(16) and st:get_int("SOL") > 0 and st:get_int("FA") > 0 and st:get_int("MI") > 0 and st:get_int("FA_2") > 0 and st:get_int("MI_2") > 0 then
    htmltext = "32122-84.htm"
    st:set_cond(17, true)
    st:unset("SOL")
    st:unset("FA")
    st:unset("MI")
    st:unset("FA_2")
    st:unset("MI_2")
  end
  return htmltext
end

q:add_start_npc(ASAMANAH)
q:add_talk_id(ASAMANAH, MUSHIKA, ULU_KAIMU, BALU_KAIMU, CHUTA_KAIMU, WARRIOR_GRAVE, SHILEN_STONE_STATUE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32115-05.htm" then
    st:start_quest()
  elseif event == "32115-10.htm" then
    st:set_cond(2, true)
  elseif event == "32119-02.htm" then
    st:set_cond(3, true)
  elseif event == "32119-09.htm" then
    st:set_cond(4, true)
  elseif event == "32119-11.htm" then
    st:set_cond(5, true)
  elseif event == "32120-07.htm" then
    st:set_cond(6, true)
  elseif event == "32120-09.htm" then
    st:set_cond(7, true)
  elseif event == "32120-11.htm" then
    st:set_cond(8, true)
  elseif event == "32121-07.htm" then
    st:set_cond(9, true)
  elseif event == "32121-10.htm" then
    st:set_cond(10, true)
  elseif event == "32121-15.htm" then
    st:set_cond(11, true)
  elseif event == "32122-03.htm" then
    st:set_cond(12, true)
  elseif event == "32122-15.htm" then
    st:set_cond(13, true)
  elseif event == "32122-18.htm" then
    st:set_cond(14, true)
  elseif event == "32122-87.htm" then
    q:give_items(player, BONEPOWDER, 1)
  elseif event == "32122-90.htm" then
    st:set_cond(18, true)
  elseif event == "32109-02.htm" then
    st:set_cond(19, true)
  elseif event == "32109-19.htm" then
    st:set_cond(20, true)
    q:take_items(player, BONEPOWDER, 1)
  elseif event == "32115-21.htm" then
    st:set_cond(21, true)
  elseif event == "32115-28.htm" then
    st:set_cond(22, true)
  elseif event == "32114-08.htm" then
    st:set_cond(23, true)
  elseif event == "32114-09.htm" then
    q:give_items(player, EWA, 1)
    st:exit_quest(false, true)
  elseif event == "DOOne" then
    htmltext = "32122-26.htm"
    if st:get_int("DO") < 1 then
      st:set("DO", "1")
    end
  elseif event == "MIOne" then
    htmltext = "32122-30.htm"
    if st:get_int("MI") < 1 then
      st:set("MI", "1")
    end
  elseif event == "FAOne" then
    htmltext = "32122-34.htm"
    if st:get_int("FA") < 1 then
      st:set("FA", "1")
    end
  elseif event == "SOLOne" then
    htmltext = "32122-38.htm"
    if st:get_int("SOL") < 1 then
      st:set("SOL", "1")
    end
  elseif event == "FA_2One" then
    if st:get_int("FA_2") < 1 then
      st:set("FA_2", "1")
    end
    htmltext = getSongOne(st)
  elseif event == "FATwo" then
    htmltext = "32122-47.htm"
    if st:get_int("FA") < 1 then
      st:set("FA", "1")
    end
  elseif event == "SOLTwo" then
    htmltext = "32122-51.htm"
    if st:get_int("SOL") < 1 then
      st:set("SOL", "1")
    end
  elseif event == "TITwo" then
    htmltext = "32122-55.htm"
    if st:get_int("TI") < 1 then
      st:set("TI", "1")
    end
  elseif event == "SOL_2Two" then
    htmltext = "32122-59.htm"
    if st:get_int("SOL_2") < 1 then
      st:set("SOL_2", "1")
    end
  elseif event == "FA_2Two" then
    if st:get_int("FA_2") < 1 then
      st:set("FA_2", "1")
    end
    htmltext = getSongTwo(st)
  elseif event == "SOLTri" then
    htmltext = "32122-68.htm"
    if st:get_int("SOL") < 1 then
      st:set("SOL", "1")
    end
  elseif event == "FATri" then
    htmltext = "32122-72.htm"
    if st:get_int("FA") < 1 then
      st:set("FA", "1")
    end
  elseif event == "MITri" then
    htmltext = "32122-76.htm"
    if st:get_int("MI") < 1 then
      st:set("MI", "1")
    end
  elseif event == "FA_2Tri" then
    htmltext = "32122-80.htm"
    if st:get_int("FA_2") < 1 then
      st:set("FA_2", "1")
    end
  elseif event == "MI_2Tri" then
    if st:get_int("MI_2") < 1 then
      st:set("MI_2", "1")
    end
    htmltext = getSongTri(st)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() < 77 then
      htmltext = "32115-02.htm"
    else
      local st2 = player:get_quest_state("Q00125_TheNameOfEvil1")
      if st2 ~= nil and st2:is_completed() then
        htmltext = "32115-01.htm"
      else
        htmltext = "32115-04.htm"
      end
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ASAMANAH then
      if cond == 1 then
        htmltext = "32115-11.htm"
        st:set_cond(2, true)
      elseif cond > 1 and cond < 20 then
        htmltext = "32115-12.htm"
      elseif cond == 20 then
        htmltext = "32115-13.htm"
      elseif cond == 21 then
        htmltext = "32115-22.htm"
      elseif cond == 22 then
        htmltext = "32115-29.htm"
      end
    elseif sw2 == ULU_KAIMU then
      if cond == 1 then
        htmltext = "32119-01a.htm"
      elseif cond == 2 then
        htmltext = "32119-02.htm"
      elseif cond == 3 then
        htmltext = "32119-08.htm"
      elseif cond == 4 then
        htmltext = "32119-09.htm"
      elseif cond > 4 then
        htmltext = "32119-12.htm"
      end
    elseif sw2 == BALU_KAIMU then
      if cond < 5 then
        htmltext = "32120-02.htm"
      elseif cond == 5 then
        htmltext = "32120-01.htm"
      elseif cond == 6 then
        htmltext = "32120-03.htm"
      elseif cond == 7 then
        htmltext = "32120-08.htm"
      elseif cond > 7 then
        htmltext = "32120-12.htm"
      end
    elseif sw2 == CHUTA_KAIMU then
      if cond < 8 then
        htmltext = "32121-02.htm"
      elseif cond == 8 then
        htmltext = "32121-01.htm"
      elseif cond == 9 then
        htmltext = "32121-03.htm"
      elseif cond == 10 then
        htmltext = "32121-10.htm"
      elseif cond > 10 then
        htmltext = "32121-16.htm"
      end
    elseif sw2 == WARRIOR_GRAVE then
      if cond < 11 then
        htmltext = "32122-02.htm"
      elseif cond == 11 then
        htmltext = "32122-01.htm"
      elseif cond == 12 then
        htmltext = "32122-15.htm"
      elseif cond == 13 then
        htmltext = "32122-18.htm"
      elseif cond == 14 then
        htmltext = "32122-24.htm"
      elseif cond == 15 then
        htmltext = "32122-45.htm"
      elseif cond == 16 then
        htmltext = "32122-66.htm"
      elseif cond == 17 then
        htmltext = "32122-84.htm"
      elseif cond == 18 then
        htmltext = "32122-91.htm"
      end
    elseif sw2 == SHILEN_STONE_STATUE then
      if cond < 18 then
        htmltext = "32109-03.htm"
      elseif cond == 18 then
        htmltext = "32109-02.htm"
      elseif cond == 19 then
        htmltext = "32109-05.htm"
      elseif cond > 19 then
        htmltext = "32109-04.htm"
      end
    elseif sw2 == MUSHIKA then
      if cond < 22 then
        htmltext = "32114-02.htm"
      elseif cond == 22 then
        htmltext = "32114-01.htm"
      elseif cond == 23 then
        htmltext = "32114-04.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

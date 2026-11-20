-- Translated by questport from quests/Q00409_PathOfTheElvenOracle (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(409, "Q00409_PathOfTheElvenOracle", "Path to an Elven Oracle")

-- NPCs
local MANUEL = 30293
local ALLANA = 30424
local PERRIN = 30428
-- Items
local CRYSTAL_MEDALLION = 1231
local SWINDLER_MONEY = 1232
local ALLANA_DIARY = 1233
local LIZARD_CAPTAIN_ORDER = 1234
local LEAF_OF_ORACLE = 1235
local HALF_OF_DIARY = 1236
local TAMIL_NECKLACE = 1275

q:register_quest_items(CRYSTAL_MEDALLION, SWINDLER_MONEY, ALLANA_DIARY, LIZARD_CAPTAIN_ORDER, HALF_OF_DIARY, TAMIL_NECKLACE)
q:add_start_npc(MANUEL)
q:add_talk_id(MANUEL, ALLANA, PERRIN)
q:add_kill_id(27032, 27033, 27034, 27035)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30293-05.htm" then
    st:start_quest()
    q:give_items(player, CRYSTAL_MEDALLION, 1)
  elseif event == "spawn_lizards" then
    st:set_cond(2, true)
    q:add_spawn(27032, {x = -92319, y = 154235, z = -3284, heading = 2000}, false, 0)
    q:add_spawn(27033, {x = -92361, y = 154190, z = -3284, heading = 2000}, false, 0)
    q:add_spawn(27034, {x = -92375, y = 154278, z = -3278, heading = 2000}, false, 0)
    return nil
  elseif event == "30428-06.htm" then
    q:add_spawn(27035, {x = -93194, y = 147587, z = -2672, heading = 2000}, false, 0)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.ELVEN_MAGE then
      htmltext = player:class_id() == ClassId.ORACLE and "30293-02a.htm" or "30293-02.htm"
    elseif player:level() < 19 then
      htmltext = "30293-03.htm"
    elseif q:has_quest_items(player, LEAF_OF_ORACLE) then
      htmltext = "30293-04.htm"
    else
      htmltext = "30293-01.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == MANUEL then
      if cond == 1 then
        htmltext = "30293-06.htm"
      elseif cond == 2 or cond == 3 then
        htmltext = "30293-09.htm"
      elseif cond > 3 and cond < 7 then
        htmltext = "30293-07.htm"
      elseif cond == 7 then
        htmltext = "30293-08.htm"
        q:take_items(player, ALLANA_DIARY, 1)
        q:take_items(player, CRYSTAL_MEDALLION, 1)
        q:take_items(player, LIZARD_CAPTAIN_ORDER, 1)
        q:take_items(player, SWINDLER_MONEY, 1)
        q:give_items(player, LEAF_OF_ORACLE, 1)
        q:add_exp_and_sp(player, 3200, 1130)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == ALLANA then
      if cond == 1 then
        htmltext = "30424-01.htm"
      elseif cond == 3 then
        htmltext = "30424-02.htm"
        st:set_cond(4, true)
        q:give_items(player, HALF_OF_DIARY, 1)
      elseif cond == 4 then
        htmltext = "30424-03.htm"
      elseif cond == 5 then
        htmltext = "30424-06.htm"
      elseif cond == 6 then
        htmltext = "30424-04.htm"
        st:set_cond(7, true)
        q:take_items(player, HALF_OF_DIARY, -1)
        q:give_items(player, ALLANA_DIARY, 1)
      elseif cond == 7 then
        htmltext = "30424-05.htm"
      end
    elseif sw2 == PERRIN then
      if cond == 4 then
        htmltext = "30428-01.htm"
      elseif cond == 5 then
        htmltext = "30428-04.htm"
        st:set_cond(6, true)
        q:take_items(player, TAMIL_NECKLACE, -1)
        q:give_items(player, SWINDLER_MONEY, 1)
      elseif cond > 5 then
        htmltext = "30428-05.htm"
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
  if npc:id() == 27035 then
    if st:is_cond(4) then
      st:set_cond(5, true)
      q:give_items(player, TAMIL_NECKLACE, 1)
    end
  elseif st:is_cond(2) then
    st:set_cond(3, true)
    q:give_items(player, LIZARD_CAPTAIN_ORDER, 1)
  end
end)

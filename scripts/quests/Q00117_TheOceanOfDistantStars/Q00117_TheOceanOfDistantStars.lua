-- Translated by questport from quests/Q00117_TheOceanOfDistantStars (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(117, "Q00117_TheOceanOfDistantStars", "The Ocean of Distant Stars")

-- NPCs
local ABEY = 32053
local GHOST = 32054
local ANCIENT_GHOST = 32055
local OBI = 32052
local BOX = 32076
-- Monsters
local BANDIT_WARRIOR = 22023
local BANDIT_INSPECTOR = 22024
-- Items
local GREY_STAR = 8495
local ENGRAVED_HAMMER = 8488

q:register_quest_items(GREY_STAR, ENGRAVED_HAMMER)
q:add_start_npc(ABEY)
q:add_talk_id(ABEY, ANCIENT_GHOST, GHOST, OBI, BOX)
q:add_kill_id(BANDIT_WARRIOR, BANDIT_INSPECTOR)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32053-02.htm" then
    st:start_quest()
  elseif event == "32055-02.htm" then
    st:set_cond(2, true)
  elseif event == "32052-02.htm" then
    st:set_cond(3, true)
  elseif event == "32053-04.htm" then
    st:set_cond(4, true)
  elseif event == "32076-02.htm" then
    st:set_cond(5, true)
    q:give_items(player, ENGRAVED_HAMMER, 1)
  elseif event == "32053-06.htm" then
    st:set_cond(6, true)
  elseif event == "32052-04.htm" then
    st:set_cond(7, true)
  elseif event == "32052-06.htm" then
    st:set_cond(9, true)
    q:take_items(player, GREY_STAR, 1)
  elseif event == "32055-04.htm" then
    st:set_cond(10, true)
    q:take_items(player, ENGRAVED_HAMMER, 1)
  elseif event == "32054-03.htm" then
    q:add_exp_and_sp(player, 63591, 0)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 39 and "32053-00.htm" or "32053-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ANCIENT_GHOST then
      if cond == 1 then
        htmltext = "32055-01.htm"
      elseif cond > 1 and cond < 9 then
        htmltext = "32055-02.htm"
      elseif cond == 9 then
        htmltext = "32055-03.htm"
      elseif cond > 9 then
        htmltext = "32055-05.htm"
      end
    elseif sw2 == OBI then
      if cond == 2 then
        htmltext = "32052-01.htm"
      elseif cond > 2 and cond < 6 then
        htmltext = "32052-02.htm"
      elseif cond == 6 then
        htmltext = "32052-03.htm"
      elseif cond == 7 then
        htmltext = "32052-04.htm"
      elseif cond == 8 then
        htmltext = "32052-05.htm"
      elseif cond > 8 then
        htmltext = "32052-06.htm"
      end
    elseif sw2 == ABEY then
      if cond == 1 or cond == 2 then
        htmltext = "32053-02.htm"
      elseif cond == 3 then
        htmltext = "32053-03.htm"
      elseif cond == 4 then
        htmltext = "32053-04.htm"
      elseif cond == 5 then
        htmltext = "32053-05.htm"
      elseif cond > 5 then
        htmltext = "32053-06.htm"
      end
    elseif sw2 == BOX then
      if cond == 4 then
        htmltext = "32076-01.htm"
      elseif cond > 4 then
        htmltext = "32076-03.htm"
      end
    elseif sw2 == GHOST then
      if cond == 10 then
        htmltext = "32054-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_random_party_member_state(killer, 7, 3, npc)
  if qs == nil or not util.in_range(Config.ALT_PARTY_RANGE, npc, killer, true) then
    return
  end
  if q:give_item_randomly(killer, npc, GREY_STAR, 1, 1, 1, 0.2, true) then
    qs:set_cond(8)
  end
end)

-- Translated by questport from quests/Q00613_ProveYourCourageVarka (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Prove Your Courage! (Varka) (613)
-- @author malyelfik
local J = require("lib.java")

local q = quest.new(613, "Q00613_ProveYourCourageVarka", "Prove your courage!")

-- NPC
local ASHAS = 31377
-- Monster
local HEKATON = 25299
-- Items
local HEKATON_HEAD = 7240
local VALOR_FEATHER = 7229
local VARKA_ALLIANCE_THREE = 7223
-- Misc
local MIN_LEVEL = 75

q:add_start_npc(ASHAS)
q:add_talk_id(ASHAS)
q:add_kill_id(HEKATON)
q:register_quest_items(HEKATON_HEAD)

q:on_action_for_each_player(function(player, npc, isSummon)
  local qs = q:get_quest_state(player, false)
  if qs ~= nil and qs:is_cond(1) and util.in_range(Config.ALT_PARTY_RANGE, npc, player, false) then
    q:give_items(player, HEKATON_HEAD, 1)
    qs:set_cond(2, true)
  end
end)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  local htmltext = event
  if event == "31377-04.htm" then
    qs:start_quest()
  elseif event == "31377-07.html" then
    if q:has_quest_items(player, HEKATON_HEAD) and qs:is_cond(2) then
      q:give_items(player, VALOR_FEATHER, 1)
      q:add_exp_and_sp(player, 10000, 0)
      qs:exit_quest(true, true)
    else
      htmltext = q:get_no_quest_msg(player)
    end
  else
    htmltext = nil
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  q:execute_for_each_player(killer, npc, isSummon, true, false)
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = qs:get_state()
  if sw == State.CREATED then
    htmltext = player:level() >= MIN_LEVEL and (q:has_quest_items(player, VARKA_ALLIANCE_THREE) and "31377-01.htm" or "31377-02.htm") or "31377-03.htm"
  elseif sw == State.STARTED then
    htmltext = qs:is_cond(2) and q:has_quest_items(player, HEKATON_HEAD) and "31377-05.html" or "31377-06.html"
  end
  return htmltext
end)

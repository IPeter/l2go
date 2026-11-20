-- Translated by questport from quests/Q00614_SlayTheEnemyCommanderVarka (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Slay the Enemy Commander! (Varka) (614)
-- @author malyelfik
local J = require("lib.java")

local q = quest.new(614, "Q00614_SlayTheEnemyCommanderVarka", "Slay the enemy commander!")

-- NPC
local ASHAS = 31377
-- Monster
local TAYR = 25302
-- Items
local TAYR_HEAD = 7241
local WISDOM_FEATHER = 7230
local VARKA_ALLIANCE_FOUR = 7224
-- Misc
local MIN_LEVEL = 75

q:add_start_npc(ASHAS)
q:add_talk_id(ASHAS)
q:add_kill_id(TAYR)
q:register_quest_items(TAYR_HEAD)

q:on_action_for_each_player(function(player, npc, isSummon)
  local qs = q:get_quest_state(player, false)
  if qs ~= nil and qs:is_cond(1) and util.in_range(Config.ALT_PARTY_RANGE, npc, player, false) then
    q:give_items(player, TAYR_HEAD, 1)
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
    if q:has_quest_items(player, TAYR_HEAD) and qs:is_cond(2) then
      q:give_items(player, WISDOM_FEATHER, 1)
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
    htmltext = player:level() >= MIN_LEVEL and (q:has_quest_items(player, VARKA_ALLIANCE_FOUR) and "31377-01.htm" or "31377-02.htm") or "31377-03.htm"
  elseif sw == State.STARTED then
    htmltext = qs:is_cond(2) and q:has_quest_items(player, TAYR_HEAD) and "31377-05.html" or "31377-06.html"
  end
  return htmltext
end)

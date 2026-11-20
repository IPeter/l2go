-- Translated by questport from quests/Q00608_SlayTheEnemyCommanderKetra (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Slay the Enemy Commander! (Ketra) (608)
-- @author malyelfik
local J = require("lib.java")

local q = quest.new(608, "Q00608_SlayTheEnemyCommanderKetra", "Slay the enemy commander!")

-- NPC
local KADUN = 31370
-- Monster
local MOS = 25312
-- Items
local MOS_HEAD = 7236
local WISDOM_TOTEM = 7220
local KETRA_ALLIANCE_FOUR = 7214
-- Misc
local MIN_LEVEL = 75

q:add_start_npc(KADUN)
q:add_talk_id(KADUN)
q:add_kill_id(MOS)
q:register_quest_items(MOS_HEAD)

q:on_action_for_each_player(function(player, npc, isSummon)
  local qs = q:get_quest_state(player, false)
  if qs ~= nil and qs:is_cond(1) and util.in_range(Config.ALT_PARTY_RANGE, npc, player, false) then
    q:give_items(player, MOS_HEAD, 1)
    qs:set_cond(2, true)
  end
end)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  local htmltext = event
  if event == "31370-04.htm" then
    qs:start_quest()
  elseif event == "31370-07.html" then
    if q:has_quest_items(player, MOS_HEAD) and qs:is_cond(2) then
      q:give_items(player, WISDOM_TOTEM, 1)
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
    htmltext = player:level() >= MIN_LEVEL and (q:has_quest_items(player, KETRA_ALLIANCE_FOUR) and "31370-01.htm" or "31370-02.htm") or "31370-03.htm"
  elseif sw == State.STARTED then
    htmltext = qs:is_cond(2) and q:has_quest_items(player, MOS_HEAD) and "31370-05.html" or "31370-06.html"
  end
  return htmltext
end)

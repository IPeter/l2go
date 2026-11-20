-- Translated by questport from quests/Q00152_ShardsOfGolem (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(152, "Q00152_ShardsOfGolem", "Shards of Golem")

-- NPCs
local HARRIS = 30035
local ALTRAN = 30283
-- Monster
local STONE_GOLEM = 20016
-- Items
local HARRIS_RECEIPT_1 = 1008
local HARRIS_RECEIPT_2 = 1009
local GOLEM_SHARD = 1010
local TOOL_BOX = 1011
-- Reward
local WOODEN_BREASTPLATE = 23

q:register_quest_items(HARRIS_RECEIPT_1, HARRIS_RECEIPT_2, GOLEM_SHARD, TOOL_BOX)
q:add_start_npc(HARRIS)
q:add_talk_id(HARRIS, ALTRAN)
q:add_kill_id(STONE_GOLEM)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30035-02.htm" then
    st:start_quest()
    q:give_items(player, HARRIS_RECEIPT_1, 1)
  elseif event == "30283-02.htm" then
    st:set_cond(2, true)
    q:take_items(player, HARRIS_RECEIPT_1, 1)
    q:give_items(player, HARRIS_RECEIPT_2, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 10 and "30035-01a.htm" or "30035-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == HARRIS then
      if cond < 4 then
        htmltext = "30035-03.htm"
      elseif cond == 4 then
        htmltext = "30035-04.htm"
        q:take_items(player, HARRIS_RECEIPT_2, 1)
        q:take_items(player, TOOL_BOX, 1)
        q:give_items(player, WOODEN_BREASTPLATE, 1)
        q:add_exp_and_sp(player, 5000, 0)
        st:exit_quest(false, true)
      end
    elseif sw2 == ALTRAN then
      if cond == 1 then
        htmltext = "30283-01.htm"
      elseif cond == 2 then
        htmltext = "30283-03.htm"
      elseif cond == 3 then
        htmltext = "30283-04.htm"
        st:set_cond(4, true)
        q:take_items(player, GOLEM_SHARD, -1)
        q:give_items(player, TOOL_BOX, 1)
      elseif cond == 4 then
        htmltext = "30283-05.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_cond(2) and rnd(100) < 30 and q:get_quest_items_count(killer, GOLEM_SHARD) < 5 then
    q:give_items(killer, GOLEM_SHARD, 1)
    if q:get_quest_items_count(killer, GOLEM_SHARD) >= 5 then
      qs:set_cond(3, true)
    else
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)

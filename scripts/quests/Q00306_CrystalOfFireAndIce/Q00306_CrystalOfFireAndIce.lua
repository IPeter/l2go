-- Translated by questport from quests/Q00306_CrystalOfFireAndIce (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(306, "Q00306_CrystalOfFireAndIce", "Crystals of Fire and Ice")

-- Items
local FLAME_SHARD = 1020
local ICE_SHARD = 1021
-- Droplist (npcId, itemId, chance)
local DROPLIST = {
  {20109, FLAME_SHARD, 300000},
  {20110, ICE_SHARD, 300000},
  {20112, FLAME_SHARD, 400000},
  {20113, ICE_SHARD, 400000},
  {20114, FLAME_SHARD, 500000},
  {20115, ICE_SHARD, 500000},
}

q:register_quest_items(FLAME_SHARD, ICE_SHARD)
q:add_start_npc(30004) -- Katerina
q:add_talk_id(30004)
q:add_kill_id(20109, 20110, 20112, 20113, 20114, 20115)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30004-03.htm" then
    st:start_quest()
  elseif event == "30004-06.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 17 and "30004-01.htm" or "30004-02.htm"
  elseif sw == State.STARTED then
    local totalItems = q:get_quest_items_count(player, FLAME_SHARD) + q:get_quest_items_count(player, ICE_SHARD)
    if totalItems == 0 then
      htmltext = "30004-04.htm"
    else
      htmltext = "30004-05.htm"
      q:take_items(player, FLAME_SHARD, -1)
      q:take_items(player, ICE_SHARD, -1)
      local reward = totalItems * 60
      if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD and totalItems >= 10 then
        reward = reward + 5000
      end
      q:give_adena(player, reward, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  for _, drop in ipairs(DROPLIST) do
    if npc:id() == drop[1] then
      if rnd(1000000) < drop[3] then
        q:give_items(player, drop[2], 1)
      end
      break
    end
  end
end)

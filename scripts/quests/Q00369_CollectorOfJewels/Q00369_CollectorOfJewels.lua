-- Translated by questport from quests/Q00369_CollectorOfJewels (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(369, "Q00369_CollectorOfJewels", "Collector of Jewels")

-- NPC
local NELL = 30376
-- Items
local FLARE_SHARD = 5882
local FREEZING_SHARD = 5883
-- Reward
local ADENA = 57
-- Droplist
local MOBS_DROP_CHANCES = {}

MOBS_DROP_CHANCES[20609] = {id = FLARE_SHARD, chance = 75, count = 1} -- salamander_lakin
MOBS_DROP_CHANCES[20612] = {id = FLARE_SHARD, chance = 91, count = 1} -- salamander_rowin
MOBS_DROP_CHANCES[20749] = {id = FLARE_SHARD, chance = 100, count = 2} -- death_fire
MOBS_DROP_CHANCES[20616] = {id = FREEZING_SHARD, chance = 81, count = 1} -- undine_lakin
MOBS_DROP_CHANCES[20619] = {id = FREEZING_SHARD, chance = 87, count = 1} -- undine_rowin
MOBS_DROP_CHANCES[20747] = {id = FREEZING_SHARD, chance = 100, count = 2} -- roxide

q:register_quest_items(FLARE_SHARD, FREEZING_SHARD)
q:add_start_npc(NELL)
q:add_talk_id(NELL)
q:add_kill_id(J.keys(MOBS_DROP_CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30376-03.htm" then
    st:start_quest()
  elseif event == "30376-07.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  elseif event == "30376-08.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 25 and "30376-01.htm" or "30376-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local flare = q:get_quest_items_count(player, FLARE_SHARD)
    local freezing = q:get_quest_items_count(player, FREEZING_SHARD)
    if cond == 1 then
      htmltext = "30376-04.htm"
    elseif cond == 2 and flare >= 50 and freezing >= 50 then
      htmltext = "30376-05.htm"
      st:set_cond(3, true)
      q:take_items(player, FLARE_SHARD, -1)
      q:take_items(player, FREEZING_SHARD, -1)
      q:reward_items(player, ADENA, 12500)
    elseif cond == 3 then
      htmltext = "30376-09.htm"
    elseif cond == 4 and flare >= 200 and freezing >= 200 then
      htmltext = "30376-10.htm"
      q:take_items(player, FLARE_SHARD, -1)
      q:take_items(player, FREEZING_SHARD, -1)
      q:reward_items(player, ADENA, 63500)
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

-- Mobius never sets memo state 1, so every kill collects towards 200 of
-- each shard and cond 4 (kept as in Mobius).
q:on_kill(function(npc, player, isSummon)
  local item = MOBS_DROP_CHANCES[npc:id()]
  if rnd(100) < item.chance then
    local luckyPlayer = q:get_random_party_member(player, npc)
    if luckyPlayer ~= nil then
      local qs = q:get_quest_state(luckyPlayer, false)
      if qs ~= nil then
        local itemCount = qs:is_memo_state(1) and 50 or 200
        local cond = qs:is_memo_state(1) and 2 or 4
        local v = item.count
        if q:give_item_randomly(luckyPlayer, npc, item.id, v, v, itemCount, 1, true) and q:get_quest_items_count(luckyPlayer, FLARE_SHARD, FREEZING_SHARD) >= itemCount * 2 then
          qs:set_cond(cond)
        end
      end
    end
  end
end)

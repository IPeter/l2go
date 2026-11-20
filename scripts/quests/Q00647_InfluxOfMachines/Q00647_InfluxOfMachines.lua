-- Translated by questport from quests/Q00647_InfluxOfMachines (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(647, "Q00647_InfluxOfMachines", "Influx of Machines")

-- NPC
local GUTENHAGEN = 32069
-- Item
local DESTROYED_GOLEM_SHARD = 8100

q:register_quest_items(DESTROYED_GOLEM_SHARD)
q:add_start_npc(GUTENHAGEN)
q:add_talk_id(GUTENHAGEN)
for i = 22052, 22078 do
  q:add_kill_id(i)
end

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32069-02.htm" then
    st:start_quest()
  elseif event == "32069-06.htm" then
    q:take_items(player, DESTROYED_GOLEM_SHARD, -1)
    q:give_items(player, rnd(4963, 4972), 1)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 46 and "32069-03.htm" or "32069-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "32069-04.htm"
    elseif cond == 2 then
      htmltext = "32069-05.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local qs = q:get_random_party_member_state(player, 1, 3, npc)
  if qs == nil then
    return
  end
  local partyMember = qs:get_player()
  local st = q:get_quest_state(partyMember, false)
  if st == nil then
    return
  end
  if rnd(10) < 3 then
    q:give_items(partyMember, DESTROYED_GOLEM_SHARD, 1)
    if q:get_quest_items_count(partyMember, DESTROYED_GOLEM_SHARD) < 500 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

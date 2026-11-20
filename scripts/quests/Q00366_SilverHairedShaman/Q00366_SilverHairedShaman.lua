-- Translated by questport from quests/Q00366_SilverHairedShaman (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(366, "Q00366_SilverHairedShaman", "Silver Haired Shaman")

-- NPC
local DIETER = 30111
-- Item
local SAIRONS_SILVER_HAIR = 5874
-- Misc
local MIN_LEVEL = 48
-- Drop chances
local CHANCES = {}

CHANCES[20986] = 56
CHANCES[20987] = 66
CHANCES[20988] = 62

q:register_quest_items(SAIRONS_SILVER_HAIR)
q:add_start_npc(DIETER)
q:add_talk_id(DIETER)
q:add_kill_id(20986, 20987, 20988)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30111-2.htm" then
    st:start_quest()
  elseif event == "30111-6.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < MIN_LEVEL and "30111-0.htm" or "30111-1.htm"
  elseif sw == State.STARTED then
    local count = q:get_quest_items_count(player, SAIRONS_SILVER_HAIR)
    if count == 0 then
      htmltext = "30111-3.htm"
    else
      htmltext = "30111-4.htm"
      q:take_items(player, SAIRONS_SILVER_HAIR, -1)
      q:give_adena(player, 12070 + 500 * count, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_random_party_member_state(killer, 1, 3, npc)
  if qs ~= nil and rnd(100) < CHANCES[npc:id()] then
    q:give_item_randomly(qs:get_player(), npc, SAIRONS_SILVER_HAIR, 1, 1, 0, 1, true)
  end
end)

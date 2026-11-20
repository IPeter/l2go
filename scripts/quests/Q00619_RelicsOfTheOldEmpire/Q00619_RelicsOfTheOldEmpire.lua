-- Translated by questport from quests/Q00619_RelicsOfTheOldEmpire (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(619, "Q00619_RelicsOfTheOldEmpire", "Relics of the Old Empire")

-- NPC
local GHOST_OF_ADVENTURER = 31538
-- Items
local RELICS = 7254
local ENTRANCE = 7075
-- Rewards ; all S grade weapons recipe (60%)
local RECIPES = {6881, 6883, 6885, 6887, 6891, 6893, 6895, 6897, 6899, 7580}

q:register_quest_items(RELICS)
q:add_start_npc(GHOST_OF_ADVENTURER)
q:add_talk_id(GHOST_OF_ADVENTURER)
for id = 21396, 21434 do
  -- IT monsters
  q:add_kill_id(id)
end
-- monsters at IT entrance
q:add_kill_id(21798, 21799, 21800)
for id = 18120, 18256 do
  -- Sepulchers monsters
  q:add_kill_id(id)
end

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31538-03.htm" then
    st:start_quest()
  elseif event == "31538-09.htm" then
    if q:get_quest_items_count(player, RELICS) >= 1000 then
      htmltext = "31538-09.htm"
      q:take_items(player, RELICS, 1000)
      q:give_items(player, rnd.entry(RECIPES), 1)
    else
      htmltext = "31538-06.htm"
    end
  elseif event == "31538-10.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 74 and "31538-02.htm" or "31538-01.htm"
  elseif sw == State.STARTED then
    if q:get_quest_items_count(player, RELICS) >= 1000 then
      htmltext = "31538-04.htm"
    elseif q:has_quest_items(player, ENTRANCE) then
      htmltext = "31538-06.htm"
    else
      htmltext = "31538-07.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_random_party_member_state(killer, -1, 3, npc)
  if qs == nil or not qs:is_started() then
    return
  end
  local partyMember = qs:get_player()
  q:give_item_randomly(partyMember, npc, RELICS, 1, 1, 0, 1, true)
  if rnd(100) < 30 then
    q:give_item_randomly(partyMember, npc, ENTRANCE, 1, 1, 0, 1, true)
  end
end)

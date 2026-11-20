-- Translated by questport from quests/Q00377_ExplorationOfTheGiantsCavePart2 (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(377, "Q00377_ExplorationOfTheGiantsCavePart2", "Exploration of the Giants Cave, Part 2")

-- Items
local ANCIENT_BOOK = 5955
local DICTIONARY_INTERMEDIATE = 5892
local BOOKS = {{5945, 5946, 5947, 5948, 5949}, {5950, 5951, 5952, 5953, 5954}}
-- Rewards
local RECIPES = {{5338, 5336}, {5420, 5422}}

local function checkItems(player)
  for type_ = 0, #BOOKS - 1 do
    local complete = true
    for _, book in ipairs(BOOKS[type_ + 1]) do
      if not q:has_quest_items(player, book) then
        complete = false
      end
    end
    if complete then
      for _, book in ipairs(BOOKS[type_ + 1]) do
        q:take_items(player, book, 1)
      end
      q:give_items(player, RECIPES[type_ + 1][rnd(#RECIPES[type_ + 1]) + 1], 1)
      return "31147-04.htm"
    end
  end
  return "31147-05.htm"
end

q:add_start_npc(31147) -- Sobling
q:add_talk_id(31147)
q:add_kill_id(20654, 20656, 20657, 20658)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31147-03.htm" then
    st:start_quest()
  elseif event == "31147-04.htm" then
    htmltext = checkItems(player)
  elseif event == "31147-07.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = (player:level() < 57 or not q:has_quest_items(player, DICTIONARY_INTERMEDIATE)) and "31147-01.htm" or "31147-02.htm"
  elseif sw == State.STARTED then
    htmltext = checkItems(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local partyMember = q:get_random_party_member_state(player, State.STARTED)
  if partyMember ~= nil then
    q:give_item_randomly(partyMember, npc, ANCIENT_BOOK, 1, 1, 0, 0.8, true)
  end
end)

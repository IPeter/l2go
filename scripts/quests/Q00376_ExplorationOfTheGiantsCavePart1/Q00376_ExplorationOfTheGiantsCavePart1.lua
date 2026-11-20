-- Translated by questport from quests/Q00376_ExplorationOfTheGiantsCavePart1 (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(376, "Q00376_ExplorationOfTheGiantsCavePart1", "Exploration of the Giants Cave, Part 1")

-- NPCs
local SOBLING = 31147
local CLIFF = 30182
-- Items
local PARCHMENT = 5944
local DICTIONARY_BASIC = 5891
local MYSTERIOUS_BOOK = 5890
local DICTIONARY_INTERMEDIATE = 5892
local BOOKS = {
  {5937, 5938, 5939, 5940, 5941},
  {5932, 5933, 5934, 5935, 5936},
  {5922, 5923, 5924, 5925, 5926},
  {5927, 5928, 5929, 5930, 5931},
}
-- Rewards
local RECIPES = {{5346, 5354}, {5332, 5334}, {5416, 5418}, {5424, 5340}}

local function checkItems(player, st)
  if q:has_quest_items(player, MYSTERIOUS_BOOK) then
    local cond = st:get_cond()
    if cond == 1 then
      st:set_cond(2, true)
      return "31147-07.htm"
    end
    return "31147-08.htm"
  end
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

q:register_quest_items(DICTIONARY_BASIC, MYSTERIOUS_BOOK)
q:add_start_npc(SOBLING)
q:add_talk_id(SOBLING, CLIFF)
q:add_kill_id(20647, 20648, 20649, 20650)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31147-03.htm" then
    st:start_quest()
    st:set("condBook", "1")
    q:give_items(player, DICTIONARY_BASIC, 1)
  elseif event == "31147-04.htm" then
    htmltext = checkItems(player, st)
  elseif event == "31147-09.htm" then
    st:exit_quest(true, true)
  elseif event == "30182-02.htm" then
    st:set_cond(3, true)
    q:take_items(player, MYSTERIOUS_BOOK, -1)
    q:give_items(player, DICTIONARY_INTERMEDIATE, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 51 and "31147-01.htm" or "31147-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == SOBLING then
      htmltext = checkItems(player, st)
    elseif sw2 == CLIFF then
      if cond == 2 and q:has_quest_items(player, MYSTERIOUS_BOOK) then
        htmltext = "30182-01.htm"
      elseif cond == 3 then
        htmltext = "30182-03.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  -- Drop parchment to anyone
  local partyMember = q:get_random_party_member_state(player, State.STARTED)
  if partyMember == nil then
    return
  end
  local st = partyMember:get_quest_state(q:name())
  if st == nil then
    return
  end
  q:give_item_randomly(partyMember, npc, PARCHMENT, 1, 1, 0, 0.2, true)
  -- Drop mysterious book to person who still need it
  partyMember = q:get_random_party_member(player, "condBook", "1")
  if partyMember == nil then
    return
  end
  st = partyMember:get_quest_state(q:name())
  if st == nil then
    return
  end
  if rnd.double(100) < 0.1 then
    q:give_items(partyMember, MYSTERIOUS_BOOK, 1)
    q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    st:unset("condBook")
  end
end)

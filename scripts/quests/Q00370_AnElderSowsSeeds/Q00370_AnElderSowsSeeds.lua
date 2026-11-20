-- Translated by questport from quests/Q00370_AnElderSowsSeeds (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(370, "Q00370_AnElderSowsSeeds", "An Elder Sows Seeds")

-- NPC
local CASIAN = 30612
-- Items
local SPELLBOOK_PAGE = 5916
local CHAPTER_OF_FIRE = 5917
local CHAPTER_OF_WATER = 5918
local CHAPTER_OF_WIND = 5919
local CHAPTER_OF_EARTH = 5920
-- Drop chances
local MOBS1 = {}
local MOBS2 = {}

MOBS1[20082] = 9 -- ant_recruit
MOBS1[20086] = 9 -- ant_guard
MOBS1[20090] = 22 -- noble_ant_leader
MOBS2[20084] = 0.101 -- ant_patrol
MOBS2[20089] = 0.1 -- noble_ant

q:add_start_npc(CASIAN)
q:add_talk_id(CASIAN)
q:add_kill_id(20082, 20084, 20086, 20089, 20090)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30612-3.htm" then
    st:start_quest()
  elseif event == "30612-6.htm" then
    if q:has_quest_items(player, CHAPTER_OF_FIRE, CHAPTER_OF_WATER, CHAPTER_OF_WIND, CHAPTER_OF_EARTH) then
      htmltext = "30612-8.htm"
      q:take_items(player, CHAPTER_OF_FIRE, 1)
      q:take_items(player, CHAPTER_OF_WATER, 1)
      q:take_items(player, CHAPTER_OF_WIND, 1)
      q:take_items(player, CHAPTER_OF_EARTH, 1)
      q:give_adena(player, 3600, true)
    end
  elseif event == "30612-9.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 28 and "30612-0a.htm" or "30612-0.htm"
  elseif sw == State.STARTED then
    htmltext = "30612-4.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local qs = q:get_random_party_member_state(player, -1, 3, npc)
  if qs ~= nil then
    local npcId = npc:id()
    local mobs1Chance = MOBS1[npcId]
    if mobs1Chance ~= nil then
      if rnd(100) < mobs1Chance then
        local luckyPlayer = q:get_random_party_member(player, npc)
        if luckyPlayer ~= nil then
          q:give_item_randomly(luckyPlayer, npc, SPELLBOOK_PAGE, 1, 1, 0, 1, true)
        end
      end
    else
      q:give_item_randomly(qs:get_player(), npc, SPELLBOOK_PAGE, 1, 1, 0, MOBS2[npcId], true)
    end
  end
end)

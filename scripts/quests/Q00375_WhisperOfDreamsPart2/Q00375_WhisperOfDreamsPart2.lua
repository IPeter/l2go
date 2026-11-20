-- Translated by questport from quests/Q00375_WhisperOfDreamsPart2 (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(375, "Q00375_WhisperOfDreamsPart2", "Whisper of Dreams, Part 2")

-- NPCs
local MANAKIA = 30515
-- Monsters
local KARIK = 20629
local CAVE_HOWLER = 20624
-- Items
local MYSTERIOUS_STONE = 5887
local KARIK_HORN = 5888
local CAVE_HOWLER_SKULL = 5889
-- Rewards : A grade robe recipes
local REWARDS = {5348, 5350, 5352}

q:register_quest_items(KARIK_HORN, CAVE_HOWLER_SKULL)
q:add_start_npc(MANAKIA)
q:add_talk_id(MANAKIA)
q:add_kill_id(KARIK, CAVE_HOWLER)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  -- Manakia
  if event == "30515-03.htm" then
    st:start_quest()
    q:take_items(player, MYSTERIOUS_STONE, 1)
  elseif event == "30515-07.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = (not q:has_quest_items(player, MYSTERIOUS_STONE) or player:level() < 60) and "30515-01.htm" or "30515-02.htm"
  elseif sw == State.STARTED then
    if q:get_quest_items_count(player, KARIK_HORN) >= 100 and q:get_quest_items_count(player, CAVE_HOWLER_SKULL) >= 100 then
      htmltext = "30515-05.htm"
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:take_items(player, KARIK_HORN, 100)
      q:take_items(player, CAVE_HOWLER_SKULL, 100)
      q:give_items(player, rnd.entry(REWARDS), 1)
    else
      htmltext = "30515-04.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  -- Drop horn or skull to anyone.
  local partyMember = q:get_random_party_member_state(player, State.STARTED)
  if partyMember == nil then
    return
  end
  local st = partyMember:get_quest_state(q:name())
  if st == nil then
    return
  end
  local sw = npc:id()
  if sw == KARIK then
    q:give_items(st:get_player(), KARIK_HORN, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  elseif sw == CAVE_HOWLER then
    if rnd(10) < 9 then
      q:give_items(st:get_player(), CAVE_HOWLER_SKULL, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)

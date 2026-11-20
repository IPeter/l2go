-- Translated by questport from quests/Q00354_ConquestOfAlligatorIsland (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(354, "Q00354_ConquestOfAlligatorIsland", "Conquest of Alligator Island")

-- Items
local ALLIGATOR_TOOTH = 5863
local TORN_MAP_FRAGMENT = 5864
local PIRATE_TREASURE_MAP = 5915
-- Drops
local MOB1 = {}
local MOB2 = {}

MOB1[20804] = 0.84 -- crokian_lad
MOB1[20805] = 0.91 -- dailaon_lad
MOB1[20806] = 0.88 -- crokian_lad_warrior
MOB1[20807] = 0.92 -- farhite_lad
MOB2[20808] = 14 -- nos_lad
MOB2[20991] = 69 -- tribe_of_swamp

q:register_quest_items(ALLIGATOR_TOOTH, TORN_MAP_FRAGMENT)
q:add_start_npc(30895) -- Kluck
q:add_talk_id(30895)
q:add_kill_id(J.keys(MOB1))
q:add_kill_id(J.keys(MOB2))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30895-02.htm" then
    st:start_quest()
  elseif event == "30895-03.htm" then
    if q:has_quest_items(player, TORN_MAP_FRAGMENT) then
      htmltext = "30895-03a.htm"
    end
  elseif event == "30895-05.htm" then
    local amount = q:get_quest_items_count(player, ALLIGATOR_TOOTH)
    if amount > 0 then
      local reward = amount * 220 + 3100
      if amount >= 100 then
        reward = reward + 7600
        htmltext = "30895-05b.htm"
      else
        htmltext = "30895-05a.htm"
      end
      q:take_items(player, ALLIGATOR_TOOTH, -1)
      q:give_adena(player, reward, true)
    end
  elseif event == "30895-07.htm" then
    if q:get_quest_items_count(player, TORN_MAP_FRAGMENT) >= 10 then
      htmltext = "30895-08.htm"
      q:take_items(player, TORN_MAP_FRAGMENT, 10)
      q:give_items(player, PIRATE_TREASURE_MAP, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30895-09.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 38 and "30895-00.htm" or "30895-01.htm"
  elseif sw == State.STARTED then
    htmltext = q:has_quest_items(player, TORN_MAP_FRAGMENT) and "30895-03a.htm" or "30895-03.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local qs = q:get_random_party_member_state(player, -1, 3, npc)
  if qs ~= nil then
    local npcId = npc:id()
    local mob1Chance = MOB1[npcId]
    if mob1Chance ~= nil then
      q:give_item_randomly(player, npc, ALLIGATOR_TOOTH, 1, 1, 0, mob1Chance, true)
    else
      local itemCount = rnd(100) < MOB2[npcId] and 2 or 1
      q:give_item_randomly(player, npc, ALLIGATOR_TOOTH, itemCount, itemCount, 0, 1, true)
    end
    q:give_item_randomly(player, npc, TORN_MAP_FRAGMENT, 1, 1, 0, 0.1, false)
  end
end)

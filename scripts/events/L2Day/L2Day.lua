-- Mobius events/L2Day (a LongTimeEvent, lib/long_time_event.lua): monsters
-- drop letters (1 % each), and the Event Managers in the five newbie
-- villages trade the words LINEAGEII, NCSOFT and CHRONICLE for prizes. Its
-- dates, drops and spawns are config.xml's (data/events/longtimeevents.json,
-- the config's events.dates on top); the pages are Mobius's
-- (data/html/scripts/events/L2Day/).

local lte = require("lib.long_time_event")

-- NPCs: the manager and the Village Scroll of Escape it gives.
local MANAGERS = {
  [31854] = 7117, -- Talking Island Village
  [31855] = 7118, -- Elven Village
  [31856] = 7119, -- Dark Elven Village
  [31857] = 7121, -- Dwarven Village
  [31858] = 7120, -- Orc Village
}
local MANAGER_IDS = { 31854, 31855, 31856, 31857, 31858 }

-- Items
local A = 3875
local C = 3876
local E = 3877
local F = 3878
local G = 3879
local H = 3880
local I = 3881
local L = 3882
local N = 3883
local O = 3884
local R = 3885
local S = 3886
local T = 3887
local II = 3888

-- Rewards (Mobius ItemChanceHolder(id, chance, count)): the first whose
-- chance the roll reaches.
local L2_REWARDS = {
  { id = 8947, chance = 19, count = 1 }, -- L2day - Rabbit Ears
  { id = 8948, chance = 16, count = 1 }, -- L2day - Little Angel Wings
  { id = 8949, chance = 13, count = 1 }, -- L2day - Fairy Antennae
  { id = 3959, chance = 10, count = 2 }, -- Blessed Scroll of Resurrection (Event)
  { id = 3958, chance = 7, count = 2 }, -- Blessed Scroll of Escape (Event)
  { id = 8752, chance = 4, count = 2 }, -- High-Grade Life Stone - Level 76
  { id = 8762, chance = 1, count = 1 }, -- Top-Grade Life Stone - Level 76
  { id = 6660, chance = 0, count = 1 }, -- Ring of Queen Ant
}
local NC_REWARDS = {
  { id = 8948, chance = 19, count = 1 }, -- L2day - Little Angel Wings
  { id = 8949, chance = 16, count = 1 }, -- L2day - Fairy Antennae
  { id = 8950, chance = 13, count = 1 }, -- L2day - Feathered Hat
  { id = 3959, chance = 10, count = 1 }, -- Blessed Scroll of Resurrection (Event)
  { id = 3958, chance = 7, count = 1 }, -- Blessed Scroll of Escape (Event)
  { id = 8742, chance = 4, count = 2 }, -- Mid-Grade Life Stone - Level 76
  { id = 8752, chance = 1, count = 1 }, -- High-Grade Life Stone - Level 76
  { id = 6661, chance = 0, count = 1 }, -- Earring of Orfen
}
local CH_REWARDS = {
  { id = 8949, chance = 19, count = 1 }, -- L2day - Fairy Antennae
  { id = 8950, chance = 16, count = 1 }, -- L2day - Feathered Hat
  { id = 8951, chance = 13, count = 1 }, -- L2day - Artisan's Goggles
  { id = 3959, chance = 10, count = 1 }, -- Blessed Scroll of Resurrection (Event)
  { id = 3958, chance = 7, count = 1 }, -- Blessed Scroll of Escape (Event)
  { id = 8742, chance = 4, count = 1 }, -- Mid-Grade Life Stone - Level 76
  { id = 8752, chance = 1, count = 1 }, -- High-Grade Life Stone - Level 76
  { id = 6662, chance = 0, count = 1 }, -- Ring of Core
}

local s = lte.new("L2Day")
s:add_start_npc(MANAGER_IDS)
s:add_first_talk_id(MANAGER_IDS)
s:add_talk_id(MANAGER_IDS)

-- reward rolls once: 95 or more gives scrolls of the manager's village,
-- else the first reward of the list whose chance the roll reaches.
local function reward(player, npc, list, scrolls)
  local random = rnd(100)
  if random >= 95 then
    s:give_items(player, MANAGERS[npc:id()], scrolls)
    return
  end
  for _, holder in ipairs(list) do
    if random >= holder.chance then
      s:give_items(player, holder.id, holder.count)
      return
    end
  end
end

lte.on_event(s, function(event, npc, player)
  local htmltext = event
  if event == "collect_l2" or event == "collect_nc" or event == "collect_ch" then
    if npc == nil or MANAGERS[npc:id()] == nil then
      return nil -- Mobius needs the manager (its village scroll)
    end
  end
  if event == "collect_l2" then
    if s:has_quest_items(player, L, I, N, E, A, G, II) and s:get_quest_items_count(player, E) > 1 then
      s:take_items_ids(player, 1, L, I, N, E, A, G, E, II)
      reward(player, npc, L2_REWARDS, 2)
      htmltext = "manager-1.htm"
    else
      htmltext = "manager-no.htm"
    end
  elseif event == "collect_nc" then
    if s:has_quest_items(player, N, C, S, O, F, T) then
      s:take_items_ids(player, 1, N, C, S, O, F, T)
      reward(player, npc, NC_REWARDS, 1)
      htmltext = "manager-1.htm"
    else
      htmltext = "manager-no.htm"
    end
  elseif event == "collect_ch" then
    if s:has_quest_items(player, C, H, R, O, N, I, L, E) and s:get_quest_items_count(player, C) > 1 then
      s:take_items_ids(player, 1, C, H, R, O, N, I, C, L, E)
      reward(player, npc, CH_REWARDS, 1)
      htmltext = "manager-1.htm"
    else
      htmltext = "manager-no.htm"
    end
  end
  return htmltext
end)

s:on_first_talk(function(npc, player)
  return "manager-1.htm"
end)

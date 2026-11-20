-- Translated by questport from quests/Q00611_AllianceWithVarkaSilenos (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Alliance with Varka Silenos (611)
-- @author malyelfik
local J = require("lib.java")

local q = quest.new(611, "Q00611_AllianceWithVarkaSilenos", "Alliance with Varka Silenos")

-- NPC
local NARAN_ASHANUK = 31378
-- Monsters
local MOBS = {}
-- Items
local KETRA_BADGE_SOLDIER = 7226
local KETRA_BADGE_OFFICER = 7227
local KETRA_BADGE_CAPTAIN = 7228
local VALOR_FEATHER = 7229
local WISDOM_FEATHER = 7230
local KETRA_MARKS = {7211, 7212, 7213, 7214, 7215}
local VARKA_MARKS = {7221, 7222, 7223, 7224, 7225}
-- Misc
local MIN_LEVEL = 74
local SOLDIER_BADGE_COUNT = {100, 200, 300, 300, 400}
local OFFICER_BADGE_COUNT = {0, 100, 200, 300, 400}
local CAPTAIN_BADGE_COUNT = {0, 0, 100, 200, 200}

local DropInfo = {}
DropInfo.__index = DropInfo

function DropInfo.new(chance, minCond)
  local self = setmetatable({}, DropInfo)
  self._chance = 0
  self._minCond = 0
  self._itemId = 0
  self._chance = chance
  self._minCond = minCond
  if self._minCond == 1 then
    self._itemId = KETRA_BADGE_SOLDIER
  elseif self._minCond == 2 then
    self._itemId = KETRA_BADGE_OFFICER
  else
    self._itemId = KETRA_BADGE_CAPTAIN
  end
  return self
end

function DropInfo:getMinCond()
  return self._minCond
end

function DropInfo:getChance()
  return self._chance
end

function DropInfo:getItemId()
  return self._itemId
end

local function canGetItem(qs, itemId)
  local count = 0
  if itemId == KETRA_BADGE_SOLDIER then
    count = SOLDIER_BADGE_COUNT[qs:get_cond()]
  elseif itemId == KETRA_BADGE_OFFICER then
    count = OFFICER_BADGE_COUNT[qs:get_cond()]
  elseif itemId == KETRA_BADGE_CAPTAIN then
    count = CAPTAIN_BADGE_COUNT[qs:get_cond()]
  end
  return q:get_quest_items_count(qs:get_player(), itemId) < count
end

MOBS[21324] = DropInfo.new(500, 1) -- Ketra Orc Footman
MOBS[21325] = DropInfo.new(500, 1) -- Ketra's War Hound
MOBS[21327] = DropInfo.new(509, 1) -- Ketra Orc Raider
MOBS[21328] = DropInfo.new(521, 1) -- Ketra Orc Scout
MOBS[21329] = DropInfo.new(519, 1) -- Ketra Orc Shaman
MOBS[21331] = DropInfo.new(500, 2) -- Ketra Orc Warrior
MOBS[21332] = DropInfo.new(500, 2) -- Ketra Orc Lieutenant
MOBS[21334] = DropInfo.new(509, 2) -- Ketra Orc Medium
MOBS[21335] = DropInfo.new(518, 2) -- Ketra Orc Elite Soldier
MOBS[21336] = DropInfo.new(518, 2) -- Ketra Orc White Captain
MOBS[21338] = DropInfo.new(527, 2) -- Ketra Orc Seer
MOBS[21339] = DropInfo.new(500, 3) -- Ketra Orc General
MOBS[21340] = DropInfo.new(500, 3) -- Ketra Orc Battalion Commander
MOBS[21342] = DropInfo.new(508, 3) -- Ketra Orc Grand Seer
MOBS[21343] = DropInfo.new(628, 2) -- Ketra Commander
MOBS[21344] = DropInfo.new(604, 2) -- Ketra Elite Guard
MOBS[21345] = DropInfo.new(627, 3) -- Ketra's Head Shaman
MOBS[21346] = DropInfo.new(604, 3) -- Ketra's Head Guard
MOBS[21347] = DropInfo.new(649, 3) -- Ketra Prophet
MOBS[21348] = DropInfo.new(626, 3) -- Prophet's Guard
MOBS[21349] = DropInfo.new(626, 3) -- Prophet's Aide

q:add_start_npc(NARAN_ASHANUK)
q:add_talk_id(NARAN_ASHANUK)
q:add_kill_id(J.keys(MOBS))
q:register_quest_items(KETRA_BADGE_CAPTAIN, KETRA_BADGE_OFFICER, KETRA_BADGE_SOLDIER)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  local htmltext = event
  if event == "31378-12a.html" or event == "31378-12b.html" or event == "31378-25.html" then
  elseif event == "31378-04.htm" then
    if q:has_at_least_one_quest_item(player, unpack(KETRA_MARKS)) then
      return "31378-03.htm"
    end
    qs:set_state(State.STARTED)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
    for i = 0, #VARKA_MARKS - 1 do
      if q:has_quest_items(player, VARKA_MARKS[i + 1]) then
        qs:set_cond(i + 2)
        return "31378-0" .. (i + 5) .. ".htm"
      end
    end
    qs:set_cond(1)
  elseif event == "31378-12.html" then
    if q:get_quest_items_count(player, KETRA_BADGE_SOLDIER) < SOLDIER_BADGE_COUNT[1] then
      return q:get_no_quest_msg(player)
    end
    q:take_items(player, KETRA_BADGE_SOLDIER, -1)
    q:give_items(player, VARKA_MARKS[1], 1)
    qs:set_cond(2, true)
  elseif event == "31378-15.html" then
    if q:get_quest_items_count(player, KETRA_BADGE_SOLDIER) < SOLDIER_BADGE_COUNT[2] or q:get_quest_items_count(player, KETRA_BADGE_OFFICER) < OFFICER_BADGE_COUNT[2] then
      return q:get_no_quest_msg(player)
    end
    q:take_items_ids(player, -1, KETRA_BADGE_SOLDIER, KETRA_BADGE_OFFICER, VARKA_MARKS[1])
    q:give_items(player, VARKA_MARKS[2], 1)
    qs:set_cond(3, true)
  elseif event == "31378-18.html" then
    if q:get_quest_items_count(player, KETRA_BADGE_SOLDIER) < SOLDIER_BADGE_COUNT[3] or q:get_quest_items_count(player, KETRA_BADGE_OFFICER) < OFFICER_BADGE_COUNT[3] or q:get_quest_items_count(player, KETRA_BADGE_CAPTAIN) < CAPTAIN_BADGE_COUNT[3] then
      return q:get_no_quest_msg(player)
    end
    q:take_items_ids(player, -1, KETRA_BADGE_SOLDIER, KETRA_BADGE_OFFICER, KETRA_BADGE_CAPTAIN, VARKA_MARKS[2])
    q:give_items(player, VARKA_MARKS[3], 1)
    qs:set_cond(4, true)
  elseif event == "31378-21.html" then
    if not q:has_quest_items(player, VALOR_FEATHER) or q:get_quest_items_count(player, KETRA_BADGE_SOLDIER) < SOLDIER_BADGE_COUNT[4] or q:get_quest_items_count(player, KETRA_BADGE_OFFICER) < OFFICER_BADGE_COUNT[4] or q:get_quest_items_count(player, KETRA_BADGE_CAPTAIN) < CAPTAIN_BADGE_COUNT[4] then
      return q:get_no_quest_msg(player)
    end
    q:take_items_ids(player, -1, KETRA_BADGE_SOLDIER, KETRA_BADGE_OFFICER, KETRA_BADGE_CAPTAIN, VALOR_FEATHER, VARKA_MARKS[3])
    q:give_items(player, VARKA_MARKS[4], 1)
    qs:set_cond(5, true)
  elseif event == "31378-26.html" then
    q:take_items_ids(player, -1, unpack(VARKA_MARKS))
    q:take_items_ids(player, -1, VALOR_FEATHER, WISDOM_FEATHER)
    qs:exit_quest(true, true)
  else
    htmltext = nil
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local member = q:get_random_party_member_state(killer, State.STARTED)
  if member ~= nil then
    local qs = q:get_quest_state(member, false)
    local info = MOBS[npc:id()]
    if qs:get_cond() >= info:getMinCond() and qs:get_cond() < 6 and canGetItem(qs, info:getItemId()) and rnd(1000) < info:getChance() then
      q:give_items(member, info:getItemId(), 1)
      q:play_sound(member, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = qs:get_state()
  if sw == State.CREATED then
    htmltext = player:level() >= MIN_LEVEL and "31378-01.htm" or "31378-02.htm"
  elseif sw == State.STARTED then
    local sw2 = qs:get_cond()
    if sw2 == 1 then
      htmltext = q:get_quest_items_count(player, KETRA_BADGE_SOLDIER) >= SOLDIER_BADGE_COUNT[1] and "31378-11.html" or "31378-10.html"
    elseif sw2 == 2 then
      htmltext = q:has_quest_items(player, VARKA_MARKS[1]) and q:get_quest_items_count(player, KETRA_BADGE_SOLDIER) >= SOLDIER_BADGE_COUNT[2] and q:get_quest_items_count(player, KETRA_BADGE_OFFICER) >= OFFICER_BADGE_COUNT[2] and "31378-14.html" or "31378-13.html"
    elseif sw2 == 3 then
      htmltext = q:has_quest_items(player, VARKA_MARKS[2]) and q:get_quest_items_count(player, KETRA_BADGE_SOLDIER) >= SOLDIER_BADGE_COUNT[3] and q:get_quest_items_count(player, KETRA_BADGE_OFFICER) >= OFFICER_BADGE_COUNT[3] and q:get_quest_items_count(player, KETRA_BADGE_CAPTAIN) >= CAPTAIN_BADGE_COUNT[3] and "31378-17.html" or "31378-16.html"
    elseif sw2 == 4 then
      htmltext = q:has_quest_items(player, VARKA_MARKS[3], VALOR_FEATHER) and q:get_quest_items_count(player, KETRA_BADGE_SOLDIER) >= SOLDIER_BADGE_COUNT[4] and q:get_quest_items_count(player, KETRA_BADGE_OFFICER) >= OFFICER_BADGE_COUNT[4] and q:get_quest_items_count(player, KETRA_BADGE_CAPTAIN) >= CAPTAIN_BADGE_COUNT[4] and "31378-20.html" or "31378-19.html"
    elseif sw2 == 5 then
      if not q:has_quest_items(player, VARKA_MARKS[4]) or not q:has_quest_items(player, WISDOM_FEATHER) or q:get_quest_items_count(player, KETRA_BADGE_SOLDIER) < SOLDIER_BADGE_COUNT[5] or q:get_quest_items_count(player, KETRA_BADGE_OFFICER) < OFFICER_BADGE_COUNT[5] or q:get_quest_items_count(player, KETRA_BADGE_CAPTAIN) < CAPTAIN_BADGE_COUNT[5] then
        return "31378-22.html"
      end
      qs:set_cond(6, true)
      q:take_items_ids(player, -1, KETRA_BADGE_SOLDIER, KETRA_BADGE_OFFICER, KETRA_BADGE_CAPTAIN, WISDOM_FEATHER, VARKA_MARKS[4])
      q:give_items(player, VARKA_MARKS[5], 1)
      htmltext = "31378-23.html"
    elseif sw2 == 6 then
      if q:has_quest_items(player, VARKA_MARKS[5]) then
        htmltext = "31378-24.html"
      end
    end
  end
  return htmltext
end)

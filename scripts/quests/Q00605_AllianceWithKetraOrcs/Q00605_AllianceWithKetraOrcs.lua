-- Translated by questport from quests/Q00605_AllianceWithKetraOrcs (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Alliance with Ketra Orcs (605)
-- @author malyelfik
local J = require("lib.java")

local q = quest.new(605, "Q00605_AllianceWithKetraOrcs", "Alliance with Ketra Orcs")

-- NPC
local WAHKAN = 31371
-- Monsters
local MOBS = {}
-- Items
local VARKA_BADGE_SOLDIER = 7216
local VARKA_BADGE_OFFICER = 7217
local VARKA_BADGE_CAPTAIN = 7218
local VALOR_TOTEM = 7219
local WISDOM_TOTEM = 7220
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
    self._itemId = VARKA_BADGE_SOLDIER
  elseif self._minCond == 2 then
    self._itemId = VARKA_BADGE_OFFICER
  else
    self._itemId = VARKA_BADGE_CAPTAIN
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
  if itemId == VARKA_BADGE_SOLDIER then
    count = SOLDIER_BADGE_COUNT[qs:get_cond()]
  elseif itemId == VARKA_BADGE_OFFICER then
    count = OFFICER_BADGE_COUNT[qs:get_cond()]
  elseif itemId == VARKA_BADGE_CAPTAIN then
    count = CAPTAIN_BADGE_COUNT[qs:get_cond()]
  end
  return q:get_quest_items_count(qs:get_player(), itemId) < count
end

MOBS[21350] = DropInfo.new(500, 1) -- Varka Silenos Recruit
MOBS[21351] = DropInfo.new(500, 1) -- Varka Silenos Footman
MOBS[21353] = DropInfo.new(509, 1) -- Varka Silenos Scout
MOBS[21354] = DropInfo.new(521, 1) -- Varka Silenos Hunter
MOBS[21355] = DropInfo.new(519, 1) -- Varka Silenos Shaman
MOBS[21357] = DropInfo.new(500, 2) -- Varka Silenos Priest
MOBS[21358] = DropInfo.new(500, 2) -- Varka Silenos Warrior
MOBS[21360] = DropInfo.new(509, 2) -- Varka Silenos Medium
MOBS[21361] = DropInfo.new(518, 2) -- Varka Silenos Magus
MOBS[21362] = DropInfo.new(518, 2) -- Varka Silenos Officer
MOBS[21364] = DropInfo.new(527, 2) -- Varka Silenos Seer
MOBS[21365] = DropInfo.new(500, 3) -- Varka Silenos Great Magus
MOBS[21366] = DropInfo.new(500, 3) -- Varka Silenos General
MOBS[21368] = DropInfo.new(508, 3) -- Varka Silenos Great Seer
MOBS[21369] = DropInfo.new(628, 2) -- Varka's Commander
MOBS[21370] = DropInfo.new(604, 2) -- Varka's Elite Guard
MOBS[21371] = DropInfo.new(627, 3) -- Varka's Head Magus
MOBS[21372] = DropInfo.new(604, 3) -- Varka's Head Guard
MOBS[21373] = DropInfo.new(649, 3) -- Varka's Prophet
MOBS[21374] = DropInfo.new(626, 3) -- Prophet's Guard
MOBS[21375] = DropInfo.new(626, 3) -- Disciple of Prophet

q:add_start_npc(WAHKAN)
q:add_talk_id(WAHKAN)
q:add_kill_id(J.keys(MOBS))
q:register_quest_items(VARKA_BADGE_SOLDIER, VARKA_BADGE_OFFICER, VARKA_BADGE_CAPTAIN)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  local htmltext = event
  if event == "31371-12a.html" or event == "31371-12b.html" or event == "31371-25.html" then
  elseif event == "31371-04.htm" then
    if q:has_at_least_one_quest_item(player, unpack(VARKA_MARKS)) then
      return "31371-03.htm"
    end
    qs:set_state(State.STARTED)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
    for i = 0, #KETRA_MARKS - 1 do
      if q:has_quest_items(player, KETRA_MARKS[i + 1]) then
        qs:set_cond(i + 2)
        return "31371-0" .. (i + 5) .. ".htm"
      end
    end
    qs:set_cond(1)
  elseif event == "31371-12.html" then
    if q:get_quest_items_count(player, VARKA_BADGE_SOLDIER) < SOLDIER_BADGE_COUNT[1] then
      return q:get_no_quest_msg(player)
    end
    q:take_items(player, VARKA_BADGE_SOLDIER, -1)
    q:give_items(player, KETRA_MARKS[1], 1)
    qs:set_cond(2, true)
  elseif event == "31371-15.html" then
    if q:get_quest_items_count(player, VARKA_BADGE_SOLDIER) < SOLDIER_BADGE_COUNT[2] or q:get_quest_items_count(player, VARKA_BADGE_OFFICER) < OFFICER_BADGE_COUNT[2] then
      return q:get_no_quest_msg(player)
    end
    q:take_items_ids(player, -1, VARKA_BADGE_SOLDIER, VARKA_BADGE_OFFICER, KETRA_MARKS[1])
    q:give_items(player, KETRA_MARKS[2], 1)
    qs:set_cond(3, true)
  elseif event == "31371-18.html" then
    if q:get_quest_items_count(player, VARKA_BADGE_SOLDIER) < SOLDIER_BADGE_COUNT[3] or q:get_quest_items_count(player, VARKA_BADGE_OFFICER) < OFFICER_BADGE_COUNT[3] or q:get_quest_items_count(player, VARKA_BADGE_CAPTAIN) < CAPTAIN_BADGE_COUNT[3] then
      return q:get_no_quest_msg(player)
    end
    q:take_items_ids(player, -1, VARKA_BADGE_SOLDIER, VARKA_BADGE_OFFICER, VARKA_BADGE_CAPTAIN, KETRA_MARKS[2])
    q:give_items(player, KETRA_MARKS[3], 1)
    qs:set_cond(4, true)
  elseif event == "31371-21.html" then
    if not q:has_quest_items(player, VALOR_TOTEM) or q:get_quest_items_count(player, VARKA_BADGE_SOLDIER) < SOLDIER_BADGE_COUNT[4] or q:get_quest_items_count(player, VARKA_BADGE_OFFICER) < OFFICER_BADGE_COUNT[4] or q:get_quest_items_count(player, VARKA_BADGE_CAPTAIN) < CAPTAIN_BADGE_COUNT[4] then
      return q:get_no_quest_msg(player)
    end
    q:take_items_ids(player, -1, VARKA_BADGE_SOLDIER, VARKA_BADGE_OFFICER, VARKA_BADGE_CAPTAIN, VALOR_TOTEM, KETRA_MARKS[3])
    q:give_items(player, KETRA_MARKS[4], 1)
    qs:set_cond(5, true)
  elseif event == "31371-26.html" then
    q:take_items_ids(player, -1, unpack(KETRA_MARKS))
    q:take_items_ids(player, -1, VALOR_TOTEM, WISDOM_TOTEM)
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
    htmltext = player:level() >= MIN_LEVEL and "31371-01.htm" or "31371-02.htm"
  elseif sw == State.STARTED then
    local sw2 = qs:get_cond()
    if sw2 == 1 then
      htmltext = q:get_quest_items_count(player, VARKA_BADGE_SOLDIER) >= SOLDIER_BADGE_COUNT[1] and "31371-11.html" or "31371-10.html"
    elseif sw2 == 2 then
      htmltext = q:has_quest_items(player, KETRA_MARKS[1]) and q:get_quest_items_count(player, VARKA_BADGE_SOLDIER) >= SOLDIER_BADGE_COUNT[2] and q:get_quest_items_count(player, VARKA_BADGE_OFFICER) >= OFFICER_BADGE_COUNT[2] and "31371-14.html" or "31371-13.html"
    elseif sw2 == 3 then
      htmltext = q:has_quest_items(player, KETRA_MARKS[2]) and q:get_quest_items_count(player, VARKA_BADGE_SOLDIER) >= SOLDIER_BADGE_COUNT[3] and q:get_quest_items_count(player, VARKA_BADGE_OFFICER) >= OFFICER_BADGE_COUNT[3] and q:get_quest_items_count(player, VARKA_BADGE_CAPTAIN) >= CAPTAIN_BADGE_COUNT[3] and "31371-17.html" or "31371-16.html"
    elseif sw2 == 4 then
      htmltext = q:has_quest_items(player, KETRA_MARKS[3], VALOR_TOTEM) and q:get_quest_items_count(player, VARKA_BADGE_SOLDIER) >= SOLDIER_BADGE_COUNT[4] and q:get_quest_items_count(player, VARKA_BADGE_OFFICER) >= OFFICER_BADGE_COUNT[4] and q:get_quest_items_count(player, VARKA_BADGE_CAPTAIN) >= CAPTAIN_BADGE_COUNT[4] and "31371-20.html" or "31371-19.html"
    elseif sw2 == 5 then
      if not q:has_quest_items(player, KETRA_MARKS[4]) or not q:has_quest_items(player, WISDOM_TOTEM) or q:get_quest_items_count(player, VARKA_BADGE_SOLDIER) < SOLDIER_BADGE_COUNT[5] or q:get_quest_items_count(player, VARKA_BADGE_OFFICER) < OFFICER_BADGE_COUNT[5] or q:get_quest_items_count(player, VARKA_BADGE_CAPTAIN) < CAPTAIN_BADGE_COUNT[5] then
        return "31371-22.html"
      end
      qs:set_cond(6, true)
      q:take_items_ids(player, -1, VARKA_BADGE_SOLDIER, VARKA_BADGE_OFFICER, VARKA_BADGE_CAPTAIN, WISDOM_TOTEM, KETRA_MARKS[4])
      q:give_items(player, KETRA_MARKS[5], 1)
      htmltext = "31371-23.html"
    elseif sw2 == 6 then
      if q:has_quest_items(player, KETRA_MARKS[5]) then
        htmltext = "31371-24.html"
      end
    end
  end
  return htmltext
end)

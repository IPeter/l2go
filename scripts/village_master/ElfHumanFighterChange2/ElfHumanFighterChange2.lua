-- The second class change of Human and Elven fighters, ported by hand from
-- Mobius village_master/ElfHumanFighterChange2. The masters turn a level 40
-- first class fighter into its second class for three marks (a trial, a
-- testimony and a test), with 15 Shadow Item Exchange Coupons (C grade).
-- Every master answers with the 30109-*.htm pages.
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/ElfHumanFighterChange2/.

local cm = require("lib.class_master")

local s = script.new("ElfHumanFighterChange2")

local NPCS = {
  30109, -- Hannavalt
  30187, -- Klaus
  30689, -- Siria
  30849, -- Sedrick
  30900, -- Marcus
  31276, -- Bernhard
  31321, -- Siegmund
  31965, -- Hector
}

-- Marks
local MARK_OF_CHALLENGER = 2627
local MARK_OF_DUTY = 2633
local MARK_OF_SEEKER = 2673
local MARK_OF_TRUST = 2734
local MARK_OF_DUELIST = 2762
local MARK_OF_SEARCHER = 2809
local MARK_OF_HEALER = 2820
local MARK_OF_LIFE = 3140
local MARK_OF_CHAMPION = 3276
local MARK_OF_SAGITTARIUS = 3293
local MARK_OF_WITCHCRAFT = 3307

-- The pages the links may ask for: 30109-02.htm to 30109-36.htm.
local PAGES = cm.numbered({}, { 30109 }, 2, 36, ".htm")

-- The classes, by event. Class IDs are numbers: the ClassId table is filled
-- only when the server runs the handlers.
local TRANSFERS = {
  ["2"] = { class = 2, from = 1, items = { MARK_OF_CHALLENGER, MARK_OF_TRUST, MARK_OF_DUELIST }, low = 40, low_no = 41, done = 42, no_proof = 43 }, -- Gladiator from Warrior
  ["3"] = { class = 3, from = 1, items = { MARK_OF_CHALLENGER, MARK_OF_TRUST, MARK_OF_CHAMPION }, low = 44, low_no = 45, done = 46, no_proof = 47 }, -- Warlord from Warrior
  ["5"] = { class = 5, from = 4, items = { MARK_OF_DUTY, MARK_OF_TRUST, MARK_OF_HEALER }, low = 48, low_no = 49, done = 50, no_proof = 51 }, -- Paladin from Knight
  ["6"] = { class = 6, from = 4, items = { MARK_OF_DUTY, MARK_OF_TRUST, MARK_OF_WITCHCRAFT }, low = 52, low_no = 53, done = 54, no_proof = 55 }, -- Dark Avenger from Knight
  ["8"] = { class = 8, from = 7, items = { MARK_OF_SEEKER, MARK_OF_TRUST, MARK_OF_SEARCHER }, low = 56, low_no = 57, done = 58, no_proof = 59 }, -- Treasure Hunter from Rogue
  ["9"] = { class = 9, from = 7, items = { MARK_OF_SEEKER, MARK_OF_TRUST, MARK_OF_SAGITTARIUS }, low = 60, low_no = 61, done = 62, no_proof = 63 }, -- Hawkeye from Rogue
  ["20"] = { class = 20, from = 19, items = { MARK_OF_DUTY, MARK_OF_LIFE, MARK_OF_HEALER }, low = 64, low_no = 65, done = 66, no_proof = 67 }, -- Temple Knight from Elven Knight
  ["21"] = { class = 21, from = 19, items = { MARK_OF_CHALLENGER, MARK_OF_LIFE, MARK_OF_DUELIST }, low = 68, low_no = 69, done = 70, no_proof = 71 }, -- Swordsinger from Elven Knight
  ["23"] = { class = 23, from = 22, items = { MARK_OF_SEEKER, MARK_OF_LIFE, MARK_OF_SEARCHER }, low = 72, low_no = 73, done = 74, no_proof = 75 }, -- Plains Walker from Elven Scout
  ["24"] = { class = 24, from = 22, items = { MARK_OF_SEEKER, MARK_OF_LIFE, MARK_OF_SAGITTARIUS }, low = 76, low_no = 77, done = 78, no_proof = 79 }, -- Silver Ranger from Elven Scout
}

-- The class lists, by the first class and its second classes: Warrior,
-- Knight, Rogue, Elven Knight and Elven Scout lines.
local CLASS_LISTS = {
  [1] = "30109-02.htm", [2] = "30109-02.htm", [3] = "30109-02.htm",
  [4] = "30109-09.htm", [5] = "30109-09.htm", [6] = "30109-09.htm",
  [7] = "30109-16.htm", [8] = "30109-16.htm", [9] = "30109-16.htm",
  [19] = "30109-23.htm", [20] = "30109-23.htm", [21] = "30109-23.htm",
  [22] = "30109-30.htm", [23] = "30109-30.htm", [24] = "30109-30.htm",
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

local function class_change_requested(player, t)
  if player:is_in_category(Category.THIRD_CLASS_GROUP) then
    return "30109-39.htm" -- fnYouAreThirdClass
  elseif player:class_id() == t.from then
    return cm.transfer(s, player, t, 40, cm.COUPON_C_GRADE, 30109, ".htm")
  end
  return nil
end

s:on_event(function(event, npc, player)
  if PAGES[event] then
    return event
  end
  local t = TRANSFERS[event]
  if t ~= nil then
    return class_change_requested(player, t)
  end
  return nil
end)

s:on_talk(function(npc, player)
  local fighter = player:is_in_category(Category.FIGHTER_GROUP)
  local fall = player:is_in_category(Category.HUMAN_FALL_CLASS) or player:is_in_category(Category.ELF_FALL_CLASS)
  if fighter and player:is_in_category(Category.FOURTH_CLASS_GROUP) and fall then
    return "30109-01.htm" -- fnYouAreFourthClass
  elseif fighter and fall then
    return CLASS_LISTS[player:class_id()] or "30109-37.htm" -- fnClassList1..5, fnYouAreFirstClass
  end
  return "30109-38.htm" -- fnClassMismatch
end)

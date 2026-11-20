-- The second class change of Orcs, ported by hand from Mobius
-- village_master/OrcChange2. The masters turn a level 40 Orc Raider into a
-- Destroyer, an Orc Monk into a Tyrant, an Orc Shaman into an Overlord or
-- Warcryer, for three marks, with 15 Shadow Item Exchange Coupons (C grade).
-- Every master answers with the 30513-*.htm pages.
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/OrcChange2/.

local cm = require("lib.class_master")

local s = script.new("OrcChange2")

local NPCS = {
  30513, -- Penatus
  30681, -- Karia
  30704, -- Garvarentz
  30865, -- Ladanza
  30913, -- Tushku
  31288, -- Aklan
  31326, -- Lambac
  31336, -- Rahorakti
  31977, -- Shaka
}

-- Marks
local MARK_OF_CHALLENGER = 2627
local MARK_OF_PILGRIM = 2721
local MARK_OF_DUELIST = 2762
local MARK_OF_WARSPIRIT = 2879
local MARK_OF_GLORY = 3203
local MARK_OF_CHAMPION = 3276
local MARK_OF_LORD = 3390

-- The pages the links may ask for (Mobius lists no 30513-06.htm here).
local PAGES = cm.numbered(cm.numbered({}, { 30513 }, 3, 5, ".htm"), { 30513 }, 7, 16, ".htm")

-- The classes, by event. Class IDs are numbers: the ClassId table is filled
-- only when the server runs the handlers.
local TRANSFERS = {
  ["46"] = { class = 46, from = 45, items = { MARK_OF_CHALLENGER, MARK_OF_GLORY, MARK_OF_CHAMPION }, low = 20, low_no = 21, done = 22, no_proof = 23 }, -- Destroyer from Orc Raider
  ["48"] = { class = 48, from = 47, items = { MARK_OF_CHALLENGER, MARK_OF_GLORY, MARK_OF_DUELIST }, low = 24, low_no = 25, done = 26, no_proof = 27 }, -- Tyrant from Orc Monk
  ["51"] = { class = 51, from = 50, items = { MARK_OF_PILGRIM, MARK_OF_GLORY, MARK_OF_LORD }, low = 28, low_no = 29, done = 30, no_proof = 31 }, -- Overlord from Orc Shaman
  ["52"] = { class = 52, from = 50, items = { MARK_OF_PILGRIM, MARK_OF_GLORY, MARK_OF_WARSPIRIT }, low = 32, low_no = 33, done = 34, no_proof = 35 }, -- Warcryer from Orc Shaman
}

-- The class lists: the Orc Raider (45, 46), Orc Monk (47, 48) and Orc
-- Shaman (50 to 52) lines.
local CLASS_LISTS = {
  [45] = "30513-02.htm", [46] = "30513-02.htm",
  [47] = "30513-06.htm", [48] = "30513-06.htm",
  [50] = "30513-10.htm", [51] = "30513-10.htm", [52] = "30513-10.htm",
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

local function class_change_requested(player, t)
  if player:is_in_category(Category.THIRD_CLASS_GROUP) then
    return "30513-19.htm" -- fnYouAreThirdClass
  elseif player:class_id() == t.from then
    return cm.transfer(s, player, t, 40, cm.COUPON_C_GRADE, 30513, ".htm")
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
  local orc = player:is_in_category(Category.ORC_MALL_CLASS) or player:is_in_category(Category.ORC_FALL_CLASS)
  if player:is_in_category(Category.FOURTH_CLASS_GROUP) and orc then
    return "30513-01.htm" -- fnYouAreFourthClass
  elseif orc then
    return CLASS_LISTS[player:class_id()] or "30513-17.htm" -- fnClassList1..3, fnYouAreFirstClass
  end
  return "30513-18.htm" -- fnClassMismatch
end)

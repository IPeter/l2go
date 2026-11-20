-- The first class change of Human and Elven fighters, ported by hand from
-- Mobius village_master/ElfHumanFighterChange1. The masters (Pabris, Rains,
-- Ramos, Schule) list the classes; at level 20 the path quest's mark turns
-- a Fighter into a Warrior, Knight or Rogue, an Elven Fighter into an Elven
-- Knight or Elven Scout, with 15 Shadow Item Exchange Coupons (D grade).
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/ElfHumanFighterChange1/.

local cm = require("lib.class_master")

local s = script.new("ElfHumanFighterChange1")

local NPCS = {
  30066, -- Pabris
  30288, -- Rains
  30373, -- Ramos
  32094, -- Schule
}

-- The pages the links may ask for: <npc>-01.htm to <npc>-17.htm.
local PAGES = cm.numbered({}, NPCS, 1, 17, ".htm")

-- The classes, by event: the class it comes from, the proof and the pages
-- (low level with and without the proof, after the change, no proof).
-- Class IDs are numbers here: the ClassId table is filled only when the
-- server runs the handlers.
local TRANSFERS = {
  ["1"] = { class = 1, from = 0, items = { 1145 }, low = 21, low_no = 22, done = 23, no_proof = 24 }, -- Warrior from Fighter: Medallion of Warrior
  ["4"] = { class = 4, from = 0, items = { 1161 }, low = 25, low_no = 26, done = 27, no_proof = 28 }, -- Knight from Fighter: Sword of Ritual
  ["7"] = { class = 7, from = 0, items = { 1190 }, low = 29, low_no = 30, done = 31, no_proof = 32 }, -- Rogue from Fighter: Bezique's Recommendation
  ["19"] = { class = 19, from = 18, items = { 1204 }, low = 33, low_no = 34, done = 35, no_proof = 36 }, -- Elven Knight from Elven Fighter: Elven Knight Brooch
  ["22"] = { class = 22, from = 18, items = { 1217 }, low = 37, low_no = 38, done = 39, no_proof = 40 }, -- Elven Scout from Elven Fighter: Reisa's Recommendation
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

local function class_change_requested(player, npc, t)
  if player:is_in_category(Category.SECOND_CLASS_GROUP) then
    return npc:id() .. "-19.htm" -- fnYouAreSecondClass
  elseif player:is_in_category(Category.THIRD_CLASS_GROUP) then
    return npc:id() .. "-20.htm" -- fnYouAreThirdClass
  elseif player:is_in_category(Category.FOURTH_CLASS_GROUP) then
    return "30066-41.htm" -- fnYouAreFourthClass
  elseif player:class_id() == t.from then
    return cm.transfer(s, player, t, 20, cm.COUPON_D_GRADE, npc:id(), ".htm")
  end
  return nil
end

s:on_event(function(event, npc, player)
  if PAGES[event] then
    return event
  end
  local t = TRANSFERS[event]
  if t ~= nil and npc ~= nil then
    return class_change_requested(player, npc, t)
  end
  return nil
end)

s:on_talk(function(npc, player)
  local race = player:race()
  if player:is_in_category(Category.FIGHTER_GROUP) and (race == Race.HUMAN or race == Race.ELF) then
    if race == Race.HUMAN then
      return npc:id() .. "-01.htm" -- fnClassList1
    end
    return npc:id() .. "-11.htm" -- fnClassList2
  end
  return npc:id() .. "-18.htm" -- fnClassMismatch
end)

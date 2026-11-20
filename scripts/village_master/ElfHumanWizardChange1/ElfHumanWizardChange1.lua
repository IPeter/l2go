-- The first class change of Human and Elven mystics, ported by hand from
-- Mobius village_master/ElfHumanWizardChange1. The masters (Levian,
-- Sylvain, Raymond, Marie, Celes) turn a level 20 Mage into a Wizard or
-- Cleric, an Elven Mage into an Elven Wizard or Oracle, for the path
-- quest's mark, with 15 Shadow Item Exchange Coupons (D grade).
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/ElfHumanWizardChange1/.

local cm = require("lib.class_master")

local s = script.new("ElfHumanWizardChange1")

local NPCS = {
  30037, -- Levian
  30070, -- Sylvain
  30289, -- Raymond
  32095, -- Marie
  32098, -- Celes
}

-- The pages the links may ask for: <npc>-01.htm to <npc>-14.htm.
local PAGES = cm.numbered({}, NPCS, 1, 14, ".htm")

-- The classes, by event (class IDs are numbers: the ClassId table is
-- filled only when the server runs the handlers).
local TRANSFERS = {
  ["11"] = { class = 11, from = 10, items = { 1292 }, low = 18, low_no = 19, done = 20, no_proof = 21 }, -- Wizard from Mage: Bead of Season
  ["15"] = { class = 15, from = 10, items = { 1201 }, low = 22, low_no = 23, done = 24, no_proof = 25 }, -- Cleric from Mage: Mark of Faith
  ["26"] = { class = 26, from = 25, items = { 1230 }, low = 26, low_no = 27, done = 28, no_proof = 29 }, -- Elven Wizard from Elven Mage: Eternity Diamond
  ["29"] = { class = 29, from = 25, items = { 1235 }, low = 30, low_no = 31, done = 32, no_proof = 33 }, -- Oracle from Elven Mage: Leaf of Oracle
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

local function class_change_requested(player, npc, t)
  if player:is_in_category(Category.SECOND_CLASS_GROUP) then
    return npc:id() .. "-16.htm" -- fnYouAreSecondClass
  elseif player:is_in_category(Category.THIRD_CLASS_GROUP) then
    return npc:id() .. "-17.htm" -- fnYouAreThirdClass
  elseif player:is_in_category(Category.FOURTH_CLASS_GROUP) then
    return "30037-34.htm" -- fnYouAreFourthClass
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
  if player:is_in_category(Category.MAGE_GROUP) and (race == Race.HUMAN or race == Race.ELF) then
    if race == Race.HUMAN then
      return npc:id() .. "-01.htm" -- fnClassList1
    end
    return npc:id() .. "-08.htm" -- fnClassList2
  end
  return npc:id() .. "-15.htm" -- fnClassMismatch
end)

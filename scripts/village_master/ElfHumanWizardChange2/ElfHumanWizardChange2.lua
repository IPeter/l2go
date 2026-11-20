-- The second class change of Human and Elven wizards, ported by hand from
-- Mobius village_master/ElfHumanWizardChange2. The masters turn a level 40
-- Wizard or Elven Wizard into its second class for three marks, with 15
-- Shadow Item Exchange Coupons (C grade). Every master answers with the
-- 30115-*.htm pages.
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/ElfHumanWizardChange2/.

local cm = require("lib.class_master")

local s = script.new("ElfHumanWizardChange2")

local NPCS = {
  30115, -- Jurek
  30174, -- Arkenias
  30176, -- Valleria
  30694, -- Scraide
  30854, -- Drikiyan
  31331, -- Valdis
  31755, -- Halaster
  31996, -- Javier
}

-- Marks
local MARK_OF_SCHOLAR = 2674
local MARK_OF_TRUST = 2734
local MARK_OF_MAGUS = 2840
local MARK_OF_WITCHCRAFT = 3307
local MARK_OF_SUMMONER = 3336
local MARK_OF_LIFE = 3140

-- The pages the links may ask for: 30115-02.htm to 30115-18.htm.
local PAGES = cm.numbered({}, { 30115 }, 2, 18, ".htm")

-- The classes, by event. Class IDs are numbers: the ClassId table is filled
-- only when the server runs the handlers.
local TRANSFERS = {
  ["12"] = { class = 12, from = 11, items = { MARK_OF_SCHOLAR, MARK_OF_TRUST, MARK_OF_MAGUS }, low = 22, low_no = 23, done = 24, no_proof = 25 }, -- Sorcerer from Wizard
  ["13"] = { class = 13, from = 11, items = { MARK_OF_SCHOLAR, MARK_OF_TRUST, MARK_OF_WITCHCRAFT }, low = 26, low_no = 27, done = 28, no_proof = 29 }, -- Necromancer from Wizard
  ["14"] = { class = 14, from = 11, items = { MARK_OF_SCHOLAR, MARK_OF_TRUST, MARK_OF_SUMMONER }, low = 30, low_no = 31, done = 32, no_proof = 33 }, -- Warlock from Wizard
  ["27"] = { class = 27, from = 26, items = { MARK_OF_SCHOLAR, MARK_OF_LIFE, MARK_OF_MAGUS }, low = 34, low_no = 35, done = 36, no_proof = 37 }, -- Spellsinger from Elven Wizard
  ["28"] = { class = 28, from = 26, items = { MARK_OF_SCHOLAR, MARK_OF_LIFE, MARK_OF_SUMMONER }, low = 38, low_no = 39, done = 40, no_proof = 41 }, -- Elemental Summoner from Elven Wizard
}

-- The class lists: the Wizard line (11 to 14) and the Elven Wizard line
-- (26 to 28).
local CLASS_LISTS = {
  [11] = "30115-02.htm", [12] = "30115-02.htm", [13] = "30115-02.htm", [14] = "30115-02.htm",
  [26] = "30115-12.htm", [27] = "30115-12.htm", [28] = "30115-12.htm",
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

local function class_change_requested(player, t)
  if player:is_in_category(Category.THIRD_CLASS_GROUP) then
    return "30115-21.htm" -- fnYouAreThirdClass
  elseif player:class_id() == t.from then
    return cm.transfer(s, player, t, 40, cm.COUPON_C_GRADE, 30115, ".htm")
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
  local wizard = player:is_in_category(Category.WIZARD_GROUP)
  local mall = player:is_in_category(Category.HUMAN_MALL_CLASS) or player:is_in_category(Category.ELF_MALL_CLASS)
  if wizard and player:is_in_category(Category.FOURTH_CLASS_GROUP) and mall then
    return "30115-01.htm" -- fnYouAreFourthClass
  elseif wizard and mall then
    return CLASS_LISTS[player:class_id()] or "30115-19.htm" -- fnClassList1/2, fnYouAreFirstClass
  end
  return "30115-20.htm" -- fnClassMismatch
end)

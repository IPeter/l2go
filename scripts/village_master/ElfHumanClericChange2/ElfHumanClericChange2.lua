-- The second class change of Human clerics and Elven oracles, ported by
-- hand from Mobius village_master/ElfHumanClericChange2. The masters turn a
-- level 40 Cleric into a Bishop or Prophet, an Oracle into an Elder, for
-- three marks, with 15 Shadow Item Exchange Coupons (C grade). Every master
-- answers with the 30120-*.htm pages.
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/ElfHumanClericChange2/.

local cm = require("lib.class_master")

local s = script.new("ElfHumanClericChange2")

local NPCS = {
  30120, -- Maximilian
  30191, -- Hollint
  30857, -- Orven
  30905, -- Squillari
  31279, -- Gregory
  31328, -- Innocentin
  31968, -- Baryl
}

-- Marks
local MARK_OF_PILGRIM = 2721
local MARK_OF_TRUST = 2734
local MARK_OF_HEALER = 2820
local MARK_OF_REFORMER = 2821
local MARK_OF_LIFE = 3140

-- The pages the links may ask for (Mobius lists no 30120-09.htm here).
local PAGES = cm.numbered(cm.numbered({}, { 30120 }, 2, 8, ".htm"), { 30120 }, 10, 12, ".htm")

-- The classes, by event. Class IDs are numbers: the ClassId table is filled
-- only when the server runs the handlers.
local TRANSFERS = {
  ["16"] = { class = 16, from = 15, items = { MARK_OF_PILGRIM, MARK_OF_TRUST, MARK_OF_HEALER }, low = 16, low_no = 17, done = 18, no_proof = 19 }, -- Bishop from Cleric
  ["17"] = { class = 17, from = 15, items = { MARK_OF_PILGRIM, MARK_OF_TRUST, MARK_OF_REFORMER }, low = 20, low_no = 21, done = 22, no_proof = 23 }, -- Prophet from Cleric
  ["30"] = { class = 30, from = 29, items = { MARK_OF_PILGRIM, MARK_OF_LIFE, MARK_OF_HEALER }, low = 24, low_no = 25, done = 26, no_proof = 27 }, -- Elder from Oracle
}

-- The class lists: the Cleric line (15 to 17) and the Oracle line (29, 30).
local CLASS_LISTS = {
  [15] = "30120-02.htm", [16] = "30120-02.htm", [17] = "30120-02.htm",
  [29] = "30120-09.htm", [30] = "30120-09.htm",
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

local function class_change_requested(player, t)
  if player:is_in_category(Category.THIRD_CLASS_GROUP) then
    return "30120-15.htm" -- fnYouAreThirdClass
  elseif player:class_id() == t.from then
    return cm.transfer(s, player, t, 40, cm.COUPON_C_GRADE, 30120, ".htm")
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
  local cleric = player:is_in_category(Category.CLERIC_GROUP)
  local call = player:is_in_category(Category.HUMAN_CALL_CLASS) or player:is_in_category(Category.ELF_CALL_CLASS)
  if cleric and player:is_in_category(Category.FOURTH_CLASS_GROUP) and call then
    return "30120-01.htm" -- fnYouAreFourthClass
  elseif cleric and call then
    return CLASS_LISTS[player:class_id()] or "30120-13.htm" -- fnClassList1/2, fnYouAreFirstClass
  end
  return "30120-14.htm" -- fnClassMismatch
end)

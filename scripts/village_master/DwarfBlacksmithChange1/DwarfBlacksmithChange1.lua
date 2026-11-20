-- The first class change of Dwarven blacksmiths, ported by hand from Mobius
-- village_master/DwarfBlacksmithChange1. The head blacksmiths (Tapoy,
-- Mendio, Opix, Bolin) turn a level 20 Dwarven Fighter into an Artisan for
-- the Final Pass Certificate, with 15 Shadow Item Exchange Coupons (D grade).
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/DwarfBlacksmithChange1/.

local cm = require("lib.class_master")

local s = script.new("DwarfBlacksmithChange1")

local NPCS = {
  30499, -- Tapoy
  30504, -- Mendio
  30595, -- Opix
  32093, -- Bolin
}

-- The pages the links may ask for: <npc>-01.htm to <npc>-04.htm.
local PAGES = cm.numbered({}, NPCS, 1, 4, ".htm")

local TRANSFERS = {
  -- Artisan (56) from Dwarven Fighter (53): Final Pass Certificate. Class
  -- IDs are numbers: the ClassId table is filled only when handlers run.
  ["56"] = { class = 56, from = 53, items = { 1635 }, low = 8, low_no = 9, done = 10, no_proof = 11 },
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

local function class_change_requested(player, npc, t)
  if player:is_in_category(Category.SECOND_CLASS_GROUP) then
    return npc:id() .. "-06.htm" -- fnYouAreSecondClass
  elseif player:is_in_category(Category.THIRD_CLASS_GROUP) then
    return npc:id() .. "-07.htm" -- fnYouAreThirdClass
  elseif player:is_in_category(Category.FOURTH_CLASS_GROUP) then
    return "30499-12.htm" -- fnYouAreFourthClass
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
  if player:is_in_category(Category.WARSMITH_GROUP) then
    return npc:id() .. "-01.htm" -- fnClassList1
  end
  return npc:id() .. "-05.htm" -- fnClassMismatch
end)

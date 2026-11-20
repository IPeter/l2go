-- The first class change of Dwarven warehouse keepers, ported by hand from
-- Mobius village_master/DwarfWarehouseChange1. The warehouse chiefs (Moke,
-- Rikadio, Ranspo, Alder) turn a level 20 Dwarven Fighter into a Scavenger
-- for the Ring of Raven, with 15 Shadow Item Exchange Coupons (D grade).
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/DwarfWarehouseChange1/.

local cm = require("lib.class_master")

local s = script.new("DwarfWarehouseChange1")

local NPCS = {
  30498, -- Moke
  30503, -- Rikadio
  30594, -- Ranspo
  32092, -- Alder
}

-- The pages the links may ask for: <npc>-01.htm to <npc>-04.htm.
local PAGES = cm.numbered({}, NPCS, 1, 4, ".htm")

local TRANSFERS = {
  -- Scavenger (54) from Dwarven Fighter (53): Ring of Raven. Class IDs are
  -- numbers: the ClassId table is filled only when handlers run.
  ["54"] = { class = 54, from = 53, items = { 1642 }, low = 8, low_no = 9, done = 10, no_proof = 11 },
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

local function class_change_requested(player, npc, t)
  if player:is_in_category(Category.SECOND_CLASS_GROUP) then
    return npc:id() .. "-06.htm" -- fnYouAreSecondClass
  elseif player:is_in_category(Category.THIRD_CLASS_GROUP) then
    return npc:id() .. "-07.htm" -- fnYouAreThirdClass
  elseif player:is_in_category(Category.FOURTH_CLASS_GROUP) then
    return "30498-12.htm" -- fnYouAreFourthClass
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
  if player:is_in_category(Category.BOUNTY_HUNTER_GROUP) then
    return npc:id() .. "-01.htm" -- fnClassList1
  end
  return npc:id() .. "-05.htm" -- fnClassMismatch
end)

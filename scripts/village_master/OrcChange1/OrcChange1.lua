-- The first class change of Orcs, ported by hand from Mobius
-- village_master/OrcChange1. The high prefects (Osborn, Drikus, Castor,
-- Finker) turn a level 20 Orc Fighter into an Orc Raider or Orc Monk, an
-- Orc Mystic into an Orc Shaman, for the path quest's mark, with 15 Shadow
-- Item Exchange Coupons (D grade).
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/OrcChange1/.

local cm = require("lib.class_master")

local s = script.new("OrcChange1")

local NPCS = {
  30500, -- Osborn
  30505, -- Drikus
  30508, -- Castor
  32097, -- Finker
}

-- The pages the links may ask for: <npc>-01.htm to <npc>-08.htm.
local PAGES = cm.numbered({}, NPCS, 1, 8, ".htm")

-- The classes, by event (class IDs are numbers: the ClassId table is
-- filled only when the server runs the handlers).
local TRANSFERS = {
  ["45"] = { class = 45, from = 44, items = { 1592 }, low = 11, low_no = 12, done = 14, no_proof = 13 }, -- Orc Raider from Orc Fighter: Mark of Raider
  ["47"] = { class = 47, from = 44, items = { 1615 }, low = 15, low_no = 16, done = 18, no_proof = 17 }, -- Orc Monk from Orc Fighter: Khavatari Totem
  ["50"] = { class = 50, from = 49, items = { 1631 }, low = 19, low_no = 20, done = 22, no_proof = 21 }, -- Orc Shaman from Orc Mystic: Mask of Medium
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

local function class_change_requested(player, npc, t)
  if player:is_in_category(Category.SECOND_CLASS_GROUP) then
    return npc:id() .. "-09.htm" -- fnYouAreSecondClass
  elseif player:is_in_category(Category.THIRD_CLASS_GROUP) then
    return npc:id() .. "-10.htm" -- fnYouAreThirdClass
  elseif player:is_in_category(Category.FOURTH_CLASS_GROUP) then
    return "30500-24.htm" -- fnYouAreFourthClass
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
  if player:race() == Race.ORC then
    if player:is_in_category(Category.FIGHTER_GROUP) then
      return npc:id() .. "-01.htm" -- fnClassList1
    elseif player:is_in_category(Category.MAGE_GROUP) then
      return npc:id() .. "-06.htm" -- fnClassList2
    end
    return nil
  end
  return npc:id() .. "-23.htm" -- fnClassMismatch
end)

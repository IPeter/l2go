-- The second class change of Dwarven blacksmiths, ported by hand from
-- Mobius village_master/DwarfBlacksmithChange2. The masters turn a level 40
-- Artisan into a Warsmith for three marks, with 15 Shadow Item Exchange
-- Coupons (C grade). Every master answers with the 30512-*.htm pages.
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/DwarfBlacksmithChange2/.

local cm = require("lib.class_master")

local s = script.new("DwarfBlacksmithChange2")

local NPCS = {
  30512, -- Kusto
  30677, -- Flutter
  30687, -- Vergara
  30847, -- Ferris
  30897, -- Roman
  31272, -- Noel
  31317, -- Lombert
  31961, -- Newyear
}

-- Marks
local MARK_OF_MAESTRO = 2867
local MARK_OF_GUILDSMAN = 3119
local MARK_OF_PROSPERITY = 3238

-- The pages the links may ask for: 30512-03.htm to 30512-05.htm.
local PAGES = cm.numbered({}, { 30512 }, 3, 5, ".htm")

local TRANSFERS = {
  -- Warsmith (57) from Artisan (56). Class IDs are numbers: the ClassId
  -- table is filled only when handlers run.
  ["57"] = { class = 57, from = 56, items = { MARK_OF_GUILDSMAN, MARK_OF_PROSPERITY, MARK_OF_MAESTRO }, low = 9, low_no = 10, done = 11, no_proof = 12 },
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

local function class_change_requested(player, t)
  if player:is_in_category(Category.THIRD_CLASS_GROUP) then
    return "30512-08.htm" -- fnYouAreThirdClass
  elseif player:class_id() == t.from then
    return cm.transfer(s, player, t, 40, cm.COUPON_C_GRADE, 30512, ".htm")
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
  if player:is_in_category(Category.FOURTH_CLASS_GROUP) then
    return "30512-01.htm" -- fnYouAreFourthClass
  elseif player:is_in_category(Category.WARSMITH_GROUP) then
    local class = player:class_id()
    if class == ClassId.ARTISAN or class == ClassId.WARSMITH then
      return "30512-02.htm" -- fnClassList1
    end
    return "30512-06.htm" -- fnYouAreFirstClass
  end
  return "30512-07.htm" -- fnClassMismatch
end)

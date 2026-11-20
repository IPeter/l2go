-- The second class change of Dwarven warehouse keepers, ported by hand from
-- Mobius village_master/DwarfWarehouseChange2. The masters turn a level 40
-- Scavenger into a Bounty Hunter for three marks, with 15 Shadow Item
-- Exchange Coupons (C grade). Every master answers with the 30511-*.htm
-- pages.
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/DwarfWarehouseChange2/.

local cm = require("lib.class_master")

local s = script.new("DwarfWarehouseChange2")

local NPCS = {
  30511, -- Gesto
  30676, -- Croop
  30685, -- Baxt
  30845, -- Klump
  30894, -- Natools
  31269, -- Mona
  31314, -- Donal
  31958, -- Yasheni
}

-- Marks
local MARK_OF_SEARCHER = 2809
local MARK_OF_GUILDSMAN = 3119
local MARK_OF_PROSPERITY = 3238

-- The pages the links may ask for: 30511-03.htm to 30511-05.htm.
local PAGES = cm.numbered({}, { 30511 }, 3, 5, ".htm")

local TRANSFERS = {
  -- Bounty Hunter (55) from Scavenger (54). Class IDs are numbers: the
  -- ClassId table is filled only when handlers run.
  ["55"] = { class = 55, from = 54, items = { MARK_OF_GUILDSMAN, MARK_OF_PROSPERITY, MARK_OF_SEARCHER }, low = 9, low_no = 10, done = 11, no_proof = 12 },
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

local function class_change_requested(player, t)
  if player:is_in_category(Category.THIRD_CLASS_GROUP) then
    return "30511-08.htm" -- fnYouAreThirdClass
  elseif player:class_id() == t.from then
    return cm.transfer(s, player, t, 40, cm.COUPON_C_GRADE, 30511, ".htm")
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
  local bounty_hunter = player:is_in_category(Category.BOUNTY_HUNTER_GROUP)
  if player:is_in_category(Category.FOURTH_CLASS_GROUP) and bounty_hunter then
    return "30511-01.htm" -- fnYouAreFourthClass
  elseif bounty_hunter then
    local class = player:class_id()
    if class == ClassId.SCAVENGER or class == ClassId.BOUNTY_HUNTER then
      return "30511-02.htm" -- fnClassList1
    end
    return "30511-06.htm" -- fnYouAreFirstClass
  end
  return "30511-07.htm" -- fnClassMismatch
end)

-- The castle warehouse keepers, ported from Mobius ai/others/CastleWarehouse.
--
-- The pages lead to the private and clan warehouse windows (the game's
-- npc_<objectId>_DepositP and the other warehouse links of a Warehouse NPC).
-- The lord can also receive the clan's Blood Alliances and exchange one for
-- 30 Blood Oaths; the clan's Blood Alliance count is not kept (Mobius
-- Clan.getBloodAllianceCount; siege rewards of later chronicles), so it is
-- always 0 here.
--
-- The pages are Mobius's, served from
-- data/html/scripts/ai/others/CastleWarehouse/.

local s = script.new("CastleWarehouse")

local NPCS = {
  35099, -- Warehouse Keeper (Gludio)
  35141, -- Warehouse Keeper (Dion)
  35183, -- Warehouse Keeper (Giran)
  35225, -- Warehouse Keeper (Oren)
  35273, -- Warehouse Keeper (Aden)
  35315, -- Warehouse Keeper (Inadril)
  35362, -- Warehouse Keeper (Goddard)
  35508, -- Warehouse Keeper (Rune)
  35554, -- Warehouse Keeper (Schuttgart)
}

local DIR = "data/html/scripts/ai/others/CastleWarehouse/"

local BLOOD_OATH = 9910
local BLOOD_ALLIANCE = 9911

local PAGES = {
  ["warehouse-01.html"] = true,
  ["warehouse-02.html"] = true,
  ["warehouse-03.html"] = true,
}

-- blood_alliance_count is Clan.getBloodAllianceCount: not kept, 0.
local function blood_alliance_count(clan)
  return 0
end

s:on_event(function(event, npc, player)
  if npc == nil or player == nil then
    return nil
  end
  if PAGES[event] then
    return DIR .. event
  elseif event == "warehouse-04.html" then
    if not npc:is_my_lord(player) then
      return DIR .. "warehouse-no.html"
    end
    return DIR .. event, { blood = tostring(blood_alliance_count(player:clan())) }
  elseif event == "Receive" then
    if not npc:is_my_lord(player) then
      return DIR .. "warehouse-no.html"
    end
    local count = blood_alliance_count(player:clan())
    if count == 0 then
      return DIR .. "warehouse-05.html"
    end
    s:give_items(player, BLOOD_ALLIANCE, count)
    return DIR .. "warehouse-06.html"
  elseif event == "Exchange" then
    if not npc:is_my_lord(player) then
      return DIR .. "warehouse-no.html"
    elseif not s:has_quest_items(player, BLOOD_ALLIANCE) then
      return DIR .. "warehouse-08.html"
    end
    s:take_items(player, BLOOD_ALLIANCE, 1)
    s:give_items(player, BLOOD_OATH, 30)
    return DIR .. "warehouse-07.html"
  end
  return nil
end)

s:on_first_talk(NPCS, function(npc, player)
  return DIR .. "warehouse-01.html"
end)

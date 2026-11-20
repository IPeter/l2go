-- The dialogs of the headmasters of the newbie villages about the first
-- class change, ported by hand from Mobius
-- village_master/FirstClassTransferTalk. None of them changes a class; they
-- only talk about it, by the player's race and class level.
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/FirstClassTransferTalk/.

local s = script.new("FirstClassTransferTalk")

-- The masters and the race each one talks to.
local MASTERS = {
  [30026] = Race.HUMAN, -- Blitz, TI Fighter Guild Head Master
  [30031] = Race.HUMAN, -- Biotin, TI Einhasad Temple High Priest
  [30154] = Race.ELF, -- Asterios, Elven Village Tetrarch
  [30358] = Race.DARK_ELF, -- Thifiell, Dark Elf Village Tetrarch
  [30565] = Race.ORC, -- Kakai, Orc Village Flame Lord
  [30520] = Race.DWARF, -- Reed, Dwarven Village Warehouse Chief
  [30525] = Race.DWARF, -- Bronk, Dwarven Village Head Blacksmith
}

local ids = {}
for id in pairs(MASTERS) do
  table.insert(ids, id)
end
table.sort(ids)
s:add_start_npc(ids)
s:add_talk_id(ids)

s:on_event(function(event, npc, player)
  return event
end)

-- by_level picks the page of a class level: the class list at level 0
-- (choose), the "already changed" pages after that.
local function by_level(level, choose)
  if level == 0 then
    return choose
  elseif level == 1 then
    return "transfer_1.html"
  end
  return "transfer_2.html"
end

s:on_talk(function(npc, player)
  local prefix = npc:id() .. "_"
  local race = MASTERS[npc:id()]
  if race ~= player:race() then
    return prefix .. "no.html"
  end
  local level = ClassId.level(player:class_id())
  if race == Race.HUMAN then
    local choose
    if player:is_mage_class() then
      choose = npc:is_type("VillageMasterPriest") and "mystic.html" or "no.html"
    else
      choose = npc:is_type("VillageMasterFighter") and "fighter.html" or "no.html"
    end
    return prefix .. by_level(level, choose)
  elseif race == Race.ELF or race == Race.DARK_ELF or race == Race.ORC then
    return prefix .. by_level(level, player:is_mage_class() and "mystic.html" or "fighter.html")
  elseif race == Race.DWARF then
    return prefix .. by_level(level, "fighter.html")
  end
  return prefix .. "no.html"
end)

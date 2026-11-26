-- The manor managers, ported from Mobius ai/others/ManorManager.
--
-- The first talk shows the manager's page (the lord of the manager's castle
-- gets the lord's greeting); the help links show the explanations. The
-- page's manor_menu_select links (the seed shop, the crop sale, the manor
-- information, the harvester buylist) are the game's manor windows, as in
-- Mobius's ON_NPC_MANOR_BYPASS listener. With the manor off the managers
-- show the default NPC page.
--
-- The pages are Mobius's, served from
-- data/html/scripts/ai/others/ManorManager/.

local s = script.new("ManorManager")

-- Mobius ManorManager.NPC.
local NPCS = {
  35644, 35645, 35319, 35366, 35512, 35558, 35229,
  35230, 35231, 35277, 35103, 35145, 35187,
}

local DIR = "data/html/scripts/ai/others/ManorManager/"

local HELP = {
  ["manager-help-01.htm"] = true,
  ["manager-help-02.htm"] = true,
  ["manager-help-03.htm"] = true,
}

s:on_event(function(event, npc, player)
  if HELP[event] then
    return DIR .. event
  end
  return nil
end)

s:on_first_talk(NPCS, function(npc, player)
  if not manor.is_enabled() then
    return "data/html/npcdefault.htm"
  end
  if manor.is_lord(npc, player) then
    return DIR .. "manager-lord.htm"
  end
  return DIR .. "manager.htm"
end)

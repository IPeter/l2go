-- The Symbol Makers, ported from Mobius ai/others/SymbolMaker.
--
-- Every Symbol Maker greets with symbol_maker.htm, whose links go to this
-- script ("Script SymbolMaker <event>"):
--   * a page name (the explanations) shows that page;
--   * "Draw" opens the window of the dyes the player can draw
--     (HennaEquipList), "Remove" the window of the dyes it wears
--     (HennaRemoveList). Drawing and removing then go through the henna
--     packets, which the game allows only next to a Symbol Maker.

local symbol_maker = script.new("SymbolMaker")

local SYMBOL_MAKERS = {
  31046, -- Marsden
  31047, -- Kell
  31048, -- McDermott
  31049, -- Pepper
  31050, -- Thora
  31051, -- Keach
  31052, -- Heid
  31053, -- Kidder
  31264, -- Olsun
  31308, -- Achim
  31953, -- Rankar
}

-- The pages the links may ask for.
local PAGES = {
  ["symbol_maker.htm"] = true,
  ["symbol_maker-1.htm"] = true,
  ["symbol_maker-2.htm"] = true,
  ["symbol_maker-3.htm"] = true,
}

symbol_maker:on_first_talk(SYMBOL_MAKERS, function(npc, player)
  return "symbol_maker.htm"
end)

symbol_maker:on_event(function(event, npc, player)
  if PAGES[event] then
    return event
  elseif event == "Draw" then
    player:show_henna_equip_list()
  elseif event == "Remove" then
    player:show_henna_remove_list()
  end
  return nil
end)

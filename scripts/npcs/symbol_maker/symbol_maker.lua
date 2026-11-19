-- The Symbol Makers, ported from Mobius ai/others/SymbolMaker.
--
-- Every Symbol Maker greets with symbol_maker.htm, whose links go to this
-- script ("Script SymbolMaker <event>"):
--   * a page name (the explanations) shows that page, as in Mobius;
--   * "Draw" and "Remove" open the symbol (henna) windows in Mobius. Symbols
--     come in a later milestone; until then they answer that the service is
--     not available yet.

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

local NOT_YET = [[<html><body>Symbol Maker:<br>
I am sorry, drawing and removing symbols is not available yet.<br><br>
<a action="bypass Script SymbolMaker symbol_maker.htm">Back</a>
</body></html>]]

symbol_maker:on_first_talk(SYMBOL_MAKERS, function(npc, player)
  return "symbol_maker.htm"
end)

symbol_maker:on_event(function(event, npc, player)
  if PAGES[event] then
    return event
  elseif event == "Draw" or event == "Remove" then
    return NOT_YET
  end
  return nil
end)

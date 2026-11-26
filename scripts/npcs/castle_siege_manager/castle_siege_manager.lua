-- The castle siege managers, ported from Mobius ai/others/CastleSiegeManager.
--
-- The lord hears that the castle is secure (CastleSiegeManager-01.html) or
-- under siege (CastleSiegeManager.html); during a siege everyone else gets
-- CastleSiegeManager-02.html, and outside one the siege window (SiegeInfo),
-- where clans register as attackers or defenders.
--
-- Mobius also lists the siege managers of the two contestable clan halls
-- (Fortress of the Dead, Devastated Castle), which show their hall's siege
-- window. Clan hall sieges come with M11j; until then those two answer that
-- the function is not available.
--
-- The pages are Mobius's, served from
-- data/html/scripts/ai/others/CastleSiegeManager/.

local s = script.new("CastleSiegeManager")

local SIEGE_MANAGERS = {
  35104, -- Gludio Castle
  35146, -- Dion Castle
  35188, -- Giran Castle
  35232, -- Oren Castle
  35278, -- Aden Castle
  35320, -- Innadril Castle
  35367, -- Goddard Castle
  35513, -- Rune Castle
  35559, -- Schuttgart Castle
  35639, -- Fortress of the Dead
  35420, -- Devastated Castle
}

-- The siege managers of the contestable clan halls (Mobius
-- npc.getConquerableHall() != null).
local HALL_MANAGERS = {
  [35639] = true,
  [35420] = true,
}

local DIR = "data/html/scripts/ai/others/CastleSiegeManager/"

local NOT_AVAILABLE = "This function is not available yet."

-- is_in_siege: the castle's siege is in progress (a clan hall's siege is
-- never in progress until M11j).
local function is_in_siege(castle)
  return castle ~= nil and castle:is_siege_in_progress()
end

s:on_first_talk(SIEGE_MANAGERS, function(npc, player)
  local castle = npc:castle()
  if castle == nil then
    return nil
  end
  if player:is_clan_leader() and player:clan_id() == castle:owner_id() then
    if is_in_siege(castle) then
      return DIR .. "CastleSiegeManager.html"
    end
    return DIR .. "CastleSiegeManager-01.html"
  elseif is_in_siege(castle) then
    return DIR .. "CastleSiegeManager-02.html"
  elseif HALL_MANAGERS[npc:id()] then
    return NOT_AVAILABLE
  end
  castle:show_siege_info(player)
  return ""
end)

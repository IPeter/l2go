-- The castle blacksmiths, ported from Mobius ai/others/CastleBlacksmith.
--
-- A GM, the lord, or a member of the owner clan with the manor privilege
-- gets <npc ID>-01.html and its crop page <npc ID>-02.html, whose link opens
-- the blacksmith's multisell (the game's npc_<objectId>_multisell);
-- everyone else gets no.html.
--
-- The pages are Mobius's, served from
-- data/html/scripts/ai/others/CastleBlacksmith/.

local s = script.new("CastleBlacksmith")

local NPCS = {
  35098, -- Blacksmith (Gludio)
  35140, -- Blacksmith (Dion)
  35182, -- Blacksmith (Giran)
  35224, -- Blacksmith (Oren)
  35272, -- Blacksmith (Aden)
  35314, -- Blacksmith (Innadril)
  35361, -- Blacksmith (Goddard)
  35507, -- Blacksmith (Rune)
  35553, -- Blacksmith (Schuttgart)
}

local DIR = "data/html/scripts/ai/others/CastleBlacksmith/"

local function has_rights(player, npc)
  if player:is_gm() or npc:is_my_lord(player) then
    return true
  end
  local castle = npc:castle()
  return castle ~= nil and player:clan_id() == castle:owner_id() and player:has_clan_privilege("CASTLE_MANOR")
end

s:on_event(function(event, npc, player)
  if npc == nil or player == nil then
    return nil
  end
  if event:lower() == (npc:id() .. "-02.html") and has_rights(player, npc) then
    return DIR .. npc:id() .. "-02.html"
  end
  return nil
end)

s:on_first_talk(NPCS, function(npc, player)
  if has_rights(player, npc) then
    return DIR .. npc:id() .. "-01.html"
  end
  return DIR .. "no.html"
end)

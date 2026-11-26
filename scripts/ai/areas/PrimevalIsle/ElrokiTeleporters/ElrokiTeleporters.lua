-- The Elroki teleporters Orahochin and Gariachin, ported from Mobius
-- ai/areas/PrimevalIsle/ElrokiTeleporters (Plim, Skache).
--
-- Each takes a player who is not fighting to the other side.

local s = script.new("ElrokiTeleporters")

local ORAHOCHIN, GARIACHIN = 32111, 32112

local TELEPORT_ORAHOCHIN = { x = 5361, y = -2638, z = -3176 }
local TELEPORT_GARIACHIN = { x = 7327, y = -5501, z = -3232 }

s:add_start_npc(ORAHOCHIN, GARIACHIN)
s:add_talk_id(ORAHOCHIN, GARIACHIN)

s:on_event(function(event, npc, player)
  if event == "TELEPORT" and npc ~= nil and player ~= nil then
    if player:is_in_combat() then
      return npc:id() .. "-no.html"
    end
    player:teleport(npc:id() == ORAHOCHIN and TELEPORT_ORAHOCHIN or TELEPORT_GARIACHIN)
  end
  return nil
end)

s:on_first_talk({ ORAHOCHIN, GARIACHIN }, function(npc, player)
  return npc:id() .. ".html"
end)

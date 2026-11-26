-- The Altar Gatekeepers of the Pagan Temple, ported from Mobius
-- ai/areas/PaganTemple/AltarGatekeeper (Altur).
--
-- Four of them spawn at once when Andreas Van Halter's altar doors open;
-- only the first shouts that the door is open, then nobody for 30 s.

local s = script.new("AltarGatekeeper")

local ALTAR_GATEKEEPER = 32051
local SHOUT_COOLDOWN = 30000 -- ms

-- last_shout is when a gatekeeper last shouted (a reload forgets it).
local last_shout = 0

s:on_spawn(ALTAR_GATEKEEPER, function(npc)
  local now = time.now_ms()
  if now - last_shout < SHOUT_COOLDOWN then
    return
  end
  last_shout = now
  npc:broadcast_say(ChatType.NPC_GENERAL, "The door to the 3rd floor of the altar is now open.")
end)

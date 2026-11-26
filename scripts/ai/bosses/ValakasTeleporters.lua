-- The way to Valakas, ported from Mobius ai/others/ValakasTeleporters
-- (Plim, Skache).
--
-- Klein (31540) lets a player with a Vacualite Floating Stone into the Hall
-- of Flames; the Heart of Volcano (31385) then lets them into the lair
-- (boss zone 12010) while Valakas sleeps or waits, at most 200 times. The
-- first one in starts Valakas's wait (Config.VALAKAS_WAIT_TIME minutes).
-- The gatekeepers open the doors on the way; the cube outside takes
-- players out.
--
-- Mobius starts the wait timer on the Valakas script itself; here the
-- Heart sends Valakas the script event VALAKAS_WAIT, and Valakas.lua starts
-- its own timer. Without a Valakas in the store the Heart sends players
-- away (Mobius: without the Valakas script).

local s = script.new("ValakasTeleporters")

local NPCS = {
  31384, -- Gatekeeper of Fire Dragon: opens some doors
  31385, -- Heart of Volcano: into the lair of Valakas
  31540, -- Watcher of Valakas Klein: into the Hall of Flames
  31686, -- Gatekeeper of Fire Dragon: the doors to the Heart of Volcano
  31687, -- Gatekeeper of Fire Dragon: the doors to the Heart of Volcano
  31759, -- Teleportation Cubic: out of the lair
}

local VALAKAS = 29028
local LAIR_ZONE = 12010
local VACUALITE_FLOATING_STONE = 7267
local ENTER_HALL_OF_FLAMES = { x = 183813, y = -115157, z = -3303 }
local INTO_LAIR = { x = 204328, y = -111874, z = 70 }
local OUT_OF_LAIR = { x = 150037, y = -57720, z = -2976 }

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

-- How many entered the lair since the scripts loaded (Mobius's static
-- playerCount).
local player_count = 0

s:on_event(function(event, npc, player)
  local qs = s:get_quest_state(player, true)
  local html = ""
  if event == "ENTER" then
    if s:has_quest_items(player, VACUALITE_FLOATING_STONE) then
      s:take_items(player, VACUALITE_FLOATING_STONE, 1)
      player:teleport(ENTER_HALL_OF_FLAMES)
      qs:set("allowEnter", "1")
    else
      html = "31540-06.htm"
    end
  elseif event == "ENTER_LAIR" then
    local status = grandboss.status(VALAKAS)
    if status == nil then
      html = "31385-01.htm"
    elseif (status == 0 or status == 1) and qs:get_int("allowEnter") == 1 then
      if player_count >= 200 then
        html = "31385-03.htm"
      else
        qs:unset("allowEnter")
        zones.allow_entry(LAIR_ZONE, player, 30)
        player:teleport(INTO_LAIR.x + rnd(600), INTO_LAIR.y + rnd(600), INTO_LAIR.z)
        player_count = player_count + 1
        if status == 0 then
          local valakas = grandboss.boss(VALAKAS)
          if valakas ~= nil and npc ~= nil then
            npc:send_event("VALAKAS_WAIT", valakas, player)
          end
          grandboss.set_status(VALAKAS, 1)
        end
      end
    else
      html = status == 2 and "31385-02.htm" or "31385-04.htm"
    end
  elseif event == "OPEN_DOOR_1" then
    doors.open(24210004)
  elseif event == "OPEN_DOOR_2" then
    doors.open(24210005)
  elseif event == "OPEN_DOOR_3" then
    doors.open(24210006)
  elseif event == "EXIT" then
    player:teleport(OUT_OF_LAIR.x + rnd(500), OUT_OF_LAIR.y + rnd(500), OUT_OF_LAIR.z)
  end
  return html
end)

s:on_talk(function(npc, player)
  if npc:id() == 31540 then
    if player_count < 50 then
      return "31540-01.htm"
    elseif player_count < 100 then
      return "31540-02.htm"
    elseif player_count < 150 then
      return "31540-03.htm"
    elseif player_count < 200 then
      return "31540-04.htm"
    end
    return "31540-05.htm"
  end
  return npc:id() .. ".htm"
end)

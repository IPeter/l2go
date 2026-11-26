-- The castle mass gatekeepers, ported from Mobius ai/others/CastleTeleporter.
--
-- "Teleport" starts the countdown: 30 seconds, or 8 minutes during a siege
-- once every control tower has fallen. Then the gatekeeper shouts to its
-- map region that the defenders go to the inner castle, and everyone in the
-- castle's teleport zone goes there (Castle.oustAllPlayers). While the
-- countdown runs (script value 1) the gatekeeper shows teleporter-03.html.
--
-- The pages are Mobius's, served from
-- data/html/scripts/ai/others/CastleTeleporter/.

local s = script.new("CastleTeleporter")

local NPCS = {
  35095, -- Mass Gatekeeper (Gludio)
  35137, -- Mass Gatekeeper (Dion)
  35179, -- Mass Gatekeeper (Giran)
  35221, -- Mass Gatekeeper (Oren)
  35266, -- Mass Gatekeeper (Aden)
  35311, -- Mass Gatekeeper (Innadril)
  35355, -- Mass Gatekeeper (Goddard)
  35502, -- Mass Gatekeeper (Rune)
  35547, -- Mass Gatekeeper (Schuttgart)
}

local DIR = "data/html/scripts/ai/others/CastleTeleporter/"

-- towers_down: the siege is on and every control tower has fallen.
local function towers_down(castle)
  return castle:is_siege_in_progress() and castle:control_tower_count() == 0
end

s:on_event(function(event, npc, player)
  if npc == nil then
    return nil
  end
  local castle = npc:castle()
  if castle == nil then
    return nil
  end
  local e = event:upper()
  if e == "TELEPORT_START" then
    if npc:is_script_value(0) then
      local delay = towers_down(castle) and 480000 or 30000
      s:start_quest_timer("TELEPORT_EXECUTE", delay, npc, nil)
      npc:set_script_value(1)
    end
    return DIR .. "teleporter-03.html"
  elseif e == "TELEPORT_EXECUTE" then
    castle:oust_all_players()
    npc:set_script_value(0)
    npc:broadcast_say_to_region(ChatType.NPC_SHOUT,
      "The defenders of " .. castle:name() .. " castle will be teleported to the inner castle.")
  end
  return nil
end)

s:on_first_talk(NPCS, function(npc, player)
  local castle = npc:castle()
  if castle == nil then
    return nil
  end
  if not npc:is_script_value(0) then
    return DIR .. "teleporter-03.html"
  elseif towers_down(castle) then
    return DIR .. "teleporter-02.html"
  end
  return DIR .. "teleporter-01.html"
end)

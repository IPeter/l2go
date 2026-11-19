-- Talking Island guards greet new adventurers.
--
-- An example NPC script in Mobius's style (see "NPC scripts" in
-- scripts/README.md):
--   * on_first_talk answers the first click on a guard: for players below
--     level 20, the page welcome.htm next to this file, with placeholder
--     values; for everyone else nil, which shows the guard's usual page;
--   * on_event handles the "Script TalkingIslandGuard <event>" links of
--     those pages: a page again, inline HTML, or NPC speech.
-- In Mobius the guards have only a short flavor page and no script, so
-- nothing else changes for them.

local guard = script.new("TalkingIslandGuard")

-- The Talking Island guards (Mobius spawns/TalkingIsland/TalkingIslandNPCs.xml).
local GUARDS = { 30733, 31032 }

-- Players below this level get the welcome page.
local NEWCOMER_LEVEL = 20

-- The welcome page and the values of its %name% and %level% placeholders.
local function welcome(player)
  return "welcome.htm", { name = player:name(), level = player:level() }
end

guard:on_first_talk(GUARDS, function(npc, player)
  if player:level() >= NEWCOMER_LEVEL then
    return nil
  end
  return welcome(player)
end)

local DIRECTIONS = [[<html><body>Guard:<br>
The village is small, %name%, you will find your way.<br>
The gatekeeper on the square sends travelers to the mainland, for a fee.<br>
The shops around the square sell weapons, armor and supplies.<br>
Past the village gates the fields begin, and so do the monsters.<br>
<a action="bypass -h Script TalkingIslandGuard welcome">Back.</a>
</body></html>]]

guard:on_event(function(event, npc, player)
  if event == "welcome" then
    return welcome(player)
  elseif event == "directions" then
    return html.replace(DIRECTIONS, { name = player:name() })
  elseif event == "salute" then
    npc:say("Stay on the roads, " .. player:name() .. "!")
    return ""
  end
  log.warn("unknown guard event", "event", event, "player", player:name())
  return nil
end)

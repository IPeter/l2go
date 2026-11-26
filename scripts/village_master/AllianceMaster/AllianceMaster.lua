-- The alliance pages of the village masters, ported by hand from Mobius
-- village_master/AllianceMaster. Every page but the first one needs a clan;
-- without one the master answers 9001-04.htm.
--
-- The alliance actions on the pages (npc_%objectId%_create_ally,
-- ..._dissolve_ally) are village master commands of the game
-- (internal/game/alliance.go: Mobius Clan.createAlly and dissolveAlly).
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/AllianceMaster/.

local s = script.new("AllianceMaster")

local NPCS = {
  30026, 30031, 30037, 30066, 30070, 30109, 30115, 30120, 30154, 30174,
  30175, 30176, 30187, 30191, 30195, 30288, 30289, 30290, 30297, 30358,
  30373, 30462, 30474, 30498, 30499, 30500, 30503, 30504, 30505, 30508,
  30511, 30512, 30513, 30520, 30525, 30565, 30594, 30595, 30676, 30677,
  30681, 30685, 30687, 30689, 30694, 30699, 30704, 30845, 30847, 30849,
  30854, 30857, 30862, 30865, 30894, 30897, 30900, 30905, 30910, 30913,
  31269, 31272, 31276, 31279, 31285, 31288, 31314, 31317, 31321, 31324,
  31326, 31328, 31331, 31334, 31336, 31755, 31958, 31961, 31965, 31968,
  31974, 31977, 31996, 32092, 32093, 32094, 32095, 32096, 32097, 32098,
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

s:on_event(function(event, npc, player)
  if event ~= "9001-01.htm" and player:clan() == nil then
    return "9001-04.htm"
  end
  return event
end)

s:on_talk(function(npc, player)
  return "9001-01.htm"
end)

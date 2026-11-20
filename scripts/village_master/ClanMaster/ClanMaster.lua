-- The clan pages of the village masters, ported by hand from Mobius
-- village_master/ClanMaster. The pages that need a clan leader show their
-- "-no" page to everyone else.
--
-- The clan actions on the pages (npc_%objectId%_create_clan,
-- increase_clan_level, dissolve_clan, recover_clan, change_clan_leader,
-- cancel_clan_leader_change and the others) are commands of the game's
-- village masters, as in Mobius VillageMaster.onBypassFeedback; the unit
-- and clan skill ones answer as not available yet until they exist.
--
-- The pages are Mobius's, served from
-- data/html/scripts/village_master/ClanMaster/.

local s = script.new("ClanMaster")

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

-- The pages only a clan leader sees, and what everyone else sees instead.
local LEADER_REQUIRED = {
  ["9000-03.htm"] = "9000-03-no.htm",
  ["9000-04.htm"] = "9000-04-no.htm",
  ["9000-05.htm"] = "9000-05-no.htm",
  ["9000-07.htm"] = "9000-07-no.htm",
  ["9000-12a.htm"] = "9000-07-no.htm",
  ["9000-12b.htm"] = "9000-07-no.htm",
  ["9000-13a.htm"] = "9000-07-no.htm",
  ["9000-13b.htm"] = "9000-07-no.htm",
  ["9000-14a.htm"] = "9000-07-no.htm",
  ["9000-14b.htm"] = "9000-07-no.htm",
  ["9000-15.htm"] = "9000-07-no.htm",
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

s:on_event(function(event, npc, player)
  if LEADER_REQUIRED[event] ~= nil and not player:is_clan_leader() then
    return LEADER_REQUIRED[event]
  end
  return event
end)

s:on_talk(function(npc, player)
  return "9000-01.htm"
end)

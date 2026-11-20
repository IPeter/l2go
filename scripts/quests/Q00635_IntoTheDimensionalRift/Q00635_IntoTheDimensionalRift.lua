-- Translated by questport from quests/Q00635_IntoTheDimensionalRift (Mobius CT_0 Interlude); owned by L2Go from here on.
-- @author Mobius
-- @note Based on python script
local J = require("lib.java")

local q = quest.new(635, "Q00635_IntoTheDimensionalRift", "In the Dimensional Rift")
-- The Dimensional Rift belongs to Seven Signs; the gatekeepers' talk also
-- needs player:active_quests(), which does not exist yet.
q:requires("seven_signs")

-- Item
local DIMENSION_FRAGMENT = 7079
-- Locations
local COORD = {
  {x = -41572, y = 209731, z = -5087}, -- Necropolis of Sacrifice
  {x = -52872, y = -250283, z = -7908}, -- Catacomb of the Heretic
  {x = 45256, y = 123906, z = -5411}, -- Pilgrim's Necropolis
  {x = 46192, y = 170290, z = -4981}, -- Catacomb of the Branded
  {x = 111273, y = 174015, z = -5437}, -- Necropolis of Worship
  {x = -20604, y = -250789, z = -8165}, -- Catacomb of Apostate
  {x = -21726, y = 77385, z = -5171}, -- Patriot's Necropolis
  {x = 140405, y = 79679, z = -5427}, -- Catacomb of the Witch
  {x = -52366, y = 79097, z = -4741}, -- Necropolis of Devotion (ex Ascetics)
  {x = 118311, y = 132797, z = -4829}, -- Necropolis of Martyrdom
  {x = 172185, y = -17602, z = -4901}, -- Disciple's Necropolis
  {x = 83000, y = 209213, z = -5439}, -- Saint's Necropolis
  {x = -19500, y = 13508, z = -4901}, -- Catacomb of Dark Omens
  {x = 113865, y = 84543, z = -6541}, -- Catacomb of the Forbidden Path
}
-- Misc
local MAX_QUEST = 23 -- How many quest slots you need to have in order to enter be able to take another quests within the Rift.

for i = 31494, 31508 do
  q:add_talk_id(i)
  q:add_start_npc(i)
end
for i = 31488, 31494 do
  q:add_talk_id(i)
end

q:on_event(function(event, npc, player)
  local htmltext = event
  local qs = player:get_quest_state(q:name())
  if qs == nil then
    return htmltext
  end
  local id = qs:get_int("id")
  local count = qs:get_int("count")
  if event == "TELEPORT" then
    if id > 0 then
      if count > 0 then
        htmltext = "5a.htm"
      end
      qs:set("count", tostring(count + 1))
      qs:set_state(State.STARTED)
      qs:set_cond(1)
      qs:get_player():teleport(-114790, -180576, -6781)
    else
      htmltext = "What are you trying to do?"
      qs:exit_quest(true)
    end
  elseif event == "6.htm" then
    qs:exit_quest(true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local qs = q:get_quest_state(player, true)
  local npcId = npc:id()
  local id = qs:get_int("id")
  if npcId >= 31494 and npcId <= 31508 then
    if player:level() < 20 then
      qs:exit_quest(true)
      htmltext = "1.htm"
    elseif #player:active_quests() > MAX_QUEST then
      qs:exit_quest(true)
      htmltext = "1a.htm"
    elseif q:get_quest_items_count(player, DIMENSION_FRAGMENT) <= 0 then
      htmltext = "3.htm"
    else
      qs:set_state(State.CREATED)
      id = npcId - 31493
      qs:set("id", tostring(id))
      htmltext = "4.htm"
    end
  elseif qs:get_state() == State.STARTED then
    if id > 0 then
      -- Mobius indexes COORD[id] with id from 1: the gatekeeper's own place
      -- is COORD[id - 1], so this sends the player one entry further (and
      -- fails for the last gatekeepers). Kept as Mobius.
      player:teleport(COORD[id + 1])
      qs:unset("cond")
      qs:set_state(State.CREATED)
      htmltext = "7.htm"
    else
      htmltext = "Where?"
      qs:exit_quest(true)
    end
  end
  return htmltext
end)

-- Translated by questport from quests/Q00505_BloodOffering (Mobius CT_0 Interlude); owned by L2Go from here on.
-- @author Mobius
-- @note Based on python script
-- Needs the Seven Signs (the rift posts ask for the seal validation period
-- and the player's cabal). The Festival of Darkness guides and witches
-- work already.
local J = require("lib.java")

local q = quest.new(505, "Q00505_BloodOffering", "Blood Offering")
q:requires("seven_signs")

-- NPCs
local TOWN_DAWN = {31078, 31079, 31080, 31081, 31083, 31084, 31082, 31692, 31694, 31997, 31168}
local TOWN_DUSK = {31085, 31086, 31087, 31088, 31090, 31091, 31089, 31693, 31695, 31998, 31169}
local DIM_GK = {31494, 31495, 31496, 31497, 31498, 31499, 31500, 31501, 31502, 31503, 31504, 31505, 31506, 31507}
local GK_ZIGGURAT = {
  31095,
  31096,
  31097,
  31098,
  31099,
  31100,
  31101,
  31102,
  31103,
  31104,
  31105,
  31106,
  31107,
  31108,
  31109,
  31110,
  31114,
  31115,
  31116,
  31117,
  31118,
  31119,
  31120,
  31121,
  31122,
  31123,
  31124,
  31125,
}
local FESTIVALGUIDE = {31127, 31128, 31129, 31130, 31131, 31137, 31138, 31139, 31140, 31141}
local FESTIVALWITCH = {31132, 31133, 31134, 31135, 31136, 31142, 31143, 31144, 31145, 31146}
local RIFTPOST = {31488, 31489, 31490, 31491, 31492, 31493}

q:add_start_npc(TOWN_DAWN)
q:add_talk_id(TOWN_DAWN)
q:add_start_npc(TOWN_DUSK)
q:add_talk_id(TOWN_DUSK)
q:add_start_npc(DIM_GK)
q:add_talk_id(DIM_GK)
q:add_start_npc(GK_ZIGGURAT)
q:add_talk_id(GK_ZIGGURAT)
q:add_start_npc(FESTIVALGUIDE)
q:add_talk_id(FESTIVALGUIDE)
q:add_start_npc(FESTIVALWITCH)
q:add_talk_id(FESTIVALWITCH)
q:add_start_npc(RIFTPOST)
q:add_talk_id(RIFTPOST)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local qs = q:get_quest_state(player, true)
  if qs == nil then
    return htmltext
  end
  local qs2 = player:get_quest_state("Q00635_IntoTheDimensionalRift")
  local npcId = npc:id()
  if J.contains(FESTIVALGUIDE, npcId) then
    player:teleport(-114796, -179334, -6752)
    if qs2 ~= nil then
      qs2:set_state(State.STARTED)
      qs2:set_cond(1)
    end
    qs:exit_quest(true, true)
    return "guide.htm"
  elseif npcId == 31132 then
    player:teleport(-80204, 87056, -5154)
    return "witch.htm"
  elseif npcId == 31133 then
    player:teleport(-77198, 87678, -5182)
    return "witch.htm"
  elseif npcId == 31134 then
    player:teleport(-76183, 87135, -5179)
    return "witch.htm"
  elseif npcId == 31135 then
    player:teleport(-76945, 86602, 5153)
    return "witch.htm"
  elseif npcId == 31136 then
    player:teleport(-79970, 85997, -5154)
    return "witch.htm"
  elseif npcId == 31142 then
    player:teleport(-79182, 111893, -4898)
    return "witch.htm"
  elseif npcId == 31143 then
    player:teleport(-76176, 112505, -4899)
    return "witch.htm"
  elseif npcId == 31144 then
    player:teleport(-75198, 111969, -4898)
    return "witch.htm"
  elseif npcId == 31145 then
    player:teleport(-75920, 111435, -4900)
    return "witch.htm"
  elseif npcId == 31146 then
    player:teleport(-78928, 110825, -4926)
    return "witch.htm"
  elseif J.contains(RIFTPOST, npcId) then
    if not seven_signs.is_seal_validation_period() then
      local cabal = seven_signs.player_cabal(player:object_id())
      if cabal == seven_signs.CABAL_DAWN then
        qs:set_state(State.STARTED)
        qs:set_cond(1)
        qs:get_player():teleport(-80157, 111344, -4901)
        if qs2 ~= nil then
          qs2:unset("cond")
        end
        return "riftpost-1.htm"
      elseif cabal == seven_signs.CABAL_DUSK then
        qs:set_state(State.STARTED)
        qs:set_cond(1)
        if qs2 ~= nil then
          qs2:unset("cond")
        end
        qs:get_player():teleport(-81261, 86531, -5157)
        return "riftpost-1.htm"
      end
      return "riftpost-2.htm"
    end
    return "riftpost-2.htm"
  end
  return htmltext
end)

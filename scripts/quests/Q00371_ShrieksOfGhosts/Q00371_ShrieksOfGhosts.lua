-- Translated by questport from quests/Q00371_ShrieksOfGhosts (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(371, "Q00371_ShrieksOfGhosts", "Shriek of Ghosts")

-- NPCs
local REVA = 30867
local PATRIN = 30929
-- Item
local URN = 5903
local PORCELAIN = 6002
-- Drop chances
local MOBS = {}

local DropInfo = {}
DropInfo.__index = DropInfo

function DropInfo.new(firstChance, secondChance)
  local self = setmetatable({}, DropInfo)
  self._firstChance = 0
  self._secondChance = 0
  self._firstChance = firstChance
  self._secondChance = secondChance
  return self
end

function DropInfo:getFirstChance()
  return self._firstChance
end

function DropInfo:getSecondChance()
  return self._secondChance
end

MOBS[20818] = DropInfo.new(350, 400) -- hallates_warrior
MOBS[20820] = DropInfo.new(583, 673) -- hallates_knight
MOBS[20824] = DropInfo.new(458, 538) -- hallates_commander

q:register_quest_items(URN, PORCELAIN)
q:add_start_npc(REVA)
q:add_talk_id(REVA, PATRIN)
q:add_kill_id(20818, 20820, 20824)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30867-03.htm" then
    st:start_quest()
  elseif event == "30867-07.htm" then
    local urns = q:get_quest_items_count(player, URN)
    if urns > 0 then
      q:take_items(player, URN, urns)
      if urns >= 100 then
        urns = urns + 13
        htmltext = "30867-08.htm"
      else
        urns = urns + 7
      end
      q:give_adena(player, urns * 1000, true)
    end
  elseif event == "30867-10.htm" then
    st:exit_quest(true, true)
  elseif event == "APPR" then
    if q:has_quest_items(player, PORCELAIN) then
      local chance = rnd(100)
      q:take_items(player, PORCELAIN, 1)
      if chance < 2 then
        q:give_items(player, 6003, 1)
        htmltext = "30929-03.htm"
      elseif chance < 32 then
        q:give_items(player, 6004, 1)
        htmltext = "30929-04.htm"
      elseif chance < 62 then
        q:give_items(player, 6005, 1)
        htmltext = "30929-05.htm"
      elseif chance < 77 then
        q:give_items(player, 6006, 1)
        htmltext = "30929-06.htm"
      else
        htmltext = "30929-07.htm"
      end
    else
      htmltext = "30929-02.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 59 and "30867-01.htm" or "30867-02.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == REVA then
      if q:has_quest_items(player, URN) then
        htmltext = q:has_quest_items(player, PORCELAIN) and "30867-05.htm" or "30867-04.htm"
      else
        htmltext = "30867-06.htm"
      end
    elseif sw2 == PATRIN then
      htmltext = "30929-01.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs == nil or not util.in_range(Config.ALT_PARTY_RANGE, npc, killer, true) then
    return
  end
  local info = MOBS[npc:id()]
  local random = rnd(1000)
  if random < info:getFirstChance() then
    q:give_item_randomly(qs:get_player(), npc, URN, 1, 1, 0, 1, true)
  elseif random < info:getSecondChance() then
    q:give_item_randomly(qs:get_player(), npc, PORCELAIN, 1, 1, 0, 1, true)
  end
end)

-- Translated by questport from quests/Q00355_FamilyHonor (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(355, "Q00355_FamilyHonor", "Family Honor")

-- NPCs
local GALIBREDO = 30181
local PATRIN = 30929
-- Monsters
local TIMAK_ORC_TROOP_LEADER = 20767
local TIMAK_ORC_TROOP_SHAMAN = 20768
local TIMAK_ORC_TROOP_WARRIOR = 20769
local TIMAK_ORC_TROOP_ARCHER = 20770
-- Items
local GALIBREDO_BUST = 4252
local WORK_OF_BERONA = 4350
local STATUE_PROTOTYPE = 4351
local STATUE_ORIGINAL = 4352
local STATUE_REPLICA = 4353
local STATUE_FORGERY = 4354
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

MOBS[TIMAK_ORC_TROOP_LEADER] = DropInfo.new(560, 684) -- timak_orc_troop_leader
MOBS[TIMAK_ORC_TROOP_SHAMAN] = DropInfo.new(530, 650) -- timak_orc_troop_shaman
MOBS[TIMAK_ORC_TROOP_WARRIOR] = DropInfo.new(420, 516) -- timak_orc_troop_warrior
MOBS[TIMAK_ORC_TROOP_ARCHER] = DropInfo.new(440, 560) -- timak_orc_troop_archer

q:register_quest_items(GALIBREDO_BUST)
q:add_start_npc(GALIBREDO)
q:add_talk_id(GALIBREDO, PATRIN)
q:add_kill_id(TIMAK_ORC_TROOP_LEADER, TIMAK_ORC_TROOP_SHAMAN, TIMAK_ORC_TROOP_WARRIOR, TIMAK_ORC_TROOP_ARCHER)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30181-2.htm" then
    st:start_quest()
  elseif event == "30181-4b.htm" then
    local count = q:get_quest_items_count(player, GALIBREDO_BUST)
    if count > 0 then
      htmltext = "30181-4.htm"
      local reward = count * 232
      if count >= 100 then
        htmltext = "30181-4a.htm"
        reward = reward + 5000
      end
      q:take_items(player, GALIBREDO_BUST, count)
      q:give_adena(player, reward, true)
    end
  elseif event == "30929-7.htm" then
    if q:has_quest_items(player, WORK_OF_BERONA) then
      q:take_items(player, WORK_OF_BERONA, 1)
      local appraising = rnd(100)
      if appraising < 20 then
        htmltext = "30929-2.htm"
      elseif appraising < 40 then
        htmltext = "30929-3.htm"
        q:give_items(player, STATUE_REPLICA, 1)
      elseif appraising < 60 then
        htmltext = "30929-4.htm"
        q:give_items(player, STATUE_ORIGINAL, 1)
      elseif appraising < 80 then
        htmltext = "30929-5.htm"
        q:give_items(player, STATUE_FORGERY, 1)
      else
        htmltext = "30929-6.htm"
        q:give_items(player, STATUE_PROTOTYPE, 1)
      end
    end
  elseif event == "30181-6.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 36 and "30181-0a.htm" or "30181-0.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == GALIBREDO then
      htmltext = q:has_quest_items(player, GALIBREDO_BUST) and "30181-3a.htm" or "30181-3.htm"
    elseif sw2 == PATRIN then
      htmltext = "30929-0.htm"
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
    q:give_item_randomly(killer, npc, GALIBREDO_BUST, 1, 1, 0, 1, true)
  elseif random < info:getSecondChance() then
    q:give_item_randomly(killer, npc, WORK_OF_BERONA, 1, 1, 0, 1, true)
  end
end)

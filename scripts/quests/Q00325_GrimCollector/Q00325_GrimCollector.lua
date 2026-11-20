-- Translated by questport from quests/Q00325_GrimCollector (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(325, "Q00325_GrimCollector", "Grim Collector")

-- NPCs
local CURTIS = 30336
local VARSAK = 30342
local SAMED = 30434
-- Items
local ANATOMY_DIAGRAM = 1349
local ZOMBIE_HEAD = 1350
local ZOMBIE_HEART = 1351
local ZOMBIE_LIVER = 1352
local SKULL = 1353
local RIB_BONE = 1354
local SPINE = 1355
local ARM_BONE = 1356
local THIGH_BONE = 1357
local COMPLETE_SKELETON = 1358
-- Drops
local DROPLIST = {}

local function getNumberOfPieces(player)
  return q:get_quest_items_count(player, ZOMBIE_HEAD) + q:get_quest_items_count(player, SPINE) + q:get_quest_items_count(player, ARM_BONE) + q:get_quest_items_count(player, ZOMBIE_HEART) + q:get_quest_items_count(player, ZOMBIE_LIVER) + q:get_quest_items_count(player, SKULL) + q:get_quest_items_count(player, RIB_BONE) + q:get_quest_items_count(player, THIGH_BONE) + q:get_quest_items_count(player, COMPLETE_SKELETON)
end

local function payback(player)
  local count = getNumberOfPieces(player)
  if count > 0 then
    local reward = 30 * q:get_quest_items_count(player, ZOMBIE_HEAD) + 20 * q:get_quest_items_count(player, ZOMBIE_HEART) + 20 * q:get_quest_items_count(player, ZOMBIE_LIVER) + 100 * q:get_quest_items_count(player, SKULL) + 40 * q:get_quest_items_count(player, RIB_BONE) + 14 * q:get_quest_items_count(player, SPINE) + 14 * q:get_quest_items_count(player, ARM_BONE) + 14 * q:get_quest_items_count(player, THIGH_BONE) + 341 * q:get_quest_items_count(player, COMPLETE_SKELETON)
    if count > 10 then
      reward = reward + 1629
    end
    if q:has_quest_items(player, COMPLETE_SKELETON) then
      reward = reward + 543
    end
    q:take_items(player, ZOMBIE_HEAD, -1)
    q:take_items(player, ZOMBIE_HEART, -1)
    q:take_items(player, ZOMBIE_LIVER, -1)
    q:take_items(player, SKULL, -1)
    q:take_items(player, RIB_BONE, -1)
    q:take_items(player, SPINE, -1)
    q:take_items(player, ARM_BONE, -1)
    q:take_items(player, THIGH_BONE, -1)
    q:take_items(player, COMPLETE_SKELETON, -1)
    q:give_adena(player, reward, true)
  end
end

DROPLIST[20026] = {{id = ZOMBIE_HEAD, count = 30}, {id = ZOMBIE_HEART, count = 50}, {id = ZOMBIE_LIVER, count = 75}}
DROPLIST[20029] = {{id = ZOMBIE_HEAD, count = 30}, {id = ZOMBIE_HEART, count = 52}, {id = ZOMBIE_LIVER, count = 75}}
DROPLIST[20035] = {{id = SKULL, count = 5}, {id = RIB_BONE, count = 15}, {id = SPINE, count = 29}, {id = THIGH_BONE, count = 79}}
DROPLIST[20042] = {{id = SKULL, count = 6}, {id = RIB_BONE, count = 19}, {id = ARM_BONE, count = 69}, {id = THIGH_BONE, count = 86}}
DROPLIST[20045] = {{id = SKULL, count = 9}, {id = SPINE, count = 59}, {id = ARM_BONE, count = 77}, {id = THIGH_BONE, count = 97}}
DROPLIST[20051] = {{id = SKULL, count = 9}, {id = RIB_BONE, count = 59}, {id = SPINE, count = 79}, {id = ARM_BONE, count = 100}}
DROPLIST[20457] = {{id = ZOMBIE_HEAD, count = 40}, {id = ZOMBIE_HEART, count = 60}, {id = ZOMBIE_LIVER, count = 80}}
DROPLIST[20458] = {{id = ZOMBIE_HEAD, count = 40}, {id = ZOMBIE_HEART, count = 70}, {id = ZOMBIE_LIVER, count = 100}}
DROPLIST[20514] = {{id = SKULL, count = 6}, {id = RIB_BONE, count = 21}, {id = SPINE, count = 30}, {id = ARM_BONE, count = 31}, {id = THIGH_BONE, count = 64}}
DROPLIST[20515] = {{id = SKULL, count = 5}, {id = RIB_BONE, count = 20}, {id = SPINE, count = 31}, {id = ARM_BONE, count = 33}, {id = THIGH_BONE, count = 69}}

q:register_quest_items(ZOMBIE_HEAD, ZOMBIE_HEART, ZOMBIE_LIVER, SKULL, RIB_BONE, SPINE, ARM_BONE, THIGH_BONE, COMPLETE_SKELETON, ANATOMY_DIAGRAM)
q:add_start_npc(CURTIS)
q:add_talk_id(CURTIS, VARSAK, SAMED)
q:add_kill_id(J.keys(DROPLIST))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30336-03.htm" then
    st:start_quest()
  elseif event == "30434-03.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, ANATOMY_DIAGRAM, 1)
  elseif event == "30434-06.htm" then
    q:take_items(player, ANATOMY_DIAGRAM, -1)
    payback(player)
    st:exit_quest(true, true)
  elseif event == "30434-07.htm" then
    payback(player)
  elseif event == "30434-09.htm" then
    local skeletons = q:get_quest_items_count(player, COMPLETE_SKELETON)
    if skeletons > 0 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:take_items(player, COMPLETE_SKELETON, -1)
      q:give_adena(player, 543 + 341 * skeletons, true)
    end
  elseif event == "30342-03.htm" then
    if not q:has_quest_items(player, SPINE, ARM_BONE, SKULL, RIB_BONE, THIGH_BONE) then
      htmltext = "30342-02.htm"
    else
      q:take_items(player, SPINE, 1)
      q:take_items(player, SKULL, 1)
      q:take_items(player, ARM_BONE, 1)
      q:take_items(player, RIB_BONE, 1)
      q:take_items(player, THIGH_BONE, 1)
      if rnd(10) < 9 then
        q:give_items(player, COMPLETE_SKELETON, 1)
      else
        htmltext = "30342-04.htm"
      end
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 15 and "30336-01.htm" or "30336-02.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == CURTIS then
      htmltext = not q:has_quest_items(player, ANATOMY_DIAGRAM) and "30336-04.htm" or "30336-05.htm"
    elseif sw2 == SAMED then
      if not q:has_quest_items(player, ANATOMY_DIAGRAM) then
        htmltext = "30434-01.htm"
      else
        if getNumberOfPieces(player) == 0 then
          htmltext = "30434-04.htm"
        else
          htmltext = not q:has_quest_items(player, COMPLETE_SKELETON) and "30434-05.htm" or "30434-08.htm"
        end
      end
    elseif sw2 == VARSAK then
      htmltext = "30342-01.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if q:has_quest_items(player, ANATOMY_DIAGRAM) then
    local chance = rnd(100)
    for _, drop in ipairs(DROPLIST[npc:id()]) do
      if chance < drop.count then
        q:give_items(player, drop.id, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        break
      end
    end
  end
end)

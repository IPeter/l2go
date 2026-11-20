-- Translated by questport from quests/Q00328_SenseForBusiness (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(328, "Q00328_SenseForBusiness", "Sense for Business")

-- Items
local MONSTER_EYE_LENS = 1366
local MONSTER_EYE_CARCASS = 1347
local BASILISK_GIZZARD = 1348
-- Drop chances
local CHANCES = {}

CHANCES[20055] = 48
CHANCES[20059] = 52
CHANCES[20067] = 68
CHANCES[20068] = 76
CHANCES[20070] = 500000
CHANCES[20072] = 510000

q:register_quest_items(MONSTER_EYE_LENS, MONSTER_EYE_CARCASS, BASILISK_GIZZARD)
q:add_start_npc(30436) -- Sarien
q:add_talk_id(30436)
q:add_kill_id(20055, 20059, 20067, 20068, 20070, 20072)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30436-03.htm" then
    st:start_quest()
  elseif event == "30436-06.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 21 and "30436-01.htm" or "30436-02.htm"
  elseif sw == State.STARTED then
    local carcasses = q:get_quest_items_count(player, MONSTER_EYE_CARCASS)
    local lenses = q:get_quest_items_count(player, MONSTER_EYE_LENS)
    local gizzards = q:get_quest_items_count(player, BASILISK_GIZZARD)
    local all = carcasses + lenses + gizzards
    if all == 0 then
      htmltext = "30436-04.htm"
    else
      htmltext = "30436-05.htm"
      q:take_items(player, MONSTER_EYE_CARCASS, -1)
      q:take_items(player, MONSTER_EYE_LENS, -1)
      q:take_items(player, BASILISK_GIZZARD, -1)
      q:give_adena(player, 30 * carcasses + 2000 * lenses + 75 * gizzards + (all >= 10 and 618 or 0), true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local npcId = npc:id()
  local chance = CHANCES[npcId]
  if npcId < 20069 then
    local rnd_ = rnd(100)
    if rnd_ < chance + 1 then
      q:give_items(player, rnd_ < chance and MONSTER_EYE_CARCASS or MONSTER_EYE_LENS, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif rnd(1000000) < chance then
    q:give_items(player, BASILISK_GIZZARD, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)

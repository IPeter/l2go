-- Translated by questport from quests/Q00631_DeliciousTopChoiceMeat (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(631, "Q00631_DeliciousTopChoiceMeat", "Delicious Top Choice Meat")

-- NPC
local TUNATUN = 31537
-- Item
local TOP_QUALITY_MEAT = 7546
-- Drop chances
local CHANCES = {}
-- Rewards
local REWARDS = {{4039, 15}, {4043, 15}, {4044, 15}, {4040, 10}, {4042, 10}, {4041, 5}}

CHANCES[21460] = 601000
CHANCES[21461] = 480000
CHANCES[21462] = 447000
CHANCES[21463] = 808000
CHANCES[21464] = 447000
CHANCES[21465] = 808000
CHANCES[21466] = 447000
CHANCES[21467] = 808000
CHANCES[21479] = 477000
CHANCES[21480] = 863000
CHANCES[21481] = 477000
CHANCES[21482] = 863000
CHANCES[21483] = 477000
CHANCES[21484] = 863000
CHANCES[21485] = 477000
CHANCES[21486] = 863000
CHANCES[21498] = 509000
CHANCES[21499] = 920000
CHANCES[21500] = 509000
CHANCES[21501] = 920000
CHANCES[21502] = 509000
CHANCES[21503] = 920000
CHANCES[21504] = 509000
CHANCES[21505] = 920000

q:register_quest_items(TOP_QUALITY_MEAT)
q:add_start_npc(TUNATUN)
q:add_talk_id(TUNATUN)
q:add_kill_id(J.keys(CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31537-03.htm" then
    if player:level() >= 65 then
      st:start_quest()
    else
      htmltext = "31537-02.htm"
      st:exit_quest(true)
    end
  elseif J.is_numeric(event) then
    if q:get_quest_items_count(player, TOP_QUALITY_MEAT) >= 120 then
      htmltext = "31537-06.htm"
      q:take_items(player, TOP_QUALITY_MEAT, -1)
      local reward = REWARDS[J.parse_int(event) + 1]
      q:reward_items(player, reward[1], reward[2])
      st:exit_quest(true, true)
    else
      st:set_cond(1)
      htmltext = "31537-07.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "31537-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "31537-03a.htm"
    elseif cond == 2 then
      if q:get_quest_items_count(player, TOP_QUALITY_MEAT) >= 120 then
        htmltext = "31537-04.htm"
      else
        st:set_cond(1)
        htmltext = "31537-03a.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local qs = q:get_random_party_member_state(player, 1, 3, npc)
  if qs == nil then
    return
  end
  local partyMember = qs:get_player()
  local st = q:get_quest_state(partyMember, false)
  if st == nil then
    return
  end
  if rnd(1000000) < CHANCES[npc:id()] then
    q:give_items(partyMember, TOP_QUALITY_MEAT, 1)
    if q:get_quest_items_count(partyMember, TOP_QUALITY_MEAT) < 120 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

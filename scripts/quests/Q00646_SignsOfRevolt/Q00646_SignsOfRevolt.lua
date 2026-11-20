-- Translated by questport from quests/Q00646_SignsOfRevolt (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(646, "Q00646_SignsOfRevolt", "Signs of Revolt")

-- NPC
local TORRANT = 32016
-- Item
local CURSED_DOLL = 8087
-- Rewards
local REWARDS = {{1880, 9}, {1881, 12}, {1882, 20}, {57, 21600}}

q:register_quest_items(CURSED_DOLL)
q:add_start_npc(TORRANT)
q:add_talk_id(TORRANT)
q:add_kill_id(22029, 22030, 22031, 22032, 22033, 22034, 22035, 22036, 22037, 22038, 22039, 22040, 22041, 22042, 22043, 22044, 22045, 22047, 22049)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32016-03.htm" then
    st:start_quest()
  elseif J.is_numeric(event) then
    htmltext = "32016-07.htm"
    q:take_items(player, CURSED_DOLL, -1)
    local reward = REWARDS[J.parse_int(event) + 1]
    q:give_items(player, reward[1], reward[2])
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 40 and "32016-02.htm" or "32016-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "32016-04.htm"
    elseif cond == 2 then
      htmltext = "32016-05.htm"
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
  if rnd(100) < 75 then
    q:give_items(partyMember, CURSED_DOLL, 1)
    if q:get_quest_items_count(partyMember, CURSED_DOLL) < 180 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

-- Translated by questport from quests/Q00644_GraveRobberAnnihilation (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(644, "Q00644_GraveRobberAnnihilation", "Grave Robber Annihilation")

-- NPC
local KARUDA = 32017
-- Item
local ORC_GRAVE_GOODS = 8088
-- Rewards
local REWARDS = {{1865, 30}, {1867, 40}, {1872, 40}, {1871, 30}, {1870, 30}, {1869, 30}}

q:register_quest_items(ORC_GRAVE_GOODS)
q:add_start_npc(KARUDA)
q:add_talk_id(KARUDA)
q:add_kill_id(22003, 22004, 22005, 22006, 22008)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32017-02.htm" then
    st:start_quest()
  elseif J.is_numeric(event) then
    htmltext = "32017-04.htm"
    q:take_items(player, ORC_GRAVE_GOODS, -1)
    local reward = REWARDS[J.parse_int(event) + 1]
    q:reward_items(player, reward[1], reward[2])
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 20 and "32017-06.htm" or "32017-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "32017-05.htm"
    elseif cond == 2 then
      htmltext = "32017-07.htm"
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
  if rnd.bool() then
    q:give_items(partyMember, ORC_GRAVE_GOODS, 1)
    if q:get_quest_items_count(partyMember, ORC_GRAVE_GOODS) < 120 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

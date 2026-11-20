-- Translated by questport from quests/Q00645_GhostsOfBatur (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(645, "Q00645_GhostsOfBatur", "Ghosts of Batur")

-- NPC
local KARUDA = 32017
-- Item
local CURSED_GRAVE_GOODS = 8089
-- Rewards
local REWARDS = {{1878, 18}, {1879, 7}, {1880, 4}, {1881, 6}, {1882, 10}, {1883, 2}}

q:add_start_npc(KARUDA)
q:add_talk_id(KARUDA)
q:add_kill_id(22007, 22009, 22010, 22011, 22012, 22013, 22014, 22015, 22016)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32017-03.htm" then
    st:start_quest()
  elseif J.is_numeric(event) then
    htmltext = "32017-07.htm"
    q:take_items(player, CURSED_GRAVE_GOODS, -1)
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
    htmltext = player:level() < 23 and "32017-02.htm" or "32017-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "32017-04.htm"
    elseif cond == 2 then
      htmltext = "32017-05.htm"
    end
  end
  return htmltext
end)

-- As Mobius: the goods go to the killer, the cond to the chosen member.
q:on_kill(function(npc, killer, isSummon)
  local player = q:get_random_party_member(killer, 1)
  if player ~= nil and util.in_range(Config.ALT_PARTY_RANGE, npc, player, false) and rnd(100) < 75 then
    local qs = q:get_quest_state(player, false)
    q:give_items(killer, CURSED_GRAVE_GOODS, 1)
    if qs:is_cond(1) and q:get_quest_items_count(killer, CURSED_GRAVE_GOODS) >= 500 then
      qs:set_cond(2, true)
    else
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)

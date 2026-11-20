-- Translated by questport from quests/Q00640_TheZeroHour (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(640, "Q00640_TheZeroHour", "The Zero Hour")

-- NPC
local KAHMAN = 31554
-- Item
local FANG_OF_STAKATO = 8085
local REWARDS = {
  {12, 4042, 1},
  {6, 4043, 1},
  {6, 4044, 1},
  {81, 1887, 10},
  {33, 1888, 5},
  {30, 1889, 10},
  {150, 5550, 10},
  {131, 1890, 10},
  {123, 1893, 5},
}

q:register_quest_items(FANG_OF_STAKATO)
q:add_start_npc(KAHMAN)
q:add_talk_id(KAHMAN)
-- All "spiked" stakatos types, except babies and cannibalistic followers.
q:add_kill_id(22105, 22106, 22107, 22108, 22109, 22110, 22111, 22113, 22114, 22115, 22116, 22117, 22118, 22119, 22121)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31554-02.htm" then
    st:start_quest()
  elseif event == "31554-05.htm" then
    if not q:has_quest_items(player, FANG_OF_STAKATO) then
      htmltext = "31554-06.htm"
    end
  elseif event == "31554-08.htm" then
    st:exit_quest(true, true)
  elseif J.is_numeric(event) then
    local reward = REWARDS[J.parse_int(event) + 1]
    if q:get_quest_items_count(player, FANG_OF_STAKATO) >= reward[1] then
      htmltext = "31554-09.htm"
      q:take_items(player, FANG_OF_STAKATO, reward[1])
      q:reward_items(player, reward[2], reward[3])
    else
      htmltext = "31554-06.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() < 66 then
      htmltext = "31554-00.htm"
    else
      local st2 = player:get_quest_state("Q00109_InSearchOfTheNest")
      htmltext = st2 ~= nil and st2:is_completed() and "31554-01.htm" or "31554-10.htm"
    end
  elseif sw == State.STARTED then
    htmltext = q:has_quest_items(player, FANG_OF_STAKATO) and "31554-04.htm" or "31554-03.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_random_party_member_state(killer, 1, 3, npc)
  if qs ~= nil then
    q:give_item_randomly(qs:get_player(), npc, FANG_OF_STAKATO, 1, 1, 0, 1, true)
  end
end)

-- Translated by questport from quests/Q00601_WatchingEyes (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(601, "Q00601_WatchingEyes", "Watching Eyes")

-- Items
local PROOF_OF_AVENGER = 7188
-- Rewards
local REWARDS = {{6699, 90000, 20}, {6698, 80000, 40}, {6700, 40000, 50}, {0, 230000, 100}}

q:register_quest_items(PROOF_OF_AVENGER)
q:add_start_npc(31683) -- Eye of Argos
q:add_talk_id(31683)
q:add_kill_id(21306, 21308, 21309, 21310, 21311)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31683-03.htm" then
    if player:level() < 71 then
      htmltext = "31683-02.htm"
    else
      st:start_quest()
    end
  elseif event == "31683-07.htm" then
    q:take_items(player, PROOF_OF_AVENGER, -1)
    local random = rnd(100)
    for _, element in ipairs(REWARDS) do
      if random < element[3] then
        q:give_adena(player, element[2], true)
        if element[1] ~= 0 then
          q:give_items(player, element[1], 5)
          q:add_exp_and_sp(player, 120000, 10000)
        end
        break
      end
    end
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "31683-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = q:has_quest_items(player, PROOF_OF_AVENGER) and "31683-05.htm" or "31683-04.htm"
    elseif cond == 2 then
      htmltext = "31683-06.htm"
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
    q:give_items(partyMember, PROOF_OF_AVENGER, 1)
    if q:get_quest_items_count(partyMember, PROOF_OF_AVENGER) < 100 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

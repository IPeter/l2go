-- Translated by questport from quests/Q00510_AClansPrestige (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Needs clans (M9): the clan level and reputation and the clan leader's
-- quest state.
local J = require("lib.java")

local q = quest.new(510, "Q00510_AClansPrestige", "A Clan's Reputation")
q:requires("clans")

-- NPC
local VALDIS = 31331
-- Quest Item
local CLAW = 8767
-- Reward
local CLAN_POINTS_REWARD = 50 -- Quantity of points

q:register_quest_items(CLAW)
q:add_start_npc(VALDIS)
q:add_talk_id(VALDIS)
q:add_kill_id(22215, 22216, 22217)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31331-3.htm" then
    st:start_quest()
  elseif event == "31331-6.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if not player:is_clan_leader() then
      st:exit_quest(true)
      htmltext = "31331-0.htm"
    elseif player:clan():level() < 5 then
      st:exit_quest(true)
      htmltext = "31331-0.htm"
    else
      htmltext = "31331-1.htm"
    end
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      local count = q:get_quest_items_count(player, CLAW)
      if count > 0 then
        local reward = CLAN_POINTS_REWARD * count
        q:take_items(player, CLAW, -1)
        local clan = player:clan()
        clan:add_reputation(reward)
        player:send_system_message(SystemMessageId.YOU_HAVE_SUCCESSFULLY_COMPLETED_A_CLAN_QUEST_S1_POINTS_HAVE_BEEN_ADDED_TO_YOUR_CLAN_S_REPUTATION_SCORE, reward)
        clan:broadcast_pledge_info_update()
        htmltext = "31331-7.htm"
      else
        htmltext = "31331-4.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  -- Retrieve the QuestState of the clan leader.
  local st = q:get_clan_leader_quest_state(player, npc)
  if st == nil or not st:is_started() then
    return
  end
  -- Check if the clan leader is within 1500 range of the raid boss.
  local clanLeader = st:get_player()
  if npc:calculate_distance_3d(clanLeader) < Config.ALT_PARTY_RANGE then
    q:give_items(clanLeader, CLAW, 1)
    q:play_sound(clanLeader, QuestSound.ITEMSOUND_QUEST_MIDDLE)
  end
end)

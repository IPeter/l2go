-- Translated by questport from quests/Q00509_AClansFame (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Needs clans (M9): the clan calls below (clan level and reputation, the
-- clan leader's quest state, the NPC distance) do not exist before.
local J = require("lib.java")

local q = quest.new(509, "Q00509_AClansFame", "The Clan's Prestige")
q:requires("clans")

-- NPCs
local VALDIS = 31331
-- Raid Bosses
local DAIMON_THE_WHITE_EYED = 25290
local HESTIA_GUARDIAN_DEITY = 25293
local PLAGUE_GOLEM = 25523
local DEMONS_AGENT_FALSTON = 25322
local QUEEN_SHYEED = 25514
-- Items
local DAIMONS_EYES = 8489
local HESTIAS_FAIRY_STONE = 8490
local NUCLEUS_OF_LESSER_GOLEM = 8491
local FALSTON_FANG = 8492
local SHAIDS_TALON = 8493
-- Reward list (itemId, minClanPoints, maxClanPoints)
local REWARD_LIST = {
  {DAIMON_THE_WHITE_EYED, DAIMONS_EYES, 180, 215},
  {HESTIA_GUARDIAN_DEITY, HESTIAS_FAIRY_STONE, 430, 465},
  {PLAGUE_GOLEM, NUCLEUS_OF_LESSER_GOLEM, 380, 415},
  {DEMONS_AGENT_FALSTON, FALSTON_FANG, 220, 255},
  {QUEEN_SHYEED, SHAIDS_TALON, 130, 165},
}
-- Radar
local radar = {
  {186320, -43904, -3175},
  {134672, -115600, -1216},
  {170000, -59900, -3848},
  {93296, -75104, -1824},
  {79635, -55612, -5980},
}

q:register_quest_items(DAIMONS_EYES, HESTIAS_FAIRY_STONE, NUCLEUS_OF_LESSER_GOLEM, FALSTON_FANG, SHAIDS_TALON)
q:add_start_npc(VALDIS)
q:add_talk_id(VALDIS)
q:add_kill_id(DAIMON_THE_WHITE_EYED, HESTIA_GUARDIAN_DEITY, PLAGUE_GOLEM, DEMONS_AGENT_FALSTON, QUEEN_SHYEED)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if J.is_numeric(event) then
    local evt = J.parse_int(event)
    st:set("raid", event)
    htmltext = "31331-" .. event .. ".htm"
    local x = radar[evt][1]
    local y = radar[evt][2]
    local z = radar[evt][3]
    if x + y + z > 0 then
      q:add_radar(player, x, y, z)
    end
    st:start_quest()
  elseif event == "31331-6.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local clan = player:clan()
  local sw = st:get_state()
  if sw == State.CREATED then
    if not player:is_clan_leader() then
      st:exit_quest(true)
      htmltext = "31331-0a.htm"
    elseif clan:level() < 6 then
      st:exit_quest(true)
      htmltext = "31331-0b.htm"
    else
      htmltext = "31331-0c.htm"
    end
  elseif sw == State.STARTED then
    local raid = st:get_int("raid")
    if st:is_cond(1) then
      local item = REWARD_LIST[raid][2]
      local count = q:get_quest_items_count(player, item)
      local reward = rnd(REWARD_LIST[raid][3], REWARD_LIST[raid][4])
      if count == 0 then
        htmltext = "31331-" .. raid .. "a.htm"
      elseif count == 1 then
        htmltext = "31331-" .. raid .. "b.htm"
        q:take_items(player, item, 1)
        clan:add_reputation(reward)
        player:send_system_message(SystemMessageId.YOU_HAVE_SUCCESSFULLY_COMPLETED_A_CLAN_QUEST_S1_POINTS_HAVE_BEEN_ADDED_TO_YOUR_CLAN_S_REPUTATION_SCORE, reward)
        clan:broadcast_pledge_info_update()
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
  -- Reward only if quest is setup on good index.
  local raid = st:get_int("raid")
  if REWARD_LIST[raid][1] == npc:id() then
    local item = REWARD_LIST[raid][2]
    local clanLeader = st:get_player()
    -- Check if the clan leader is within 1500 range of the raid boss.
    if clanLeader ~= nil and npc:distance_3d(clanLeader) < Config.ALT_PARTY_RANGE and not q:has_quest_items(clanLeader, item) then
      q:give_items(clanLeader, item, 1)
      q:play_sound(clanLeader, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
  end
end)

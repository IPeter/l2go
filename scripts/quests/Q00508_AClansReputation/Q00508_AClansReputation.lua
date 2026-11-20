-- Translated by questport from quests/Q00508_AClansReputation (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Needs clans (M9): the clan calls below (clan level and reputation, the
-- clan leader's quest state, the NPC distance) do not exist before.
local J = require("lib.java")

local q = quest.new(508, "Q00508_AClansReputation", "A Clan's Reputation")
q:requires("clans")

-- NPC
local SIR_ERIC_RODEMAI = 30868
-- Raidbosses
local FLAMESTONE_GIANT = 25524
local PALIBATI_QUEEN_THEMIS = 25252
local HEKATON_PRIME = 25140
local GARGOYLE_LORD_TIPHON = 25255
local LAST_LESSER_GIANT_GLAKI = 25245
local RAHHA = 25051
-- Items
local NUCLEUS_OF_FLAMESTONE_GIANT = 8494
local THEMIS_SCALE = 8277
local NUCLEUS_OF_HEKATON_PRIME = 8279
local TIPHON_SHARD = 8280
local GLAKIS_NUCLEUS = 8281
local RAHHAS_FANG = 8282
-- Reward list (itemId, minClanPoints, maxClanPoints)
local REWARD_LIST = {
  {PALIBATI_QUEEN_THEMIS, THEMIS_SCALE, 65, 100},
  {HEKATON_PRIME, NUCLEUS_OF_HEKATON_PRIME, 40, 75},
  {GARGOYLE_LORD_TIPHON, TIPHON_SHARD, 30, 65},
  {LAST_LESSER_GIANT_GLAKI, GLAKIS_NUCLEUS, 105, 140},
  {RAHHA, RAHHAS_FANG, 40, 75},
  {FLAMESTONE_GIANT, NUCLEUS_OF_FLAMESTONE_GIANT, 60, 95},
}
-- Radar
local radar = {
  {192346, 21528, -3648},
  {191979, 54902, -7658},
  {170038, -26236, -3824},
  {171762, 55028, -5992},
  {117232, -9476, -3320},
  {144218, -5816, -4722},
}

q:register_quest_items(THEMIS_SCALE, NUCLEUS_OF_HEKATON_PRIME, TIPHON_SHARD, GLAKIS_NUCLEUS, RAHHAS_FANG, NUCLEUS_OF_FLAMESTONE_GIANT)
q:add_start_npc(SIR_ERIC_RODEMAI)
q:add_talk_id(SIR_ERIC_RODEMAI)
q:add_kill_id(FLAMESTONE_GIANT, PALIBATI_QUEEN_THEMIS, HEKATON_PRIME, GARGOYLE_LORD_TIPHON, LAST_LESSER_GIANT_GLAKI, RAHHA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if J.is_numeric(event) then
    local evt = J.parse_int(event)
    st:set("raid", event)
    htmltext = "30868-" .. event .. ".htm"
    local x = radar[evt][1]
    local y = radar[evt][2]
    local z = radar[evt][3]
    if x + y + z > 0 then
      q:add_radar(player, x, y, z)
    end
    st:start_quest()
  elseif event == "30868-7.htm" then
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
      htmltext = "30868-0a.htm"
    elseif clan:level() < 5 then
      st:exit_quest(true)
      htmltext = "30868-0b.htm"
    else
      htmltext = "30868-0c.htm"
    end
  elseif sw == State.STARTED then
    local raid = st:get_int("raid")
    if st:is_cond(1) then
      local item = REWARD_LIST[raid][2]
      local count = q:get_quest_items_count(player, item)
      local reward = rnd(REWARD_LIST[raid][3], REWARD_LIST[raid][4])
      if count == 0 then
        htmltext = "30868-" .. raid .. "a.htm"
      elseif count == 1 then
        htmltext = "30868-" .. raid .. "b.htm"
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
  -- Check if the clan leader is within 1500 range of the raid boss.
  local clanLeader = st:get_player()
  if npc:distance_3d(clanLeader) < Config.ALT_PARTY_RANGE then
    -- Reward only if quest is set up on the correct index.
    local raid = st:get_int("raid")
    if REWARD_LIST[raid][1] == npc:id() then
      local item = REWARD_LIST[raid][2]
      if not q:has_quest_items(clanLeader, item) then
        q:give_items(clanLeader, item, 1)
        q:play_sound(clanLeader, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  end
end)

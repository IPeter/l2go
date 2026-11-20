-- Translated by questport from quests/Q00372_LegacyOfInsolence (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(372, "Q00372_LegacyOfInsolence", "Legacy of Insolence")

-- NPCs
local WALDERAL = 30844
local PATRIN = 30929
local HOLLY = 30839
local CLAUDIA = 31001
local DESMOND = 30855
-- Monsters
local MONSTERS_DROPS = {
  {20817, 20821, 20825, 20829, 21069, 21063},
  {5966, 5966, 5966, 5967, 5968, 5969},
  {300000, 400000, 460000, 400000, 250000, 250000},
}
-- Items
local SCROLLS = {{5989, 6001}, {5984, 5988}, {5979, 5983}, {5972, 5978}, {5972, 5978}}
-- Rewards matrice.
local REWARDS_MATRICE = {
  {{13, 5496}, {26, 5508}, {40, 5525}, {58, 5368}, {76, 5392}, {100, 5426}},
  {{13, 5497}, {26, 5509}, {40, 5526}, {58, 5370}, {76, 5394}, {100, 5428}},
  {{20, 5502}, {40, 5514}, {58, 5527}, {73, 5380}, {87, 5404}, {100, 5430}},
  {{20, 5503}, {40, 5515}, {58, 5528}, {73, 5382}, {87, 5406}, {100, 5432}},
  {{33, 5496}, {66, 5508}, {89, 5525}, {100, 57}},
  {{33, 5497}, {66, 5509}, {89, 5526}, {100, 57}},
  {{35, 5502}, {70, 5514}, {87, 5527}, {100, 57}},
  {{35, 5503}, {70, 5515}, {87, 5528}, {100, 57}},
}

local function checkAndRewardItems(player, itemType, rewardType, npcId)
  -- Retrieve array with items to check.
  local itemsToCheck = SCROLLS[itemType + 1]
  -- Check set of items.
  for item = itemsToCheck[1], itemsToCheck[2] do
    if not q:has_quest_items(player, item) then
      return npcId .. (npcId == WALDERAL and "-07a.htm" or "-01.htm")
    end
  end
  -- Remove set of items.
  for item = itemsToCheck[1], itemsToCheck[2] do
    q:take_items(player, item, 1)
  end
  -- Retrieve array with rewards.
  local rewards = REWARDS_MATRICE[rewardType + 1]
  local chance = rnd(100)
  for _, reward in ipairs(rewards) do
    if chance < reward[1] then
      q:reward_items(player, reward[2], 1)
      return npcId .. "-02.htm"
    end
  end
  return npcId .. (npcId == WALDERAL and "-07a.htm" or "-01.htm")
end

q:add_start_npc(WALDERAL)
q:add_talk_id(WALDERAL, PATRIN, HOLLY, CLAUDIA, DESMOND)
q:add_kill_id(MONSTERS_DROPS[1])

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30844-04.htm" then
    st:start_quest()
  elseif event == "30844-05b.htm" then
    if st:is_cond(1) then
      st:set_cond(2, true)
    end
  elseif event == "30844-07.htm" then
    for blueprint = 5989, 6001 do
      if not q:has_quest_items(player, blueprint) then
        htmltext = "30844-06.htm"
        break
      end
    end
  elseif J.starts_with(event, "30844-07-") then
    checkAndRewardItems(player, 0, J.parse_int(J.substring(event, 9, 10)), WALDERAL)
  elseif event == "30844-09.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 59 and "30844-01.htm" or "30844-02.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == WALDERAL then
      htmltext = "30844-05.htm"
    elseif sw2 == HOLLY then
      htmltext = checkAndRewardItems(player, 1, 4, HOLLY)
    elseif sw2 == PATRIN then
      htmltext = checkAndRewardItems(player, 2, 5, PATRIN)
    elseif sw2 == CLAUDIA then
      htmltext = checkAndRewardItems(player, 3, 6, CLAUDIA)
    elseif sw2 == DESMOND then
      htmltext = checkAndRewardItems(player, 4, 7, DESMOND)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil then
    return
  end
  local npcId = npc:id()
  for i = 0, #MONSTERS_DROPS[1] - 1 do
    if MONSTERS_DROPS[1][i + 1] == npcId and rnd(1000000) < MONSTERS_DROPS[3][i + 1] then
      local partyMember = st:get_player()
      q:give_items(partyMember, MONSTERS_DROPS[2][i + 1], 1)
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      break
    end
  end
end)

-- Translated by questport from quests/Q00648_AnIceMerchantsDream (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(648, "Q00648_AnIceMerchantsDream", "An Ice Merchant's Dream")

-- NPCs
local RAFFORTY = 32020
local ICE_SHELF = 32023
-- Items
local SILVER_HEMOCYTE = 8057
local SILVER_ICE_CRYSTAL = 8077
local BLACK_ICE_CRYSTAL = 8078
-- Rewards
local REWARDS = {}
-- Drop chances
local CHANCES = {}

REWARDS["a"] = {SILVER_ICE_CRYSTAL, 23, 1894} -- Crafted Leather
REWARDS["b"] = {SILVER_ICE_CRYSTAL, 6, 1881} -- Coarse Bone Powder
REWARDS["c"] = {SILVER_ICE_CRYSTAL, 8, 1880} -- Steel
REWARDS["d"] = {BLACK_ICE_CRYSTAL, 1800, 729} -- Scroll: Enchant Weapon (A-Grade)
REWARDS["e"] = {BLACK_ICE_CRYSTAL, 240, 730} -- Scroll: Enchant Armor (A-Grade)
REWARDS["f"] = {BLACK_ICE_CRYSTAL, 500, 947} -- Scroll: Enchant Weapon (B-Grade)
REWARDS["g"] = {BLACK_ICE_CRYSTAL, 80, 948} -- Scroll: Enchant Armor (B-Grade)
CHANCES[22080] = {285000, 48000} -- Massive Maze Bandersnatch
CHANCES[22081] = {443000, 0} -- Lost Watcher
CHANCES[22082] = {510000, 0} -- Baby Panthera
CHANCES[22083] = {510000, 0} -- Elder Lost Watcher
CHANCES[22084] = {477000, 49000} -- Panthera
CHANCES[22085] = {420000, 43000} -- Lost Gargoyle
CHANCES[22086] = {490000, 50000} -- Lost Gargoyle Youngling
CHANCES[22087] = {787000, 81000} -- Pronghorn Spirit
CHANCES[22088] = {480000, 49000} -- Pronghorn
CHANCES[22089] = {550000, 56000} -- Ice Tarantula
CHANCES[22090] = {570000, 58000} -- Frost Tarantula
CHANCES[22092] = {623000, 0} -- Frost Iron Golem
CHANCES[22093] = {910000, 93000} -- Lost Buffalo
CHANCES[22094] = {553000, 57000} -- Frost Buffalo
CHANCES[22096] = {593000, 61000} -- Ursus
CHANCES[22097] = {693000, 71000} -- Lost Yeti
CHANCES[22098] = {717000, 74000} -- Frost Yeti

q:add_start_npc(RAFFORTY, ICE_SHELF)
q:add_talk_id(RAFFORTY, ICE_SHELF)
q:add_kill_id(J.keys(CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = player:get_quest_state(q:name())
  if st == nil then
    return htmltext
  end
  if event == "32020-04.htm" then
    st:start_quest()
  elseif event == "32020-05.htm" then
    st:set_state(State.STARTED)
    st:set_cond(2)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
  else
    local exCond
    local val
    if not (event == "32020-14.htm") and not (event == "32020-15.htm") then
      if J.starts_with(event, "32020-17") then
        local reward = REWARDS[J.substring(event, 8, 9)]
        if q:get_quest_items_count(player, reward[1]) >= reward[2] then
          q:take_items(player, reward[1], reward[2])
          q:reward_items(player, reward[3], 1)
        else
          htmltext = "32020-15a.htm"
        end
      elseif not (event == "32020-20.htm") and not (event == "32020-22.htm") then
        if event == "32023-05.htm" then
          if st:get_int("exCond") == 0 then
            st:set("exCond", tostring((rnd(4) + 1) * 10))
          end
        elseif J.starts_with(event, "32023-06-") then
          exCond = st:get_int("exCond")
          if exCond > 0 then
            htmltext = "32023-06.htm"
            st:set("exCond", tostring(exCond + (J.ends_with(event, "chisel") and 1 or 2)))
            q:play_sound(player, QuestSound.ITEMSOUND_BROKEN_KEY)
            q:take_items(player, 8077, 1)
          end
        elseif J.starts_with(event, "32023-07-") then
          exCond = st:get_int("exCond")
          if exCond > 0 then
            val = J.idiv(exCond, 10)
            if val == exCond - val * 10 + (J.ends_with(event, "knife") and 0 or 2) then
              htmltext = "32023-07.htm"
              q:play_sound(player, QuestSound.ITEMSOUND_ENCHANT_SUCCESS)
              q:reward_items(player, 8078, 1)
            else
              htmltext = "32023-08.htm"
              q:play_sound(player, QuestSound.ITEMSOUND_ENCHANT_FAILED)
            end
            st:set("exCond", "0")
          end
        end
      else
        st:exit_quest(true, true)
      end
    else
      exCond = q:get_quest_items_count(player, 8078)
      val = q:get_quest_items_count(player, 8077)
      if val + exCond > 0 then
        q:take_items(player, 8078, -1)
        q:take_items(player, 8077, -1)
        q:give_adena(player, val * 300 + exCond * 1200, true)
      else
        htmltext = "32020-16a.htm"
      end
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if npc:id() == RAFFORTY then
      if player:level() < 53 then
        htmltext = "32020-01.htm"
      else
        local st2 = player:get_quest_state("Q00115_TheOtherSideOfTruth")
        htmltext = st2 ~= nil and st2:is_completed() and "32020-02.htm" or "32020-03.htm"
      end
    else
      htmltext = "32023-01.htm"
    end
  elseif sw == State.STARTED then
    if npc:id() == RAFFORTY then
      local hasItem = q:has_at_least_one_quest_item(player, SILVER_ICE_CRYSTAL, BLACK_ICE_CRYSTAL)
      local st2 = player:get_quest_state("Q00115_TheOtherSideOfTruth")
      if st2 ~= nil and st2:is_completed() then
        htmltext = hasItem and "32020-11.htm" or "32020-09.htm"
        if st:is_cond(1) then
          st:set_cond(2, true)
        end
      else
        htmltext = hasItem and "32020-10.htm" or "32020-08.htm"
      end
    else
      if not q:has_quest_items(player, SILVER_ICE_CRYSTAL) then
        htmltext = "32023-02.htm"
      else
        if math.fmod(st:get_int("exCond"), 10) == 0 then
          htmltext = "32023-03.htm"
          st:set("exCond", "0")
        else
          htmltext = "32023-04.htm"
        end
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil or not st:is_started() then
    return
  end
  local partyMember = st:get_player()
  local chance = CHANCES[npc:id()]
  if rnd(1000000) < chance[1] then
    q:give_items(partyMember, SILVER_ICE_CRYSTAL, 1)
    q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
  if st:is_cond(2) and chance[2] > 0 and rnd(1000000) < chance[2] then
    q:give_items(partyMember, SILVER_HEMOCYTE, 1)
    q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)

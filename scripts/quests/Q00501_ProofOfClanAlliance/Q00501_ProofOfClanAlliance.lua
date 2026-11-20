-- Translated by questport from quests/Q00501_ProofOfClanAlliance (Mobius CT_0 Interlude); owned by L2Go from here on.
-- @author Rootware, Skache
-- Needs clans (M9): the clan, its leader, clan levels, NPC skill casts and
-- skill effects, and the poison check do not exist before.
local J = require("lib.java")

local q = quest.new(501, "Q00501_ProofOfClanAlliance", "Proof of Clan Alliance")
q:requires("clans")

-- NPC
local SIR_KRISTOF_RODEMAI = 30756
local STATUE_OF_OFFERING = 30757
local ATHREA = 30758
local KALIS = 30759
-- Monsters
local VANOR_SILENOS_SHAMAN = 20685
local HARIT_LIZARDMAN_SHAMAN = 20644
local OEL_MAHUM_WITCH_DOCTOR = 20576
-- Chests
local BOX_OF_ATHREA_1 = 27173
local BOX_OF_ATHREA_2 = 27174
local BOX_OF_ATHREA_3 = 27175
local BOX_OF_ATHREA_4 = 27176
local BOX_OF_ATHREA_5 = 27177
-- Items
local ADENA = 57
local POTION_OF_RECOVERY = 3889
-- Quest Items
local HERB_OF_HARIT = 3832
local HERB_OF_VANOR = 3833
local HERB_OF_OEL_MAHUM = 3834
local BLOOD_OF_EVA = 3835
local ATHREAS_COIN = 3836
local SYMBOL_OF_LOYALTY = 3837
local VOUCHER_OF_FAITH = 3873
local ANTIDOTE_RECIPE_LIST = 3872
-- Reward
local PROOF_OF_ALLIANCE = 3874
-- Quest variables
local NEED_HARIT = "need_harit"
local NEED_VANOR = "need_vanor"
local NEED_OEL = "need_oel"
-- Drops
local DROP = {}
-- Chests spawns
local CHESTS_SPAWN = {
  {102273, 103433, -3512},
  {102190, 103379, -3524},
  {102107, 103325, -3533},
  {102024, 103271, -3500},
  {102327, 103350, -3511},
  {102244, 103296, -3518},
  {102161, 103242, -3529},
  {102078, 103188, -3500},
  {102381, 103267, -3538},
  {102298, 103213, -3532},
  {102215, 103159, -3520},
  {102132, 103105, -3513},
  {102435, 103184, -3515},
  {102352, 103130, -3522},
  {102269, 103076, -3533},
  {102186, 103022, -3541},
}
-- Chests
local CHEST_IDS = {}
-- Trigger (a static field in Mobius: one chest game at a time, reset by a
-- script reload)
local _isSpawned = false

-- Verifies if the player has the poison.
-- @param player the player to check
-- @return {@code true} if the player has {@link AbnormalType#FATAL_POISON} abnormal
local function hasAbnormal(player)
  return player:buff_info_by_abnormal_type(AbnormalType.FATAL_POISON) ~= nil
end

local function castSkill(npc, player, skillId)
  local skill = skills.get(skillId, 1)
  npc:set_target(player)
  npc:cast(skill)
end

local function getClanLeaderQuestState(player, npc)
  local clan = player:clan()
  if clan == nil then
    return nil
  end
  local leader = clan:leader():player()
  if leader == nil then
    return nil
  end
  return leader:get_quest_state(q:name())
end

DROP[VANOR_SILENOS_SHAMAN] = HERB_OF_VANOR
DROP[HARIT_LIZARDMAN_SHAMAN] = HERB_OF_HARIT
DROP[OEL_MAHUM_WITCH_DOCTOR] = HERB_OF_OEL_MAHUM
table.insert(CHEST_IDS, BOX_OF_ATHREA_1)
table.insert(CHEST_IDS, BOX_OF_ATHREA_2)
table.insert(CHEST_IDS, BOX_OF_ATHREA_3)
table.insert(CHEST_IDS, BOX_OF_ATHREA_4)
table.insert(CHEST_IDS, BOX_OF_ATHREA_5)

q:register_quest_items(HERB_OF_HARIT, HERB_OF_VANOR, HERB_OF_OEL_MAHUM, BLOOD_OF_EVA, ATHREAS_COIN, SYMBOL_OF_LOYALTY, VOUCHER_OF_FAITH, ANTIDOTE_RECIPE_LIST)
q:add_start_npc(SIR_KRISTOF_RODEMAI, STATUE_OF_OFFERING)
q:add_talk_id(SIR_KRISTOF_RODEMAI, KALIS, STATUE_OF_OFFERING, ATHREA)
q:add_kill_id(J.keys(DROP))
q:add_kill_id(CHEST_IDS)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30756-07.htm" then
    st:start_quest()
    st:set("state", "1")
  elseif event == "30759-03.htm" then
    st:set_cond(2, true)
    st:set("state", "2")
  elseif event == "30759-07.htm" then
    st:set_cond(3, true)
    st:set("state", "3")
    st:set(NEED_HARIT, "1")
    st:set(NEED_VANOR, "1")
    st:set(NEED_OEL, "1")
    q:take_items(player, SYMBOL_OF_LOYALTY, 3)
    q:give_items(player, ANTIDOTE_RECIPE_LIST, 1)
    skills.apply_effects(skills.get(4082, 1), npc, player)
    q:start_quest_timer("poison", 60000, npc, player, true)
  elseif event == "30757-03.htm" then
    if rnd(10) > 5 then
      local st2 = getClanLeaderQuestState(player, npc)
      st:set_state(State.STARTED)
      st:set("symbol", "1")
      st2:set("symbols", tostring(st2:get_int("symbols") + 1))
      q:give_items(player, SYMBOL_OF_LOYALTY, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
      htmltext = "30757-04.htm"
    else
      castSkill(npc, player, 4083)
      q:start_quest_timer("die", 4000, npc, player, false)
    end
  elseif event == "30758-03.htm" then
    if not _isSpawned then
      local st2 = getClanLeaderQuestState(player, npc)
      if st2 == nil then
        return htmltext
      end
      local clan = st2:get_player():clan()
      if clan == nil then
        return htmltext
      end
      st2:set("state", "4")
      st2:set("bingo", "0")
      st2:set("chests", "0")
      for _, coords in ipairs(CHESTS_SPAWN) do
        local chest = q:add_spawn(CHEST_IDS[rnd(#CHEST_IDS) + 1], {x = coords[1], y = coords[2], z = coords[3], heading = 0}, false, 0)
        chest:set_script_value(clan:id())
      end
      _isSpawned = true
      q:start_quest_timer("despawn", 300000, nil, player, false)
    else
      htmltext = "30758-03a.htm"
    end
  elseif event == "30758-07.htm" then
    if player:adena() >= 10000 then
      if not _isSpawned and getClanLeaderQuestState(player, npc) ~= nil then
        q:take_items(player, ADENA, 10000)
      end
    else
      htmltext = "30758-06.htm"
    end
  elseif event == "die" then
    local st2 = getClanLeaderQuestState(player, npc)
    st:set_state(State.STARTED)
    st:set("symbol", "1")
    st2:set("symbols", tostring(st2:get_int("symbols") + 1))
    q:give_items(player, SYMBOL_OF_LOYALTY, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
    return nil
  elseif event == "poison" then
    if not hasAbnormal(player) then
      q:cancel_quest_timer("poison", npc, player) -- Cancel check timer
    end
    return nil
  elseif event == "despawn" then
    _isSpawned = false
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  if npc:id() == ATHREA then
    if player:is_clan_leader() then
      return htmltext
    end
    local cl = getClanLeaderQuestState(player, npc)
    if cl == nil then
      return htmltext
    end
    if cl:get_int("state") == 3 and q:has_quest_items(cl:get_player(), ANTIDOTE_RECIPE_LIST) and not q:has_quest_items(cl:get_player(), BLOOD_OF_EVA) and cl:get_int(NEED_HARIT) == 0 and cl:get_int(NEED_VANOR) == 0 and cl:get_int(NEED_OEL) == 0 then
      return "30758-01.htm"
    end
    if cl:get_int("state") == 4 then
      if cl:get_int("bingo") == 4 then
        cl:set("state", "5")
        q:give_items(player, BLOOD_OF_EVA, 1)
        cl:unset("chests")
        cl:unset("bingo")
        _isSpawned = false
        return "30758-08.htm"
      end
      return "30758-05.htm"
    end
    if cl:get_int("state") == 5 then
      return "30758-09.htm"
    end
    return htmltext
  end
  local sw = st:get_state()
  if sw == State.CREATED then
    local sw2 = npc:id()
    if sw2 == SIR_KRISTOF_RODEMAI then
      if player:is_clan_leader() then
        if player:clan():level() == 3 then
          if q:has_quest_items(player, PROOF_OF_ALLIANCE) then
            htmltext = "30756-03.htm"
          else
            htmltext = "30756-04.htm"
          end
        elseif player:clan():level() < 3 then
          htmltext = "30756-01.htm"
        else
          htmltext = "30756-02.htm"
        end
      else
        htmltext = "30756-05.htm"
      end
    elseif sw2 == STATUE_OF_OFFERING then
      local cl = getClanLeaderQuestState(player, npc)
      if cl ~= nil and cl:get_int("state") == 2 and cl:get_int("symbols") < 3 then
        if not player:is_clan_leader() then
          if player:level() > 39 then
            htmltext = "30757-01.htm"
          else
            htmltext = "30757-02.htm"
          end
        else
          htmltext = "30757-01a.htm"
        end
      elseif player:clan() ~= nil then
        htmltext = "30757-06.htm"
      end
    end
  elseif sw == State.STARTED then
    local state = st:get_int("state")
    local sw3 = npc:id()
    if sw3 == SIR_KRISTOF_RODEMAI then
      if state == 6 and q:has_quest_items(player, VOUCHER_OF_FAITH) then
        htmltext = "30756-09.htm"
        q:add_exp_and_sp(player, 0, 120000)
        q:take_items(player, VOUCHER_OF_FAITH, -1)
        q:give_items(player, PROOF_OF_ALLIANCE, 1)
        -- htmltext = getAlreadyCompletedMsg();
        st:exit_quest(true, true)
      elseif state > 0 then
        htmltext = "30756-10.htm"
      end
    elseif sw3 == STATUE_OF_OFFERING then
      local cl = getClanLeaderQuestState(player, npc)
      if cl ~= nil and cl:get_int("state") == 2 and st:get_int("symbol") == 1 then
        htmltext = "30757-01b.htm"
      end
    elseif sw3 == KALIS then
      if player:is_clan_leader() then
        if state == 1 and not q:has_quest_items(player, SYMBOL_OF_LOYALTY) then
          htmltext = "30759-01.htm"
        elseif state == 2 then
          if q:get_quest_items_count(player, SYMBOL_OF_LOYALTY) < 3 then
            htmltext = "30759-05.htm"
          else
            htmltext = "30759-06.htm"
          end
        elseif state > 2 and state < 6 and not hasAbnormal(player) then
          st:set_cond(1)
          st:set("state", "1")
          st:unset(NEED_HARIT)
          st:unset(NEED_VANOR)
          st:unset(NEED_OEL)
          q:take_items(player, ANTIDOTE_RECIPE_LIST, -1)
          htmltext = "30759-09.htm"
        elseif state == 5 and hasAbnormal(player) and q:has_quest_items(player, HERB_OF_HARIT, HERB_OF_VANOR, HERB_OF_OEL_MAHUM, BLOOD_OF_EVA) then
          st:set_cond(4, true)
          st:set("state", "6")
          st:unset(NEED_HARIT)
          st:unset(NEED_VANOR)
          st:unset(NEED_OEL)
          q:take_items(player, ANTIDOTE_RECIPE_LIST, -1)
          q:take_items(player, HERB_OF_HARIT, -1)
          q:take_items(player, HERB_OF_VANOR, -1)
          q:take_items(player, HERB_OF_OEL_MAHUM, -1)
          q:take_items(player, BLOOD_OF_EVA, -1)
          q:give_items(player, VOUCHER_OF_FAITH, 1)
          q:give_items(player, POTION_OF_RECOVERY, 1)
          q:cancel_quest_timer("poison", npc, player) -- Cancel check timer
          htmltext = "30759-08.htm"
        elseif state > 2 and state < 6 and hasAbnormal(player) and not q:has_quest_items(player, HERB_OF_HARIT, HERB_OF_VANOR, HERB_OF_OEL_MAHUM, BLOOD_OF_EVA) then
          htmltext = "30759-10.htm"
        elseif state == 6 then
          htmltext = "30759-11.htm"
        end
      else
        htmltext = "30759-12.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = player:get_quest_state(q:name())
  local cl = getClanLeaderQuestState(player, npc)
  if st == nil or cl == nil then
    return
  end
  local npcId = npc:id()
  if DROP[npcId] ~= nil and cl:get_int("state") >= 3 and cl:get_int("state") < 6 then
    local itemId = DROP[npcId]
    local herbFlag
    if itemId == HERB_OF_HARIT then
      herbFlag = NEED_HARIT
    elseif itemId == HERB_OF_VANOR then
      herbFlag = NEED_VANOR
    elseif itemId == HERB_OF_OEL_MAHUM then
      herbFlag = NEED_OEL
    else
      return
    end
    if cl:get_int(herbFlag) == 1 and rnd(10) == 1 then
      q:give_items(player, itemId, 1)
      cl:unset(herbFlag)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif J.contains(CHEST_IDS, npcId) and cl:get_int("state") == 4 then
    if player:clan() == nil or npc:get_script_value() ~= player:clan():id() then
      return
    end
    local chests = cl:get_int("chests")
    local bingo = cl:get_int("bingo")
    if chests == 15 and bingo == 3 or chests == 14 and bingo == 2 or chests == 13 and bingo == 1 or chests == 12 and bingo == 0 or bingo < 4 and rnd(4) == 0 then
      npc:broadcast_say(ChatType.NPC_GENERAL, "##########Bingo!##########")
      cl:set("bingo", tostring(bingo + 1))
    end
    cl:set("chests", tostring(chests + 1))
    if chests >= 15 then
      _isSpawned = false
    end
  end
end)

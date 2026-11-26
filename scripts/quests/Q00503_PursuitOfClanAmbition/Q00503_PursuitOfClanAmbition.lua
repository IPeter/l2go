-- Translated by questport from quests/Q00503_PursuitOfClanAmbition (Mobius CT_0 Interlude); owned by L2Go from here on.
-- @author Jackass, Skache
-- Needs clans (M9): nearly every handler asks for the clan leader's quest
-- state (q:get_clan_leader_quest_state) and the clan.
local J = require("lib.java")

local q = quest.new(503, "Q00503_PursuitOfClanAmbition", "Pursuit of Clan Ambition!")
q:requires("clans")

-- NPCs
local MARTIEN = 30645
local ATHREA = 30758
local KALIS = 30759
local SIR_GUSTAV_ATHEBALDT = 30760
local CORPSE_OF_FRITZ = 30761
local CORPSE_OF_LUTZ = 30762
local CORPSE_OF_KURTZ = 30763
local KUSTO = 30512
local BALTHAZAR = 30764
local SIR_ERIC_RODEMAI = 30868
local IMPERIAL_COFFER = 30765
local CLEO = 30766
-- Monsters
local THUNDER_WYRM = 20243
local THUNDER_WYRM_HOLD = 20282
local DRAKE = 20137
local DRAKE_HOLD = 20285
local BLITZ_WYRM = 27178
local LESSER_GIANT_SOLDIER = 20654
local LESSER_GIANT_SCOUT = 20656
local GRAVE_GUARD = 20668
local GRAVE_KEYMASTER = 27179
local IMPERIAL_SLAVE = 27180
local IMPERIAL_GRAVEKEEPER = 27181
-- Imperial Gravekeeper teleports
local TELEPORT_1_X = 179520
local TELEPORT_1_Y = 6464
local TELEPORT_1_Z = -2706
local TELEPORT_2_X = 171104
local TELEPORT_2_Y = 6496
local TELEPORT_2_Z = -2706
-- Skill
local DARK_HEAL_SKILL = 4080
-- First part items
local GUSTAVS_1ST_LETTER = 3866
local MIST_DRAKE_EGG = 3839
local BLITZ_WYRM_EGG = 3840
local DRAKE_EGG = 3841
local THUNDER_WYRM_EGG = 3842
local BROOCH_OF_THE_MAGPIE = 3843
local BLACK_ANVIL_COIN = 3871
-- Second part items
local GUSTAVS_2ND_LETTER = 3867
local RECIPE_TITANS_POWERSTONE = 3838
local NEBULITE_CRYSTALS = 3844
local BROKEN_TITANS_POWERSTONE = 3845
local TITANS_POWERSTONE = 3846
-- Third part items
local GUSTAVS_3RD_LETTER = 3868
local SCEPTER_OF_JUDGMENT = 3869
local IMPERIAL_KEY = 3847
-- Final item
local SEAL_OF_ASPIRATION = 3870
-- Droplist
local DROPLIST = {
  {THUNDER_WYRM, 2, 10, 500000, THUNDER_WYRM_EGG},
  {THUNDER_WYRM_HOLD, 2, 10, 500000, THUNDER_WYRM_EGG},
  {DRAKE, 2, 10, 500000, DRAKE_EGG},
  {DRAKE_HOLD, 2, 10, 500000, DRAKE_EGG},
  {BLITZ_WYRM, 2, 10, 1000000, BLITZ_WYRM_EGG},
}

local function spawnImperialSlaves(npc)
  for i = 0, 3 do
    q:add_spawn(IMPERIAL_SLAVE, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 0)
  end
end

-- Mobius broadcasts the teleport, sets the position and revalidates the
-- zones: npc:teleport does all three.
local function teleportImperialGravekeeper(npc, x, y, z)
  npc:teleport(x, y, z)
end

q:register_quest_items(MIST_DRAKE_EGG, BLITZ_WYRM_EGG, DRAKE_EGG, THUNDER_WYRM_EGG, BROOCH_OF_THE_MAGPIE, NEBULITE_CRYSTALS, BROKEN_TITANS_POWERSTONE, TITANS_POWERSTONE, IMPERIAL_KEY, GUSTAVS_1ST_LETTER, GUSTAVS_2ND_LETTER, GUSTAVS_3RD_LETTER, SCEPTER_OF_JUDGMENT)
q:add_start_npc(SIR_GUSTAV_ATHEBALDT)
q:add_talk_id(MARTIEN, ATHREA, KALIS, SIR_GUSTAV_ATHEBALDT, CORPSE_OF_FRITZ, CORPSE_OF_LUTZ, CORPSE_OF_KURTZ, KUSTO, BALTHAZAR, SIR_ERIC_RODEMAI, IMPERIAL_COFFER, CLEO)
q:add_kill_id(THUNDER_WYRM_HOLD, THUNDER_WYRM, DRAKE, DRAKE_HOLD, BLITZ_WYRM, LESSER_GIANT_SOLDIER, LESSER_GIANT_SCOUT, GRAVE_GUARD, GRAVE_KEYMASTER, IMPERIAL_GRAVEKEEPER)
q:add_attack_id(IMPERIAL_GRAVEKEEPER)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30760-08.htm" then
    st:start_quest()
    q:give_items(player, GUSTAVS_1ST_LETTER, 1)
  elseif event == "30760-12.htm" then
    q:give_items(player, GUSTAVS_2ND_LETTER, 1)
    st:set_cond(4)
  elseif event == "30760-16.htm" then
    q:give_items(player, GUSTAVS_3RD_LETTER, 1)
    st:set_cond(7)
  elseif event == "30760-20.htm" then
    q:take_items(player, SCEPTER_OF_JUDGMENT, -1)
    q:give_items(player, SEAL_OF_ASPIRATION, 1)
    q:add_exp_and_sp(player, 0, 250000)
    st:exit_quest(false, true)
    q:finish_quest_to_clan(player)
  elseif event == "30760-22.htm" then
    if st:get_cond() == 11 or q:has_quest_items(player, SCEPTER_OF_JUDGMENT) then
      st:set_cond(13)
    end
  elseif event == "30760-23.htm" then
    q:take_items(player, SCEPTER_OF_JUDGMENT, -1)
    q:give_items(player, SEAL_OF_ASPIRATION, 1)
    q:add_exp_and_sp(player, 0, 250000)
    st:exit_quest(false, true)
    q:finish_quest_to_clan(player)
  elseif event == "30645-03.htm" then
    q:set_quest_to_clan_members(player)
    q:take_items(player, GUSTAVS_1ST_LETTER, -1)
    st:set_cond(2)
    st:set("kurt", "0")
  elseif event == "30763-02.htm" then
    q:give_items(player, MIST_DRAKE_EGG, 6)
    q:give_items(player, BROOCH_OF_THE_MAGPIE, 1)
    st:set("kurt", "1")
  elseif event == "30762-02.htm" then
    q:give_items(player, MIST_DRAKE_EGG, 4)
    q:give_items(player, BLITZ_WYRM_EGG, 3)
    q:add_spawn(BLITZ_WYRM, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 0)
    q:add_spawn(BLITZ_WYRM, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 0)
    st:set("lutz", "1")
  elseif event == "30761-02.htm" then
    q:give_items(player, BLITZ_WYRM_EGG, 3)
    q:add_spawn(BLITZ_WYRM, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 0)
    q:add_spawn(BLITZ_WYRM, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 0)
    st:set("fritz", "1")
  elseif event == "30512-03.htm" then
    q:take_items(player, BROOCH_OF_THE_MAGPIE, 1)
    q:give_items(player, BLACK_ANVIL_COIN, 1)
    st:set("kurt", "2")
  elseif event == "30764-03.htm" then
    q:take_items(player, GUSTAVS_2ND_LETTER, -1)
    st:set_cond(5)
  elseif event == "30764-05.htm" then
    q:take_items(player, GUSTAVS_2ND_LETTER, -1)
    st:set_cond(5)
  elseif event == "30764-06.htm" then
    q:take_items(player, BLACK_ANVIL_COIN, -1)
    q:give_items(player, RECIPE_TITANS_POWERSTONE, 1)
  elseif event == "30868-04.htm" then
    q:take_items(player, GUSTAVS_3RD_LETTER, -1)
    st:set_cond(8)
  elseif event == "30868-06a.htm" then
    st:set_cond(10)
  elseif event == "30868-10.htm" then
    st:set_cond(11)
  elseif event == "30766-04.htm" then
    st:set_cond(9)
    npc:broadcast_say(ChatType.NPC_GENERAL, "Blood and honor!")
    local sister1 = q:add_spawn(KALIS, {x = 160665, y = 21209, z = -3710, heading = npc:heading()}, false, 180000)
    sister1:broadcast_say(ChatType.NPC_GENERAL, "Ambition and power!")
    local sister2 = q:add_spawn(ATHREA, {x = 160665, y = 21291, z = -3710, heading = npc:heading()}, false, 180000)
    sister2:broadcast_say(ChatType.NPC_GENERAL, "War and death!")
  elseif event == "OPEN" then
    if not player:is_clan_leader() or not st:is_started() or st:get_cond() ~= 10 or q:has_quest_items(player, SCEPTER_OF_JUDGMENT) then
      htmltext = "30765-02.htm"
    elseif q:get_quest_items_count(player, IMPERIAL_KEY) < 6 then
      htmltext = "30765-03a.htm"
    else
      htmltext = "30765-03.htm"
      st:set_cond(11)
      q:take_items(player, IMPERIAL_KEY, 6)
      q:give_items(player, SCEPTER_OF_JUDGMENT, 1)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local leaderSt = q:get_clan_leader_quest_state(player, npc)
  if npc:id() == IMPERIAL_COFFER and not player:is_clan_leader() and leaderSt ~= nil and leaderSt:is_started() and leaderSt:get_cond() == 10 then
    return "30765-02.htm"
  end
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local state = st:get_state()
  if not player:is_clan_leader() and state == State.CREATED and leaderSt ~= nil and leaderSt:is_started() then
    state = State.STARTED
  end
  if state == State.CREATED then
    repeat
      if npc:id() ~= SIR_GUSTAV_ATHEBALDT then
        break
      end
      if player:clan() == nil then
        htmltext = "30760-01.htm"
        st:exit_quest(true)
      elseif player:is_clan_leader() then
        if q:has_quest_items(player, SEAL_OF_ASPIRATION) then
          htmltext = "30760-03.htm"
          st:exit_quest(true)
        elseif player:clan():level() ~= 4 then
          htmltext = "30760-02.htm"
          st:exit_quest(true)
        else
          htmltext = "30760-04.htm"
        end
      else
        htmltext = "30760-04t.htm"
        st:exit_quest(true)
      end
    until true
  elseif state == State.STARTED then
    local cond = st:get_cond()
    local memberCond = 0
    if leaderSt ~= nil then
      memberCond = leaderSt:get_cond()
    end
    local sw = npc:id()
    if sw == SIR_GUSTAV_ATHEBALDT then
      if player:is_clan_leader() then
        if cond == 1 then
          htmltext = "30760-09.htm"
        elseif cond == 2 then
          htmltext = "30760-10.htm"
        elseif cond == 3 then
          htmltext = "30760-11.htm"
        elseif cond == 4 then
          htmltext = "30760-13.htm"
        elseif cond == 5 then
          htmltext = "30760-14.htm"
        elseif cond == 6 then
          htmltext = "30760-15.htm"
        elseif cond == 7 then
          htmltext = "30760-17.htm"
        elseif cond == 11 then
          htmltext = "30760-19.htm"
        elseif cond == 13 then
          htmltext = "30760-24.htm"
        elseif q:has_quest_items(player, SCEPTER_OF_JUDGMENT) then
          htmltext = "30760-19.htm"
        else
          htmltext = "30760-18.htm"
        end
      else
        if memberCond == 3 then
          htmltext = "30760-11t.htm"
        elseif memberCond == 4 then
          htmltext = "30760-15t.htm"
        elseif memberCond == 11 then
          htmltext = "30760-19t.htm"
        elseif memberCond == 13 then
          htmltext = "30760-24t.htm"
        end
      end
    elseif sw == MARTIEN then
      if player:is_clan_leader() then
        if cond == 1 then
          htmltext = "30645-02.htm"
        elseif cond == 2 then
          if q:get_quest_items_count(player, MIST_DRAKE_EGG) > 9 and q:get_quest_items_count(player, BLITZ_WYRM_EGG) > 9 and q:get_quest_items_count(player, DRAKE_EGG) > 9 and q:get_quest_items_count(player, THUNDER_WYRM_EGG) > 9 then
            htmltext = "30645-05.htm"
            st:set_cond(3)
            q:take_items(player, MIST_DRAKE_EGG, -1)
            q:take_items(player, BLITZ_WYRM_EGG, -1)
            q:take_items(player, DRAKE_EGG, -1)
            q:take_items(player, THUNDER_WYRM_EGG, -1)
          else
            htmltext = "30645-04.htm"
          end
        elseif cond == 3 then
          htmltext = "30645-07.htm"
        else
          htmltext = "30645-08.htm"
        end
      else
        if memberCond == 1 or memberCond == 2 or memberCond == 3 then
          htmltext = "30645-01.htm"
        end
      end
    elseif sw == CORPSE_OF_LUTZ then
      if player:is_clan_leader() and cond == 2 then
        if st:get_int("lutz") == 1 then
          htmltext = "30762-03.htm"
        else
          htmltext = "30762-01.htm"
        end
      end
    elseif sw == CORPSE_OF_KURTZ then
      if player:is_clan_leader() and cond == 2 then
        if st:get_int("kurt") == 1 then
          htmltext = "30763-03.htm"
        else
          htmltext = "30763-01.htm"
        end
      end
    elseif sw == CORPSE_OF_FRITZ then
      if player:is_clan_leader() and cond == 2 then
        if st:get_int("fritz") == 1 then
          htmltext = "30761-03.htm"
        else
          htmltext = "30761-01.htm"
        end
      end
    elseif sw == KUSTO then
      if player:is_clan_leader() then
        if q:get_quest_items_count(player, BROOCH_OF_THE_MAGPIE) == 1 then
          if st:get_int("kurt") == 0 then
            htmltext = "30512-01.htm"
          elseif st:get_int("kurt") == 1 then
            htmltext = "30512-02.htm"
          else
            htmltext = "30512-04.htm"
          end
        end
      else
        if memberCond > 2 and memberCond < 6 then
          htmltext = "30512-01a.htm"
        end
      end
    elseif sw == BALTHAZAR then
      if player:is_clan_leader() then
        if cond == 4 then
          if st:get_int("kurt") == 2 then
            htmltext = "30764-04.htm"
          else
            htmltext = "30764-02.htm"
          end
        elseif cond == 5 then
          if q:get_quest_items_count(player, TITANS_POWERSTONE) > 9 and q:get_quest_items_count(player, NEBULITE_CRYSTALS) > 9 then
            htmltext = "30764-08.htm"
            q:take_items(player, TITANS_POWERSTONE, -1)
            q:take_items(player, NEBULITE_CRYSTALS, -1)
            q:take_items(player, BROOCH_OF_THE_MAGPIE, -1)
            st:set_cond(6)
          else
            htmltext = "30764-07.htm"
          end
        elseif cond == 6 then
          htmltext = "30764-09.htm"
        end
      else
        if memberCond == 4 then
          htmltext = "30764-01.htm"
        end
      end
    elseif sw == SIR_ERIC_RODEMAI then
      if player:is_clan_leader() then
        if cond == 7 then
          htmltext = "30868-02.htm"
        elseif cond == 8 then
          htmltext = "30868-05.htm"
        elseif cond == 9 then
          htmltext = "30868-06.htm"
        elseif cond == 10 then
          htmltext = "30868-08.htm"
        elseif cond == 11 then
          htmltext = "30868-09.htm"
        elseif cond == 12 then
          htmltext = "30868-11.htm"
        end
      else
        if memberCond == 7 then
          htmltext = "30868-01.htm"
        elseif memberCond == 9 or memberCond == 10 then
          htmltext = "30868-07.htm"
        end
      end
    elseif sw == CLEO then
      if player:is_clan_leader() then
        if cond == 8 then
          htmltext = "30766-02.htm"
        elseif cond == 9 then
          htmltext = "30766-05.htm"
        elseif cond == 10 then
          htmltext = "30766-06.htm"
        elseif cond == 11 or cond == 12 or cond == 13 then
          htmltext = "30766-07.htm"
        end
      else
        if memberCond == 8 then
          htmltext = "30766-01.htm"
        end
      end
    elseif sw == IMPERIAL_COFFER then
      if player:is_clan_leader() then
        if cond == 10 and not q:has_quest_items(player, SCEPTER_OF_JUDGMENT) then
          htmltext = q:get_quest_items_count(player, IMPERIAL_KEY) >= 6 and "30765-01.htm" or "30765-03a.htm"
        end
      else
        if memberCond == 10 then
          htmltext = "30765-02.htm"
        end
      end
    elseif sw == KALIS then
      if player:is_clan_leader() then
        htmltext = "30759-01.htm"
      end
    elseif sw == ATHREA then
      if player:is_clan_leader() then
        htmltext = "30758-01.htm"
      end
    end
  end
  return htmltext
end)

q:on_attack(function(npc, attacker, damage, isSummon, skill)
  if npc:is_dead() or npc:id() ~= IMPERIAL_GRAVEKEEPER then
    return
  end
  local realAttacker
  if isSummon and attacker ~= nil then
    realAttacker = attacker
  else
    realAttacker = attacker
  end
  if realAttacker == nil or not realAttacker:is_online() then
    return
  end
  local qs = q:get_clan_leader_quest_state(realAttacker, npc)
  if qs == nil or not qs:is_started() or qs:get_cond() ~= 10 then
    return
  end
  local hpPercent = npc:hp() / npc:max_hp() * 100
  local lastSpawnTrigger = npc:var_int("lastSpawnTrigger", 100)
  if lastSpawnTrigger > 80 and hpPercent <= 80 then
    spawnImperialSlaves(npc)
    npc:set_var("lastSpawnTrigger", 80)
  elseif lastSpawnTrigger > 40 and hpPercent <= 40 then
    spawnImperialSlaves(npc)
    npc:set_var("lastSpawnTrigger", 40)
  elseif lastSpawnTrigger > 20 and hpPercent <= 20 then
    spawnImperialSlaves(npc)
    npc:set_var("lastSpawnTrigger", 20)
  end
  local lastTeleport = npc:var_int("lastTeleport", 100)
  if lastTeleport > 50 and hpPercent <= 50 then
    npc:set_can_return_to_spawn_point(false)
    teleportImperialGravekeeper(npc, TELEPORT_1_X, TELEPORT_1_Y, TELEPORT_1_Z)
    npc:set_var("lastTeleport", 50)
  elseif lastTeleport > 30 and hpPercent <= 30 then
    npc:set_can_return_to_spawn_point(false)
    teleportImperialGravekeeper(npc, TELEPORT_2_X, TELEPORT_2_Y, TELEPORT_2_Z)
    npc:set_var("lastTeleport", 30)
  end
  local selfHealSkill = skills.get(DARK_HEAL_SKILL, 1)
  if selfHealSkill ~= nil then
    local now = time.now_ms()
    local lastCast = npc:var_int("lastDarkHealCast", 0)
    local reuseDelay = 60000
    -- Mobius also checks the skill's reuse and its MP and HP costs here;
    -- npc:cast refuses the cast on those itself (doCast's conditions).
    if now - lastCast >= reuseDelay and not npc:is_casting_now() then
      npc:set_target(npc)
      npc:cast(selfHealSkill, npc)
      npc:set_var("lastDarkHealCast", now)
    end
  end
end)

q:on_kill(function(npc, player, isPet)
  local st = nil
  st = q:get_clan_leader_quest_state(player, npc)
  if st == nil or not st:is_started() then
    return
  end
  local npcId = npc:id()
  local cond = st:get_cond()
  if npcId == IMPERIAL_GRAVEKEEPER then
    local clanLeader = st:get_player()
    if clanLeader ~= nil and clanLeader:is_online() and cond == 10 and not q:has_quest_items(clanLeader, SCEPTER_OF_JUDGMENT) and clanLeader:calculate_distance_3d(npc) <= 1500 then
      local coffer = q:add_spawn(IMPERIAL_COFFER, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 180000)
      coffer:broadcast_say(ChatType.NPC_GENERAL, "Curse of the gods on the one that defiles the property of the empire!")
    end
    return
  end
  if npcId == GRAVE_GUARD then
    local clanLeader = st:get_player()
    if clanLeader ~= nil and clanLeader:is_online() and cond == 10 and not q:has_quest_items(clanLeader, SCEPTER_OF_JUDGMENT) and clanLeader:calculate_distance_3d(npc) <= 1500 then
      local graveGuardKills = st:get_int("grave_guard_kills") + 1
      st:set("grave_guard_kills", tostring(graveGuardKills))
      if graveGuardKills >= 5 and rnd(100) < 50 or graveGuardKills >= 10 then
        st:set("grave_guard_kills", "0")
        q:add_spawn(GRAVE_KEYMASTER, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 0)
      end
    end
    return
  end
  if npcId == GRAVE_KEYMASTER then
    local clanLeader = st:get_player()
    if clanLeader ~= nil and clanLeader:is_online() and cond == 10 and not q:has_quest_items(clanLeader, SCEPTER_OF_JUDGMENT) and q:get_quest_items_count(clanLeader, IMPERIAL_KEY) < 6 and clanLeader:calculate_distance_3d(npc) <= 1500 then
      q:give_items(clanLeader, IMPERIAL_KEY, 1)
      if q:get_quest_items_count(clanLeader, IMPERIAL_KEY) >= 6 then
        q:play_sound(clanLeader, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(clanLeader, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
    return
  end
  if (npcId == DRAKE or npcId == DRAKE_HOLD) and cond == 2 then
    local clanLeader = st:get_player()
    q:give_item_randomly(clanLeader, npc, MIST_DRAKE_EGG, 1, 1, 10, 0.1, true)
    q:give_item_randomly(clanLeader, npc, DRAKE_EGG, 1, 1, 10, 0.5, true)
    return
  end
  -- Handling for LESSER_GIANT_SOLDIER and LESSER_GIANT_SCOUT for retail chances
  if (npcId == LESSER_GIANT_SOLDIER or npcId == LESSER_GIANT_SCOUT) and cond == 5 then
    local clanLeader = st:get_player()
    local roll = rnd(100)
    if roll < 10 and q:get_quest_items_count(clanLeader, TITANS_POWERSTONE) < 10 then
      q:give_items(clanLeader, TITANS_POWERSTONE, 1)
    elseif roll < 30 and q:get_quest_items_count(clanLeader, NEBULITE_CRYSTALS) < 10 then
      q:give_items(clanLeader, NEBULITE_CRYSTALS, 1)
    elseif roll < 80 then
      q:give_items(clanLeader, BROKEN_TITANS_POWERSTONE, 1)
    end
    return -- Skip rest of the drop logic for these NPCs
  end
  for _, element in ipairs(DROPLIST) do
    if element[1] == npcId then
      if cond == element[2] then
        local maxCount = element[3]
        local chance = element[4]
        local item1 = element[5]
        if item1 ~= 0 then
          q:give_item_randomly(st:get_player(), nil, item1, 1, 1, maxCount, chance / 1e+06, true)
        elseif element[1] == GRAVE_GUARD and q:get_quest_items_count(st:get_player(), IMPERIAL_KEY) < 6 and rnd(1000000) < chance then
          q:add_spawn(GRAVE_KEYMASTER, {x = player:x(), y = player:y(), z = player:z(), heading = player:heading()}, true, 0)
        end
      end
    end
  end
end)

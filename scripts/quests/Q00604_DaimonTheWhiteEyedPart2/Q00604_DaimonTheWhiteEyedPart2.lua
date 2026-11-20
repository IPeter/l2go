-- Translated by questport from quests/Q00604_DaimonTheWhiteEyedPart2 (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Daimon the White-Eyed - Part 2 (604)
-- @author Adry_85
local J = require("lib.java")

local q = quest.new(604, "Q00604_DaimonTheWhiteEyedPart2", "Daimon the White-Eyed - Part 2")

-- NPCs
local DAIMONS_ALTAR = 31541
local EYE_OF_ARGOS = 31683
-- Raid Boss
local DAIMON_THE_WHITE_EYED = 25290
-- Items
local UNFINISHED_SUMMON_CRYSTAL = 7192
local SUMMON_CRYSTAL = 7193
local ESSENCE_OF_DAIMON = 7194
-- Rewards
local DYE_I2M2_C = 4595 -- Greater Dye of INT <Int+2 Men-2>
local DYE_I2W2_C = 4596 -- Greater Dye of INT <Int+2 Wit-2>
local DYE_M2I2_C = 4597 -- Greater Dye of MEN <Men+2 Int-2>
local DYE_M2W2_C = 4598 -- Greater Dye of MEN <Men+2 Wit-2>
local DYE_W2I2_C = 4599 -- Greater Dye of WIT <Wit+2 Int-2>
local DYE_W2M2_C = 4600 -- Greater Dye of WIT <Wit+2 Men-2>
-- Location
local DAIMON_THE_WHITE_EYED_LOC = {x = 186320, y = -43904, z = -3175}
-- Misc
local DAIMON_THE_WHITE_EYED_RESPAWN_TIME = "DAIMON_THE_WHITE_EYED_RESPAWN_TIME"
local MIN_LEVEL = 73

local function isDaimonSpawned()
  if time.now_ms() > globals.get_long(DAIMON_THE_WHITE_EYED_RESPAWN_TIME, 0) then
    return world.npc_exists(DAIMON_THE_WHITE_EYED)
  end
  return true
end

q:add_start_npc(EYE_OF_ARGOS)
q:add_talk_id(EYE_OF_ARGOS, DAIMONS_ALTAR)
q:add_spawn_id(DAIMON_THE_WHITE_EYED)
q:add_kill_id(DAIMON_THE_WHITE_EYED)
q:register_quest_items(SUMMON_CRYSTAL, ESSENCE_OF_DAIMON)

q:on_event(function(event, npc, player)
  if "DESPAWN" == event then
    if isDaimonSpawned() then
      npc:broadcast_say(ChatType.NPC_GENERAL, "Can light exist without darkness?")
      npc:delete()
    end
    return nil
  end
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  local htmltext = nil
  if event == "31683-04.htm" then
    q:take_items(player, UNFINISHED_SUMMON_CRYSTAL, 1)
    qs:start_quest()
    qs:set_memo_state(11)
    q:give_items(player, SUMMON_CRYSTAL, 1)
    htmltext = event
  elseif event == "31683-07.html" then
    if q:has_quest_items(player, ESSENCE_OF_DAIMON) then
      local reward
      local random = rnd(1000)
      q:take_items(player, ESSENCE_OF_DAIMON, 1)
      if random < 167 then
        reward = DYE_I2M2_C
      elseif random < 334 then
        reward = DYE_I2W2_C
      elseif random < 501 then
        reward = DYE_M2I2_C
      elseif random < 668 then
        reward = DYE_M2W2_C
      elseif random < 835 then
        reward = DYE_W2I2_C
      else
        reward = DYE_W2M2_C
      end
      q:reward_items(player, reward, 5)
      qs:exit_quest(true, true)
      htmltext = event
    else
      htmltext = "31683-08.html"
    end
  elseif event == "31541-02.html" then
    if q:has_quest_items(player, SUMMON_CRYSTAL) then
      if not isDaimonSpawned() then
        q:take_items(player, SUMMON_CRYSTAL, 1)
        htmltext = event
        q:add_spawn(DAIMON_THE_WHITE_EYED, DAIMON_THE_WHITE_EYED_LOC)
        qs:set_memo_state(21)
        qs:set_cond(2, true)
      else
        htmltext = "31541-03.html"
      end
    else
      htmltext = "31541-04.html"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  if qs:is_created() then
    if player:level() < MIN_LEVEL then
      htmltext = "31683-01.htm"
    elseif not q:has_quest_items(player, UNFINISHED_SUMMON_CRYSTAL) then
      htmltext = "31683-02.htm"
    else
      htmltext = "31683-03.htm"
    end
  elseif qs:is_started() then
    if npc:id() == EYE_OF_ARGOS then
      if qs:is_memo_state(11) then
        htmltext = "31683-05.html"
      elseif qs:get_memo_state() >= 22 and q:has_quest_items(player, ESSENCE_OF_DAIMON) then
        htmltext = "31683-06.html"
      else
        htmltext = "31683-09.html"
      end
    else
      if qs:is_memo_state(11) then
        if isDaimonSpawned() then
          htmltext = "31541-05.html"
        elseif q:has_quest_items(player, SUMMON_CRYSTAL) then
          htmltext = "31541-01.html"
        else
          htmltext = "31541-04.html"
        end
      elseif qs:is_memo_state(21) then
        if not isDaimonSpawned() then
          if q:has_quest_items(player, SUMMON_CRYSTAL) then
            q:take_items(player, SUMMON_CRYSTAL, 1)
            q:add_spawn(DAIMON_THE_WHITE_EYED, DAIMON_THE_WHITE_EYED_LOC)
            htmltext = "31541-02.html"
          else
            htmltext = "31541-04.html"
          end
        else
          htmltext = "31541-03.html"
        end
      elseif qs:get_memo_state() >= 22 then
        htmltext = "31541-05.html"
      end
    end
  end
  return htmltext
end)

q:on_spawn(function(npc)
  q:start_quest_timer("DESPAWN", 1200000, npc, nil)
  npc:broadcast_say(ChatType.NPC_GENERAL, "Who is calling me?")
end)

q:on_kill(function(npc, killer, isSummon)
  q:execute_for_each_player(killer, npc, isSummon, true, false)
  local respawnMinDelay = J.int(43200000 * Config.RAID_MIN_RESPAWN_MULTIPLIER)
  local respawnMaxDelay = J.int(129600000 * Config.RAID_MAX_RESPAWN_MULTIPLIER)
  local respawnDelay = rnd(respawnMinDelay, respawnMaxDelay)
  globals.set(DAIMON_THE_WHITE_EYED_RESPAWN_TIME, time.now_ms() + respawnDelay)
end)

q:on_action_for_each_player(function(player, npc, isSummon)
  local qs = q:get_quest_state(player, false)
  if qs ~= nil and qs:get_memo_state() >= 11 and qs:get_memo_state() <= 21 and util.in_range(Config.ALT_PARTY_RANGE, npc, player, false) then
    -- if (hasQuestItems(player, ESSENCE_OF_DAIMON))
    -- {
    qs:set_cond(3, true)
    qs:set_memo_state(22)
    -- }
    q:give_items(player, ESSENCE_OF_DAIMON, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)

-- Translated by questport from quests/Q00625_TheFinestIngredientsPart2 (Mobius CT_0 Interlude); owned by L2Go from here on.
-- The Finest Ingredients Part - 2 (625)
-- @author Janiko
local J = require("lib.java")

local q = quest.new(625, "Q00625_TheFinestIngredientsPart2", "The Finest Ingredients - Part 2")

-- NPCs
local JEREMY = 31521
local YETIS_TABLE = 31542
-- Monster
local ICICLE_EMPEROR_BUMBALUMP = 25296
-- Required Item
local SOY_SOURCE_JAR = {id = 7205, count = 1}
-- Quest Items
local FOOD_FOR_BUMBALUMP = {id = 7209, count = 1}
local SPECIAL_YETI_MEAT = {id = 7210, count = 1}
-- Rewards
local GREATER_DYE_OF_STR_1 = {id = 4589, count = 5}
local GREATER_DYE_OF_STR_2 = {id = 4590, count = 5}
local GREATER_DYE_OF_CON_1 = {id = 4591, count = 5}
local GREATER_DYE_OF_CON_2 = {id = 4592, count = 5}
local GREATER_DYE_OF_DEX_1 = {id = 4593, count = 5}
local GREATER_DYE_OF_DEX_2 = {id = 4594, count = 5}
-- Location
local ICICLE_EMPEROR_BUMBALUMP_LOC = {x = 158240, y = -121536, z = -2222}
-- Misc
local ICICLE_EMPEROR_BUMBALUMP_RESPAWN_TIME = "ICICLE_EMPEROR_BUMBALUMP_RESPAWN_TIME"
local MIN_LEVEL = 73

local function isBumbalumpSpawned()
  if time.now_ms() > globals.get_long(ICICLE_EMPEROR_BUMBALUMP_RESPAWN_TIME, 0) then
    return world.npc_exists(ICICLE_EMPEROR_BUMBALUMP)
  end
  return true
end

q:add_start_npc(JEREMY)
q:add_talk_id(JEREMY, YETIS_TABLE)
q:add_spawn_id(ICICLE_EMPEROR_BUMBALUMP)
q:add_kill_id(ICICLE_EMPEROR_BUMBALUMP)
q:register_quest_items(FOOD_FOR_BUMBALUMP.id, SPECIAL_YETI_MEAT.id)

q:on_event(function(event, npc, player)
  if event == "NPC_TALK" then
    if isBumbalumpSpawned() then
      npc:broadcast_say(ChatType.NPC_GENERAL, "Oooh!")
    end
    return nil
  end
  local qs = q:get_quest_state(player, false)
  local htmltext = nil
  if qs == nil then
    return htmltext
  end
  if event == "31521-04.htm" then
    if qs:is_created() then
      qs:start_quest()
      q:take_items(player, SOY_SOURCE_JAR)
      q:give_items(player, FOOD_FOR_BUMBALUMP)
      htmltext = event
    end
  elseif event == "31521-08.html" then
    if qs:is_cond(3) then
      if q:get_quest_items_count(player, SPECIAL_YETI_MEAT.id) >= SPECIAL_YETI_MEAT.count then
        local random = rnd(1000)
        if random < 167 then
          q:reward_items(player, GREATER_DYE_OF_STR_1)
        elseif random < 334 then
          q:reward_items(player, GREATER_DYE_OF_STR_2)
        elseif random < 501 then
          q:reward_items(player, GREATER_DYE_OF_CON_1)
        elseif random < 668 then
          q:reward_items(player, GREATER_DYE_OF_CON_2)
        elseif random < 835 then
          q:reward_items(player, GREATER_DYE_OF_DEX_1)
        elseif random < 1000 then
          q:reward_items(player, GREATER_DYE_OF_DEX_2)
        end
        qs:exit_quest(true, true)
        htmltext = event
      else
        htmltext = "31521-09.html"
      end
    end
  elseif event == "31542-02.html" then
    if qs:is_cond(1) then
      if q:get_quest_items_count(player, FOOD_FOR_BUMBALUMP.id) >= FOOD_FOR_BUMBALUMP.count then
        qs:set_cond(2, true)
        if not isBumbalumpSpawned() then
          q:take_items(player, FOOD_FOR_BUMBALUMP)
          q:add_spawn(ICICLE_EMPEROR_BUMBALUMP, ICICLE_EMPEROR_BUMBALUMP_LOC)
          htmltext = event
        else
          htmltext = "31542-03.html"
        end
      else
        htmltext = "31542-04.html"
      end
    elseif qs:is_cond(2) then
      if q:get_quest_items_count(player, FOOD_FOR_BUMBALUMP.id) >= FOOD_FOR_BUMBALUMP.count then
        if not isBumbalumpSpawned() then
          q:take_items(player, FOOD_FOR_BUMBALUMP)
          q:add_spawn(ICICLE_EMPEROR_BUMBALUMP, ICICLE_EMPEROR_BUMBALUMP_LOC)
          htmltext = event
        else
          htmltext = "31542-03.html"
        end
      end
    end
  end
  return htmltext
end)

q:on_talk(function(npc, talker)
  local qs = q:get_quest_state(talker, true)
  local htmltext = q:get_no_quest_msg(talker)
  local sw = npc:id()
  if sw == JEREMY then
    if qs:is_created() then
      if talker:level() >= MIN_LEVEL then
        htmltext = q:get_quest_items_count(talker, SOY_SOURCE_JAR.id) >= SOY_SOURCE_JAR.count and "31521-01.htm" or "31521-02.htm"
      else
        htmltext = "31521-03.htm"
      end
    elseif qs:is_started() then
      local sw2 = qs:get_cond()
      if sw2 == 1 then
        htmltext = "31521-05.html"
      elseif sw2 == 2 then
        htmltext = "31521-06.html"
      elseif sw2 == 3 then
        htmltext = "31521-07.html"
      end
    elseif qs:is_completed() then
      htmltext = q:get_already_completed_msg(talker)
    end
  elseif sw == YETIS_TABLE then
    local sw3 = qs:get_cond()
    if sw3 == 1 then
      if not isBumbalumpSpawned() then
        htmltext = "31542-01.html"
      else
        htmltext = "31542-05.html"
      end
    elseif sw3 == 2 then
      if not isBumbalumpSpawned() then
        if q:get_quest_items_count(talker, FOOD_FOR_BUMBALUMP.id) >= FOOD_FOR_BUMBALUMP.count then
          q:take_items(talker, FOOD_FOR_BUMBALUMP)
          q:add_spawn(ICICLE_EMPEROR_BUMBALUMP, ICICLE_EMPEROR_BUMBALUMP_LOC)
          htmltext = "31542-02.html"
        else
          htmltext = "31542-04.html"
        end
      else
        htmltext = "31542-03.html"
      end
    elseif sw3 == 3 then
      htmltext = "31542-05.html"
    end
  end
  return htmltext
end)

q:on_spawn(function(npc)
  q:start_quest_timer("NPC_TALK", 1000 * 1200, npc, nil)
  npc:broadcast_say(ChatType.NPC_GENERAL, "I smell something delicious...")
end)

q:on_kill(function(npc, killer, isSummon)
  q:execute_for_each_player(killer, npc, isSummon, true, false)
  local respawnMinDelay = J.int(43200000 * Config.RAID_MIN_RESPAWN_MULTIPLIER)
  local respawnMaxDelay = J.int(129600000 * Config.RAID_MAX_RESPAWN_MULTIPLIER)
  local respawnDelay = rnd(respawnMinDelay, respawnMaxDelay)
  globals.set(ICICLE_EMPEROR_BUMBALUMP_RESPAWN_TIME, time.now_ms() + respawnDelay)
end)

q:on_action_for_each_player(function(player, npc, isSummon)
  local qs = q:get_quest_state(player, false)
  if qs ~= nil and qs:is_cond(2) and util.in_range(Config.ALT_PARTY_RANGE, npc, player, true) then
    qs:set_cond(3, true)
    q:give_items(player, SPECIAL_YETI_MEAT)
  end
end)

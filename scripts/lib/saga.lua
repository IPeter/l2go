-- Sagas: the third class change quests Q070 to Q100, ported by hand from
-- Mobius CT_0 Interlude quests/AbstractSagaQuest.java. Each saga file is
-- data only (written by cmd/questport):
--
--   local saga = require("lib.saga")
--   saga.define({
--     id = 70, name = "Q00070_SagaOfThePhoenixKnight", description = "Saga of the Phoenix Knight",
--     npc = {...},        -- 12 NPC IDs (Mobius _npc)
--     items = {...},      -- 12 item IDs (_items; the 12th may be 0)
--     mob = {...},        -- 3 monster IDs (_mob)
--     class_id = {...},   -- the new class (_classId)
--     prev_class = {...}, -- the class the saga is for (_prevClass)
--     x = {...}, y = {...}, z = {...}, -- 3 spawn points (_x, _y, _z)
--     text = {...},       -- 18 NPC lines (_text); PLAYERNAME is the player's name
--   })
--
-- saga.define makes the quest, registers its NPCs and handlers, and returns
-- it. Every table is 1-based: Mobius's _npc[4] is npc[5] here, _text[13]
-- is text[14], and so on.
--
-- Mobius keeps the NPCs a saga spawned (SPAWN_LIST, shared by every saga)
-- in a static map Npc -> player object ID. Here it is a table keyed by the
-- NPC's object ID, also shared by every saga (this module loads once), and
-- like Mobius's map it starts empty after a reload.
--
-- Not there yet (the calls light up once the server has them):
--   * NPC AI: Mobius makes the spawned monsters fight (addDamageHate and
--     setIntentionAttack); npc:add_damage_hate and npc:attack (Plan 11,
--     M6/M7) are used when they exist, otherwise the NPCs only talk.
--   * The skill animation Mobius shows with each Tablet of Vision and the
--     class change (MagicSkillUse 4546 and 4339): x:magic_skill_use.
--   * Parties and subclasses: player:party() is nil and
--     player:is_sub_class_active() false until they exist, which takes the
--     solo, base class branches.

local J = require("lib.java")

local saga = {}

-- QUEST_CLASSES[id - 66] are the classes that may do saga `id` (Mobius
-- QUEST_CLASSES[getId() - 67]); the first three are Kamael sagas, which
-- Interlude does not have.
local QUEST_CLASSES = {
  { 0x7f }, { 0x80, 0x81 }, { 0x82 }, { 0x05 }, { 0x14 }, { 0x15 },
  { 0x02 }, { 0x03 }, { 0x2e }, { 0x30 }, { 0x33 }, { 0x34 }, { 0x08 },
  { 0x17 }, { 0x24 }, { 0x09 }, { 0x18 }, { 0x25 }, { 0x10 }, { 0x11 },
  { 0x1e }, { 0x0c }, { 0x1b }, { 0x28 }, { 0x0e }, { 0x1c }, { 0x29 },
  { 0x0d }, { 0x06 }, { 0x22 }, { 0x21 }, { 0x2b }, { 0x37 }, { 0x39 },
}

local ARCHON_HALISHA_NORM = { 18212, 18214, 18215, 18216, 18218 }
local ARCHON_MINION_FIRST, ARCHON_MINION_LAST = 21646, 21651
local GUARDIAN_ANGEL_FIRST, GUARDIAN_ANGEL_LAST = 27214, 27216

-- The Kamael class Mobius checks for the second entry of class_id and
-- prev_class (dead code in Interlude).
local KAMAEL_SECOND = 0x81

-- Mobius World.getFirstVisibleObject looks in the surrounding world
-- regions; this range stands for them.
local VISIBLE_RANGE = 4096

local SKILL_CLASS_CHANGE = 4339
local SKILL_TABLET = 4546

-- spawn_list is Mobius SPAWN_LIST: NPC object ID -> the object ID of the
-- player it was spawned for.
local spawn_list = {}

-------------------------------------------------------------------------------
-- Checking the data
-------------------------------------------------------------------------------

-- check_list returns what is wrong with def[field] (min to max entries of
-- kind "number" or "string"; zero allows 0), or nil.
local function check_list(def, field, min, max, kind, zero)
  local t = def[field]
  if type(t) ~= "table" then
    return field .. " must be a list"
  end
  if #t < min or #t > max then
    if min == max then
      return field .. " must have " .. min .. " entries, has " .. #t
    end
    return field .. " must have " .. min .. " to " .. max .. " entries, has " .. #t
  end
  for i = 1, #t do
    local v = t[i]
    if kind == "string" then
      if type(v) ~= "string" then
        return field .. "[" .. i .. "] must be a string"
      end
    elseif type(v) ~= "number" or v ~= math.floor(v) or (v == 0 and not zero) then
      return field .. "[" .. i .. "] must be " .. (zero and "an integer" or "a non-zero integer")
    end
  end
  return nil
end

-- check_def returns what is wrong with def for AbstractSagaQuest, or nil.
local function check_def(def)
  if type(def) ~= "table" then
    return "a table expected, got " .. type(def)
  end
  if type(def.id) ~= "number" or QUEST_CLASSES[def.id - 66] == nil then
    return "id must be a saga quest ID from 67 to 100"
  end
  if type(def.name) ~= "string" or type(def.description) ~= "string" then
    return "name and description must be strings"
  end
  return check_list(def, "npc", 12, 12, "number", false)
    or check_list(def, "items", 12, 12, "number", true)
    or check_list(def, "mob", 3, 3, "number", false)
    or check_list(def, "class_id", 1, 2, "number", false)
    or check_list(def, "prev_class", 1, 2, "number", false)
    or check_list(def, "x", 3, 3, "number", true)
    or check_list(def, "y", 3, 3, "number", true)
    or check_list(def, "z", 3, 3, "number", true)
    or check_list(def, "text", 18, 18, "string", false)
end

-------------------------------------------------------------------------------
-- Helpers shared by every saga (Mobius's private methods)
-------------------------------------------------------------------------------

-- auto_chat is Mobius autoChat: the NPC says text in general chat.
local function auto_chat(npc, text)
  if npc ~= nil then
    npc:broadcast_say(ChatType.GENERAL, text)
  end
end

-- cast is Mobius cast: the skill animation on the target and on the NPC.
local function cast(npc, target, skill_id, level)
  if target ~= nil and target.magic_skill_use ~= nil then
    target:magic_skill_use(target, skill_id, level, 6000, 1)
  end
  if npc ~= nil and npc.magic_skill_use ~= nil then
    npc:magic_skill_use(npc, skill_id, level, 6000, 1)
  end
end

-- attack is addDamageHate(target, 0, hate) (when hate is given) and
-- setIntentionAttack(target).
local function attack(npc, target, hate)
  if npc == nil or target == nil then
    return
  end
  if hate ~= nil and npc.add_damage_hate ~= nil then
    npc:add_damage_hate(target, 0, hate)
  end
  if npc.attack ~= nil then
    npc:attack(target)
  end
end

-- track is Mobius addSpawn(st, mob): mob was spawned for st's player.
local function track(st, mob)
  spawn_list[mob:object_id()] = st:get_player():object_id()
end

-- find_spawn is Mobius findSpawn: npc if it was spawned for player.
local function find_spawn(player, npc)
  if npc ~= nil and npc.object_id ~= nil and spawn_list[npc:object_id()] == player:object_id() then
    return npc
  end
  return nil
end

-- delete_spawn is Mobius deleteSpawn: forgets npc and deletes it, if it was
-- a saga spawn.
local function delete_spawn(npc)
  if npc == nil then
    return
  end
  local id = npc:object_id()
  if spawn_list[id] ~= nil then
    spawn_list[id] = nil
    npc:delete()
  end
end

-- is_visible stands for World.getFirstVisibleObject(npc, Npc.class,
-- it -> it == other) != null.
local function is_visible(npc, other)
  return other ~= nil and not other:is_dead() and npc:is_inside_radius_3d(other, VISIBLE_RANGE)
end

-------------------------------------------------------------------------------
-- saga.define
-------------------------------------------------------------------------------

-- saga.define(def) makes the saga quest described by def (see the top of
-- this file) and returns it.
function saga.define(def)
  local problem = check_def(def)
  if problem ~= nil then
    local name = type(def) == "table" and def.name ~= nil and (" " .. tostring(def.name)) or ""
    -- Level 3 in gopher-lua: the saga file that called saga.define.
    error("saga.define" .. name .. ": " .. problem, 3)
  end
  local ID = def.id
  local NPC, ITEMS, MOB = def.npc, def.items, def.mob
  local CLASS_ID, PREV_CLASS = def.class_id, def.prev_class
  local X, Y, Z, TEXT = def.x, def.y, def.z, def.text

  local q = quest.new(ID, def.name, def.description)

  -- text returns TEXT[i] with PLAYERNAME replaced by the player's name.
  local function text(i, player)
    return J.replace(TEXT[i], "PLAYERNAME", player:name())
  end

  -- spawn is Mobius addSpawn(id, x, y, z, -1). Mobius's -1 heading is a
  -- random one; the spawn here faces 0.
  local function spawn(npc_id, x, y, z)
    local npc = q:add_spawn(npc_id, { x = x, y = y, z = z })
    if npc == nil then
      error("saga " .. def.name .. ": could not spawn NPC " .. npc_id, 3)
    end
    return npc
  end

  -- find_right_state is Mobius findRightState: the state of the player npc
  -- was spawned for, if that player is online.
  local function find_right_state(npc)
    local id = spawn_list[npc:object_id()]
    if id == nil then
      return nil
    end
    local player = world.get_player(id)
    if player == nil then
      return nil
    end
    return player:get_quest_state(def.name)
  end

  -- find_quest is Mobius findQuest: the player's state, if the player has
  -- the class of this saga.
  local function find_quest(player)
    local st = q:get_quest_state(player, false)
    if st ~= nil then
      local classes = QUEST_CLASSES[ID - 66]
      local class_id = player:class_id()
      if ID == 68 then
        for i = 1, 2 do
          if class_id == classes[i] then
            return st
          end
        end
      elseif class_id == classes[1] then
        return st
      end
    end
    return nil
  end

  -- get_player_class is Mobius getPlayerClass: the class the saga gives.
  local function get_player_class(player)
    if player:class_id() == KAMAEL_SECOND then
      return CLASS_ID[2]
    end
    return CLASS_ID[1]
  end

  -- get_prev_class is Mobius getPrevClass: the class the saga is for.
  local function get_prev_class(player)
    if player:class_id() == KAMAEL_SECOND then
      if #PREV_CLASS == 1 then
        return -1
      end
      return PREV_CLASS[2]
    end
    return PREV_CLASS[1]
  end

  -- change_class gives the new class (both rewards share it).
  local function change_class(npc, player)
    local player_class = get_player_class(player)
    local prev_class = get_prev_class(player)
    player:set_class_id(player_class)
    if not player:is_sub_class_active() and player:base_class() == prev_class then
      player:set_base_class(player_class)
    end
    player:broadcast_user_info()
    cast(npc, player, SKILL_CLASS_CHANGE, 1)
  end

  -- reward is the exp, adena and the Book of Giants (6622).
  local function reward(player)
    q:add_exp_and_sp(player, 2299404, 0)
    q:give_adena(player, 5000000, true)
    q:give_items(player, 6622, 1)
  end

  -- give_halisha_mark is Mobius giveHalishaMark: marks of Halisha until
  -- 700, then Archon of Halisha comes for the player.
  local function give_halisha_mark(st)
    if st:get_int("spawned") ~= 0 then
      return
    end
    local player = st:get_player()
    if q:get_quest_items_count(player, ITEMS[4]) >= 700 then
      q:take_items(player, ITEMS[4], 20)
      local archon = spawn(MOB[2], player:x(), player:y(), player:z())
      track(st, archon)
      st:set("spawned", "1")
      q:start_quest_timer("Archon Hellisha has despawned", 600000, archon, player)
      auto_chat(archon, text(14, player))
      attack(archon, player, 99999)
    else
      q:give_items(player, ITEMS[4], rnd(1, 4))
    end
  end

  -- registerNPCs
  q:add_start_npc(NPC[1])
  q:add_attack_id(MOB[3])
  q:add_attack_id(MOB[2])
  q:add_skill_see_id(MOB[2])
  q:add_first_talk_id(NPC[5])
  q:add_talk_id(NPC)
  q:add_kill_id(MOB)
  -- Not the Ice Crystal and the Divine Stone of Wisdom (items 1 and 3).
  local quest_items = J.copy(ITEMS)
  quest_items[1] = 0
  quest_items[3] = 0
  q:register_quest_items(quest_items)
  for id = ARCHON_MINION_FIRST, ARCHON_MINION_LAST do
    q:add_kill_id(id)
  end
  q:add_kill_id(ARCHON_HALISHA_NORM)
  for id = GUARDIAN_ANGEL_FIRST, GUARDIAN_ANGEL_LAST do
    q:add_kill_id(id)
  end

  q:on_event(function(event, npc, player)
    if player == nil then
      return nil
    end
    local st = q:get_quest_state(player, false)
    if st == nil then
      return nil
    end

    if event == "0-011.htm" or event == "0-012.htm" or event == "0-013.htm" or event == "0-014.htm"
      or event == "0-015.htm" then
      return event
    elseif event == "accept" then
      st:start_quest()
      q:give_items(player, ITEMS[11], 1)
      return "0-03.htm"
    elseif event == "0-1" then
      if player:level() < 76 then
        if st:is_created() then
          st:exit_quest(true)
        end
        return "0-02.htm"
      end
      return "0-05.htm"
    elseif event == "0-2" then
      if player:level() < 76 then
        q:take_items(player, ITEMS[11], -1)
        st:set_cond(20, true)
        return "0-08.htm"
      end
      q:take_items(player, ITEMS[11], -1)
      reward(player)
      change_class(npc, player)
      st:exit_quest(false)
      return "0-07.htm"
    elseif event == "1-3" then
      st:set_cond(3)
      return "1-05.htm"
    elseif event == "1-4" then
      st:set_cond(4)
      q:take_items(player, ITEMS[1], 1)
      if ITEMS[12] ~= 0 then
        q:take_items(player, ITEMS[12], 1)
      end
      q:give_items(player, ITEMS[2], 1)
      return "1-06.htm"
    elseif event == "2-1" then
      st:set_cond(2)
      return "2-05.htm"
    elseif event == "2-2" then
      st:set_cond(5)
      q:take_items(player, ITEMS[2], 1)
      q:give_items(player, ITEMS[5], 1)
      return "2-06.htm"
    elseif event == "3-5" then
      return "3-07.htm"
    elseif event == "3-6" then
      st:set_cond(11)
      return "3-02.htm"
    elseif event == "3-7" then
      st:set_cond(12)
      return "3-03.htm"
    elseif event == "3-8" then
      st:set_cond(13)
      q:take_items(player, ITEMS[3], 1)
      q:give_items(player, ITEMS[8], 1)
      return "3-08.htm"
    elseif event == "4-1" then
      return "4-010.htm"
    elseif event == "4-2" then
      q:give_items(player, ITEMS[10], 1)
      st:set_cond(18, true)
      return "4-011.htm"
    elseif event == "4-3" then
      q:give_items(player, ITEMS[10], 1)
      st:set_cond(18, true)
      if npc ~= nil then
        auto_chat(npc, text(14, player))
      end
      st:set("Quest0", "0")
      q:cancel_quest_timer("Mob_2 has despawned", npc, player)
      delete_spawn(npc)
      return nil
    elseif event == "5-1" then
      st:set_cond(6, true)
      q:take_items(player, ITEMS[5], 1)
      cast(npc, player, SKILL_TABLET, 1)
      return "5-02.htm"
    elseif event == "6-1" then
      st:set_cond(8, true)
      st:set("spawned", "0")
      q:take_items(player, ITEMS[6], 1)
      cast(npc, player, SKILL_TABLET, 1)
      return "6-03.htm"
    elseif event == "7-1" then
      local spawned = st:get_int("spawned")
      if spawned == 1 then
        return "7-03.htm"
      elseif spawned == 0 then
        local mob1 = spawn(MOB[1], X[1], Y[1], Z[1])
        st:set("spawned", "1")
        q:start_quest_timer("Mob_1 Timer 1", 500, mob1, player)
        q:start_quest_timer("Mob_1 has despawned", 300000, mob1, player)
        track(st, mob1)
        return "7-02.htm"
      end
      return "7-04.htm"
    elseif event == "7-2" then
      st:set_cond(10, true)
      q:take_items(player, ITEMS[7], 1)
      cast(npc, player, SKILL_TABLET, 1)
      return "7-06.htm"
    elseif event == "8-1" then
      st:set_cond(14, true)
      q:take_items(player, ITEMS[8], 1)
      cast(npc, player, SKILL_TABLET, 1)
      return "8-02.htm"
    elseif event == "9-1" then
      st:set_cond(17, true)
      st:set("Quest0", "0")
      st:set("Tab", "0")
      q:take_items(player, ITEMS[9], 1)
      cast(npc, player, SKILL_TABLET, 1)
      return "9-03.htm"
    elseif event == "10-1" then
      if st:get_int("Quest0") == 0 then
        local mob3 = spawn(MOB[3], X[2], Y[2], Z[2])
        local mob2 = spawn(NPC[5], X[3], Y[3], Z[3])
        track(st, mob3)
        track(st, mob2)
        st:set("Mob_2", tostring(mob2:object_id()))
        st:set("Quest0", "1")
        st:set("Quest1", "45")
        q:start_quest_timer("Mob_3 Timer 1", 500, mob3, player, true)
        q:start_quest_timer("Mob_3 has despawned", 59000, mob3, player)
        q:start_quest_timer("Mob_2 Timer 1", 500, mob2, player)
        q:start_quest_timer("Mob_2 has despawned", 60000, mob2, player)
        return "10-02.htm"
      elseif st:get_int("Quest1") == 45 then
        return "10-03.htm"
      end
      return "10-04.htm"
    elseif event == "10-2" then
      st:set_cond(19, true)
      q:take_items(player, ITEMS[10], 1)
      cast(npc, player, SKILL_TABLET, 1)
      return "10-06.htm"
    elseif event == "11-9" then
      st:set_cond(15)
      return "11-03.htm"
    end

    -- Timers: they all have the spawned NPC.
    if npc == nil then
      return nil
    end
    if event == "Mob_1 Timer 1" then
      auto_chat(npc, text(1, player))
    elseif event == "Mob_1 has despawned" then
      auto_chat(npc, text(2, player))
      st:set("spawned", "0")
      delete_spawn(npc)
    elseif event == "Archon Hellisha has despawned" then
      auto_chat(npc, text(7, player))
      st:set("spawned", "0")
      delete_spawn(npc)
    elseif event == "Mob_3 Timer 1" then
      local mob2 = find_spawn(player, world.find_object(st:get_int("Mob_2")))
      if is_visible(npc, mob2) then
        attack(npc, mob2, 99999)
        attack(mob2, npc)
        auto_chat(npc, text(15, player))
        q:cancel_quest_timer("Mob_3 Timer 1", npc, player)
      end
    elseif event == "Mob_3 has despawned" then
      auto_chat(npc, text(16, player))
      st:set("Quest0", "2")
      delete_spawn(npc)
    elseif event == "Mob_2 Timer 1" then
      auto_chat(npc, text(8, player))
      q:start_quest_timer("Mob_2 Timer 2", 1500, npc, player)
      if st:get_int("Quest1") == 45 then
        st:set("Quest1", "0")
      end
    elseif event == "Mob_2 Timer 2" then
      auto_chat(npc, text(9, player))
      q:start_quest_timer("Mob_2 Timer 3", 10000, npc, player)
    elseif event == "Mob_2 Timer 3" then
      if st:get_int("Quest0") == 0 then
        q:start_quest_timer("Mob_2 Timer 3", 13000, npc, player)
        if rnd.bool() then
          auto_chat(npc, text(10, player))
        else
          auto_chat(npc, text(11, player))
        end
      end
    elseif event == "Mob_2 has despawned" then
      st:set("Quest1", tostring(st:get_int("Quest1") + 1))
      local quest0 = st:get_int("Quest0")
      if quest0 == 1 or quest0 == 2 or st:get_int("Quest1") > 3 then
        st:set("Quest0", "0")
        -- Mobius: "This if will never be true": Quest0 was just set to 0.
        if st:get_int("Quest0") == 1 then
          auto_chat(npc, text(12, player))
        else
          auto_chat(npc, text(13, player))
        end
        delete_spawn(npc)
      else
        q:start_quest_timer("Mob_2 has despawned", 1000, npc, player)
      end
    end
    return nil
  end)

  q:on_attack(function(npc, player, damage, is_summon, skill)
    local st2 = find_right_state(npc)
    if st2 == nil then
      return
    end
    local cond = st2:get_cond()
    local st = q:get_quest_state(player, false)
    local npc_id = npc:id()
    if npc_id == MOB[3] and st == st2 and cond == 17 then
      local quest0 = st:get_int("Quest0") + 1
      if quest0 == 1 then
        auto_chat(npc, text(17, player))
      end
      if quest0 > 15 then
        quest0 = 1
        auto_chat(npc, text(18, player))
        q:cancel_quest_timer("Mob_3 has despawned", npc, st2:get_player())
        st:set("Tab", "1")
        delete_spawn(npc)
      end
      st:set("Quest0", tostring(quest0))
    elseif npc_id == MOB[2] and cond == 15 and (st ~= st2 or player:is_in_party()) then
      auto_chat(npc, text(6, player))
      q:cancel_quest_timer("Archon Hellisha has despawned", npc, st2:get_player())
      st2:set("spawned", "0")
      delete_spawn(npc)
    end
  end)

  q:on_first_talk(function(npc, player)
    local htmltext = ""
    local st = q:get_quest_state(player, false)
    if st ~= nil and npc:id() == NPC[5] then
      local cond = st:get_cond()
      if cond == 17 then
        local st2 = find_right_state(npc)
        if st2 ~= nil then
          -- Mobius sets the player's last quest NPC here; the game does
          -- that on every click of a quest's talk NPC.
          local tab = st:get_int("Tab")
          local quest0 = st:get_int("Quest0")
          if st == st2 then
            if tab == 1 then
              if quest0 == 0 then
                htmltext = "4-04.htm"
              elseif quest0 == 1 then
                htmltext = "4-06.htm"
              end
            elseif quest0 == 0 then
              htmltext = "4-01.htm"
            elseif quest0 == 1 then
              htmltext = "4-03.htm"
            end
          elseif tab == 1 then
            if quest0 == 0 then
              htmltext = "4-05.htm"
            elseif quest0 == 1 then
              htmltext = "4-07.htm"
            end
          elseif quest0 == 0 then
            htmltext = "4-02.htm"
          end
        end
      elseif cond == 18 then
        htmltext = "4-08.htm"
      end
    end
    if htmltext == "" then
      return nil -- npc.showChatWindow(player)
    end
    return htmltext
  end)

  q:on_kill(function(npc, player, is_summon)
    local npc_id = npc:id()
    local st = q:get_quest_state(player, false)

    if npc_id >= ARCHON_MINION_FIRST and npc_id <= ARCHON_MINION_LAST then
      if player:party() ~= nil then
        local party_quest_members = {}
        for _, player1 in ipairs(player:party_members()) do
          local st1 = find_quest(player1)
          if st1 ~= nil and st1:is_cond(15) and player1:is_inside_radius_2d(player, Config.ALT_PARTY_RANGE) then
            party_quest_members[#party_quest_members + 1] = st1
          end
        end
        if #party_quest_members > 0 then
          give_halisha_mark(party_quest_members[rnd(#party_quest_members) + 1])
        end
      else
        local st1 = find_quest(player)
        if st1 ~= nil and st1:is_cond(15) then
          give_halisha_mark(st1)
        end
      end
      return
    end

    if J.contains(ARCHON_HALISHA_NORM, npc_id) then
      local st1 = find_quest(player)
      if st1 ~= nil and st1:is_cond(15) then
        -- Mobius: "This is just a guess....not really sure what it actually
        -- says, if anything".
        auto_chat(npc, text(5, st1:get_player()))
        q:give_items(player, ITEMS[9], 1)
        q:take_items(player, ITEMS[4], -1)
        st1:set_cond(16, true)
      end
      return
    end

    if npc_id >= GUARDIAN_ANGEL_FIRST and npc_id <= GUARDIAN_ANGEL_LAST then
      local st1 = find_quest(player)
      if st1 ~= nil and st1:is_cond(6) then
        local kills = st1:get_int("kills")
        if kills < 9 then
          st1:set("kills", tostring(kills + 1))
        else
          q:give_items(player, ITEMS[6], 1)
          st:set_cond(7, true)
        end
      end
      return
    end

    if st ~= nil and npc_id ~= MOB[3] then
      local st2 = find_right_state(npc)
      if st2 ~= nil then
        local cond = st:get_cond()
        if npc_id == MOB[1] and cond == 8 then
          if not player:is_in_party() and st == st2 then
            auto_chat(npc, text(13, player))
            q:give_items(player, ITEMS[7], 1)
            st:set_cond(9, true)
          end
          q:cancel_quest_timer("Mob_1 has despawned", npc, st2:get_player())
          st2:set("spawned", "0")
          delete_spawn(npc)
        elseif npc_id == MOB[2] and cond == 15 then
          if not player:is_in_party() then
            if st == st2 then
              auto_chat(npc, text(5, player))
              q:give_items(player, ITEMS[9], 1)
              q:take_items(player, ITEMS[4], -1)
              st:set_cond(16, true)
            else
              auto_chat(npc, text(6, player))
            end
          end
          q:cancel_quest_timer("Archon Hellisha has despawned", npc, st2:get_player())
          st2:set("spawned", "0")
          delete_spawn(npc)
        end
      end
    elseif npc_id == MOB[1] then
      st = find_right_state(npc)
      if st ~= nil then
        q:cancel_quest_timer("Mob_1 has despawned", npc, st:get_player())
        st:set("spawned", "0")
        delete_spawn(npc)
      end
    elseif npc_id == MOB[2] then
      st = find_right_state(npc)
      if st ~= nil then
        q:cancel_quest_timer("Archon Hellisha has despawned", npc, st:get_player())
        st:set("spawned", "0")
        delete_spawn(npc)
      end
    elseif npc_id == MOB[3] then
      st = find_right_state(npc)
      if st ~= nil then
        st:set_cond(18, true)
        auto_chat(npc, text(14, player))
        st:set("Quest0", "0")
        -- As Mobius: the timer named here belongs to the other spawn
        -- (NPC 5), so this cancels nothing.
        q:cancel_quest_timer("Mob_2 has despawned", npc, player)
        delete_spawn(npc)
      end
    end
  end)

  q:on_skill_see(function(npc, player, skill, targets, is_summon)
    local owner = spawn_list[npc:object_id()]
    if owner == nil or owner == player:object_id() then
      return
    end
    local quest_player = world.find_object(owner)
    if quest_player == nil then
      return
    end
    for _, obj in ipairs(targets or {}) do
      if obj == quest_player or obj == npc then
        local st2 = find_right_state(npc)
        if st2 == nil then
          return
        end
        auto_chat(npc, text(6, player))
        q:cancel_quest_timer("Archon Hellisha has despawned", npc, st2:get_player())
        st2:set("spawned", "0")
        delete_spawn(npc)
      end
    end
  end)

  q:on_talk(function(npc, player)
    local st = q:get_quest_state(player, true)
    local htmltext = q:get_no_quest_msg(player)
    local npc_id = npc:id()
    if npc_id == NPC[1] and st:is_completed() then
      return q:get_already_completed_msg(player)
    elseif player:class_id() ~= get_prev_class(player) then
      return htmltext
    end
    local cond = st:get_cond()
    if cond == 0 then
      if npc_id == NPC[1] then
        htmltext = "0-01.htm"
      end
    elseif cond == 1 then
      if npc_id == NPC[1] then
        htmltext = "0-04.htm"
      elseif npc_id == NPC[3] then
        htmltext = "2-01.htm"
      end
    elseif cond == 2 then
      if npc_id == NPC[3] then
        htmltext = "2-02.htm"
      elseif npc_id == NPC[2] then
        htmltext = "1-01.htm"
      end
    elseif cond == 3 then
      if npc_id == NPC[2] and q:has_quest_items(player, ITEMS[1]) then
        if ITEMS[12] == 0 or q:has_quest_items(player, ITEMS[12]) then
          htmltext = "1-03.htm"
        else
          htmltext = "1-02.htm"
        end
      end
    elseif cond == 4 then
      if npc_id == NPC[2] then
        htmltext = "1-04.htm"
      elseif npc_id == NPC[3] then
        htmltext = "2-03.htm"
      end
    elseif cond == 5 then
      if npc_id == NPC[3] then
        htmltext = "2-04.htm"
      elseif npc_id == NPC[6] then
        htmltext = "5-01.htm"
      end
    elseif cond == 6 then
      if npc_id == NPC[6] then
        htmltext = "5-03.htm"
      elseif npc_id == NPC[7] then
        htmltext = "6-01.htm"
      end
    elseif cond == 7 then
      if npc_id == NPC[7] then
        htmltext = "6-02.htm"
      end
    elseif cond == 8 then
      if npc_id == NPC[7] then
        htmltext = "6-04.htm"
      elseif npc_id == NPC[8] then
        htmltext = "7-01.htm"
      end
    elseif cond == 9 then
      if npc_id == NPC[8] then
        htmltext = "7-05.htm"
      end
    elseif cond == 10 then
      if npc_id == NPC[8] then
        htmltext = "7-07.htm"
      elseif npc_id == NPC[4] then
        htmltext = "3-01.htm"
      end
    elseif cond == 11 or cond == 12 then
      if npc_id == NPC[4] then
        if q:has_quest_items(player, ITEMS[3]) then
          htmltext = "3-05.htm"
        else
          htmltext = "3-04.htm"
        end
      end
    elseif cond == 13 then
      if npc_id == NPC[4] then
        htmltext = "3-06.htm"
      elseif npc_id == NPC[9] then
        htmltext = "8-01.htm"
      end
    elseif cond == 14 then
      if npc_id == NPC[9] then
        htmltext = "8-03.htm"
      elseif npc_id == NPC[12] then
        htmltext = "11-01.htm"
      end
    elseif cond == 15 then
      if npc_id == NPC[12] then
        htmltext = "11-02.htm"
      elseif npc_id == NPC[10] then
        htmltext = "9-01.htm"
      end
    elseif cond == 16 then
      if npc_id == NPC[10] then
        htmltext = "9-02.htm"
      end
    elseif cond == 17 then
      if npc_id == NPC[10] then
        htmltext = "9-04.htm"
      elseif npc_id == NPC[11] then
        htmltext = "10-01.htm"
      end
    elseif cond == 18 then
      if npc_id == NPC[11] then
        htmltext = "10-05.htm"
      end
    elseif cond == 19 then
      if npc_id == NPC[11] then
        htmltext = "10-07.htm"
      elseif npc_id == NPC[1] then
        htmltext = "0-06.htm"
      end
    elseif cond == 20 then
      if npc_id == NPC[1] then
        if player:level() >= 76 then
          htmltext = "0-09.htm"
          st:exit_quest(false)
          reward(player)
          change_class(npc, player)
        else
          htmltext = "0-010.htm"
        end
      end
    end
    return htmltext
  end)

  return q
end

return saga

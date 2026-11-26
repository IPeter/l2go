-- The Four Sepulchers, ported from Mobius
-- ai/areas/ImperialTomb/FourSepulchers/FourSepulchers.
--
-- A party of four or more on quest 620 (Four Goblets) enters a sepulcher
-- through its manager between xx:55 and the full hour; at the full hour a
-- mysterious chest appears in the first hall. Each hall is a wave of
-- monsters (the room spawns of lib/four_sepulchers.lua, Mobius's
-- FourSepulchers.xml); a chapel key from the wave opens the gate to the
-- next. Hall 3 has a victim to protect, hall 4 the charm devices that
-- switch off the hall's effect zones, hall 5 petrified statues, and the
-- last hall the sepulcher's boss, whose death gives the party its goblet.
-- After 50 minutes everyone is sent out. The managers call at xx:55.
--
-- The progress of each sepulcher lives in this script, as in Mobius: a
-- reload starts every sepulcher from the first hall again.
--
-- Differences from Mobius:
-- * Entering takes the sepulcher from its manager. Mobius reads it from
--   the zone the party leader stands in, which is none at the manager
--   (outside the halls), so it never reset the progress, closed the doors
--   or switched the effect zones off for the next party.
-- * The entry teleport adds the 80-unit random offset here, without
--   Mobius's re-rolls against geodata.
-- * Wall-clock minutes are UTC minutes, which match the server's local
--   ones in every time zone with whole-hour offsets.

local SPAWNS = require("lib.four_sepulchers")

local s = script.new("FourSepulchers")

-- NPCs
local CONQUEROR_MANAGER = 31921
local EMPEROR_MANAGER = 31922
local GREAT_SAGES_MANAGER = 31923
local JUDGE_MANAGER = 31924
local MYSTERIOUS_CHEST = 31468
local KEY_CHEST = 31467
local ROOM_3_VICTIM = 18150
local ROOM_3_CHEST_REWARDER = 18158
local ROOM_4_CHARM_1 = 18196
local ROOM_4_CHARM_2 = 18197
local ROOM_4_CHARM_3 = 18198
local ROOM_4_CHARM_4 = 18199
local ROOM_5_STATUE_GUARD = 18232
local ROOM_6_REWARD_CHEST = 18256
local CONQUEROR_BOSS = 25346
local EMPEROR_BOSS = 25342
local GREAT_SAGES_BOSS = 25339
local JUDGE_BOSS = 25349
local TELEPORTER = 31452
local FIRST_TALK_NPCS = {
  TELEPORTER,
  31453, 31454, 31919, 31920, 31925, 31926, 31927, 31928,
  31929, 31930, 31931, 31932, 31933, 31934, 31935, 31936,
  31937, 31938, 31939, 31940, 31941, 31942, 31943, 31944,
}
local CHEST_REWARD_MONSTERS = {
  18120, -- room 1
  ROOM_3_CHEST_REWARDER,
  18177, -- room 4
  18212, -- room 5 - wave 2
}
local MANAGERS = { CONQUEROR_MANAGER, EMPEROR_MANAGER, GREAT_SAGES_MANAGER, JUDGE_MANAGER }
local BOSSES = { [CONQUEROR_BOSS] = true, [EMPEROR_BOSS] = true, [GREAT_SAGES_BOSS] = true, [JUDGE_BOSS] = true }

-- Items
local ENTRANCE_PASS = 7075
local USED_PASS = 7261
local CHAPEL_KEY = 7260
local ANTIQUE_BROOCH = 7262
local GOBLETS = 7255 -- + the sepulcher: 7256 to 7259

-- Locations
local START_HALL_SPAWNS = {
  [CONQUEROR_MANAGER] = { x = 181632, y = -85587, z = -7218 },
  [EMPEROR_MANAGER] = { x = 179963, y = -88978, z = -7218 },
  [GREAT_SAGES_MANAGER] = { x = 173217, y = -86132, z = -7218 },
  [JUDGE_MANAGER] = { x = 175608, y = -82296, z = -7218 },
}

-- Zones, by sepulcher (1 to 4) and by manager.
local CONQUEROR_ZONE = 200221
local EMPEROR_ZONE = 200222
local GREAT_SAGES_ZONE = 200224
local JUDGE_ZONE = 200223
local SEPULCHER_ZONES = { CONQUEROR_ZONE, EMPEROR_ZONE, GREAT_SAGES_ZONE, JUDGE_ZONE }
local MANAGER_ZONES = {
  [CONQUEROR_MANAGER] = CONQUEROR_ZONE,
  [EMPEROR_MANAGER] = EMPEROR_ZONE,
  [GREAT_SAGES_MANAGER] = GREAT_SAGES_ZONE,
  [JUDGE_MANAGER] = JUDGE_ZONE,
}

-- { sepulcher, room, x, y, z, heading }
local CHEST_SPAWN_LOCATIONS = {
  { 1, 1, 182074, -85579, -7216, 32768 },
  { 1, 2, 183868, -85577, -7216, 32768 },
  { 1, 3, 185681, -85573, -7216, 32768 },
  { 1, 4, 187498, -85566, -7216, 32768 },
  { 1, 5, 189306, -85571, -7216, 32768 },
  { 2, 1, 180375, -88968, -7216, 32768 },
  { 2, 2, 182151, -88962, -7216, 32768 },
  { 2, 3, 183960, -88964, -7216, 32768 },
  { 2, 4, 185792, -88966, -7216, 32768 },
  { 2, 5, 187625, -88953, -7216, 32768 },
  { 3, 1, 173218, -85703, -7216, 49152 },
  { 3, 2, 173206, -83929, -7216, 49152 },
  { 3, 3, 173208, -82085, -7216, 49152 },
  { 3, 4, 173191, -80290, -7216, 49152 },
  { 3, 5, 173198, -78465, -7216, 49152 },
  { 4, 1, 175601, -81905, -7216, 49152 },
  { 4, 2, 175619, -80094, -7216, 49152 },
  { 4, 3, 175608, -78268, -7216, 49152 },
  { 4, 4, 175588, -76472, -7216, 49152 },
  { 4, 5, 175594, -74655, -7216, 49152 },
}

-- { sepulcher, wave, door }
local DOORS = {
  { 1, 2, 25150012 }, { 1, 3, 25150013 }, { 1, 4, 25150014 }, { 1, 5, 25150015 }, { 1, 7, 25150016 },
  { 2, 2, 25150002 }, { 2, 3, 25150003 }, { 2, 4, 25150004 }, { 2, 5, 25150005 }, { 2, 7, 25150006 },
  { 3, 2, 25150032 }, { 3, 3, 25150033 }, { 3, 4, 25150034 }, { 3, 5, 25150035 }, { 3, 7, 25150036 },
  { 4, 2, 25150022 }, { 4, 3, 25150023 }, { 4, 4, 25150024 }, { 4, 5, 25150025 }, { 4, 7, 25150026 },
}

-- Skills
local PETRIFY = 4616
local CHARM_SKILLS = {
  [ROOM_4_CHARM_1] = 4146,
  [ROOM_4_CHARM_2] = 4145,
  [ROOM_4_CHARM_3] = 4148,
  [ROOM_4_CHARM_4] = 4624,
}

-- Misc
local CHARM_MSG = {
  [ROOM_4_CHARM_1] = "The P. Atk. reduction device has now been destroyed.",
  [ROOM_4_CHARM_2] = "The Defense reduction device has been destroyed.",
  [ROOM_4_CHARM_3] = "The poison device has now been destroyed.",
  [ROOM_4_CHARM_4] = "The poison device has now been destroyed.", -- Mobius TODO: the HP regeneration reduction device
}
local VICTIM_MSG = {
  "Help me!",
  "Don't miss!",
  "Keep pushing!",
}
local MANAGER_MESSAGES = {
  "You may now enter the Sepulcher.",
  "If you place your hand on the stone statue in front of each sepulcher, you will be able to enter.",
}
local FOUR_GOBLETS = "Q00620_FourGoblets"
local PARTY_MEMBER_COUNT = 4
local ENTRY_TIME = 55 -- Minute for entry.
local EVENT_TIME = 50 -- Minutes for the event.

-- The hall each sepulcher is at (0 is a player outside them, which Mobius
-- also keeps), and the monsters of the waves that must die before the next
-- step: { id = object ID, loc = where it was last seen }.
local stored_progress = { [0] = 1, 1, 1, 1, 1 }
local stored_monster_spawns = { [0] = {}, {}, {}, {}, {} }

s:add_first_talk_id(CONQUEROR_MANAGER, EMPEROR_MANAGER, GREAT_SAGES_MANAGER, JUDGE_MANAGER, MYSTERIOUS_CHEST, KEY_CHEST)
s:add_talk_id(CONQUEROR_MANAGER, EMPEROR_MANAGER, GREAT_SAGES_MANAGER, JUDGE_MANAGER, MYSTERIOUS_CHEST, KEY_CHEST)
s:add_first_talk_id(FIRST_TALK_NPCS)
s:add_talk_id(FIRST_TALK_NPCS)
s:add_kill_id(CHEST_REWARD_MONSTERS)
s:add_kill_id(ROOM_3_VICTIM, ROOM_4_CHARM_1, ROOM_4_CHARM_2, ROOM_4_CHARM_3, ROOM_4_CHARM_4, ROOM_6_REWARD_CHEST,
  CONQUEROR_BOSS, EMPEROR_BOSS, GREAT_SAGES_BOSS, JUDGE_BOSS)
s:add_spawn_id(ROOM_3_VICTIM, ROOM_4_CHARM_1, ROOM_4_CHARM_2, ROOM_4_CHARM_3, ROOM_4_CHARM_4, ROOM_5_STATUE_GUARD,
  ROOM_6_REWARD_CHEST, CONQUEROR_BOSS, EMPEROR_BOSS, GREAT_SAGES_BOSS, JUDGE_BOSS)

-- The wall-clock minute of the hour.
local function current_minute()
  return math.floor(time.now_ms() / 60000) % 60
end

-- How long until the next xx:55 (this hour's, or the next one's from
-- xx:55 on).
local function ms_until_entry_time()
  local now = time.now_ms()
  local target = now - now % 3600000 + ENTRY_TIME * 60000
  if current_minute() >= ENTRY_TIME then
    target = target + 3600000
  end
  return target - now
end

local function location_of(obj)
  return { x = obj:x(), y = obj:y(), z = obj:z(), heading = obj:heading() }
end

-- The sepulcher the player is in, 0 for none.
local function get_sepulcher_id(player)
  for i, zone in ipairs(SEPULCHER_ZONES) do
    if player:is_inside_zone(zone) then
      return i
    end
  end
  return 0
end

-- The level of a skill in an EffectZone's skill list ("4146-3;"), 0 when
-- it has none (Mobius EffectZone.getSkillLevel).
local function zone_skill_level(zone_id, skill_id)
  local list = zones.param(zone_id, "skillIdLvl") or ""
  for id, level in list:gmatch("(%d+)-(%d+)") do
    if tonumber(id) == skill_id then
      return tonumber(level)
    end
  end
  return 0
end

local function set_effect_zones_at(pos, enabled)
  for _, zone in ipairs(zones.at(pos)) do
    if zone:type() == "EffectZone" then
      zones.set_enabled(zone:id(), enabled)
    end
  end
end

local function spawn_next_wave(player)
  local sepulcher_id = get_sepulcher_id(player)
  if sepulcher_id == 0 then
    return -- Mobius fails here: there is no spawn list for 0.
  end
  local current_wave = stored_progress[sepulcher_id]
  local stored = stored_monster_spawns[sepulcher_id]
  for _, row in ipairs(SPAWNS) do
    if row[1] == sepulcher_id and row[2] == current_wave then
      local loc = { x = row[4], y = row[5], z = row[6], heading = row[7] }
      local monster = s:add_spawn(row[3], loc, false, 0)
      if monster ~= nil then
        stored[#stored + 1] = { id = monster:object_id(), loc = loc }
      end
    end
  end
  if current_wave == 4 then
    set_effect_zones_at(player, true)
  end
  if current_wave == 2 or current_wave == 5 then
    s:start_quest_timer("WAVE_DEFEATED_CHECK", 5000, nil, player, false)
  else
    stored_monster_spawns[sepulcher_id] = {} -- no need check for these waves
  end
end

local function spawn_mysterious_chest(player)
  local sepulcher_id = get_sepulcher_id(player)
  local current_wave = stored_progress[sepulcher_id]
  for _, row in ipairs(CHEST_SPAWN_LOCATIONS) do
    if row[1] == sepulcher_id and row[2] == current_wave then
      s:add_spawn(MYSTERIOUS_CHEST, { x = row[3], y = row[4], z = row[5], heading = row[6] }, false, 0)
      break
    end
  end
end

local function spawn_key_chest(player, loc)
  s:add_spawn(KEY_CHEST, loc or player)
end

-- try_enter is Mobius tryEnter; it returns the manager's page.
local function try_enter(npc, player)
  local npc_id = npc:id()
  local party = player:party()
  if party == nil then
    return npc_id .. "-SP.html"
  end
  if party.leader:object_id() ~= player:object_id() then
    return npc_id .. "-NL.html"
  end
  local zone = MANAGER_ZONES[npc_id]
  if #zones.players_inside(zone) > 0 then
    return npc_id .. "-FULL.html"
  end
  if not player:is_in_party() or #party.members < PARTY_MEMBER_COUNT then
    return npc_id .. "-SP.html"
  end

  -- Get current time in minutes for entering.
  local current_minute_ = current_minute()
  if current_minute_ >= 0 and current_minute_ < ENTRY_TIME then
    return npc_id .. "-NE.html"
  end

  -- Set the timer to spawn the chest when event start.
  local time_until_next_hour = (60 - current_minute_) * 60 * 1000
  s:start_quest_timer("SPAWN_MYSTERIOUS_CHEST", time_until_next_hour, npc, player, false)

  for _, mem in ipairs(party.members) do
    local qs = mem:get_quest_state(FOUR_GOBLETS)
    if qs == nil or (not qs:is_started() and not qs:is_completed()) then
      return npc_id .. "-NS.html", { member = mem:name() }
    end
    if not s:has_quest_items(mem, ENTRANCE_PASS) then
      return npc_id .. "-SE.html", { member = mem:name() }
    end
    if player:weight_penalty() >= 3 then
      mem:send_system_message(SystemMessageId.PROGRESS_IN_A_QUEST_IS_POSSIBLE_ONLY_WHEN_YOUR_INVENTORY_S_WEIGHT_AND_VOLUME_ARE_LESS_THAN_80_PERCENT_OF_CAPACITY)
      return ""
    end
  end

  local sepulcher_id = 0
  for i, z in ipairs(SEPULCHER_ZONES) do
    if z == zone then
      sepulcher_id = i
    end
  end

  -- Delete any existing spawns.
  for _, other in ipairs(zones.npcs_inside(zone)) do
    local id = other:id()
    if other:is_type("Monster") or other:is_type("RaidBoss") or id == MYSTERIOUS_CHEST or id == KEY_CHEST or id == TELEPORTER then
      other:delete()
    end
  end

  -- Disable EffectZones.
  for _, row in ipairs(CHEST_SPAWN_LOCATIONS) do
    if row[1] == sepulcher_id and row[2] == 4 then
      set_effect_zones_at({ x = row[3], y = row[4], z = row[5] }, false)
      break
    end
  end

  -- Close all doors.
  for _, door in ipairs(DOORS) do
    if door[1] == sepulcher_id then
      doors.close(door[3])
    end
  end

  -- Teleport players inside.
  local members = {}
  for _, mem in ipairs(party.members) do
    if util.in_range(700, player, mem, true) then
      members[#members + 1] = mem
    end
  end
  local hall = START_HALL_SPAWNS[npc_id]
  for _, mem in ipairs(members) do
    mem:teleport(hall.x + rnd(-80, 80), hall.y + rnd(-80, 80), hall.z)
    s:take_items(mem, ENTRANCE_PASS, 1)
    s:take_items(mem, CHAPEL_KEY, -1)
    if not s:has_quest_items(mem, ANTIQUE_BROOCH) then
      s:give_items(mem, USED_PASS, 1)
    end
  end

  -- Kick all players when time is over.
  s:start_quest_timer("OUST " .. zone .. " " .. time.now_ms(), EVENT_TIME * 60 * 1000, nil, nil)

  -- Init progress.
  stored_progress[sepulcher_id] = 1 -- Start from 1.
  stored_monster_spawns[sepulcher_id] = {}
  return npc_id .. "-OK.html"
end

s:on_start(function(ev)
  s:start_quest_timer("ANNOUNCE", ms_until_entry_time(), nil, nil)
end)

s:on_event(function(event, npc, player)
  local htmltext = event
  if event == "ENTER" then
    local qs = player:get_quest_state(FOUR_GOBLETS)
    if qs == nil then
      return s:get_no_quest_msg(player)
    end
    if qs:is_started() then
      return try_enter(npc, player)
    end
  elseif event == "OPEN_GATE" then
    local qs = player:get_quest_state(FOUR_GOBLETS)
    if qs == nil then
      return s:get_no_quest_msg(player)
    end
    if qs:is_started() and npc:is_script_value(0) then
      if not s:has_quest_items(player, CHAPEL_KEY) then
        return "Gatekeeper-no.html", { npcname = npc:name() }
      end
      npc:set_script_value(1)
      s:take_items(player, CHAPEL_KEY, -1)
      local sepulcher_id = get_sepulcher_id(player)
      local current_wave = stored_progress[sepulcher_id] + 1
      stored_progress[sepulcher_id] = current_wave -- update progress
      for _, door in ipairs(DOORS) do
        if door[1] == sepulcher_id and door[2] == current_wave then
          doors.open(door[3])
          s:start_quest_timer("CLOSE_DOOR " .. door[3], 15000, nil, nil)
          break
        end
      end
      if current_wave < 7 then
        spawn_mysterious_chest(player)
      else
        spawn_next_wave(player)
      end
      npc:broadcast_say(ChatType.NPC_GENERAL, "The monsters have spawned!")
      return nil
    end
    htmltext = s:get_no_quest_msg(player) -- Mobius TODO: a proper page?
  elseif event == "SPAWN_MYSTERIOUS_CHEST" then
    spawn_mysterious_chest(player)
    return nil
  elseif event == "VICTIM_FLEE" then
    if npc ~= nil and npc:valid() and not npc:is_dead() then
      local home = npc:spawn_location() or location_of(npc)
      local destination = world.valid_location(npc, { x = home.x + rnd(-400, 400), y = home.y + rnd(-400, 400), z = npc:z() })
      if npc:calculate_distance_2d(home) < 600 then
        npc:set_intention(Intention.MOVE_TO, destination)
      end
      npc:broadcast_say(ChatType.NPC_GENERAL, rnd.entry(VICTIM_MSG))
      s:start_quest_timer("VICTIM_FLEE", 3000, npc, nil, false)
    end
    return nil
  elseif event == "REMOVE_PETRIFY" then
    if npc ~= nil and npc:valid() then
      npc:stop_skill_effects(PETRIFY)
      npc:set_invul(false)
    end
    return nil
  elseif event == "WAVE_DEFEATED_CHECK" then
    local sepulcher_id = get_sepulcher_id(player)
    local current_wave = stored_progress[sepulcher_id]
    local last_location = nil
    local alive = {}
    for _, spawn in ipairs(stored_monster_spawns[sepulcher_id]) do
      local monster = world.find_object(spawn.id)
      if monster ~= nil then
        spawn.loc = location_of(monster)
      end
      last_location = spawn.loc
      if monster ~= nil and not monster:is_dead() then
        alive[#alive + 1] = spawn
      end
    end
    stored_monster_spawns[sepulcher_id] = alive
    if #alive == 0 then
      if current_wave == 2 then
        if rnd.bool() then
          spawn_next_wave(player)
        else
          spawn_key_chest(player, last_location)
        end
      elseif current_wave == 5 then
        stored_progress[sepulcher_id] = current_wave + 1
        spawn_next_wave(player)
      end
    elseif sepulcher_id > 0 then
      s:start_quest_timer("WAVE_DEFEATED_CHECK", 5000, nil, player, false)
    end
    return nil
  elseif event == "ANNOUNCE" then
    for _, manager_id in ipairs(MANAGERS) do
      local manager = world.npcs_by_id(manager_id)[1]
      if manager ~= nil then
        for _, message in ipairs(MANAGER_MESSAGES) do
          manager:broadcast_say(ChatType.NPC_SHOUT, message)
        end
      else
        log.warn("Four Sepulcher Manager is missing", "npc", manager_id)
      end
    end
    s:start_quest_timer("ANNOUNCE", ms_until_entry_time(), nil, nil)
    return nil
  else
    -- The timers Mobius runs on its thread pool.
    local door = event:match("^CLOSE_DOOR (%d+)$")
    if door ~= nil then
      doors.close(tonumber(door))
      return nil
    end
    local zone = event:match("^OUST (%d+) %d+$")
    if zone ~= nil then
      zones.oust_all(tonumber(zone))
      return nil
    end
  end
  return htmltext
end)

s:on_first_talk(function(npc, player)
  if npc:id() == MYSTERIOUS_CHEST then
    if npc:is_script_value(0) then
      npc:set_script_value(1)
      npc:delete()
      spawn_next_wave(player)
    end
    return ""
  end
  if npc:id() == KEY_CHEST then
    if npc:is_script_value(0) then
      npc:set_script_value(1)
      npc:delete()
      s:give_items(player, CHAPEL_KEY, 1)
    end
    return ""
  end
  return npc:id() .. ".html"
end)

s:on_spawn(function(npc)
  npc:set_random_walking(false)
  if npc:id() == ROOM_3_VICTIM then
    npc:disable_core_ai(true)
    npc:set_running()
    s:start_quest_timer("VICTIM_FLEE", 1000, npc, nil, false)
  end
  if npc:id() == ROOM_5_STATUE_GUARD then
    npc:set_target(npc)
    npc:cast(skills.get(PETRIFY, 1))
    npc:set_can_return_to_spawn_point(false)
    s:cancel_quest_timer("REMOVE_PETRIFY", npc, nil)
    s:start_quest_timer("REMOVE_PETRIFY", 5 * 60 * 1000, npc, nil, false) -- 5 minutes
  end
end)

s:on_kill(function(npc, killer, is_summon)
  local id = npc:id()
  if id == ROOM_3_VICTIM then
    s:add_spawn(ROOM_3_CHEST_REWARDER, npc)
  elseif CHARM_SKILLS[id] ~= nil then
    for _, zone in ipairs(zones.at(killer)) do
      if zone:type() == "EffectZone" and zone_skill_level(zone:id(), CHARM_SKILLS[id]) > 0 then
        zones.set_enabled(zone:id(), false)
        break
      end
    end
    npc:broadcast_say(ChatType.NPC_GENERAL, CHARM_MSG[id])
  elseif BOSSES[id] then
    local sepulcher_id = get_sepulcher_id(killer)
    stored_progress[sepulcher_id] = stored_progress[sepulcher_id] + 1
    local party = killer:party()
    if party ~= nil and sepulcher_id > 0 then
      for _, mem in ipairs(party.members) do
        if util.in_range(Config.ALT_PARTY_RANGE, killer, mem, true) then
          local qs = mem:get_quest_state(FOUR_GOBLETS)
          if qs ~= nil and qs:is_started() and not s:has_at_least_one_quest_item(mem, ANTIQUE_BROOCH)
            and not s:has_at_least_one_quest_item(mem, GOBLETS + sepulcher_id) then
            s:give_items(mem, GOBLETS + sepulcher_id, 1)
            s:play_sound(mem, QuestSound.ITEMSOUND_QUEST_MIDDLE)
          end
        end
      end
    end
    spawn_next_wave(killer)
    s:add_spawn(TELEPORTER, npc, true, 0, false)
  elseif id == ROOM_6_REWARD_CHEST then
    npc:drop_item(killer, 57, rnd(300, 1300))
  else
    spawn_key_chest(killer, location_of(npc))
  end
end)

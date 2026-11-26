-- The shared grand boss pattern of Mobius's grand boss scripts (Plan 13,
-- Phase 10.7): the saved status, the respawn timer from GrandBoss.ini
-- (Config.<BOSS>_SPAWN_INTERVAL and _RANDOM), the zone oust and a
-- restart-safe start.
--
--   local boss = require("lib.boss")
--   s:on_start(function(ev)
--     boss.start(s, QUEEN, {
--       unlock   = "queen_unlock",            -- the respawn timer's event
--       live     = function(npc) ... end,     -- after a reload: the boss lives
--       unlocked = function() ... end,        -- dead, its time has come
--       restore  = function(info) ... end,    -- alive in the store: spawn it
--     })
--   end)
--
-- Mobius's scripts spawn their boss in their constructor; here that is
-- on_start. A reload keeps the NPCs but not the timers, so start finds a
-- live boss first (spec.live re-arms its timers) instead of spawning a
-- second one, and re-arms the respawn timer of a dead boss from its saved
-- time.

local M = {}

-- The statuses most scripts use (Queen Ant, Core, Orfen, Zaken). Baium,
-- Antharas and Valakas number theirs differently and pass spec.dead.
M.ALIVE = 0
M.DEAD = 1

M.HOUR_MS = 3600000

-- respawn_ms is the respawn delay of Queen Ant, Core, Orfen and Zaken:
-- interval hours plus a random number of milliseconds within random hours
-- either way (Mobius interval * 3600000 + getRandom(-r, r)).
function M.respawn_ms(interval_hours, random_hours)
  local r = random_hours * M.HOUR_MS
  return interval_hours * M.HOUR_MS + rnd(-r, r)
end

-- live_boss returns the boss of the template in the world and alive: the
-- one the store knows, else one found by template (registered again), or
-- nil.
function M.live_boss(id)
  local npc = grandboss.boss(id)
  if npc ~= nil and not npc:is_dead() then
    return npc
  end
  for _, n in ipairs(world.npcs_by_id(id)) do
    if not n:is_dead() then
      grandboss.add_boss(n)
      return n
    end
  end
  return nil
end

-- start is the constructor part of a grand boss script (see the top).
-- spec.dead is the dead status (M.DEAD when left out). It returns what it
-- did: "live", "waiting", "unlocked" or "restored".
function M.start(s, id, spec)
  local live = M.live_boss(id)
  if live ~= nil then
    if spec.live ~= nil then
      spec.live(live)
    end
    return "live"
  end
  local info = grandboss.info(id) or { respawn_time = 0, x = 0, y = 0, z = 0, heading = 0, hp = 0, mp = 0 }
  if grandboss.status(id) == (spec.dead or M.DEAD) then
    local remaining = info.respawn_time - time.now_ms()
    if remaining > 0 then
      s:start_quest_timer(spec.unlock, remaining, nil, nil)
      return "waiting"
    end
    spec.unlocked()
    return "unlocked"
  end
  spec.restore(info)
  return "restored"
end

-- spawn_saved spawns the boss where the store saw it last, with its saved
-- HP and MP (Mobius addSpawn(id, loc_x, loc_y, loc_z, heading) and
-- setCurrentHpMp). loc replaces the saved place when given. Saved HP of 0
-- (a seed without HP) leaves the boss at full HP.
function M.spawn_saved(s, id, info, loc)
  loc = loc or { x = info.x, y = info.y, z = info.z }
  local npc = s:add_spawn(id, { x = loc.x, y = loc.y, z = loc.z, heading = info.heading }, false, 0)
  if npc ~= nil and info.hp > 0 then
    npc:set_hp_mp(info.hp, info.mp)
  end
  return npc
end

-- died saves a killed boss: the dead status at once, the respawn timer
-- (event unlock after delay ms) and the respawn time in the store (Mobius
-- setStatus, startQuestTimer, setStatSet). It returns the respawn time
-- (Unix ms).
function M.died(s, id, dead, delay, unlock, name)
  grandboss.set_status(id, dead)
  local at = time.now_ms() + delay
  log.info((name or "grand boss") .. " will respawn", "boss", id, "at_ms", at, "delay_ms", delay)
  s:start_quest_timer(unlock, delay, nil, nil)
  local info = grandboss.info(id) or { x = 0, y = 0, z = 0, heading = 0, hp = 0, mp = 0 }
  info.respawn_time = at
  grandboss.set_info(id, info)
  return at
end

-- sound plays a boss sound from npc to the players around it (Mobius
-- broadcastPacket(new PlaySound(1, sound, 1, npc, x, y, z))).
function M.sound(npc, sound)
  cinematics.sound(npc, sound, npc)
end

-- oust sends the players inside the zone out (Mobius oustAllPlayers).
function M.oust(zone)
  zones.oust_all(zone)
end

-- attack_desire makes npc go for target (Mobius Quest.addAttackDesire:
-- hate 999 by default, running, the ATTACK intention).
function M.attack_desire(npc, target, desire)
  npc:add_damage_hate(target, 0, desire or 999)
  npc:set_running()
  npc:attack(target)
end

-- cast_desire makes npc cast skill at target (Mobius
-- Quest.addSkillCastDesire: hate as desire, the target, the CAST
-- intention).
function M.cast_desire(npc, target, skill, desire)
  npc:add_damage_hate(target, 0, desire)
  npc:set_target(target)
  npc:set_intention_cast(skill, target)
end

-- same reports whether two players or NPCs are the same object (nil only
-- equals nil).
function M.same(a, b)
  if a == nil or b == nil then
    return a == b
  end
  return a:object_id() == b:object_id()
end

-- alive reports whether an NPC handle is still in the world and alive.
function M.alive(npc)
  return npc ~= nil and npc:valid() and not npc:is_dead()
end

return M

-- Mobius's LongTimeEvent (mechanics/script/LongTimeEvent.java, Plan 14
-- M11n): a seasonal event with a date range, an optional drop period, a
-- drop list, a spawn list, start and end messages and items destroyed at
-- the end. The data is the event's config.xml (data/events/, with the
-- config's events.dates on top), kept by the server with the drop list,
-- the login notice and the item cleanup (the long_events module); this
-- library runs the event's life.
--
--   local lte = require("lib.long_time_event")
--   local s = lte.new("L2Day")            -- script.new("L2Day") with the event
--   lte.on_event(s, function(event, npc, player) ... end)  -- the event's links
--   s:on_first_talk(...)                  -- and the rest, as in any script
--
-- The event script must leave s:on_event and s:on_start to the library.
--
-- At startup (on_start) an event within its dates starts; one that is still
-- to come starts on a timer; one that is over destroys its items (Mobius's
-- constructor). Starting marks it active, starts the drops if the drop
-- period runs, spawns its NPCs until the end, announces the enter message
-- and keeps it as a login notice, and sets the end timer. The end stops the
-- drops, destroys the items, announces the end message, drops the notice
-- and removes the NPCs.
--
-- Unlike Mobius, the drops follow the drop period while the event runs
-- (Mobius adds them only if the event starts within it, until the event
-- ends). A reload keeps the NPCs, drops and notice of a running event and
-- sets its timers again. The GM commands //event_start and //event_stop
-- run the START and STOP events (Mobius's AdminEvents only knows its custom
-- events): a GM start ignores the dates and runs the drops (it ends at the
-- end date if that is still to come).

local M = {}

-- The events the game and the timers send to the script.
M.START = "long_time_event_start" -- //event_start
M.STOP = "long_time_event_stop" -- //event_stop
local BEGIN = "long_time_event_begin" -- the start date came (Mobius ScheduleStart)
local FINISH = "long_time_event_end" -- the end date came (Mobius ScheduleEnd)
local DROPS_ON = "long_time_event_drops_on"
local DROPS_OFF = "long_time_event_drops_off"

-- handlers are the event scripts' own on_event handlers, by script name.
local handlers = {}

-- config returns the event's data, or nil after logging why.
local function config(s)
  local cfg, err = long_events.config(s:name())
  if cfg == nil then
    log.warn("seasonal event has no usable data", "event", s:name(), "err", err)
  end
  return cfg
end

-- within reports whether t is in [from, to] (Mobius isWithinRange).
local function within(t, from, to)
  return t >= from and t <= to
end

-- timer starts a timer of the script for the event name in ms (none if ms
-- is not positive).
local function timer(s, name, ms)
  if ms > 0 then
    s:start_quest_timer(name, ms, nil, nil)
  end
end

-- arm sets the timers of a running event from now: its end and its drop
-- period.
local function arm(s, cfg, now)
  timer(s, FINISH, cfg.end_ms - now)
  if cfg.drop_start_ms <= cfg.drop_end_ms then
    timer(s, DROPS_ON, cfg.drop_start_ms - now)
    if cfg.drop_end_ms < cfg.end_ms then
      timer(s, DROPS_OFF, cfg.drop_end_ms - now)
    end
  end
end

-- spawn puts the event's NPCs into the world until the end (Mobius
-- spawnNpcs: addSpawn with the time left as the despawn delay, none when
-- the end has passed). The server spawns them right after the handler, so
-- Christmas's 207 NPCs do not count against its time limit.
local function spawn(s, cfg, now)
  long_events.spawn(s:name(), math.max(cfg.end_ms - now, 0))
end

-- start is Mobius startEvent. A GM start (forced) runs the drops whatever
-- the drop period.
local function start(s, cfg, forced)
  local name = s:name()
  local now = time.now_ms()
  long_events.set_active(name, true)
  if forced or within(now, cfg.drop_start_ms, cfg.drop_end_ms) then
    long_events.set_drops(name, true)
  end
  spawn(s, cfg, now)
  if cfg.on_enter ~= "" then
    long_events.announce(cfg.on_enter)
    local until_ms = nil
    if cfg.end_ms > now then
      until_ms = cfg.end_ms
    end
    long_events.set_notice(name, cfg.on_enter, math.min(cfg.start_ms, now), until_ms)
  end
  if forced then
    timer(s, FINISH, cfg.end_ms - now)
  else
    arm(s, cfg, now)
  end
  log.info("seasonal event started", "event", name, "until_ms", cfg.end_ms)
end

-- stop is Mobius stopEvent. The NPCs leave too: at the end they would on
-- their despawn timers anyway, and a GM stop must not leave them.
local function stop(s, cfg)
  local name = s:name()
  long_events.set_active(name, false)
  long_events.set_drops(name, false)
  long_events.destroy_items(name)
  if cfg.on_end ~= "" then
    long_events.announce(cfg.on_end)
  end
  long_events.set_notice(name, nil)
  long_events.despawn(name)
  for _, t in ipairs({ BEGIN, FINISH, DROPS_ON, DROPS_OFF }) do
    s:cancel_quest_timer(t, nil, nil)
  end
  log.info("seasonal event ended", "event", name)
end

-- init is Mobius's constructor, at startup and after a reload.
local function init(s, reload)
  local cfg = config(s)
  if cfg == nil then
    return
  end
  local name = s:name()
  local now = time.now_ms()
  if long_events.is_active(name) then
    -- A reload: the NPCs, drops and notice are still there.
    arm(s, cfg, now)
    return
  end
  if within(now, cfg.start_ms, cfg.end_ms) then
    start(s, cfg, false)
  elseif cfg.start_ms > now then
    timer(s, BEGIN, cfg.start_ms - now)
    log.info("seasonal event will start", "event", name, "at_ms", cfg.start_ms)
  elseif not reload then
    long_events.destroy_items(name) -- the items exist only during the event
    log.debug("seasonal event has passed", "event", name)
  end
end

-- lifecycle handles the library's events; it returns true for them.
local function lifecycle(s, event)
  local name = s:name()
  if event == BEGIN then
    local cfg = config(s)
    if cfg ~= nil and not long_events.is_active(name) then
      start(s, cfg, false)
    end
  elseif event == FINISH then
    local cfg = config(s)
    if cfg ~= nil and long_events.is_active(name) then
      stop(s, cfg)
    end
  elseif event == DROPS_ON then
    if long_events.is_active(name) then
      long_events.set_drops(name, true)
    end
  elseif event == DROPS_OFF then
    long_events.set_drops(name, false)
  elseif event == M.START then
    local cfg = config(s)
    if cfg ~= nil and not long_events.is_active(name) then
      s:cancel_quest_timer(BEGIN, nil, nil)
      start(s, cfg, true)
    end
  elseif event == M.STOP then
    local cfg = config(s)
    if cfg ~= nil and long_events.is_active(name) then
      stop(s, cfg)
    end
  else
    return false
  end
  return true
end

-- new creates the event's script (script.new(name)) and takes its on_event
-- and on_start.
function M.new(name)
  local s = script.new(name)
  s:on_event(function(event, npc, player)
    if lifecycle(s, event) then
      return ""
    end
    local fn = handlers[name]
    if fn ~= nil then
      return fn(event, npc, player)
    end
    return nil
  end)
  s:on_start(function(ev)
    init(s, ev.reload)
  end)
  return s
end

-- on_event sets the event script's own on_event handler.
function M.on_event(s, fn)
  handlers[s:name()] = fn
end

-- is_active reports whether the event runs (Mobius isEventPeriod).
function M.is_active(s)
  return long_events.is_active(s:name())
end

return M

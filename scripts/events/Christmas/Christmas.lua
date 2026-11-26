-- Mobius events/Christmas (a LongTimeEvent, lib/long_time_event.lua): monsters
-- drop Christmas tree parts and carol Echo Crystals, Santa's Trainees stand
-- in every town with their gift multisell, Christmas trees decorate the
-- towns, and every 15 s a Santa buffs the players around it. Its dates,
-- drops and spawns are config.xml's (data/events/longtimeevents.json, the
-- config's events.dates on top); the pages are Mobius's
-- (data/html/scripts/events/Christmas/).

local lte = require("lib.long_time_event")

-- NPC
local SANTA = 31863

-- Skills: {id, level}
local FIGHTER_BUFFS = {
  { 4322, 1 }, -- Wind Walk
  { 4323, 1 }, -- Shield
  { 4396, 2 }, -- Magic Barrier
  { 4324, 1 }, -- Bless the Body
  { 4325, 1 }, -- Vampiric Rage
  { 4326, 1 }, -- Regeneration
  { 4327, 1 }, -- Haste
}
local MAGE_BUFFS = {
  { 4322, 1 }, -- Wind Walk
  { 4323, 1 }, -- Shield
  { 4396, 2 }, -- Magic Barrier
  { 4328, 1 }, -- Bless the Soul
  { 4329, 1 }, -- Acumen
  { 4330, 1 }, -- Concentration
  { 4331, 1 }, -- Empower
}

-- Misc
local ALLOW_SANTA_BUFFS = true
local SANTA_BUFF_RANGE = 500
local BLESSINGS_MS = 15000

local s = lte.new("Christmas")
s:add_start_npc(SANTA)
s:add_first_talk_id(SANTA)
s:add_talk_id(SANTA)
s:add_spawn_id(SANTA)

lte.on_event(s, function(event, npc, player)
  if event == "SantaBlessings" then
    if npc ~= nil and npc:valid() and npc:id() == SANTA then
      for _, nearby in ipairs(world.players_around(npc, SANTA_BUFF_RANGE)) do
        local buffed = false
        npc:set_target(nearby)
        local buffs = nearby:is_mage_class() and MAGE_BUFFS or FIGHTER_BUFFS
        for _, b in ipairs(buffs) do
          if not nearby:is_affected_by_skill(b[1]) then
            local skill = skills.get(b[1], b[2])
            if skill ~= nil then
              npc:cast(skill, nearby)
            end
            buffed = true
          end
        end
        if buffed then
          nearby:send_message("Santa Trainee has gifted you buffs!")
        end
      end
      s:start_quest_timer("SantaBlessings", BLESSINGS_MS, npc, nil)
    end
  end
  return nil
end)

s:on_first_talk(function(npc, player)
  return "31863.htm"
end)

s:on_spawn(function(npc)
  if ALLOW_SANTA_BUFFS then
    s:start_quest_timer("SantaBlessings", BLESSINGS_MS, npc, nil)
  end
end)

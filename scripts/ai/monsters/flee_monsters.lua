-- Monsters that run away when hit, ported from Mobius
-- ai/others/FleeMonsters (Mobius, NosBit).
--
-- The monster stops its own AI and runs 500 units straight away from the
-- attacker (or the attacker's summon), as far as geodata lets it.

local s = script.new("FleeMonsters")

local MOBS = {
  18150, -- Victim
  18151, -- Victim
  18152, -- Victim
  18153, -- Victim
  18154, -- Victim
  18155, -- Victim
  18156, -- Victim
  18157, -- Victim
  20002, -- Rabbit
  20432, -- Elpy
}

local FLEE_DISTANCE = 500

s:add_attack_id(MOBS)

-- trunc is Java's (int) cast: toward zero.
local function trunc(x)
  if x < 0 then
    return math.ceil(x)
  end
  return math.floor(x)
end

s:on_attack(function(npc, attacker, damage, is_summon)
  npc:disable_core_ai(true)
  npc:set_running()

  local from = attacker
  if is_summon then
    from = attacker:summon() or attacker
  end
  local radians = math.rad(util.calculate_angle_from(from, npc))
  local x = trunc(npc:x() + FLEE_DISTANCE * math.cos(radians))
  local y = trunc(npc:y() + FLEE_DISTANCE * math.sin(radians))
  local destination = world.valid_location(npc, { x = x, y = y, z = npc:z() })
  npc:move_to(destination)
end)

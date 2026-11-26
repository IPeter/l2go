-- The Monastery of Silence, ported from Mobius
-- ai/areas/MonasteryOfSilence.
--
-- The monks hate only players who carry a weapon. A guard (mobs1) that
-- sees an armed player, when it spawns or when the player comes into its
-- aggro range, shouts a warning and attacks, the casters after Silence
-- (4589). The others (mobs2) answer a hostile skill aimed at them.

local J = require("lib.java")

local s = script.new("MonasteryOfSilence")

local MOBS1 = { 22124, 22125, 22126, 22127, 22129 }
local MOBS2 = { 22134, 22135 }
local TEXT = {
  "You cannot carry a weapon without authorization!",
  "name, why would you choose the path of darkness?!",
  "name! How dare you defy the will of Einhasad!",
}
local SILENCE = 4589

local contains = J.contains

-- attack is the guard's charge at an armed player.
local function attack(npc, target)
  npc:set_running()
  npc:add_damage_hate(target, 0, 999)
  npc:attack(target)
end

-- warn makes a guard warn and attack an armed player; casters is the list
-- of the guards that cast Silence first.
local function warn(npc, target, casters)
  npc:set_target(target)
  npc:broadcast_say(ChatType.GENERAL, TEXT[1])
  if contains(casters, npc:id()) then
    npc:cast(skills.get(SILENCE, 8))
  else
    attack(npc, target)
  end
end

for _, mobs in ipairs({ MOBS1, MOBS2 }) do
  s:add_npc_hate_id(mobs)
  s:add_aggro_range_enter_id(mobs)
  s:add_skill_see_id(mobs)
  s:add_spell_finished_id(mobs)
  s:add_spawn_id(mobs)
end

s:on_npc_hate(function(npc, player, is_summon)
  return player:active_weapon() ~= nil
end)

s:on_aggro_range_enter(function(npc, player, is_summon)
  if contains(MOBS1, npc:id()) and not npc:is_in_combat() and npc:target() == nil
    and player:active_weapon() ~= nil then
    warn(npc, player, { 22124, 22126 })
  end
end)

s:on_skill_see(function(npc, caster, skill, targets, is_summon)
  if not contains(MOBS2, npc:id()) or not skill:has_effect_type("AGGRESSION") then
    return
  end
  for _, obj in ipairs(targets) do
    if obj == npc then
      npc:broadcast_say(ChatType.GENERAL, J.replace(TEXT[rnd(2) + 2], "name", caster:name()))
      npc:add_damage_hate(caster, 0, 999)
      npc:attack(caster)
      break
    end
  end
end)

s:on_spell_finished(function(npc, player, skill)
  if contains(MOBS1, npc:id()) and skill:id() == SILENCE then
    attack(npc, player)
  end
end)

-- in_aggro_range is Mobius checkIfInRange(getAggroRange(), npc, obj, true)
-- for a living obj.
local function in_aggro_range(npc, obj)
  return util.in_range(npc:aggro_range(), npc, obj, true) and not obj:is_dead()
end

s:on_spawn(function(npc)
  if not contains(MOBS1, npc:id()) then
    return
  end
  -- The players and pets within aggro range; a pet stands for its owner.
  local found = {}
  for _, p in ipairs(world.players_around(npc)) do
    if in_aggro_range(npc, p) then
      found[#found + 1] = p
    end
    local pet = p:summon()
    if pet ~= nil and pet:is_type("Pet") and in_aggro_range(npc, pet) then
      found[#found + 1] = p
    end
  end
  for _, target in ipairs(found) do
    if target:active_weapon() ~= nil and not npc:is_in_combat() and npc:target() == nil then
      warn(npc, target, { 22124, 22126, 22127 })
    end
  end
end)

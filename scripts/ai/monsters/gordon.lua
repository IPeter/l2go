-- Gordon, ported from Mobius ai/others/Gordon (TOFIZ, malyelfik).
--
-- The walking raid boss of Goddard hunts whoever he sees with a cursed
-- weapon, and never walks back to his spawn point.

local s = script.new("Gordon")

local GORDON = 29095

s:add_spawn_id(GORDON)
s:add_creature_see_id(GORDON)

s:on_creature_see(function(npc, creature)
  if creature:is_player() and creature:is_cursed_weapon_equipped() then
    -- Mobius addAttackDesire.
    npc:add_damage_hate(creature, 0, 999)
    npc:set_running()
    npc:attack(creature)
  end
end)

s:on_spawn(function(npc)
  npc:set_can_return_to_spawn_point(false)
end)

-- Monsters that change shape when hit, ported from Mobius
-- ai/others/PolymorphingOnAttack (Slyce).
--
-- Hit at or below its HP threshold, the monster may shout, vanish and come
-- back 20 higher as its next form, which runs at the attacker (or its
-- summon).

local s = script.new("PolymorphingOnAttack")

-- Per monster: the next form, the HP threshold in percent, the chance in
-- percent and the line it shouts (an index into MOB_TEXTS, 0 for none).
local MOB_SPAWNS = {
  [21258] = { 21259, 100, 100, 0 }, -- Fallen Orc Shaman -> Sharp Talon Tiger (always polymorphs)
  [21261] = { 21262, 100, 20, 1 }, -- Ol Mahum Transcender 1st stage
  [21262] = { 21263, 100, 10, 2 }, -- Ol Mahum Transcender 2nd stage
  [21263] = { 21264, 100, 5, 3 }, -- Ol Mahum Transcender 3rd stage
  [21265] = { 21271, 100, 33, 1 }, -- Cave Ant Larva -> Cave Ant
  [21266] = { 21269, 100, 100, 0 }, -- Cave Ant Larva -> Cave Ant (always polymorphs)
  [21267] = { 21270, 100, 100, 0 }, -- Cave Ant Larva -> Cave Ant Soldier (always polymorphs)
  [21271] = { 21272, 66, 10, 2 }, -- Cave Ant -> Cave Ant Soldier
  [21272] = { 21273, 33, 5, 3 }, -- Cave Ant Soldier -> Cave Noble Ant
  [21521] = { 21522, 100, 30, 0 }, -- Claws of Splendor
  [21524] = { 21525, 100, 30, 0 }, -- Blade of Splendor
  [21527] = { 21528, 100, 30, 0 }, -- Anger of Splendor
  [21531] = { 21658, 100, 30, 0 }, -- Punishment of Splendor
  [21533] = { 21534, 100, 30, 0 }, -- Alliance of Splendor
  [21537] = { 21538, 100, 30, 0 }, -- Fang of Splendor
  [21539] = { 21540, 100, 30, 0 }, -- Wailing of Splendor
}

local MOB_TEXTS = {
  {
    "Enough fooling around. Get ready to die!",
    "You idiot! I've just been toying with you!",
    "Now the fun starts!",
  },
  {
    "I must admit, no one makes my blood boil quite like you do!",
    "Now the battle begins!",
    "Witness my true power!",
  },
  {
    "Prepare to die!",
    "I'll double my strength!",
    "You have more skill than I thought!",
  },
}

for id in pairs(MOB_SPAWNS) do
  s:add_attack_id(id)
end

s:on_attack(function(npc, attacker, damage, is_summon)
  if not npc:valid() or npc:is_dead() then
    return
  end
  local change = MOB_SPAWNS[npc:id()]
  if change == nil or npc:hp() > npc:max_hp() * change[2] / 100 or rnd(100) >= change[3] then
    return
  end
  if change[4] > 0 then
    npc:broadcast_say(ChatType.NPC_GENERAL, rnd.entry(MOB_TEXTS[change[4]]))
  end

  -- The handle is gone once the NPC is deleted: take its place first.
  local pos = { x = npc:x(), y = npc:y(), z = npc:z() + 20, heading = npc:heading() }
  npc:delete()
  local next_form = s:add_spawn(change[1], pos, false, 0, true)
  if next_form == nil then
    return
  end
  local target = attacker
  if is_summon then
    target = attacker:summon() or attacker
  end
  next_form:set_running()
  next_form:add_damage_hate(target, 0, 500)
  next_form:attack(target)
end)

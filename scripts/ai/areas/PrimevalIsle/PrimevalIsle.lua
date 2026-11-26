-- The dinosaurs of Primeval Isle, ported from Mobius
-- ai/areas/PrimevalIsle (St3eT).
--
-- * Sprigants cast their poison every 15 s.
-- * An Ancient Egg that is hit may call the monsters within 500.
-- * A Tyrannosaurus says "?" at a player in its aggro range and stuns him
--   6 s later; it buffs itself (Berserk) as it weakens and uses its skills
--   more after each buff; it hunts the plant eaters it sees; ghost
--   Tyrannosauruses (CollectGhost) leave after a while out of combat.
-- * The other dinosaurs use their template's skills (PhysicalSpecial1/2,
--   SelfRangeBuff1) as they weaken; Deinonychus and Ornithomimus may run
--   from a player they see.

local J = require("lib.java")

local s = script.new("PrimevalIsle")

-- NPCs
local EGG = 18344 -- Ancient Egg
local ORNIT = 22202 -- Ornithomimus
local DEINO = 22205 -- Deinonychus
local SPRIGANT = {
  18345, -- Sprigant (Anesthesia)
  18346, -- Sprigant (Deadly Poison)
}
local MONSTERS = {
  22196, -- Velociraptor
  22198, -- Velociraptor
  22200, -- Ornithomimus
  22202, -- Ornithomimus
  22203, -- Deinonychus
  22205, -- Deinonychus
  22208, -- Pachycephalosaurus
  22210, -- Pachycephalosaurus
  22211, -- Wild Strider
  22213, -- Wild Strider
  22223, -- Velociraptor
  22224, -- Ornithomimus
  22225, -- Deinonychus
  22226, -- Pachycephalosaurus
  22227, -- Wild Strider
}
local TREX = {
  22215, -- Tyrannosaurus
  22216, -- Tyrannosaurus
  22217, -- Tyrannosaurus
}
-- The plant eaters a Tyrannosaurus hunts.
local VEGETABLE = {
  22200, -- Ornithomimus
  22201, -- Ornithomimus
  22202, -- Ornithomimus
  22203, -- Deinonychus
  22204, -- Deinonychus
  22205, -- Deinonychus
  22224, -- Ornithomimus
  22225, -- Deinonychus
}

-- Skills: {id, level}.
local ANESTHESIA = { 5085, 1 }
local DEADLY_POISON = { 5086, 1 }
local SELFBUFF1 = { 5087, 1 } -- Berserk
local SELFBUFF2 = { 5087, 2 } -- Berserk
local LONGRANGEDMAGIC1 = { 5120, 1 } -- Stun
local PHYSICALSPECIAL1 = { 5083, 4 } -- Stun
local PHYSICALSPECIAL2 = { 5081, 4 } -- Silence
local PHYSICALSPECIAL3 = { 5082, 4 } -- NPC Spinning, Slashing Trick

local USE_SKILL_MS = 15000
local GHOST_RECHECK_MS = 1800000

local function skill(holder)
  return skills.get(holder[1], holder[2])
end

local function hp_percent(npc)
  return npc:hp() / npc:max_hp() * 100
end

-- attack_desire is Mobius addAttackDesire: hate 999, run, attack.
local function attack_desire(npc, target)
  npc:add_damage_hate(target, 0, 999)
  npc:set_running()
  npc:attack(target)
end

-- cast_at casts sk at target (nil: at no one in particular), as Mobius's
-- setTarget then doCast.
local function cast_at(npc, target, sk)
  npc:set_target(target)
  npc:cast(sk)
end

s:add_spawn_id(TREX, SPRIGANT, MONSTERS)
s:add_aggro_range_enter_id(TREX)
s:add_spell_finished_id(TREX)
s:add_attack_id(EGG, TREX, MONSTERS)
s:add_creature_see_id(TREX, MONSTERS)

s:on_spell_finished(function(npc, player, sk)
  if not npc:is_in_combat() then
    return
  end
  -- Mobius also has a branch below 30 % for SELFBUFF2, which this test
  -- (below 60 %) always takes first, so it never runs.
  if hp_percent(npc) < 60 and sk:id() == SELFBUFF1[1] then
    npc:set_script_value(3)
    local target = npc:most_hated()
    if target ~= nil then
      npc:set_target(target)
      npc:set_running()
      npc:add_damage_hate(target, 0, 555)
      npc:attack(target)
    end
  end
end)

s:on_event(function(event, npc, player)
  if event == "USE_SKILL" then
    if npc ~= nil and not npc:is_dead() then
      npc:cast(skill(npc:id() == SPRIGANT[1] and ANESTHESIA or DEADLY_POISON))
      s:start_quest_timer("USE_SKILL", USE_SKILL_MS, npc, nil)
    end
  elseif event == "GHOST_DESPAWN" then
    if npc ~= nil and not npc:is_dead() then
      if not npc:is_in_combat() then
        npc:delete()
      else
        s:start_quest_timer("GHOST_DESPAWN", GHOST_RECHECK_MS, npc, nil)
      end
    end
  elseif event == "TREX_ATTACK" then
    if npc ~= nil and player ~= nil then
      npc:set_script_value(0)
      if player:is_inside_radius_3d(npc, 800) then
        cast_at(npc, player, skill(LONGRANGEDMAGIC1))
        attack_desire(npc, player)
      end
    end
  end
  return nil
end)

-- run_away sends the dinosaur 3,000 away from creature, as far as the
-- ground lets it.
local function run_away(npc, creature)
  npc:clear_aggro_list()
  npc:set_script_value(1)
  npc:set_running()
  -- Mobius calculateHeadingFrom and convertHeadingToDegree: the angle goes
  -- through a heading (an integer).
  local heading = math.floor(util.calculate_angle_from(creature, npc) * 182.044444444)
  local radian = math.rad(heading / 182.044444444)
  local here = { x = npc:x(), y = npc:y(), z = npc:z() }
  local to = { x = J.int(npc:x() + math.cos(radian) * 3000), y = J.int(npc:y() + math.sin(radian) * 3000), z = npc:z() }
  npc:move_to(world.valid_location(here, to))
end

s:on_creature_see(function(npc, creature)
  if J.contains(MONSTERS, npc:id()) then
    if not creature:is_player() then
      return
    end
    local ag_type = npc:param_int("ag_type", 0)
    local prob1 = npc:param_int("ProbPhysicalSpecial1", 0)
    local prob2 = npc:param_int("ProbPhysicalSpecial2", 0)
    local special1 = npc:param_skill("PhysicalSpecial1")
    local special2 = npc:param_skill("PhysicalSpecial2")
    local multiplier = npc:var_int("SKILL_MULTIPLER")
    if (rnd(100) < 30 and npc:id() == DEINO) or (npc:id() == ORNIT and npc:is_script_value(0)) then
      run_away(npc, creature)
    elseif ag_type == 1 then
      if rnd(100) <= prob1 * multiplier then
        if not npc:is_skill_disabled(special1) then
          cast_at(npc, creature, special1)
        end
      elseif rnd(100) <= prob2 * multiplier and not npc:is_skill_disabled(special2) then
        cast_at(npc, creature, special2)
      end
    end
  elseif not creature:is_player() and J.contains(VEGETABLE, creature:id()) then
    npc:set_target(creature)
    npc:set_running()
    npc:attack(creature)
  end
end)

s:on_aggro_range_enter(function(npc, player, is_summon)
  if npc:is_script_value(0) then
    npc:set_script_value(1)
    npc:broadcast_say(ChatType.NPC_GENERAL, "?")
    npc:clear_aggro_list()
    s:start_quest_timer("TREX_ATTACK", 6000, npc, player)
  end
end)

-- trex_attacked is a Tyrannosaurus's answer to a hit: Berserk as it
-- weakens, then its skills, more often after each Berserk.
local function trex_attacked(npc, attacker)
  local target = npc:most_hated()
  local hp = hp_percent(npc)
  local buff1, buff2 = skill(SELFBUFF1), skill(SELFBUFF2)
  if hp <= 30 then
    if npc:is_script_value(3) then
      if not npc:is_skill_disabled(buff1) then
        npc:cast(buff1)
      end
    elseif npc:is_script_value(1) and not npc:is_skill_disabled(buff2) then
      npc:cast(buff2)
    end
  elseif hp <= 60 and npc:is_script_value(3) and not npc:is_skill_disabled(buff1) then
    npc:cast(buff1)
  end

  local level = npc:get_script_value()
  local stun = skill(LONGRANGEDMAGIC1)
  if npc:calculate_distance_3d(attacker) > 100 then
    if not npc:is_skill_disabled(stun) and rnd(100) <= 10 * level then
      cast_at(npc, attacker, stun)
    end
    return
  end
  if not npc:is_skill_disabled(stun) and rnd(100) <= 10 * level then
    cast_at(npc, target, stun)
  end
  for _, special in ipairs({ { PHYSICALSPECIAL1, 5 }, { PHYSICALSPECIAL2, 3 }, { PHYSICALSPECIAL3, 5 } }) do
    local sk = skill(special[1])
    if not npc:is_skill_disabled(sk) and rnd(100) <= special[2] * level then
      cast_at(npc, target, sk)
    end
  end
end

-- monster_attacked is another dinosaur's answer to a hit: below 30 % it
-- forgets its enemies, buffs itself once and uses its skills on the one it
-- hated most.
local function monster_attacked(npc)
  local prob1 = npc:param_int("ProbPhysicalSpecial1", 0)
  local prob2 = npc:param_int("ProbPhysicalSpecial2", 0)
  local self_buff = npc:param_skill("SelfRangeBuff1")
  local special1 = npc:param_skill("PhysicalSpecial1")
  local special2 = npc:param_skill("PhysicalSpecial2")

  local hp = hp_percent(npc)
  npc:set_var("SKILL_MULTIPLER", hp <= 50 and 2 or 1)
  local target
  if hp <= 30 and npc:var_int("SELFBUFF_USED") == 0 then
    target = npc:most_hated()
    npc:clear_aggro_list()
    if not npc:is_skill_disabled(self_buff) then
      npc:set_var("SELFBUFF_USED", 1)
      npc:cast(self_buff)
      npc:set_running()
      if target ~= nil then
        npc:attack(target)
      end
    end
  end
  if target == nil then
    return
  end
  local multiplier = npc:var_int("SKILL_MULTIPLER")
  if rnd(100) <= prob1 * multiplier and not npc:is_skill_disabled(special1) then
    cast_at(npc, target, special1)
  end
  if rnd(100) <= prob2 * multiplier and not npc:is_skill_disabled(special2) then
    cast_at(npc, target, special2)
  end
end

s:on_attack(function(npc, attacker, damage, is_summon)
  if npc:id() == EGG then
    if rnd(100) <= 80 and npc:is_script_value(0) then
      npc:set_script_value(1)
      local playable = is_summon and attacker:summon() or attacker
      for _, monster in ipairs(world.visible_objects(npc, "Attackable", 500)) do
        if rnd.bool() then
          attack_desire(monster, playable)
        end
      end
    end
  elseif J.contains(TREX, npc:id()) then
    trex_attacked(npc, attacker)
  else
    monster_attacked(npc)
  end
end)

-- arm starts a sprigant's poison or a ghost Tyrannosaurus's departure.
local function arm(npc)
  if J.contains(SPRIGANT, npc:id()) then
    s:cancel_quest_timer("USE_SKILL", npc, nil)
    s:start_quest_timer("USE_SKILL", USE_SKILL_MS, npc, nil)
  elseif J.contains(TREX, npc:id()) then
    if npc:param_int("CollectGhost", 0) == 1 then
      s:cancel_quest_timer("GHOST_DESPAWN", npc, nil)
      s:start_quest_timer("GHOST_DESPAWN", npc:param_int("CollectGhostDespawnTime", 30) * 60000, npc, nil)
    end
  end
end

s:on_spawn(function(npc)
  if J.contains(SPRIGANT, npc:id()) or J.contains(TREX, npc:id()) then
    arm(npc)
  else
    npc:set_var("SELFBUFF_USED", 0)
    npc:set_var("SKILL_MULTIPLER", 1)
  end
end)

-- A reload stops the timers: the living sprigants and ghosts get theirs
-- again (a ghost's departure starts over).
s:on_start(function(ev)
  if not ev.reload then
    return
  end
  for _, id in ipairs(SPRIGANT) do
    for _, npc in ipairs(world.npcs_by_id(id)) do
      if not npc:is_dead() then
        arm(npc)
      end
    end
  end
  for _, id in ipairs(TREX) do
    for _, npc in ipairs(world.npcs_by_id(id)) do
      if not npc:is_dead() then
        arm(npc)
      end
    end
  end
end)

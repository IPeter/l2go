-- The Hot Springs monsters spread diseases, ported from Mobius
-- ai/areas/HotSprings/HotSprings.
--
-- Hit, or a skill at them, and they may infect the player: Malaria
-- (10 %), then their own disease (10 %). Each infection raises the
-- disease's level by one, up to 10.

local s = script.new("HotSprings")

-- NPCs
local BANDERSNATCHLING = 21314
local FLAVA = 21316
local ATROXSPAWN = 21317
local NEPENTHES = 21319
local ATROX = 21321
local BANDERSNATCH = 21322

-- Skills
local RHEUMATISM = 4551
local CHOLERA = 4552
local FLU = 4553
local MALARIA = 4554

-- Misc
local DISEASE_CHANCE = 10

-- Each monster's own disease.
local DISEASES = {
  [BANDERSNATCHLING] = RHEUMATISM,
  [ATROX] = RHEUMATISM,
  [FLAVA] = CHOLERA,
  [NEPENTHES] = CHOLERA,
  [ATROXSPAWN] = FLU,
  [BANDERSNATCH] = FLU,
}

local MONSTERS = { BANDERSNATCHLING, FLAVA, ATROXSPAWN, NEPENTHES, ATROX, BANDERSNATCH }

s:add_attack_id(MONSTERS)
s:add_skill_see_id(MONSTERS)

local function try_to_infect(npc, player, disease_id)
  local level = player:effect_level(disease_id)
  if level == nil then
    level = 1
  elseif level < 10 then
    level = level + 1
  else
    level = 10
  end
  local skill = skills.get(disease_id, level)
  if skill ~= nil and not npc:is_casting_now() and npc:check_do_cast_conditions(skill) then
    npc:set_target(player)
    npc:cast(skill)
  end
end

local function spread_disease(npc, player)
  if rnd(100) < DISEASE_CHANCE then
    try_to_infect(npc, player, MALARIA)
  end
  if rnd(100) < DISEASE_CHANCE then
    local disease = DISEASES[npc:id()]
    if disease ~= nil then
      try_to_infect(npc, player, disease)
    end
  end
end

s:on_attack(function(npc, attacker, damage, is_summon)
  spread_disease(npc, attacker)
end)

s:on_skill_see(function(npc, caster, skill, targets, is_summon)
  for _, target in ipairs(targets) do
    if target:object_id() == npc:object_id() then
      spread_disease(npc, caster)
      return
    end
  end
end)

-- Treasure chests, ported from Mobius ai/others/Chests (Fulminus).
--
-- A box opened by force explodes: on the first hit, or on an offensive
-- skill aimed at it. Keys and unlock skills (target type UNLOCKABLE) are
-- the OpenChest effect's business, and a mimic fights back as a normal
-- monster.

local s = script.new("Chests")

local TREASURE_CHESTS = {
  18265, 18266, 18267, 18268, 18269, 18270, 18271, 18272, 18273, 18274,
  18275, 18276, 18277, 18278, 18279, 18280, 18281, 18282, 18283, 18284,
  18285, 18286, 18287, 18288, 18289, 18290, 18291, 18292, 18293, 18294,
  18295, 18296, 18297, 18298, 21671, 21694, 21717, 21740, 21763, 21786,
  21801, 21802, 21803, 21804, 21805, 21806, 21807, 21808, 21809, 21810,
  21811, 21812, 21813, 21814, 21815, 21816, 21817, 21818, 21819, 21820,
  21821, 21822,
}

s:add_attack_id(TREASURE_CHESTS)
s:add_skill_see_id(TREASURE_CHESTS)

-- explode blows a box up the first time it is forced.
local function explode(chest)
  if chest:is_box() and not chest:is_interacted() then
    chest:set_interacted()
    chest:treasure_bomb()
  end
end

s:on_skill_see(function(npc, caster, skill, targets, is_summon)
  if not npc:is_type("Chest") then
    return
  end
  -- Only when the player tries to open this chest with the skill.
  local found = false
  for _, target in ipairs(targets) do
    if target == npc then
      found = true
      break
    end
  end
  if not found then
    return
  end
  if skill:target_type() == "UNLOCKABLE" then
    return
  end
  explode(npc)
end)

s:on_attack(function(npc, attacker, damage, is_summon)
  if npc:is_type("Chest") then
    explode(npc)
  end
end)

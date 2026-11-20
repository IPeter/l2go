-- Translated by questport from quests/Q00373_SupplierOfReagents (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(373, "Q00373_SupplierOfReagents", "Supplier of Reagents")

-- NPCs
local WESLEY = 30166
local URN = 31149
-- Monsters
local CRENDION = 20813
local HALLATE_MAID = 20822
local HALLATE_GUARDIAN = 21061
local PLATINUM_TRIBE_SHAMAN = 20828
local PLATINUM_GUARDIAN_SHAMAN = 21066
local LAVA_WYRM = 21111
local HAMES_ORC_SHAMAN = 21115
-- Quest items
local MIXING_STONE = 5904
local MIXING_MANUAL = 6317
-- Items - pouches
local REAGENT_POUCH_1 = 6007
local REAGENT_POUCH_2 = 6008
local REAGENT_POUCH_3 = 6009
local REAGENT_BOX = 6010
-- Items - ingredients
local WYRMS_BLOOD = 6011
local LAVA_STONE = 6012
local MOONSTONE_SHARD = 6013
local ROTTEN_BONE = 6014
local DEMONS_BLOOD = 6015
local INFERNIUM_ORE = 6016
-- Items - catalysts
local BLOOD_ROOT = 6017
local VOLCANIC_ASH = 6018
local QUICKSILVER = 6019
local SULFUR = 6020
local DEMONIC_ESSENCE = 6031
local MIDNIGHT_OIL = 6030
-- Items - products
local DRACOPLASM = 6021
local MAGMA_DUST = 6022
local MOON_DUST = 6023
local NECROPLASM = 6024
local DEMONPLASM = 6025
local INFERNO_DUST = 6026
local FIRE_ESSENCE = 6028
local LUNARGENT = 6029
-- Items - products final
local DRACONIC_ESSENCE = 6027
local ABYSS_OIL = 6032
local HELLFIRE_OIL = 6033
local NIGHTMARE_OIL = 6034
local PURE_SILVER = 6320
-- Variables
local INGREDIENT = "ingredient"
local CATALYST = "catalyst"
-- This droplist defines the npcId, the item dropped and the luck.
-- <ul>
-- <li>HAMES_ORC_SHAMAN : 47% chance to drop - reagent pouch (47%)</li>
-- <li>HALLATES_MAID : 84,4% chance to drop - reageant pouch (66,4%) and volcanic ash (18%)</li>
-- <li>HALLATES_GUARDIAN : 83,3% chance to drop - demon's blood (72,9%) and moonstone shard (10,4%)</li>
-- <li>PLATINUM_GUARDIAN_SHAMAN : 44,2% chance to drop - reagent box (44,2%)</li>
-- <li>PLATINUM_TRIBE_SHAMAN : 100% chance to drop - reagent pouch (68%) and quichsilver (32%)</li>
-- <li>CRENDION : 100% chance to drop - rotten bone piece (61,8%) and quicksilver (38,2%)</li>
-- <li>LAVA_WYRM : 75% chance to drop - wyrm's blood (50,5%) and lava stone (24,5%)</li>
-- </ul>
local DROPLIST = {}
local FORMULAS = {
  {10, WYRMS_BLOOD, BLOOD_ROOT, DRACOPLASM},
  {10, LAVA_STONE, VOLCANIC_ASH, MAGMA_DUST},
  {10, MOONSTONE_SHARD, VOLCANIC_ASH, MOON_DUST},
  {10, ROTTEN_BONE, BLOOD_ROOT, NECROPLASM},
  {10, DEMONS_BLOOD, BLOOD_ROOT, DEMONPLASM},
  {10, INFERNIUM_ORE, VOLCANIC_ASH, INFERNO_DUST},
  {10, DRACOPLASM, QUICKSILVER, DRACONIC_ESSENCE},
  {10, MAGMA_DUST, SULFUR, FIRE_ESSENCE},
  {10, MOON_DUST, QUICKSILVER, LUNARGENT},
  {10, NECROPLASM, QUICKSILVER, MIDNIGHT_OIL},
  {10, DEMONPLASM, SULFUR, DEMONIC_ESSENCE},
  {10, INFERNO_DUST, SULFUR, ABYSS_OIL},
  {1, FIRE_ESSENCE, DEMONIC_ESSENCE, HELLFIRE_OIL},
  {1, LUNARGENT, MIDNIGHT_OIL, NIGHTMARE_OIL},
  {1, LUNARGENT, QUICKSILVER, PURE_SILVER},
}
local TEMPERATURES = {{1, 100, 1}, {2, 45, 2}, {3, 15, 3}}

DROPLIST[PLATINUM_GUARDIAN_SHAMAN] = {REAGENT_BOX, 442000, 0}
DROPLIST[HAMES_ORC_SHAMAN] = {REAGENT_POUCH_3, 470000, 0}
DROPLIST[PLATINUM_TRIBE_SHAMAN] = {REAGENT_POUCH_2, QUICKSILVER, 680, 1000}
DROPLIST[HALLATE_MAID] = {REAGENT_POUCH_1, VOLCANIC_ASH, 664, 844}
DROPLIST[HALLATE_GUARDIAN] = {DEMONS_BLOOD, MOONSTONE_SHARD, 729, 833}
DROPLIST[CRENDION] = {ROTTEN_BONE, QUICKSILVER, 618, 1000}
DROPLIST[LAVA_WYRM] = {WYRMS_BLOOD, LAVA_STONE, 505, 750}

q:register_quest_items(MIXING_STONE, MIXING_MANUAL)
q:add_start_npc(WESLEY)
q:add_talk_id(WESLEY, URN)
q:add_kill_id(CRENDION, HALLATE_MAID, HALLATE_GUARDIAN, PLATINUM_TRIBE_SHAMAN, PLATINUM_GUARDIAN_SHAMAN, LAVA_WYRM, HAMES_ORC_SHAMAN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  -- Wesley
  if event == "30166-04.htm" then
    st:start_quest()
    q:give_items(player, MIXING_STONE, 1)
    q:give_items(player, MIXING_MANUAL, 1)
  elseif event == "30166-09.htm" then
    st:exit_quest(true, true)
  elseif event == "31149-02.htm" then
    if not q:has_quest_items(player, MIXING_STONE) then
      htmltext = "31149-04.htm"
    end
  elseif J.starts_with(event, "31149-03-") then
    local regentId = J.parse_int(J.substring(event, 9, 13))
    for _, formula in ipairs(FORMULAS) do
      if not (formula[2] ~= regentId) then
        -- Not enough items, cancel the operation.
        if q:get_quest_items_count(player, regentId) < formula[1] then
          break
        end
        st:set(INGREDIENT, tostring(regentId))
        return htmltext
      end
    end
    htmltext = "31149-04.htm"
  elseif J.starts_with(event, "31149-06-") then
    local catalyst = J.parse_int(J.substring(event, 9, 13))
    -- Not enough items, cancel the operation.
    if not q:has_quest_items(player, catalyst) then
      return "31149-04.htm"
    end
    st:set(CATALYST, tostring(catalyst))
  elseif J.starts_with(event, "31149-12-") then
    local regent = st:get_int(INGREDIENT)
    local catalyst = st:get_int(CATALYST)
    for _, formula in ipairs(FORMULAS) do
      if not (formula[2] ~= regent or formula[3] ~= catalyst) then
        -- Not enough regents.
        if q:get_quest_items_count(player, regent) < formula[1] then
          break
        end
        -- Not enough catalysts.
        if not q:has_quest_items(player, catalyst) then
          break
        end
        q:take_items(player, regent, formula[1])
        q:take_items(player, catalyst, 1)
        local tempIndex = J.parse_int(J.substring(event, 9, 10))
        for _, temperature in ipairs(TEMPERATURES) do
          if not (temperature[1] ~= tempIndex) then
            if rnd(100) < temperature[2] then
              q:give_items(player, formula[4], temperature[3])
              return "31149-12-" .. formula[4] .. ".htm"
            end
            return "31149-11.htm"
          end
        end
      end
    end
    htmltext = "31149-13.htm"
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 57 and "30166-01.htm" or "30166-02.htm"
  elseif sw == State.STARTED then
    if npc:id() == WESLEY then
      htmltext = "30166-05.htm"
    else
      htmltext = "31149-01.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil then
    return
  end
  local partyMember = st:get_player()
  local drop = DROPLIST[npc:id()]
  if drop[3] == 0 then
    if rnd(1000000) < drop[2] then
      q:give_items(partyMember, drop[1], 1)
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  else
    local random = rnd(1000)
    if random < drop[4] then
      q:give_items(partyMember, (function() if random < drop[3] then return drop[1] end return drop[2] end)(), 1)
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)

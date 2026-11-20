-- Translated by questport from quests/Q00330_AdeptOfTaste (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(330, "Q00330_AdeptOfTaste", "Adept of Taste")

-- NPCs
local SONIA = 30062
local GLYVKA = 30067
local ROLLANT = 30069
local JACOB = 30073
local PANO = 30078
local MIRIEN = 30461
local JONAS = 30469
-- Items
local INGREDIENT_LIST = 1420
local SONIA_BOTANY_BOOK = 1421
local RED_MANDRAGORA_ROOT = 1422
local WHITE_MANDRAGORA_ROOT = 1423
local RED_MANDRAGORA_SAP = 1424
local WHITE_MANDRAGORA_SAP = 1425
local JACOB_INSECT_BOOK = 1426
local NECTAR = 1427
local ROYAL_JELLY = 1428
local HONEY = 1429
local GOLDEN_HONEY = 1430
local PANO_CONTRACT = 1431
local HOBGOBLIN_AMULET = 1432
local DIONIAN_POTATO = 1433
local GLYVKA_BOTANY_BOOK = 1434
local GREEN_MARSH_MOSS = 1435
local BROWN_MARSH_MOSS = 1436
local GREEN_MOSS_BUNDLE = 1437
local BROWN_MOSS_BUNDLE = 1438
local ROLANT_CREATURE_BOOK = 1439
local MONSTER_EYE_BODY = 1440
local MONSTER_EYE_MEAT = 1441
local JONAS_STEAK_DISH_1 = 1442
local JONAS_STEAK_DISH_2 = 1443
local JONAS_STEAK_DISH_3 = 1444
local JONAS_STEAK_DISH_4 = 1445
local JONAS_STEAK_DISH_5 = 1446
local MIRIEN_REVIEW_1 = 1447
local MIRIEN_REVIEW_2 = 1448
local MIRIEN_REVIEW_3 = 1449
local MIRIEN_REVIEW_4 = 1450
local MIRIEN_REVIEW_5 = 1451
-- Rewards
local JONAS_SALAD_RECIPE = 1455
local JONAS_SAUCE_RECIPE = 1456
local JONAS_STEAK_RECIPE = 1457
-- Drop chances
local CHANCES = {}

local function hasAllIngredients(player)
  return q:has_quest_items(player, DIONIAN_POTATO, MONSTER_EYE_MEAT) and q:has_at_least_one_quest_item(player, WHITE_MANDRAGORA_SAP, RED_MANDRAGORA_SAP) and q:has_at_least_one_quest_item(player, GOLDEN_HONEY, HONEY) and q:has_at_least_one_quest_item(player, BROWN_MOSS_BUNDLE, GREEN_MOSS_BUNDLE)
end

CHANCES[20204] = {92, 100}
CHANCES[20229] = {80, 95}
CHANCES[20223] = {70, 77}
CHANCES[20154] = {70, 77}
CHANCES[20155] = {87, 96}
CHANCES[20156] = {77, 85}

q:register_quest_items(INGREDIENT_LIST, RED_MANDRAGORA_SAP, WHITE_MANDRAGORA_SAP, HONEY, GOLDEN_HONEY, DIONIAN_POTATO, GREEN_MOSS_BUNDLE, BROWN_MOSS_BUNDLE, MONSTER_EYE_MEAT, MIRIEN_REVIEW_1, MIRIEN_REVIEW_2, MIRIEN_REVIEW_3, MIRIEN_REVIEW_4, MIRIEN_REVIEW_5, JONAS_STEAK_DISH_1, JONAS_STEAK_DISH_2, JONAS_STEAK_DISH_3, JONAS_STEAK_DISH_4, JONAS_STEAK_DISH_5, SONIA_BOTANY_BOOK, RED_MANDRAGORA_ROOT, WHITE_MANDRAGORA_ROOT, JACOB_INSECT_BOOK, NECTAR, ROYAL_JELLY, PANO_CONTRACT, HOBGOBLIN_AMULET, GLYVKA_BOTANY_BOOK, GREEN_MARSH_MOSS, BROWN_MARSH_MOSS, ROLANT_CREATURE_BOOK, MONSTER_EYE_BODY)
q:add_start_npc(JONAS) -- Jonas
q:add_talk_id(JONAS, SONIA, GLYVKA, ROLLANT, JACOB, PANO, MIRIEN)
q:add_kill_id(20147, 20154, 20155, 20156, 20204, 20223, 20226, 20228, 20229, 20265, 20266)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30469-03.htm" then
    st:start_quest()
    q:give_items(player, INGREDIENT_LIST, 1)
  elseif event == "30062-05.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, SONIA_BOTANY_BOOK, 1)
    q:take_items(player, RED_MANDRAGORA_ROOT, -1)
    q:take_items(player, WHITE_MANDRAGORA_ROOT, -1)
    q:give_items(player, RED_MANDRAGORA_SAP, 1)
  elseif event == "30073-05.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, JACOB_INSECT_BOOK, 1)
    q:take_items(player, NECTAR, -1)
    q:take_items(player, ROYAL_JELLY, -1)
    q:give_items(player, HONEY, 1)
  elseif event == "30067-05.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, GLYVKA_BOTANY_BOOK, 1)
    q:take_items(player, GREEN_MARSH_MOSS, -1)
    q:take_items(player, BROWN_MARSH_MOSS, -1)
    q:give_items(player, GREEN_MOSS_BUNDLE, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 24 and "30469-01.htm" or "30469-02.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == JONAS then
      if q:has_quest_items(player, INGREDIENT_LIST) then
        if not hasAllIngredients(player) then
          htmltext = "30469-04.htm"
        else
          local dish
          local specialIngredientsNumber = q:get_quest_items_count(player, WHITE_MANDRAGORA_SAP) + q:get_quest_items_count(player, GOLDEN_HONEY) + q:get_quest_items_count(player, BROWN_MOSS_BUNDLE)
          if rnd.bool() then
            htmltext = "30469-05t" .. tostring(specialIngredientsNumber + 2) .. ".htm"
            dish = 1443 + specialIngredientsNumber
          else
            htmltext = "30469-05t" .. tostring(specialIngredientsNumber + 1) .. ".htm"
            dish = 1442 + specialIngredientsNumber
          end
          -- Sound according dish.
          q:play_sound(player, dish == JONAS_STEAK_DISH_5 and QuestSound.ITEMSOUND_QUEST_JACKPOT or QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, INGREDIENT_LIST, 1)
          q:take_items(player, RED_MANDRAGORA_SAP, 1)
          q:take_items(player, WHITE_MANDRAGORA_SAP, 1)
          q:take_items(player, HONEY, 1)
          q:take_items(player, GOLDEN_HONEY, 1)
          q:take_items(player, DIONIAN_POTATO, 1)
          q:take_items(player, GREEN_MOSS_BUNDLE, 1)
          q:take_items(player, BROWN_MOSS_BUNDLE, 1)
          q:take_items(player, MONSTER_EYE_MEAT, 1)
          q:give_items(player, dish, 1)
        end
      elseif q:has_at_least_one_quest_item(player, JONAS_STEAK_DISH_1, JONAS_STEAK_DISH_2, JONAS_STEAK_DISH_3, JONAS_STEAK_DISH_4, JONAS_STEAK_DISH_5) then
        htmltext = "30469-06.htm"
      elseif q:has_at_least_one_quest_item(player, MIRIEN_REVIEW_1, MIRIEN_REVIEW_2, MIRIEN_REVIEW_3, MIRIEN_REVIEW_4, MIRIEN_REVIEW_5) then
        if q:has_quest_items(player, MIRIEN_REVIEW_1) then
          htmltext = "30469-06t1.htm"
          q:take_items(player, MIRIEN_REVIEW_1, 1)
          q:give_adena(player, 7500, true)
          q:add_exp_and_sp(player, 6000, 0)
        elseif q:has_quest_items(player, MIRIEN_REVIEW_2) then
          htmltext = "30469-06t2.htm"
          q:take_items(player, MIRIEN_REVIEW_2, 1)
          q:give_adena(player, 9000, true)
          q:add_exp_and_sp(player, 7000, 0)
        elseif q:has_quest_items(player, MIRIEN_REVIEW_3) then
          htmltext = "30469-06t3.htm"
          q:take_items(player, MIRIEN_REVIEW_3, 1)
          q:give_adena(player, 5800, true)
          q:give_items(player, JONAS_SALAD_RECIPE, 1)
          q:add_exp_and_sp(player, 9000, 0)
        elseif q:has_quest_items(player, MIRIEN_REVIEW_4) then
          htmltext = "30469-06t4.htm"
          q:take_items(player, MIRIEN_REVIEW_4, 1)
          q:give_adena(player, 6800, true)
          q:give_items(player, JONAS_SAUCE_RECIPE, 1)
          q:add_exp_and_sp(player, 10500, 0)
        elseif q:has_quest_items(player, MIRIEN_REVIEW_5) then
          htmltext = "30469-06t5.htm"
          q:take_items(player, MIRIEN_REVIEW_5, 1)
          q:give_adena(player, 7800, true)
          q:give_items(player, JONAS_STEAK_RECIPE, 1)
          q:add_exp_and_sp(player, 12000, 0)
        end
        st:exit_quest(true, true)
      end
    elseif sw2 == MIRIEN then
      if q:has_quest_items(player, INGREDIENT_LIST) then
        htmltext = "30461-01.htm"
      elseif q:has_at_least_one_quest_item(player, JONAS_STEAK_DISH_1, JONAS_STEAK_DISH_2, JONAS_STEAK_DISH_3, JONAS_STEAK_DISH_4, JONAS_STEAK_DISH_5) then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        if q:has_quest_items(player, JONAS_STEAK_DISH_1) then
          htmltext = "30461-02t1.htm"
          q:take_items(player, JONAS_STEAK_DISH_1, 1)
          q:give_items(player, MIRIEN_REVIEW_1, 1)
        elseif q:has_quest_items(player, JONAS_STEAK_DISH_2) then
          htmltext = "30461-02t2.htm"
          q:take_items(player, JONAS_STEAK_DISH_2, 1)
          q:give_items(player, MIRIEN_REVIEW_2, 1)
        elseif q:has_quest_items(player, JONAS_STEAK_DISH_3) then
          htmltext = "30461-02t3.htm"
          q:take_items(player, JONAS_STEAK_DISH_3, 1)
          q:give_items(player, MIRIEN_REVIEW_3, 1)
        elseif q:has_quest_items(player, JONAS_STEAK_DISH_4) then
          htmltext = "30461-02t4.htm"
          q:take_items(player, JONAS_STEAK_DISH_4, 1)
          q:give_items(player, MIRIEN_REVIEW_4, 1)
        elseif q:has_quest_items(player, JONAS_STEAK_DISH_5) then
          htmltext = "30461-02t5.htm"
          q:take_items(player, JONAS_STEAK_DISH_5, 1)
          q:give_items(player, MIRIEN_REVIEW_5, 1)
        end
      elseif q:has_at_least_one_quest_item(player, MIRIEN_REVIEW_1, MIRIEN_REVIEW_2, MIRIEN_REVIEW_3, MIRIEN_REVIEW_4, MIRIEN_REVIEW_5) then
        htmltext = "30461-04.htm"
      end
    elseif sw2 == SONIA then
      if not q:has_quest_items(player, RED_MANDRAGORA_SAP) and not q:has_quest_items(player, WHITE_MANDRAGORA_SAP) then
        if not q:has_quest_items(player, SONIA_BOTANY_BOOK) then
          htmltext = "30062-01.htm"
          q:give_items(player, SONIA_BOTANY_BOOK, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          if q:get_quest_items_count(player, RED_MANDRAGORA_ROOT) < 40 or q:get_quest_items_count(player, WHITE_MANDRAGORA_ROOT) < 40 then
            htmltext = "30062-02.htm"
          elseif q:get_quest_items_count(player, WHITE_MANDRAGORA_ROOT) >= 40 then
            htmltext = "30062-06.htm"
            q:take_items(player, SONIA_BOTANY_BOOK, 1)
            q:take_items(player, RED_MANDRAGORA_ROOT, -1)
            q:take_items(player, WHITE_MANDRAGORA_ROOT, -1)
            q:give_items(player, WHITE_MANDRAGORA_SAP, 1)
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          else
            htmltext = "30062-03.htm"
          end
        end
      else
        htmltext = "30062-07.htm"
      end
    elseif sw2 == JACOB then
      if not q:has_quest_items(player, HONEY) and not q:has_quest_items(player, GOLDEN_HONEY) then
        if not q:has_quest_items(player, JACOB_INSECT_BOOK) then
          htmltext = "30073-01.htm"
          q:give_items(player, JACOB_INSECT_BOOK, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          if q:get_quest_items_count(player, NECTAR) < 20 then
            htmltext = "30073-02.htm"
          else
            if q:get_quest_items_count(player, ROYAL_JELLY) < 10 then
              htmltext = "30073-03.htm"
            else
              htmltext = "30073-06.htm"
              q:take_items(player, JACOB_INSECT_BOOK, 1)
              q:take_items(player, NECTAR, -1)
              q:take_items(player, ROYAL_JELLY, -1)
              q:give_items(player, GOLDEN_HONEY, 1)
              q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            end
          end
        end
      else
        htmltext = "30073-07.htm"
      end
    elseif sw2 == PANO then
      if not q:has_quest_items(player, DIONIAN_POTATO) then
        if not q:has_quest_items(player, PANO_CONTRACT) then
          htmltext = "30078-01.htm"
          q:give_items(player, PANO_CONTRACT, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          if q:get_quest_items_count(player, HOBGOBLIN_AMULET) < 30 then
            htmltext = "30078-02.htm"
          else
            htmltext = "30078-03.htm"
            q:take_items(player, PANO_CONTRACT, 1)
            q:take_items(player, HOBGOBLIN_AMULET, -1)
            q:give_items(player, DIONIAN_POTATO, 1)
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        end
      else
        htmltext = "30078-04.htm"
      end
    elseif sw2 == GLYVKA then
      if not q:has_quest_items(player, GREEN_MOSS_BUNDLE) and not q:has_quest_items(player, BROWN_MOSS_BUNDLE) then
        if not q:has_quest_items(player, GLYVKA_BOTANY_BOOK) then
          q:give_items(player, GLYVKA_BOTANY_BOOK, 1)
          htmltext = "30067-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          if q:get_quest_items_count(player, GREEN_MARSH_MOSS) < 20 or q:get_quest_items_count(player, BROWN_MARSH_MOSS) < 20 then
            htmltext = "30067-02.htm"
          elseif q:get_quest_items_count(player, BROWN_MARSH_MOSS) >= 20 then
            htmltext = "30067-06.htm"
            q:take_items(player, GLYVKA_BOTANY_BOOK, 1)
            q:take_items(player, GREEN_MARSH_MOSS, -1)
            q:take_items(player, BROWN_MARSH_MOSS, -1)
            q:give_items(player, BROWN_MOSS_BUNDLE, 1)
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          else
            htmltext = "30067-03.htm"
          end
        end
      else
        htmltext = "30067-07.htm"
      end
    elseif sw2 == ROLLANT then
      if not q:has_quest_items(player, MONSTER_EYE_MEAT) then
        if not q:has_quest_items(player, ROLANT_CREATURE_BOOK) then
          htmltext = "30069-01.htm"
          q:give_items(player, ROLANT_CREATURE_BOOK, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          if q:get_quest_items_count(player, MONSTER_EYE_BODY) < 30 then
            htmltext = "30069-02.htm"
          else
            htmltext = "30069-03.htm"
            q:take_items(player, ROLANT_CREATURE_BOOK, 1)
            q:take_items(player, MONSTER_EYE_BODY, -1)
            q:give_items(player, MONSTER_EYE_MEAT, 1)
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        end
      else
        htmltext = "30069-04.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local npcId = npc:id()
  if npcId == 20265 then
    if q:has_quest_items(player, ROLANT_CREATURE_BOOK) and q:get_quest_items_count(player, MONSTER_EYE_BODY) < 30 and rnd(100) < 97 then
      q:give_items(player, MONSTER_EYE_BODY, rnd(97) < 77 and 2 or 3)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == 20266 then
    if q:has_quest_items(player, ROLANT_CREATURE_BOOK) and q:get_quest_items_count(player, MONSTER_EYE_BODY) < 30 then
      q:give_items(player, MONSTER_EYE_BODY, rnd(10) < 7 and 1 or 2)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == 20226 then
    if q:has_quest_items(player, GLYVKA_BOTANY_BOOK) then
      local itemId = rnd(96) < 87 and GREEN_MARSH_MOSS or BROWN_MARSH_MOSS
      if q:get_quest_items_count(player, itemId) < 20 and rnd(100) < 96 then
        q:give_items(player, itemId, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == 20228 then
    if q:has_quest_items(player, GLYVKA_BOTANY_BOOK) then
      local itemId = rnd(10) < 9 and GREEN_MARSH_MOSS or BROWN_MARSH_MOSS
      if q:get_quest_items_count(player, itemId) < 20 then
        q:give_items(player, itemId, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == 20147 then
    if q:has_quest_items(player, PANO_CONTRACT) and q:get_quest_items_count(player, HOBGOBLIN_AMULET) < 30 then
      q:give_items(player, HOBGOBLIN_AMULET, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == 20204 or npcId == 20229 then
    if q:has_quest_items(player, JACOB_INSECT_BOOK) then
      local random = rnd(100)
      local chances = CHANCES[npcId]
      if random < chances[1] then
        if q:get_quest_items_count(player, NECTAR) < 20 then
          q:give_items(player, NECTAR, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif random < chances[2] and q:get_quest_items_count(player, ROYAL_JELLY) < 10 then
        q:give_items(player, ROYAL_JELLY, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == 20223 or npcId == 20154 or npcId == 20155 or npcId == 20156 then
    if q:has_quest_items(player, SONIA_BOTANY_BOOK) then
      local random = rnd(100)
      local chances = CHANCES[npcId]
      if random < chances[2] then
        local itemId = random < chances[1] and RED_MANDRAGORA_ROOT or WHITE_MANDRAGORA_ROOT
        if q:get_quest_items_count(player, itemId) < 40 then
          q:give_items(player, itemId, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  end
end)

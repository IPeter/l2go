-- Translated by questport from quests/Q00334_TheWishingPotion (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Adapted from FirstTeam Interlude
local J = require("lib.java")

local q = quest.new(334, "Q00334_TheWishingPotion", "The Wishing Potion")

local GRIMA = 27135
local SUCCUBUS_OF_SEDUCTION = 27136
local GREAT_DEMON_KING = 27138
local SECRET_KEEPER_TREE = 27139
local SANCHES = 27153
local BONAPARTERIUS = 27154
local RAMSEBALIUS = 27155
local TORAI = 30557
local ALCHEMIST_MATILD = 30738
local RUPINA = 30742
local WISDOM_CHEST = 30743
local WHISPERING_WIND = 20078
local ANT_SOLDIER = 20087
local ANT_WARRIOR_CAPTAIN = 20088
local SILENOS = 20168
local TYRANT = 20192
local TYRANT_KINGPIN = 20193
local AMBER_BASILISK = 20199
local HORROR_MIST_RIPPER = 20227
local TURAK_BUGBEAR = 20248
local TURAK_BUGBEAR_WARRIOR = 20249
local GLASS_JAGUAR = 20250
local DEMONS_TUNIC_ID = 441
local DEMONS_STOCKINGS_ID = 472
local NECKLACE_OF_GRACE_ID = 931
local SPELLBOOK_ICEBOLT_ID = 1049
local SPELLBOOK_BATTLEHEAL_ID = 1050
local DEMONS_TUNIC_FABRIC_ID = 1979
local DEMONS_STOCKINGS_PATTERN_ID = 1980
local DEMONS_BOOTS_ID = 2435
local DEMONS_GLOVES_ID = 2459
local DEMONS_BOOTS_FABRIC_ID = 2952
local DEMONS_GLOVES_FABRIC_ID = 2953
local WISH_POTION_ID = 3467
local ANCIENT_CROWN_ID = 3468
local CERTIFICATE_OF_ROYALTY_ID = 3469
local GOLD_BAR_ID = 3470
local ALCHEMY_TEXT_ID = 3678
local SECRET_BOOK_ID = 3679
local POTION_RECIPE_1_ID = 3680
local POTION_RECIPE_2_ID = 3681
local MATILDS_ORB_ID = 3682
local FORBIDDEN_LOVE_SCROLL_ID = 3683
local HEART_OF_PAAGRIO_ID = 3943
local AMBER_SCALE_ID = 3684
local WIND_SOULSTONE_ID = 3685
local GLASS_EYE_ID = 3686
local HORROR_ECTOPLASM_ID = 3687
local SILENOS_HORN_ID = 3688
local ANT_SOLDIER_APHID_ID = 3689
local TYRANTS_CHITIN_ID = 3690
local BUGBEAR_BLOOD_ID = 3691
local DROP_CHANCE_FORBIDDEN_LOVE_SCROLL_ID = 3
local DROP_CHANCE_NECKLACE_OF_GRACE_ID = 5
local DROP_CHANCE_GOLD_BAR_ID = 10
local DROPLIST_COND = {
  {1, 2, SECRET_KEEPER_TREE, 0, SECRET_BOOK_ID, 1, 100, 1},
  {3, 0, AMBER_BASILISK, 0, AMBER_SCALE_ID, 1, 15, 1},
  {3, 0, WHISPERING_WIND, 0, WIND_SOULSTONE_ID, 1, 20, 1},
  {3, 0, GLASS_JAGUAR, 0, GLASS_EYE_ID, 1, 35, 1},
  {3, 0, HORROR_MIST_RIPPER, 0, HORROR_ECTOPLASM_ID, 1, 15, 1},
  {3, 0, SILENOS, 0, SILENOS_HORN_ID, 1, 30, 1},
  {3, 0, ANT_SOLDIER, 0, ANT_SOLDIER_APHID_ID, 1, 40, 1},
  {3, 0, ANT_WARRIOR_CAPTAIN, 0, ANT_SOLDIER_APHID_ID, 1, 40, 1},
  {3, 0, TYRANT, 0, TYRANTS_CHITIN_ID, 1, 50, 1},
  {3, 0, TYRANT_KINGPIN, 0, TYRANTS_CHITIN_ID, 1, 50, 1},
  {3, 0, TURAK_BUGBEAR, 0, BUGBEAR_BLOOD_ID, 1, 15, 1},
  {3, 0, TURAK_BUGBEAR_WARRIOR, 0, BUGBEAR_BLOOD_ID, 1, 25, 1},
}

local function checkIngr(st, player)
  if q:get_quest_items_count(player, AMBER_SCALE_ID) == 1 and q:get_quest_items_count(player, WIND_SOULSTONE_ID) == 1 and q:get_quest_items_count(player, GLASS_EYE_ID) == 1 and q:get_quest_items_count(player, HORROR_ECTOPLASM_ID) == 1 and q:get_quest_items_count(player, SILENOS_HORN_ID) == 1 and q:get_quest_items_count(player, ANT_SOLDIER_APHID_ID) == 1 and q:get_quest_items_count(player, TYRANTS_CHITIN_ID) == 1 and q:get_quest_items_count(player, BUGBEAR_BLOOD_ID) == 1 then
    st:set_cond(4, true)
    return true
  end
  st:set_cond(3, true)
  return false
end

q:add_start_npc(ALCHEMIST_MATILD)
q:add_talk_id(ALCHEMIST_MATILD, TORAI, WISDOM_CHEST, RUPINA)
q:register_quest_items(ALCHEMY_TEXT_ID, SECRET_BOOK_ID, POTION_RECIPE_1_ID, POTION_RECIPE_2_ID, AMBER_SCALE_ID, WIND_SOULSTONE_ID, GLASS_EYE_ID, HORROR_ECTOPLASM_ID, SILENOS_HORN_ID, ANT_SOLDIER_APHID_ID, TYRANTS_CHITIN_ID, BUGBEAR_BLOOD_ID)
for _, element in ipairs(DROPLIST_COND) do
  q:add_kill_id(element[3])
end
-- Monsters summoned by the Wishing Potion, handled after the drop list.
q:add_kill_id(GRIMA, SUCCUBUS_OF_SEDUCTION, SANCHES, BONAPARTERIUS, RAMSEBALIUS, GREAT_DEMON_KING)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30738-03.htm" then
    st:start_quest()
    q:give_items(player, ALCHEMY_TEXT_ID, 1)
  elseif event == "30738-06.htm" then
    if q:get_quest_items_count(player, WISH_POTION_ID) == 0 then
      q:take_items(player, ALCHEMY_TEXT_ID, -1)
      q:take_items(player, SECRET_BOOK_ID, -1)
      if q:get_quest_items_count(player, POTION_RECIPE_1_ID) == 0 then
        q:give_items(player, POTION_RECIPE_1_ID, 1)
      end
      if q:get_quest_items_count(player, POTION_RECIPE_2_ID) == 0 then
        q:give_items(player, POTION_RECIPE_2_ID, 1)
      end
      if q:get_quest_items_count(player, MATILDS_ORB_ID) == 0 then
        htmltext = "30738-06.htm"
      else
        htmltext = "30738-12.htm"
      end
      st:set_cond(3)
    elseif q:get_quest_items_count(player, MATILDS_ORB_ID) >= 1 and q:get_quest_items_count(player, WISH_POTION_ID) >= 1 then
      htmltext = "30738-13.htm"
    end
  elseif event == "30738-10.htm" then
    if checkIngr(st, player) then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_FINISH)
      q:take_items(player, ALCHEMY_TEXT_ID, -1)
      q:take_items(player, SECRET_BOOK_ID, -1)
      q:take_items(player, POTION_RECIPE_1_ID, -1)
      q:take_items(player, POTION_RECIPE_2_ID, -1)
      q:take_items(player, AMBER_SCALE_ID, -1)
      q:take_items(player, WIND_SOULSTONE_ID, -1)
      q:take_items(player, GLASS_EYE_ID, -1)
      q:take_items(player, HORROR_ECTOPLASM_ID, -1)
      q:take_items(player, SILENOS_HORN_ID, -1)
      q:take_items(player, ANT_SOLDIER_APHID_ID, -1)
      q:take_items(player, TYRANTS_CHITIN_ID, -1)
      q:take_items(player, BUGBEAR_BLOOD_ID, -1)
      if q:get_quest_items_count(player, MATILDS_ORB_ID) == 0 then
        q:give_items(player, MATILDS_ORB_ID, 1)
      end
      q:give_items(player, WISH_POTION_ID, 1)
      st:set_cond(0)
    else
      htmltext = "<html><head><body>You don't have required items</body></html>"
    end
  elseif event == "30738-14.htm" then
    if q:get_quest_items_count(player, WISH_POTION_ID) >= 1 then
      htmltext = "30738-15.htm"
    end
  elseif event == "30738-16.htm" then
    if q:get_quest_items_count(player, WISH_POTION_ID) >= 1 then
      q:take_items(player, WISH_POTION_ID, 1)
      if rnd(100) < 50 then
        q:add_spawn(SUCCUBUS_OF_SEDUCTION, player)
        q:add_spawn(SUCCUBUS_OF_SEDUCTION, player)
        q:add_spawn(SUCCUBUS_OF_SEDUCTION, player)
      else
        q:add_spawn(RUPINA, player)
      end
    else
      htmltext = "30738-14.htm"
    end
  elseif event == "30738-17.htm" then
    if q:get_quest_items_count(player, WISH_POTION_ID) >= 1 then
      q:take_items(player, WISH_POTION_ID, 1)
      local WISH_CHANCE = rnd(100) + 1
      if WISH_CHANCE <= 33 then
        q:add_spawn(GRIMA, player)
        q:add_spawn(GRIMA, player)
        q:add_spawn(GRIMA, player)
      elseif WISH_CHANCE >= 66 then
        q:give_adena(player, 10000, true)
      elseif rnd(100) < 2 then
        q:give_adena(player, (rnd(10) + 1) * 1000000, true)
      else
        q:add_spawn(GRIMA, player)
        q:add_spawn(GRIMA, player)
        q:add_spawn(GRIMA, player)
      end
    else
      htmltext = "30738-14.htm"
    end
  elseif event == "30738-18.htm" then
    if q:get_quest_items_count(player, WISH_POTION_ID) >= 1 then
      q:take_items(player, WISH_POTION_ID, 1)
      local WISH_CHANCE = rnd(100) + 1
      if WISH_CHANCE <= 33 then
        q:give_items(player, CERTIFICATE_OF_ROYALTY_ID, 1)
      elseif WISH_CHANCE >= 66 then
        q:give_items(player, ANCIENT_CROWN_ID, 1)
      else
        q:add_spawn(SANCHES, player)
      end
    else
      htmltext = "30738-14.htm"
    end
  elseif event == "30738-19.htm" then
    if q:get_quest_items_count(player, WISH_POTION_ID) >= 1 then
      q:take_items(player, 3467, 1)
      local WISH_CHANCE = rnd(100) + 1
      if WISH_CHANCE <= 33 then
        q:give_items(player, SPELLBOOK_ICEBOLT_ID, 1)
      elseif WISH_CHANCE <= 66 then
        q:give_items(player, SPELLBOOK_BATTLEHEAL_ID, 1)
      else
        q:add_spawn(WISDOM_CHEST, player)
      end
    else
      htmltext = "30738-14.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local npcId = npc:id()
  local id = st:get_state()
  local cond = 0
  if id ~= 0 then
    cond = st:get_cond()
  end
  if npcId == ALCHEMIST_MATILD then
    if cond == 0 then
      if st:get_player():level() <= 29 then
        htmltext = "30738-21.htm"
        st:exit_quest(true)
      elseif q:get_quest_items_count(player, MATILDS_ORB_ID) == 0 then
        htmltext = "30738-01.htm"
      elseif q:get_quest_items_count(player, 3467) == 0 then
        -- A repeat after an abort starts from a new quest state, which has to be started before the condition applies.
        st:set_state(State.STARTED)
        st:set_cond(3)
        if q:get_quest_items_count(player, POTION_RECIPE_1_ID) == 0 then
          q:give_items(player, POTION_RECIPE_1_ID, 1)
        end
        if q:get_quest_items_count(player, POTION_RECIPE_2_ID) == 0 then
          q:give_items(player, POTION_RECIPE_2_ID, 1)
        end
        htmltext = "30738-12.htm"
      else
        htmltext = "30738-11.htm"
      end
    elseif cond == 1 and q:get_quest_items_count(player, ALCHEMY_TEXT_ID) == 1 then
      htmltext = "30738-04.htm"
    elseif cond == 2 then
      if q:get_quest_items_count(player, SECRET_BOOK_ID) == 1 and q:get_quest_items_count(player, ALCHEMY_TEXT_ID) == 1 then
        htmltext = "30738-05.htm"
      end
    elseif cond == 4 then
      if checkIngr(st, player) then
        htmltext = "30738-08.htm"
      else
        htmltext = "30738-07.htm"
      end
    end
  elseif npcId == TORAI then
    if q:get_quest_items_count(player, FORBIDDEN_LOVE_SCROLL_ID) >= 1 then
      q:take_items(player, FORBIDDEN_LOVE_SCROLL_ID, 1)
      q:give_adena(player, 500000, true)
      htmltext = "30557-01.htm"
    else
      htmltext = q:get_no_quest_msg(player)
    end
  elseif npcId == WISDOM_CHEST then
    -- L2Go: Mobius decays st.getPlayer().getTarget(), the NPC the player talks
    -- to; without player:target() (API gap) the talked NPC is removed directly.
    local dropChance = rnd(100)
    if dropChance < 20 then
      q:give_items(player, SPELLBOOK_ICEBOLT_ID, 1)
      q:give_items(player, SPELLBOOK_BATTLEHEAL_ID, 1)
      npc:delete()
      htmltext = "30743-06.htm"
    elseif dropChance < 30 then
      q:give_items(player, HEART_OF_PAAGRIO_ID, 1)
      npc:delete()
      htmltext = "30743-06.htm"
    else
      npc:delete()
      htmltext = "30743-0" .. (rnd(5) + 1) .. ".htm"
    end
  elseif npcId == RUPINA then
    if rnd(100) < DROP_CHANCE_NECKLACE_OF_GRACE_ID then
      q:give_items(player, NECKLACE_OF_GRACE_ID, 1)
    else
      local dropChance = rnd(100) + 1
      if dropChance <= 25 then
        q:give_items(player, DEMONS_TUNIC_FABRIC_ID, 1)
      elseif dropChance <= 50 then
        q:give_items(player, DEMONS_STOCKINGS_PATTERN_ID, 1)
      elseif dropChance <= 75 then
        q:give_items(player, DEMONS_BOOTS_FABRIC_ID, 1)
      else
        q:give_items(player, DEMONS_GLOVES_FABRIC_ID, 1)
      end
    end
    npc:delete() -- L2Go: Mobius decays the player's target (see the Wisdom Chest).
    htmltext = "30742-01.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = player:get_quest_state(q:name())
  if st == nil then
    return
  end
  local npcId = npc:id()
  local cond = st:get_cond()
  for _, element in ipairs(DROPLIST_COND) do
    if cond == element[1] and npcId == element[3] and (element[4] == 0 or q:get_quest_items_count(player, element[4]) > 0) then
      if element[6] == 0 then
        if rnd(100) < element[7] then
          q:give_items(player, element[5], element[8])
        end
      elseif rnd(100) < element[7] and q:get_quest_items_count(player, element[5]) < element[6] then
        q:give_items(player, element[5], element[8])
        if cond == 3 then
          checkIngr(st, player)
        end
        if element[2] ~= cond and element[2] ~= 0 then
          st:set_cond(element[2])
          st:set_state(State.STARTED)
        end
      end
    end
  end
  local dropChance = rnd(100) + 1
  if npcId == SUCCUBUS_OF_SEDUCTION and dropChance <= DROP_CHANCE_FORBIDDEN_LOVE_SCROLL_ID then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, FORBIDDEN_LOVE_SCROLL_ID, 1)
  elseif npcId == GRIMA and dropChance <= DROP_CHANCE_GOLD_BAR_ID then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, GOLD_BAR_ID, rnd(5) + 1)
  elseif npcId == SANCHES and rnd(100) < 50 then
    q:add_spawn(BONAPARTERIUS, player)
  elseif npcId == BONAPARTERIUS and rnd(100) < 50 then
    q:add_spawn(RAMSEBALIUS, player)
  elseif npcId == RAMSEBALIUS and rnd(100) < 50 then
    q:add_spawn(GREAT_DEMON_KING, player)
  elseif npcId == GREAT_DEMON_KING and rnd(100) < 50 then
    if dropChance <= 25 then
      q:give_items(player, DEMONS_BOOTS_ID, 1)
    elseif dropChance <= 50 then
      q:give_items(player, DEMONS_GLOVES_ID, 1)
    elseif dropChance <= 75 then
      q:give_items(player, DEMONS_STOCKINGS_ID, 1)
    else
      q:give_items(player, DEMONS_TUNIC_ID, 1)
    end
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)

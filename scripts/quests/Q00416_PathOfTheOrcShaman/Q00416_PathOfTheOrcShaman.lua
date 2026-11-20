-- Translated by questport from quests/Q00416_PathOfTheOrcShaman (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(416, "Q00416_PathOfTheOrcShaman", "Path to an Orc Shaman")

-- NPCs
local TATARU_ZU_HESTUI = 30585
local UMOS = 30502
local HESTUI_TOTEM_SPIRIT = 30592
local DUDA_MARA_TOTEM_SPIRIT = 30593
local MOIRA = 31979
local TOTEM_SPIRIT_OF_GANDI = 32057
local DEAD_LEOPARD_CARCASS = 32090
-- Monsters
local VENOMOUS_SPIDER = 20038
local ARACHNID_TRACKER = 20043
local GRIZZLY_BEAR = 20335
local SCARLET_SALAMANDER = 20415
local KASHA_BLADE_SPIDER = 20478
local KASHA_BEAR = 20479
local DURKA_SPIRIT = 27056
local BLACK_LEOPARD = 27319
-- Items
local FIRE_CHARM = 1616
local KASHA_BEAR_PELT = 1617
local KASHA_BLADE_SPIDER_HUSK = 1618
local FIERY_EGG_1 = 1619
local HESTUI_MASK = 1620
local FIERY_EGG_2 = 1621
local TOTEM_SPIRIT_CLAW = 1622
local TATARU_LETTER = 1623
local FLAME_CHARM = 1624
local GRIZZLY_BLOOD = 1625
local BLOOD_CAULDRON = 1626
local SPIRIT_NET = 1627
local BOUND_DURKA_SPIRIT = 1628
local DURKA_PARASITE = 1629
local TOTEM_SPIRIT_BLOOD = 1630
local MASK_OF_MEDIUM = 1631

q:register_quest_items(FIRE_CHARM, KASHA_BEAR_PELT, KASHA_BLADE_SPIDER_HUSK, FIERY_EGG_1, HESTUI_MASK, FIERY_EGG_2, TOTEM_SPIRIT_CLAW, TATARU_LETTER, FLAME_CHARM, GRIZZLY_BLOOD, BLOOD_CAULDRON, SPIRIT_NET, BOUND_DURKA_SPIRIT, DURKA_PARASITE, TOTEM_SPIRIT_BLOOD)
q:add_start_npc(TATARU_ZU_HESTUI)
q:add_talk_id(TATARU_ZU_HESTUI, UMOS, HESTUI_TOTEM_SPIRIT, DUDA_MARA_TOTEM_SPIRIT, MOIRA, TOTEM_SPIRIT_OF_GANDI, DEAD_LEOPARD_CARCASS)
q:add_kill_id(VENOMOUS_SPIDER, ARACHNID_TRACKER, GRIZZLY_BEAR, SCARLET_SALAMANDER, KASHA_BLADE_SPIDER, KASHA_BEAR, DURKA_SPIRIT, BLACK_LEOPARD)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30585-05.htm" then
    if player:class_id() ~= ClassId.ORC_MAGE then
      htmltext = player:class_id() == ClassId.ORC_SHAMAN and "30585-02a.htm" or "30585-02.htm"
    elseif player:level() < 19 then
      htmltext = "30585-03.htm"
    elseif q:has_quest_items(player, MASK_OF_MEDIUM) then
      htmltext = "30585-04.htm"
    end
  elseif event == "30585-06.htm" then
    st:start_quest()
    q:give_items(player, FIRE_CHARM, 1)
  elseif event == "30585-11b.htm" then
    st:set_cond(5, true)
    q:take_items(player, TOTEM_SPIRIT_CLAW, 1)
    q:give_items(player, TATARU_LETTER, 1)
  elseif event == "30585-11c.htm" then
    st:set_cond(12, true)
    q:take_items(player, TOTEM_SPIRIT_CLAW, 1)
  elseif event == "30592-03.htm" then
    st:set_cond(4, true)
    q:take_items(player, HESTUI_MASK, 1)
    q:take_items(player, FIERY_EGG_2, 1)
    q:give_items(player, TOTEM_SPIRIT_CLAW, 1)
  elseif event == "30593-03.htm" then
    st:set_cond(9, true)
    q:take_items(player, BLOOD_CAULDRON, 1)
    q:give_items(player, SPIRIT_NET, 1)
  elseif event == "32057-02.htm" then
    st:set_cond(14, true)
  elseif event == "32057-05.htm" then
    st:set_cond(21, true)
  elseif event == "32090-04.htm" then
    st:set_cond(18, true)
  elseif event == "30502-07.htm" then
    q:take_items(player, TOTEM_SPIRIT_BLOOD, -1)
    q:give_items(player, MASK_OF_MEDIUM, 1)
    q:add_exp_and_sp(player, 3200, 2600)
    player:social_action(3)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30585-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == TATARU_ZU_HESTUI then
      if cond == 1 then
        htmltext = "30585-07.htm"
      elseif cond == 2 then
        htmltext = "30585-08.htm"
        st:set_cond(3, true)
        q:take_items(player, FIERY_EGG_1, 1)
        q:take_items(player, FIRE_CHARM, 1)
        q:take_items(player, KASHA_BEAR_PELT, 1)
        q:take_items(player, KASHA_BLADE_SPIDER_HUSK, 1)
        q:give_items(player, FIERY_EGG_2, 1)
        q:give_items(player, HESTUI_MASK, 1)
      elseif cond == 3 then
        htmltext = "30585-09.htm"
      elseif cond == 4 then
        htmltext = "30585-10.htm"
      elseif cond == 5 then
        htmltext = "30585-12.htm"
      elseif cond > 5 and cond < 12 then
        htmltext = "30585-13.htm"
      elseif cond == 12 then
        htmltext = "30585-11c.htm"
      end
    elseif sw2 == HESTUI_TOTEM_SPIRIT then
      if cond == 3 then
        htmltext = "30592-01.htm"
      elseif cond == 4 then
        htmltext = "30592-04.htm"
      elseif cond > 4 and cond < 12 then
        htmltext = "30592-05.htm"
      end
    elseif sw2 == UMOS then
      if cond == 5 then
        htmltext = "30502-01.htm"
        st:set_cond(6, true)
        q:take_items(player, TATARU_LETTER, 1)
        q:give_items(player, FLAME_CHARM, 1)
      elseif cond == 6 then
        htmltext = "30502-02.htm"
      elseif cond == 7 then
        htmltext = "30502-03.htm"
        st:set_cond(8, true)
        q:take_items(player, FLAME_CHARM, 1)
        q:take_items(player, GRIZZLY_BLOOD, 3)
        q:give_items(player, BLOOD_CAULDRON, 1)
      elseif cond == 8 then
        htmltext = "30502-04.htm"
      elseif cond == 9 or cond == 10 then
        htmltext = "30502-05.htm"
      elseif cond == 11 then
        htmltext = "30502-06.htm"
      end
    elseif sw2 == MOIRA then
      if cond == 12 then
        htmltext = "31979-01.htm"
        st:set_cond(13, true)
      elseif cond > 12 and cond < 21 then
        htmltext = "31979-02.htm"
      elseif cond == 21 then
        htmltext = "31979-03.htm"
        q:give_items(player, MASK_OF_MEDIUM, 1)
        q:add_exp_and_sp(player, 3200, 3250)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == TOTEM_SPIRIT_OF_GANDI then
      if cond == 13 then
        htmltext = "32057-01.htm"
      elseif cond > 13 and cond < 20 then
        htmltext = "32057-03.htm"
      elseif cond == 20 then
        htmltext = "32057-04.htm"
      end
    elseif sw2 == DUDA_MARA_TOTEM_SPIRIT then
      if cond == 8 then
        htmltext = "30593-01.htm"
      elseif cond == 9 then
        htmltext = "30593-04.htm"
      elseif cond == 10 then
        htmltext = "30593-05.htm"
        st:set_cond(11, true)
        q:take_items(player, BOUND_DURKA_SPIRIT, 1)
        q:give_items(player, TOTEM_SPIRIT_BLOOD, 1)
      elseif cond == 11 then
        htmltext = "30593-06.htm"
      end
    elseif sw2 == DEAD_LEOPARD_CARCASS then
      if cond == 14 then
        htmltext = "32090-01a.htm"
      elseif cond == 15 then
        htmltext = "32090-01.htm"
        st:set_cond(16, true)
      elseif cond == 16 then
        htmltext = "32090-01b.htm"
      elseif cond == 17 then
        htmltext = "32090-02.htm"
      elseif cond == 18 then
        htmltext = "32090-05.htm"
      elseif cond == 19 then
        htmltext = "32090-06.htm"
        st:set_cond(20, true)
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
  local sw = npc:id()
  if sw == KASHA_BEAR then
    if st:is_cond(1) and not q:has_quest_items(player, KASHA_BEAR_PELT) then
      q:give_items(player, KASHA_BEAR_PELT, 1)
      if q:has_quest_items(player, FIERY_EGG_1, KASHA_BLADE_SPIDER_HUSK) then
        st:set_cond(2, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == KASHA_BLADE_SPIDER then
    if st:is_cond(1) and not q:has_quest_items(player, KASHA_BLADE_SPIDER_HUSK) then
      q:give_items(player, KASHA_BLADE_SPIDER_HUSK, 1)
      if q:has_quest_items(player, KASHA_BEAR_PELT, FIERY_EGG_1) then
        st:set_cond(2, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == SCARLET_SALAMANDER then
    if st:is_cond(1) and not q:has_quest_items(player, FIERY_EGG_1) then
      q:give_items(player, FIERY_EGG_1, 1)
      if q:has_quest_items(player, KASHA_BEAR_PELT, KASHA_BLADE_SPIDER_HUSK) then
        st:set_cond(2, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == GRIZZLY_BEAR then
    if st:is_cond(6) then
      if q:get_quest_items_count(player, GRIZZLY_BLOOD) < 3 then
        q:give_items(player, GRIZZLY_BLOOD, 1)
        if q:get_quest_items_count(player, GRIZZLY_BLOOD) >= 3 then
          st:set_cond(7, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == VENOMOUS_SPIDER or sw == ARACHNID_TRACKER then
    if st:is_cond(9) then
      local count = q:get_quest_items_count(player, DURKA_PARASITE)
      local rnd_ = rnd(10)
      if count == 5 and rnd_ < 1 or (count == 6 or count == 7) and rnd_ < 2 or count >= 8 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_BEFORE_BATTLE)
        q:take_items(player, DURKA_PARASITE, -1)
        q:add_spawn(DURKA_SPIRIT, npc, false, 120000)
      else
        q:give_items(player, DURKA_PARASITE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == DURKA_SPIRIT then
    if st:is_cond(9) then
      st:set_cond(10, true)
      q:take_items(player, DURKA_PARASITE, -1)
      q:take_items(player, SPIRIT_NET, 1)
      q:give_items(player, BOUND_DURKA_SPIRIT, 1)
    end
  elseif sw == BLACK_LEOPARD then
    if st:is_cond(14) then
      if st:get_int("leopard") > 0 then
        st:set_cond(15, true)
        if rnd(3) < 2 then
          npc:broadcast_say(ChatType.GENERAL, "My dear friend of " .. player:name() .. ", who has gone on ahead of me!")
        end
      else
        st:set("leopard", "1")
      end
    elseif st:is_cond(16) then
      st:set_cond(17, true)
      if rnd(3) < 2 then
        npc:broadcast_say(ChatType.GENERAL, "Listen to Tejakar Gandi, young Oroka! The spirit of the slain leopard is calling you, " .. player:name() .. "!")
      end
    elseif st:is_cond(18) then
      st:set_cond(19, true)
    end
  end
end)

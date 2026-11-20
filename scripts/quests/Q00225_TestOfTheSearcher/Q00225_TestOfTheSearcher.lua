-- Translated by questport from quests/Q00225_TestOfTheSearcher (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(225, "Q00225_TestOfTheSearcher", "Test of the Searcher")

-- NPCs
local ALEX = 30291
local TYRA = 30420
local TREE = 30627
local STRONG_WOODEN_CHEST = 30628
local LUTHER = 30690
local LEIRYNN = 30728
local BORYS = 30729
local JAX = 30730
-- Monsters
local HANGMAN_TREE = 20144
local ROAD_SCAVENGER = 20551
local GIANT_FUNGUS = 20555
local DELU_LIZARDMAN_SHAMAN = 20781
local DELU_CHIEF_KALKIS = 27093
local NEER_BODYGUARD = 27092
-- Items
local LUTHER_LETTER = 2784
local ALEX_WARRANT = 2785
local LEIRYNN_ORDER_1 = 2786
local DELU_TOTEM = 2787
local LEIRYNN_ORDER_2 = 2788
local CHIEF_KALKI_FANG = 2789
local LEIRYNN_REPORT = 2790
local STRANGE_MAP = 2791
local LAMBERT_MAP = 2792
local ALEX_LETTER = 2793
local ALEX_ORDER = 2794
local WINE_CATALOG = 2795
local TYRA_CONTRACT = 2796
local RED_SPORE_DUST = 2797
local MALRUKIAN_WINE = 2798
local OLD_ORDER = 2799
local JAX_DIARY = 2800
local TORN_MAP_PIECE_1 = 2801
local TORN_MAP_PIECE_2 = 2802
local SOLT_MAP = 2803
local MAKEL_MAP = 2804
local COMBINED_MAP = 2805
local RUSTED_KEY = 2806
local GOLD_BAR = 2807
local ALEX_RECOMMEND = 2808
-- Rewards
local MARK_OF_SEARCHER = 2809
local DIMENSIONAL_DIAMOND = 7562
-- Misc
local _strongWoodenChest -- Used to avoid to spawn multiple instances.

q:register_quest_items(LUTHER_LETTER, ALEX_WARRANT, LEIRYNN_ORDER_1, DELU_TOTEM, LEIRYNN_ORDER_2, CHIEF_KALKI_FANG, LEIRYNN_REPORT, STRANGE_MAP, LAMBERT_MAP, ALEX_LETTER, ALEX_ORDER, WINE_CATALOG, TYRA_CONTRACT, RED_SPORE_DUST, MALRUKIAN_WINE, OLD_ORDER, JAX_DIARY, TORN_MAP_PIECE_1, TORN_MAP_PIECE_2, SOLT_MAP, MAKEL_MAP, COMBINED_MAP, RUSTED_KEY, GOLD_BAR, ALEX_RECOMMEND)
q:add_start_npc(LUTHER)
q:add_talk_id(ALEX, TYRA, TREE, STRONG_WOODEN_CHEST, LUTHER, LEIRYNN, BORYS, JAX)
q:add_attack_id(DELU_LIZARDMAN_SHAMAN)
q:add_kill_id(HANGMAN_TREE, ROAD_SCAVENGER, GIANT_FUNGUS, DELU_LIZARDMAN_SHAMAN, DELU_CHIEF_KALKIS, NEER_BODYGUARD)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30690-05.htm" then
    st:start_quest()
    q:give_items(player, LUTHER_LETTER, 1)
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30690-05a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30291-07.htm" then
    st:set_cond(8, true)
    q:take_items(player, LEIRYNN_REPORT, 1)
    q:take_items(player, STRANGE_MAP, 1)
    q:give_items(player, ALEX_LETTER, 1)
    q:give_items(player, ALEX_ORDER, 1)
    q:give_items(player, LAMBERT_MAP, 1)
  elseif event == "30420-01a.htm" then
    st:set_cond(10, true)
    q:take_items(player, WINE_CATALOG, 1)
    q:give_items(player, TYRA_CONTRACT, 1)
  elseif event == "30730-01d.htm" then
    st:set_cond(14, true)
    q:take_items(player, OLD_ORDER, 1)
    q:give_items(player, JAX_DIARY, 1)
  elseif event == "30627-01a.htm" then
    if _strongWoodenChest == nil then
      if st:is_cond(16) then
        st:set_cond(17, true)
        q:give_items(player, RUSTED_KEY, 1)
      end
      _strongWoodenChest = J.oid(q:add_spawn(STRONG_WOODEN_CHEST, {x = 10098, y = 157287, z = -2406, heading = 0}, false, 0))
      q:start_quest_timer("chest_despawn", 300000, nil, player, false)
    end
  elseif event == "30628-01a.htm" then
    if not q:has_quest_items(player, RUSTED_KEY) then
      htmltext = "30628-02.htm"
    else
      st:set_cond(18, true)
      q:take_items(player, RUSTED_KEY, -1)
      q:give_items(player, GOLD_BAR, 20)
      J.obj(_strongWoodenChest):delete()
      _strongWoodenChest = nil
      q:cancel_quest_timer("chest_despawn", nil, player)
    end
  elseif event == "chest_despawn" then
    J.obj(_strongWoodenChest):delete()
    _strongWoodenChest = nil
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.ROGUE and player:class_id() ~= ClassId.ELVEN_SCOUT and player:class_id() ~= ClassId.ASSASSIN and player:class_id() ~= ClassId.SCAVENGER then
      htmltext = "30690-01.htm"
    elseif player:level() < 39 then
      htmltext = "30690-02.htm"
    else
      htmltext = player:class_id() == ClassId.SCAVENGER and "30690-04.htm" or "30690-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == LUTHER then
      if cond == 1 then
        htmltext = "30690-06.htm"
      elseif cond > 1 and cond < 19 then
        htmltext = "30690-07.htm"
      elseif cond == 19 then
        htmltext = "30690-08.htm"
        q:take_items(player, ALEX_RECOMMEND, 1)
        q:give_items(player, MARK_OF_SEARCHER, 1)
        q:add_exp_and_sp(player, 37831, 18750)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == ALEX then
      if cond == 1 then
        htmltext = "30291-01.htm"
        st:set_cond(2, true)
        q:take_items(player, LUTHER_LETTER, 1)
        q:give_items(player, ALEX_WARRANT, 1)
      elseif cond == 2 then
        htmltext = "30291-02.htm"
      elseif cond > 2 and cond < 7 then
        htmltext = "30291-03.htm"
      elseif cond == 7 then
        htmltext = "30291-04.htm"
      elseif cond > 7 and cond < 13 then
        htmltext = "30291-08.htm"
      elseif cond > 12 and cond < 16 then
        htmltext = "30291-09.htm"
      elseif cond > 15 and cond < 18 then
        htmltext = "30291-10.htm"
      elseif cond == 18 then
        htmltext = "30291-11.htm"
        st:set_cond(19, true)
        q:take_items(player, ALEX_ORDER, 1)
        q:take_items(player, COMBINED_MAP, 1)
        q:take_items(player, GOLD_BAR, -1)
        q:give_items(player, ALEX_RECOMMEND, 1)
      elseif cond == 19 then
        htmltext = "30291-12.htm"
      end
    elseif sw2 == LEIRYNN then
      if cond == 2 then
        htmltext = "30728-01.htm"
        st:set_cond(3, true)
        q:take_items(player, ALEX_WARRANT, 1)
        q:give_items(player, LEIRYNN_ORDER_1, 1)
      elseif cond == 3 then
        htmltext = "30728-02.htm"
      elseif cond == 4 then
        htmltext = "30728-03.htm"
        st:set_cond(5, true)
        q:take_items(player, DELU_TOTEM, -1)
        q:take_items(player, LEIRYNN_ORDER_1, 1)
        q:give_items(player, LEIRYNN_ORDER_2, 1)
      elseif cond == 5 then
        htmltext = "30728-04.htm"
      elseif cond == 6 then
        htmltext = "30728-05.htm"
        st:set_cond(7, true)
        q:take_items(player, CHIEF_KALKI_FANG, 1)
        q:take_items(player, LEIRYNN_ORDER_2, 1)
        q:give_items(player, LEIRYNN_REPORT, 1)
      elseif cond == 7 then
        htmltext = "30728-06.htm"
      elseif cond > 7 then
        htmltext = "30728-07.htm"
      end
    elseif sw2 == BORYS then
      if cond == 8 then
        htmltext = "30729-01.htm"
        st:set_cond(9, true)
        q:take_items(player, ALEX_LETTER, 1)
        q:give_items(player, WINE_CATALOG, 1)
      elseif cond > 8 and cond < 12 then
        htmltext = "30729-02.htm"
      elseif cond == 12 then
        htmltext = "30729-03.htm"
        st:set_cond(13, true)
        q:take_items(player, MALRUKIAN_WINE, 1)
        q:take_items(player, WINE_CATALOG, 1)
        q:give_items(player, OLD_ORDER, 1)
      elseif cond == 13 then
        htmltext = "30729-04.htm"
      elseif cond > 13 then
        htmltext = "30729-05.htm"
      end
    elseif sw2 == TYRA then
      if cond == 9 then
        htmltext = "30420-01.htm"
      elseif cond == 10 then
        htmltext = "30420-02.htm"
      elseif cond == 11 then
        htmltext = "30420-03.htm"
        st:set_cond(12, true)
        q:take_items(player, RED_SPORE_DUST, -1)
        q:take_items(player, TYRA_CONTRACT, 1)
        q:give_items(player, MALRUKIAN_WINE, 1)
      elseif cond > 11 then
        htmltext = "30420-04.htm"
      end
    elseif sw2 == JAX then
      if cond == 13 then
        htmltext = "30730-01.htm"
      elseif cond == 14 then
        htmltext = "30730-02.htm"
      elseif cond == 15 then
        htmltext = "30730-03.htm"
        st:set_cond(16, true)
        q:take_items(player, LAMBERT_MAP, 1)
        q:take_items(player, MAKEL_MAP, 1)
        q:take_items(player, JAX_DIARY, 1)
        q:take_items(player, SOLT_MAP, 1)
        q:give_items(player, COMBINED_MAP, 1)
      elseif cond > 15 then
        htmltext = "30730-04.htm"
      end
    elseif sw2 == TREE then
      if cond == 16 or cond == 17 then
        htmltext = "30627-01.htm"
      end
    elseif sw2 == STRONG_WOODEN_CHEST then
      if cond == 17 then
        htmltext = "30628-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_attack(function(npc, attacker, damage, isPet)
  local st = q:get_quest_state(attacker, false)
  if st == nil then
    return
  end
  if q:has_quest_items(attacker, LEIRYNN_ORDER_1) and not npc:is_script_value(1) then
    npc:set_script_value(1)
    q:add_spawn(NEER_BODYGUARD, npc, false, 200000)
  end
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == DELU_LIZARDMAN_SHAMAN then
    if st:is_cond(3) then
      q:give_items(player, DELU_TOTEM, 1)
      if q:get_quest_items_count(player, DELU_TOTEM) >= 10 then
        st:set_cond(4, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == DELU_CHIEF_KALKIS then
    if st:is_cond(5) then
      st:set_cond(6, true)
      q:give_items(player, CHIEF_KALKI_FANG, 1)
      q:give_items(player, STRANGE_MAP, 1)
    end
  elseif sw == GIANT_FUNGUS then
    if st:is_cond(10) then
      q:give_items(player, RED_SPORE_DUST, 1)
      if q:get_quest_items_count(player, RED_SPORE_DUST) >= 10 then
        st:set_cond(11, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == ROAD_SCAVENGER then
    if st:is_cond(14) and not q:has_quest_items(player, SOLT_MAP) and rnd.bool() then
      q:give_items(player, TORN_MAP_PIECE_1, 1)
      if q:get_quest_items_count(player, TORN_MAP_PIECE_1) >= 4 then
        q:take_items(player, TORN_MAP_PIECE_1, -1)
        q:give_items(player, SOLT_MAP, 1)
        if q:has_quest_items(player, MAKEL_MAP) then
          st:set_cond(15, true)
        end
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == HANGMAN_TREE then
    if st:is_cond(14) and not q:has_quest_items(player, MAKEL_MAP) and rnd.bool() then
      q:give_items(player, TORN_MAP_PIECE_2, 1)
      if q:get_quest_items_count(player, TORN_MAP_PIECE_2) >= 4 then
        q:take_items(player, TORN_MAP_PIECE_2, -1)
        q:give_items(player, MAKEL_MAP, 1)
        if q:has_quest_items(player, SOLT_MAP) then
          st:set_cond(15, true)
        end
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)

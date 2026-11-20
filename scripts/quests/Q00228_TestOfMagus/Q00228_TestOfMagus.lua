-- Translated by questport from quests/Q00228_TestOfMagus (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(228, "Q00228_TestOfMagus", "Test of Magus")

-- NPCs
local PARINA = 30391
local EARTH_SNAKE = 30409
local FLAME_SALAMANDER = 30411
local WIND_SYLPH = 30412
local WATER_UNDINE = 30413
local CASIAN = 30612
local RUKAL = 30629
-- Monsters
local HARPY = 20145
local MARSH_STAKATO = 20157
local WYRM = 20176
local MARSH_STAKATO_WORKER = 20230
local TOAD_LORD = 20231
local MARSH_STAKATO_SOLDIER = 20232
local MARSH_STAKATO_DRONE = 20234
local WINDSUS = 20553
local ENCHANTED_MONSTEREYE = 20564
local ENCHANTED_STONE_GOLEM = 20565
local ENCHANTED_IRON_GOLEM = 20566
local SINGING_FLOWER_PHANTASM = 27095
local SINGING_FLOWER_NIGHTMARE = 27096
local SINGING_FLOWER_DARKLING = 27097
local GHOST_FIRE = 27098
-- Items
local RUKAL_LETTER = 2841
local PARINA_LETTER = 2842
local LILAC_CHARM = 2843
local GOLDEN_SEED_1 = 2844
local GOLDEN_SEED_2 = 2845
local GOLDEN_SEED_3 = 2846
local SCORE_OF_ELEMENTS = 2847
local DAZZLING_DROP = 2848
local FLAME_CRYSTAL = 2849
local HARPY_FEATHER = 2850
local WYRM_WINGBONE = 2851
local WINDSUS_MANE = 2852
local EN_MONSTEREYE_SHELL = 2853
local EN_STONEGOLEM_POWDER = 2854
local EN_IRONGOLEM_SCRAP = 2855
local TONE_OF_WATER = 2856
local TONE_OF_FIRE = 2857
local TONE_OF_WIND = 2858
local TONE_OF_EARTH = 2859
local SALAMANDER_CHARM = 2860
local SYLPH_CHARM = 2861
local UNDINE_CHARM = 2862
local SERPENT_CHARM = 2863
-- Rewards
local MARK_OF_MAGUS = 2840
local DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(RUKAL_LETTER, PARINA_LETTER, LILAC_CHARM, GOLDEN_SEED_1, GOLDEN_SEED_2, GOLDEN_SEED_3, SCORE_OF_ELEMENTS, DAZZLING_DROP, FLAME_CRYSTAL, HARPY_FEATHER, WYRM_WINGBONE, WINDSUS_MANE, EN_MONSTEREYE_SHELL, EN_STONEGOLEM_POWDER, EN_IRONGOLEM_SCRAP, TONE_OF_WATER, TONE_OF_FIRE, TONE_OF_WIND, TONE_OF_EARTH, SALAMANDER_CHARM, SYLPH_CHARM, UNDINE_CHARM, SERPENT_CHARM)
q:add_start_npc(RUKAL)
q:add_talk_id(PARINA, EARTH_SNAKE, FLAME_SALAMANDER, WIND_SYLPH, WATER_UNDINE, CASIAN, RUKAL)
q:add_kill_id(HARPY, MARSH_STAKATO, WYRM, MARSH_STAKATO_WORKER, TOAD_LORD, MARSH_STAKATO_SOLDIER, MARSH_STAKATO_DRONE, WINDSUS, ENCHANTED_MONSTEREYE, ENCHANTED_STONE_GOLEM, ENCHANTED_IRON_GOLEM, SINGING_FLOWER_PHANTASM, SINGING_FLOWER_NIGHTMARE, SINGING_FLOWER_DARKLING, GHOST_FIRE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30629-04.htm" then
    st:start_quest()
    q:give_items(player, RUKAL_LETTER, 1)
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30629-04a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30629-10.htm" then
    st:set_cond(5, true)
    q:take_items(player, GOLDEN_SEED_1, 1)
    q:take_items(player, GOLDEN_SEED_2, 1)
    q:take_items(player, GOLDEN_SEED_3, 1)
    q:take_items(player, LILAC_CHARM, 1)
    q:give_items(player, SCORE_OF_ELEMENTS, 1)
  elseif event == "30391-02.htm" then
    st:set_cond(2, true)
    q:take_items(player, RUKAL_LETTER, 1)
    q:give_items(player, PARINA_LETTER, 1)
  elseif event == "30612-02.htm" then
    st:set_cond(3, true)
    q:take_items(player, PARINA_LETTER, 1)
    q:give_items(player, LILAC_CHARM, 1)
  elseif event == "30412-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, SYLPH_CHARM, 1)
  elseif event == "30409-03.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, SERPENT_CHARM, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.WIZARD and player:class_id() ~= ClassId.ELVEN_WIZARD and player:class_id() ~= ClassId.DARK_WIZARD then
      htmltext = "30629-01.htm"
    elseif player:level() < 39 then
      htmltext = "30629-02.htm"
    else
      htmltext = "30629-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == RUKAL then
      if cond == 1 then
        htmltext = "30629-05.htm"
      elseif cond == 2 then
        htmltext = "30629-06.htm"
      elseif cond == 3 then
        htmltext = "30629-07.htm"
      elseif cond == 4 then
        htmltext = "30629-08.htm"
      elseif cond == 5 then
        htmltext = "30629-11.htm"
      elseif cond == 6 then
        htmltext = "30629-12.htm"
        q:take_items(player, SCORE_OF_ELEMENTS, 1)
        q:take_items(player, TONE_OF_EARTH, 1)
        q:take_items(player, TONE_OF_FIRE, 1)
        q:take_items(player, TONE_OF_WATER, 1)
        q:take_items(player, TONE_OF_WIND, 1)
        q:give_items(player, MARK_OF_MAGUS, 1)
        q:add_exp_and_sp(player, 139039, 40000)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == PARINA then
      if cond == 1 then
        htmltext = "30391-01.htm"
      elseif cond == 2 then
        htmltext = "30391-03.htm"
      elseif cond == 3 or cond == 4 then
        htmltext = "30391-04.htm"
      elseif cond > 4 then
        htmltext = "30391-05.htm"
      end
    elseif sw2 == CASIAN then
      if cond == 2 then
        htmltext = "30612-01.htm"
      elseif cond == 3 then
        htmltext = "30612-03.htm"
      elseif cond == 4 then
        htmltext = "30612-04.htm"
      elseif cond > 4 then
        htmltext = "30612-05.htm"
      end
    elseif sw2 == WATER_UNDINE then
      if cond == 5 then
        if q:has_quest_items(player, UNDINE_CHARM) then
          if q:get_quest_items_count(player, DAZZLING_DROP) < 20 then
            htmltext = "30413-02.htm"
          else
            htmltext = "30413-03.htm"
            q:take_items(player, DAZZLING_DROP, 20)
            q:take_items(player, UNDINE_CHARM, 1)
            q:give_items(player, TONE_OF_WATER, 1)
            if q:has_quest_items(player, TONE_OF_FIRE, TONE_OF_WIND, TONE_OF_EARTH) then
              st:set_cond(6, true)
            else
              q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            end
          end
        elseif not q:has_quest_items(player, TONE_OF_WATER) then
          htmltext = "30413-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:give_items(player, UNDINE_CHARM, 1)
        else
          htmltext = "30413-04.htm"
        end
      elseif cond == 6 then
        htmltext = "30413-04.htm"
      end
    elseif sw2 == FLAME_SALAMANDER then
      if cond == 5 then
        if q:has_quest_items(player, SALAMANDER_CHARM) then
          if q:get_quest_items_count(player, FLAME_CRYSTAL) < 5 then
            htmltext = "30411-02.htm"
          else
            htmltext = "30411-03.htm"
            q:take_items(player, FLAME_CRYSTAL, 5)
            q:take_items(player, SALAMANDER_CHARM, 1)
            q:give_items(player, TONE_OF_FIRE, 1)
            if q:has_quest_items(player, TONE_OF_WATER, TONE_OF_WIND, TONE_OF_EARTH) then
              st:set_cond(6, true)
            else
              q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            end
          end
        elseif not q:has_quest_items(player, TONE_OF_FIRE) then
          htmltext = "30411-01.htm"
          q:give_items(player, SALAMANDER_CHARM, 1)
        else
          htmltext = "30411-04.htm"
        end
      elseif cond == 6 then
        htmltext = "30411-04.htm"
      end
    elseif sw2 == WIND_SYLPH then
      if cond == 5 then
        if q:has_quest_items(player, SYLPH_CHARM) then
          if q:get_quest_items_count(player, HARPY_FEATHER) + q:get_quest_items_count(player, WYRM_WINGBONE) + q:get_quest_items_count(player, WINDSUS_MANE) < 40 then
            htmltext = "30412-03.htm"
          else
            htmltext = "30412-04.htm"
            q:take_items(player, HARPY_FEATHER, 20)
            q:take_items(player, SYLPH_CHARM, 1)
            q:take_items(player, WINDSUS_MANE, 10)
            q:take_items(player, WYRM_WINGBONE, 10)
            q:give_items(player, TONE_OF_WIND, 1)
            if q:has_quest_items(player, TONE_OF_WATER, TONE_OF_FIRE, TONE_OF_EARTH) then
              st:set_cond(6, true)
            else
              q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            end
          end
        elseif not q:has_quest_items(player, TONE_OF_WIND) then
          htmltext = "30412-01.htm"
        else
          htmltext = "30412-05.htm"
        end
      elseif cond == 6 then
        htmltext = "30412-05.htm"
      end
    elseif sw2 == EARTH_SNAKE then
      if cond == 5 then
        if q:has_quest_items(player, SERPENT_CHARM) then
          if q:get_quest_items_count(player, EN_MONSTEREYE_SHELL) + q:get_quest_items_count(player, EN_STONEGOLEM_POWDER) + q:get_quest_items_count(player, EN_IRONGOLEM_SCRAP) < 30 then
            htmltext = "30409-04.htm"
          else
            htmltext = "30409-05.htm"
            q:take_items(player, EN_IRONGOLEM_SCRAP, 10)
            q:take_items(player, EN_MONSTEREYE_SHELL, 10)
            q:take_items(player, EN_STONEGOLEM_POWDER, 10)
            q:take_items(player, SERPENT_CHARM, 1)
            q:give_items(player, TONE_OF_EARTH, 1)
            if q:has_quest_items(player, TONE_OF_WATER, TONE_OF_FIRE, TONE_OF_WIND) then
              st:set_cond(6, true)
            else
              q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            end
          end
        elseif not q:has_quest_items(player, TONE_OF_EARTH) then
          htmltext = "30409-01.htm"
        else
          htmltext = "30409-06.htm"
        end
      elseif cond == 6 then
        htmltext = "30409-06.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local cond = st:get_cond()
  if cond == 3 then
    local sw = npc:id()
    if sw == SINGING_FLOWER_PHANTASM then
      if not q:has_quest_items(player, GOLDEN_SEED_1) then
        npc:broadcast_say(ChatType.GENERAL, "I am a tree of nothing... a tree... that knows where to return...")
        q:give_items(player, GOLDEN_SEED_1, 1)
        if q:has_quest_items(player, GOLDEN_SEED_2, GOLDEN_SEED_3) then
          st:set_cond(4, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == SINGING_FLOWER_NIGHTMARE then
      if not q:has_quest_items(player, GOLDEN_SEED_2) then
        npc:broadcast_say(ChatType.GENERAL, "I am a creature that shows the truth of the place deep in my heart...")
        q:give_items(player, GOLDEN_SEED_2, 1)
        if q:has_quest_items(player, GOLDEN_SEED_1, GOLDEN_SEED_3) then
          st:set_cond(4, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw == SINGING_FLOWER_DARKLING then
      if not q:has_quest_items(player, GOLDEN_SEED_3) then
        npc:broadcast_say(ChatType.GENERAL, "I am a mirror of darkness... a virtual image of darkness...")
        q:give_items(player, GOLDEN_SEED_3, 1)
        if q:has_quest_items(player, GOLDEN_SEED_1, GOLDEN_SEED_2) then
          st:set_cond(4, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif cond == 5 then
    local sw2 = npc:id()
    if sw2 == GHOST_FIRE then
      if q:has_quest_items(player, SALAMANDER_CHARM) and rnd(10) < 5 and q:get_quest_items_count(player, FLAME_CRYSTAL) < 5 then
        q:give_items(player, FLAME_CRYSTAL, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw2 == TOAD_LORD or sw2 == MARSH_STAKATO or sw2 == MARSH_STAKATO_WORKER then
      if q:has_quest_items(player, UNDINE_CHARM) and rnd(10) < 3 and q:get_quest_items_count(player, DAZZLING_DROP) < 20 then
        q:give_items(player, DAZZLING_DROP, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw2 == MARSH_STAKATO_SOLDIER then
      if q:has_quest_items(player, UNDINE_CHARM) and rnd(10) < 4 and q:get_quest_items_count(player, DAZZLING_DROP) < 20 then
        q:give_items(player, DAZZLING_DROP, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw2 == MARSH_STAKATO_DRONE then
      if q:has_quest_items(player, UNDINE_CHARM) and rnd(10) < 5 and q:get_quest_items_count(player, DAZZLING_DROP) < 20 then
        q:give_items(player, DAZZLING_DROP, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw2 == HARPY then
      if q:has_quest_items(player, SYLPH_CHARM) then
        if q:get_quest_items_count(player, HARPY_FEATHER) < 20 then
          q:give_items(player, HARPY_FEATHER, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    elseif sw2 == WYRM then
      if q:has_quest_items(player, SYLPH_CHARM) and rnd(10) < 5 and q:get_quest_items_count(player, WYRM_WINGBONE) < 10 then
        q:give_items(player, WYRM_WINGBONE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw2 == WINDSUS then
      if q:has_quest_items(player, SYLPH_CHARM) and rnd(10) < 5 and q:get_quest_items_count(player, WINDSUS_MANE) < 10 then
        q:give_items(player, WINDSUS_MANE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw2 == ENCHANTED_MONSTEREYE then
      if q:has_quest_items(player, SERPENT_CHARM) and q:get_quest_items_count(player, EN_MONSTEREYE_SHELL) < 10 then
        q:give_items(player, EN_MONSTEREYE_SHELL, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw2 == ENCHANTED_STONE_GOLEM then
      if q:has_quest_items(player, SERPENT_CHARM) and q:get_quest_items_count(player, EN_STONEGOLEM_POWDER) < 10 then
        q:give_items(player, EN_STONEGOLEM_POWDER, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw2 == ENCHANTED_IRON_GOLEM then
      if q:has_quest_items(player, SERPENT_CHARM) and q:get_quest_items_count(player, EN_IRONGOLEM_SCRAP) < 10 then
        q:give_items(player, EN_IRONGOLEM_SCRAP, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)

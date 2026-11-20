-- Translated by questport from quests/Q00233_TestOfTheWarSpirit (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(233, "Q00233_TestOfTheWarSpirit", "Test of the War Spirit")
local K = {}

K.VIVYAN = 30030
K.SARIEN = 30436
K.RACOY = 30507
K.SOMAK = 30510
K.MANAKIA = 30515
K.ORIM = 30630
K.ANCESTOR_MARTANKUS = 30649
K.PEKIRON = 30682
K.NOBLE_ANT = 20089
K.NOBLE_ANT_LEADER = 20090
K.MEDUSA = 20158
K.PORTA = 20213
K.EXCURO = 20214
K.MORDEO = 20215
K.LETO_LIZARDMAN_SHAMAN = 20581
K.LETO_LIZARDMAN_OVERLORD = 20582
K.TAMLIN_ORC = 20601
K.TAMLIN_ORC_ARCHER = 20602
K.STENOA_GORGON_QUEEN = 27108
K.VENDETTA_TOTEM = 2880
K.TAMLIN_ORC_HEAD = 2881
K.WARSPIRIT_TOTEM = 2882
K.ORIM_CONTRACT = 2883
K.PORTA_EYE = 2884
K.EXCURO_SCALE = 2885
K.MORDEO_TALON = 2886
K.BRAKI_REMAINS_1 = 2887
K.PEKIRON_TOTEM = 2888
K.TONAR_SKULL = 2889
K.TONAR_RIBBONE = 2890
K.TONAR_SPINE = 2891
K.TONAR_ARMBONE = 2892
K.TONAR_THIGHBONE = 2893
K.TONAR_REMAINS_1 = 2894
K.MANAKIA_TOTEM = 2895
K.HERMODT_SKULL = 2896
K.HERMODT_RIBBONE = 2897
K.HERMODT_SPINE = 2898
K.HERMODT_ARMBONE = 2899
K.HERMODT_THIGHBONE = 2900
K.HERMODT_REMAINS_1 = 2901
K.RACOY_TOTEM = 2902
K.VIVYAN_LETTER = 2903
K.INSECT_DIAGRAM_BOOK = 2904
K.KIRUNA_SKULL = 2905
K.KIRUNA_RIBBONE = 2906
K.KIRUNA_SPINE = 2907
K.KIRUNA_ARMBONE = 2908
K.KIRUNA_THIGHBONE = 2909
K.KIRUNA_REMAINS_1 = 2910
K.BRAKI_REMAINS_2 = 2911
K.TONAR_REMAINS_2 = 2912
K.HERMODT_REMAINS_2 = 2913
K.KIRUNA_REMAINS_2 = 2914
K.MARK_OF_WARSPIRIT = 2879
K.DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(K.VENDETTA_TOTEM, K.TAMLIN_ORC_HEAD, K.WARSPIRIT_TOTEM, K.ORIM_CONTRACT, K.PORTA_EYE, K.EXCURO_SCALE, K.MORDEO_TALON, K.BRAKI_REMAINS_1, K.PEKIRON_TOTEM, K.TONAR_SKULL, K.TONAR_RIBBONE, K.TONAR_SPINE, K.TONAR_ARMBONE, K.TONAR_THIGHBONE, K.TONAR_REMAINS_1, K.MANAKIA_TOTEM, K.HERMODT_SKULL, K.HERMODT_RIBBONE, K.HERMODT_SPINE, K.HERMODT_ARMBONE, K.HERMODT_THIGHBONE, K.HERMODT_REMAINS_1, K.RACOY_TOTEM, K.VIVYAN_LETTER, K.INSECT_DIAGRAM_BOOK, K.KIRUNA_SKULL, K.KIRUNA_RIBBONE, K.KIRUNA_SPINE, K.KIRUNA_ARMBONE, K.KIRUNA_THIGHBONE, K.KIRUNA_REMAINS_1, K.BRAKI_REMAINS_2, K.TONAR_REMAINS_2, K.HERMODT_REMAINS_2, K.KIRUNA_REMAINS_2)
q:add_start_npc(K.SOMAK)
q:add_talk_id(K.SOMAK, K.VIVYAN, K.SARIEN, K.RACOY, K.MANAKIA, K.ORIM, K.ANCESTOR_MARTANKUS, K.PEKIRON)
q:add_kill_id(K.NOBLE_ANT, K.NOBLE_ANT_LEADER, K.MEDUSA, K.PORTA, K.EXCURO, K.MORDEO, K.LETO_LIZARDMAN_SHAMAN, K.LETO_LIZARDMAN_OVERLORD, K.TAMLIN_ORC, K.TAMLIN_ORC_ARCHER, K.STENOA_GORGON_QUEEN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30510-05.htm" then
    st:start_quest()
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30510-05e.htm"
      q:give_items(player, K.DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30630-04.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.ORIM_CONTRACT, 1)
  elseif event == "30507-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.RACOY_TOTEM, 1)
  elseif event == "30030-04.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.VIVYAN_LETTER, 1)
  elseif event == "30682-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.PEKIRON_TOTEM, 1)
  elseif event == "30515-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.MANAKIA_TOTEM, 1)
  elseif event == "30649-03.htm" then
    q:take_items(player, K.TAMLIN_ORC_HEAD, -1)
    q:take_items(player, K.WARSPIRIT_TOTEM, -1)
    q:take_items(player, K.BRAKI_REMAINS_2, -1)
    q:take_items(player, K.HERMODT_REMAINS_2, -1)
    q:take_items(player, K.KIRUNA_REMAINS_2, -1)
    q:take_items(player, K.TONAR_REMAINS_2, -1)
    q:give_items(player, K.MARK_OF_WARSPIRIT, 1)
    q:add_exp_and_sp(player, 63483, 17500)
    player:social_action(3)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() == ClassId.ORC_SHAMAN then
      htmltext = player:level() < 39 and "30510-03.htm" or "30510-04.htm"
    else
      htmltext = player:race() == Race.ORC and "30510-02.htm" or "30510-01.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == K.SOMAK then
      if cond == 1 then
        htmltext = "30510-06.htm"
      elseif cond == 2 then
        htmltext = "30510-07.htm"
        st:set_cond(3, true)
        q:take_items(player, K.BRAKI_REMAINS_1, 1)
        q:take_items(player, K.HERMODT_REMAINS_1, 1)
        q:take_items(player, K.KIRUNA_REMAINS_1, 1)
        q:take_items(player, K.TONAR_REMAINS_1, 1)
        q:give_items(player, K.VENDETTA_TOTEM, 1)
      elseif cond == 3 then
        htmltext = "30510-08.htm"
      elseif cond == 4 then
        htmltext = "30510-09.htm"
        st:set_cond(5, true)
        q:take_items(player, K.VENDETTA_TOTEM, 1)
        q:give_items(player, K.BRAKI_REMAINS_2, 1)
        q:give_items(player, K.HERMODT_REMAINS_2, 1)
        q:give_items(player, K.KIRUNA_REMAINS_2, 1)
        q:give_items(player, K.TONAR_REMAINS_2, 1)
        q:give_items(player, K.WARSPIRIT_TOTEM, 1)
      elseif cond == 5 then
        htmltext = "30510-10.htm"
      end
    elseif sw2 == K.ORIM then
      if cond == 1 and not q:has_quest_items(player, K.BRAKI_REMAINS_1) then
        if not q:has_quest_items(player, K.ORIM_CONTRACT) then
          htmltext = "30630-01.htm"
        elseif q:get_quest_items_count(player, K.PORTA_EYE) + q:get_quest_items_count(player, K.EXCURO_SCALE) + q:get_quest_items_count(player, K.MORDEO_TALON) == 30 then
          htmltext = "30630-06.htm"
          q:take_items(player, K.EXCURO_SCALE, 10)
          q:take_items(player, K.MORDEO_TALON, 10)
          q:take_items(player, K.PORTA_EYE, 10)
          q:take_items(player, K.ORIM_CONTRACT, 1)
          q:give_items(player, K.BRAKI_REMAINS_1, 1)
          if q:has_quest_items(player, K.HERMODT_REMAINS_1, K.KIRUNA_REMAINS_1, K.TONAR_REMAINS_1) then
            st:set_cond(2, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        else
          htmltext = "30630-05.htm"
        end
      else
        htmltext = "30630-07.htm"
      end
    elseif sw2 == K.RACOY then
      if cond == 1 and not q:has_quest_items(player, K.KIRUNA_REMAINS_1) then
        if not q:has_quest_items(player, K.RACOY_TOTEM) then
          htmltext = "30507-01.htm"
        elseif q:has_quest_items(player, K.VIVYAN_LETTER) then
          htmltext = "30507-04.htm"
        elseif q:has_quest_items(player, K.INSECT_DIAGRAM_BOOK) then
          if q:has_quest_items(player, K.KIRUNA_ARMBONE, K.KIRUNA_RIBBONE, K.KIRUNA_SKULL, K.KIRUNA_SPINE, K.KIRUNA_THIGHBONE) then
            htmltext = "30507-06.htm"
            q:take_items(player, K.INSECT_DIAGRAM_BOOK, 1)
            q:take_items(player, K.RACOY_TOTEM, 1)
            q:take_items(player, K.KIRUNA_ARMBONE, 1)
            q:take_items(player, K.KIRUNA_RIBBONE, 1)
            q:take_items(player, K.KIRUNA_SKULL, 1)
            q:take_items(player, K.KIRUNA_SPINE, 1)
            q:take_items(player, K.KIRUNA_THIGHBONE, 1)
            q:give_items(player, K.KIRUNA_REMAINS_1, 1)
            if q:has_quest_items(player, K.BRAKI_REMAINS_1, K.HERMODT_REMAINS_1, K.TONAR_REMAINS_1) then
              st:set_cond(2, true)
            else
              q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            end
          else
            htmltext = "30507-05.htm"
          end
        else
          htmltext = "30507-03.htm"
        end
      else
        htmltext = "30507-07.htm"
      end
    elseif sw2 == K.VIVYAN then
      if cond == 1 and q:has_quest_items(player, K.RACOY_TOTEM) then
        if q:has_quest_items(player, K.VIVYAN_LETTER) then
          htmltext = "30030-05.htm"
        elseif q:has_quest_items(player, K.INSECT_DIAGRAM_BOOK) then
          htmltext = "30030-06.htm"
        else
          htmltext = "30030-01.htm"
        end
      else
        htmltext = "30030-07.htm"
      end
    elseif sw2 == K.SARIEN then
      if cond == 1 and q:has_quest_items(player, K.RACOY_TOTEM) then
        if q:has_quest_items(player, K.VIVYAN_LETTER) then
          htmltext = "30436-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.VIVYAN_LETTER, 1)
          q:give_items(player, K.INSECT_DIAGRAM_BOOK, 1)
        elseif q:has_quest_items(player, K.INSECT_DIAGRAM_BOOK) then
          htmltext = "30436-02.htm"
        end
      else
        htmltext = "30436-03.htm"
      end
    elseif sw2 == K.PEKIRON then
      if cond == 1 and not q:has_quest_items(player, K.TONAR_REMAINS_1) then
        if not q:has_quest_items(player, K.PEKIRON_TOTEM) then
          htmltext = "30682-01.htm"
        elseif q:has_quest_items(player, K.TONAR_ARMBONE, K.TONAR_RIBBONE, K.TONAR_SKULL, K.TONAR_SPINE, K.TONAR_THIGHBONE) then
          htmltext = "30682-04.htm"
          q:take_items(player, K.PEKIRON_TOTEM, 1)
          q:take_items(player, K.TONAR_ARMBONE, 1)
          q:take_items(player, K.TONAR_RIBBONE, 1)
          q:take_items(player, K.TONAR_SKULL, 1)
          q:take_items(player, K.TONAR_SPINE, 1)
          q:take_items(player, K.TONAR_THIGHBONE, 1)
          q:give_items(player, K.TONAR_REMAINS_1, 1)
          if q:has_quest_items(player, K.BRAKI_REMAINS_1, K.HERMODT_REMAINS_1, K.KIRUNA_REMAINS_1) then
            st:set_cond(2, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        else
          htmltext = "30682-03.htm"
        end
      else
        htmltext = "30682-05.htm"
      end
    elseif sw2 == K.MANAKIA then
      if cond == 1 and not q:has_quest_items(player, K.HERMODT_REMAINS_1) then
        if not q:has_quest_items(player, K.MANAKIA_TOTEM) then
          htmltext = "30515-01.htm"
        elseif q:has_quest_items(player, K.HERMODT_ARMBONE, K.HERMODT_RIBBONE, K.HERMODT_SKULL, K.HERMODT_SPINE, K.HERMODT_THIGHBONE) then
          htmltext = "30515-04.htm"
          q:take_items(player, K.MANAKIA_TOTEM, 1)
          q:take_items(player, K.HERMODT_ARMBONE, 1)
          q:take_items(player, K.HERMODT_RIBBONE, 1)
          q:take_items(player, K.HERMODT_SKULL, 1)
          q:take_items(player, K.HERMODT_SPINE, 1)
          q:take_items(player, K.HERMODT_THIGHBONE, 1)
          q:give_items(player, K.HERMODT_REMAINS_1, 1)
          if q:has_quest_items(player, K.BRAKI_REMAINS_1, K.KIRUNA_REMAINS_1, K.TONAR_REMAINS_1) then
            st:set_cond(2, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        else
          htmltext = "30515-03.htm"
        end
      else
        htmltext = "30515-05.htm"
      end
    elseif sw2 == K.ANCESTOR_MARTANKUS then
      if cond == 5 then
        htmltext = "30649-01.htm"
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
  local sw = npc:id()
  if sw == K.PORTA then
    if q:has_quest_items(player, K.ORIM_CONTRACT) and q:get_quest_items_count(player, K.PORTA_EYE) < 10 then
      q:give_items(player, K.PORTA_EYE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.EXCURO then
    if q:has_quest_items(player, K.ORIM_CONTRACT) and q:get_quest_items_count(player, K.EXCURO_SCALE) < 10 then
      q:give_items(player, K.EXCURO_SCALE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.MORDEO then
    if q:has_quest_items(player, K.ORIM_CONTRACT) and q:get_quest_items_count(player, K.MORDEO_TALON) < 10 then
      q:give_items(player, K.MORDEO_TALON, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.NOBLE_ANT or sw == K.NOBLE_ANT_LEADER then
    if q:has_quest_items(player, K.INSECT_DIAGRAM_BOOK) then
      local rndAnt = rnd(100)
      if rndAnt > 70 then
        if q:has_quest_items(player, K.KIRUNA_THIGHBONE) then
          if not q:has_quest_items(player, K.KIRUNA_ARMBONE) then
            q:give_items(player, K.KIRUNA_ARMBONE, 1)
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        elseif not q:has_quest_items(player, K.KIRUNA_THIGHBONE) then
          q:give_items(player, K.KIRUNA_THIGHBONE, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif rndAnt > 40 then
        if q:has_quest_items(player, K.KIRUNA_SPINE) then
          if not q:has_quest_items(player, K.KIRUNA_RIBBONE) then
            q:give_items(player, K.KIRUNA_RIBBONE, 1)
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        elseif not q:has_quest_items(player, K.KIRUNA_SPINE) then
          q:give_items(player, K.KIRUNA_SPINE, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif rndAnt > 10 and not q:has_quest_items(player, K.KIRUNA_SKULL) then
        q:give_items(player, K.KIRUNA_SKULL, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.LETO_LIZARDMAN_SHAMAN or sw == K.LETO_LIZARDMAN_OVERLORD then
    if q:has_quest_items(player, K.PEKIRON_TOTEM) and rnd.bool() then
      if not q:has_quest_items(player, K.TONAR_SKULL) then
        q:give_items(player, K.TONAR_SKULL, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif not q:has_quest_items(player, K.TONAR_RIBBONE) then
        q:give_items(player, K.TONAR_RIBBONE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif not q:has_quest_items(player, K.TONAR_SPINE) then
        q:give_items(player, K.TONAR_SPINE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif not q:has_quest_items(player, K.TONAR_ARMBONE) then
        q:give_items(player, K.TONAR_ARMBONE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif not q:has_quest_items(player, K.TONAR_THIGHBONE) then
        q:give_items(player, K.TONAR_THIGHBONE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.MEDUSA then
    if q:has_quest_items(player, K.MANAKIA_TOTEM) and rnd.bool() then
      if not q:has_quest_items(player, K.HERMODT_RIBBONE) then
        q:give_items(player, K.HERMODT_RIBBONE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif not q:has_quest_items(player, K.HERMODT_SPINE) then
        q:give_items(player, K.HERMODT_SPINE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif not q:has_quest_items(player, K.HERMODT_ARMBONE) then
        q:give_items(player, K.HERMODT_ARMBONE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif not q:has_quest_items(player, K.HERMODT_THIGHBONE) then
        q:give_items(player, K.HERMODT_THIGHBONE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.STENOA_GORGON_QUEEN then
    if q:has_quest_items(player, K.MANAKIA_TOTEM) and not q:has_quest_items(player, K.HERMODT_SKULL) then
      q:give_items(player, K.HERMODT_SKULL, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.TAMLIN_ORC or sw == K.TAMLIN_ORC_ARCHER then
    if q:has_quest_items(player, K.VENDETTA_TOTEM) and not st:is_cond(4) and rnd.bool() and q:get_quest_items_count(player, K.TAMLIN_ORC_HEAD) < 13 then
      q:give_items(player, K.TAMLIN_ORC_HEAD, 1)
      if q:get_quest_items_count(player, K.TAMLIN_ORC_HEAD) >= 13 then
        st:set_cond(4, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)

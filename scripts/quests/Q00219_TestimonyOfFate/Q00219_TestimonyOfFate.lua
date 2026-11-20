-- Translated by questport from quests/Q00219_TestimonyOfFate (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(219, "Q00219_TestimonyOfFate", "Testimony of Fate")
local K = {}

K.KAIRA = 30476
K.METHEUS = 30614
K.IXIA = 30463
K.ALDER_SPIRIT = 30613
K.ROA = 30114
K.NORMAN = 30210
K.THIFIELL = 30358
K.ARKENIA = 30419
K.BLOODY_PIXY = 31845
K.BLIGHT_TREANT = 31850
K.HANGMAN_TREE = 20144
K.MARSH_STAKATO = 20157
K.MEDUSA = 20158
K.TYRANT = 20192
K.TYRANT_KINGPIN = 20193
K.DEAD_SEEKER = 20202
K.MARSH_STAKATO_WORKER = 20230
K.MARSH_STAKATO_SOLDIER = 20232
K.MARSH_SPIDER = 20233
K.MARSH_STAKATO_DRONE = 20234
K.BREKA_ORC_OVERLORD = 20270
K.GRANDIS = 20554
K.LETO_LIZARDMAN_OVERLORD = 20582
K.KARUL_BUGBEAR = 20600
K.BLACK_WILLOW_LURKER = 27079
K.KAIRA_LETTER = 3173
K.METHEUS_FUNERAL_JAR = 3174
K.KASANDRA_REMAINS = 3175
K.HERBALISM_TEXTBOOK = 3176
K.IXIA_LIST = 3177
K.MEDUSA_ICHOR = 3178
K.MARSH_SPIDER_FLUIDS = 3179
K.DEAD_SEEKER_DUNG = 3180
K.TYRANT_BLOOD = 3181
K.NIGHTSHADE_ROOT = 3182
K.BELLADONNA = 3183
K.ALDER_SKULL_1 = 3184
K.ALDER_SKULL_2 = 3185
K.ALDER_RECEIPT = 3186
K.REVELATIONS_MANUSCRIPT = 3187
K.KAIRA_RECOMMENDATION = 3189
K.KAIRA_INSTRUCTIONS = 3188
K.PALUS_CHARM = 3190
K.THIFIELL_LETTER = 3191
K.ARKENIA_NOTE = 3192
K.PIXY_GARNET = 3193
K.GRANDIS_SKULL = 3194
K.KARUL_BUGBEAR_SKULL = 3195
K.BREKA_OVERLORD_SKULL = 3196
K.LETO_OVERLORD_SKULL = 3197
K.RED_FAIRY_DUST = 3198
K.BLIGHT_TREANT_SEED = 3199
K.BLACK_WILLOW_LEAF = 3200
K.BLIGHT_TREANT_SAP = 3201
K.ARKENIA_LETTER = 3202
K.MARK_OF_FATE = 3172
K.DIMENSIONAL_DIAMOND = 7562
K.CHANCES = {}

K.CHANCES[K.DEAD_SEEKER] = 500000
K.CHANCES[K.TYRANT] = 500000
K.CHANCES[K.TYRANT_KINGPIN] = 600000
K.CHANCES[K.MEDUSA] = 500000
K.CHANCES[K.MARSH_STAKATO] = 400000
K.CHANCES[K.MARSH_STAKATO_WORKER] = 300000
K.CHANCES[K.MARSH_STAKATO_SOLDIER] = 500000
K.CHANCES[K.MARSH_STAKATO_DRONE] = 600000
K.CHANCES[K.MARSH_SPIDER] = 500000

q:register_quest_items(K.KAIRA_LETTER, K.METHEUS_FUNERAL_JAR, K.KASANDRA_REMAINS, K.HERBALISM_TEXTBOOK, K.IXIA_LIST, K.MEDUSA_ICHOR, K.MARSH_SPIDER_FLUIDS, K.DEAD_SEEKER_DUNG, K.TYRANT_BLOOD, K.NIGHTSHADE_ROOT, K.BELLADONNA, K.ALDER_SKULL_1, K.ALDER_SKULL_2, K.ALDER_RECEIPT, K.REVELATIONS_MANUSCRIPT, K.KAIRA_RECOMMENDATION, K.KAIRA_INSTRUCTIONS, K.PALUS_CHARM, K.THIFIELL_LETTER, K.ARKENIA_NOTE, K.PIXY_GARNET, K.GRANDIS_SKULL, K.KARUL_BUGBEAR_SKULL, K.BREKA_OVERLORD_SKULL, K.LETO_OVERLORD_SKULL, K.RED_FAIRY_DUST, K.BLIGHT_TREANT_SEED, K.BLACK_WILLOW_LEAF, K.BLIGHT_TREANT_SAP, K.ARKENIA_LETTER)
q:add_start_npc(K.KAIRA)
q:add_talk_id(K.KAIRA, K.METHEUS, K.IXIA, K.ALDER_SPIRIT, K.ROA, K.NORMAN, K.THIFIELL, K.ARKENIA, K.BLOODY_PIXY, K.BLIGHT_TREANT)
q:add_kill_id(K.HANGMAN_TREE, K.MARSH_STAKATO, K.MEDUSA, K.TYRANT, K.TYRANT_KINGPIN, K.DEAD_SEEKER, K.MARSH_STAKATO_WORKER, K.MARSH_STAKATO_SOLDIER, K.MARSH_SPIDER, K.MARSH_STAKATO_DRONE, K.BREKA_ORC_OVERLORD, K.GRANDIS, K.LETO_LIZARDMAN_OVERLORD, K.KARUL_BUGBEAR, K.BLACK_WILLOW_LURKER)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30476-05.htm" then
    st:start_quest()
    q:give_items(player, K.KAIRA_LETTER, 1)
    if not player:var_bool("secondClassChange37", false) then
      htmltext = "30476-05a.htm"
      q:give_items(player, K.DIMENSIONAL_DIAMOND, quest.DF_REWARD_37[player:race()])
      player:set_var("secondClassChange37", true)
    end
  elseif event == "30114-04.htm" then
    st:set_cond(12, true)
    q:take_items(player, K.ALDER_SKULL_2, 1)
    q:give_items(player, K.ALDER_RECEIPT, 1)
  elseif event == "30476-12.htm" then
    if player:level() < 38 then
      htmltext = "30476-13.htm"
      st:set_cond(14, true)
      q:give_items(player, K.KAIRA_INSTRUCTIONS, 1)
    else
      st:set_cond(15, true)
      q:take_items(player, K.REVELATIONS_MANUSCRIPT, 1)
      q:give_items(player, K.KAIRA_RECOMMENDATION, 1)
    end
  elseif event == "30419-02.htm" then
    st:set_cond(17, true)
    q:take_items(player, K.THIFIELL_LETTER, 1)
    q:give_items(player, K.ARKENIA_NOTE, 1)
  elseif event == "31845-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.PIXY_GARNET, 1)
  elseif event == "31850-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.BLIGHT_TREANT_SEED, 1)
  elseif event == "30419-05.htm" then
    st:set_cond(18, true)
    q:take_items(player, K.ARKENIA_NOTE, 1)
    q:take_items(player, K.BLIGHT_TREANT_SAP, 1)
    q:take_items(player, K.RED_FAIRY_DUST, 1)
    q:give_items(player, K.ARKENIA_LETTER, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DARK_ELF then
      htmltext = "30476-02.htm"
    elseif player:level() < 37 or ClassId.level(player:class_id()) ~= 1 then
      htmltext = "30476-01.htm"
    else
      htmltext = "30476-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == K.KAIRA then
      if cond == 1 then
        htmltext = "30476-06.htm"
      elseif cond == 2 or cond == 3 then
        htmltext = "30476-07.htm"
      elseif cond > 3 and cond < 9 then
        htmltext = "30476-08.htm"
      elseif cond == 9 then
        htmltext = "30476-09.htm"
        st:set_cond(10, true)
        q:take_items(player, K.ALDER_SKULL_1, 1)
        q:add_spawn(K.ALDER_SPIRIT, player, false, 0)
      elseif cond > 9 and cond < 13 then
        htmltext = "30476-10.htm"
      elseif cond == 13 then
        htmltext = "30476-11.htm"
      elseif cond == 14 then
        if player:level() < 38 then
          htmltext = "30476-14.htm"
        else
          htmltext = "30476-12.htm"
          st:set_cond(15, true)
          q:take_items(player, K.KAIRA_INSTRUCTIONS, 1)
          q:take_items(player, K.REVELATIONS_MANUSCRIPT, 1)
          q:give_items(player, K.KAIRA_RECOMMENDATION, 1)
        end
      elseif cond == 15 then
        htmltext = "30476-16.htm"
      elseif cond > 15 then
        htmltext = "30476-17.htm"
      end
    elseif sw2 == K.METHEUS then
      if cond == 1 then
        htmltext = "30614-01.htm"
        st:set_cond(2, true)
        q:take_items(player, K.KAIRA_LETTER, 1)
        q:give_items(player, K.METHEUS_FUNERAL_JAR, 1)
      elseif cond == 2 then
        htmltext = "30614-02.htm"
      elseif cond == 3 then
        htmltext = "30614-03.htm"
        st:set_cond(4)
        st:set_cond(5, true)
        q:take_items(player, K.KASANDRA_REMAINS, 1)
        q:give_items(player, K.HERBALISM_TEXTBOOK, 1)
      elseif cond > 3 and cond < 8 then
        htmltext = "30614-04.htm"
      elseif cond == 8 then
        htmltext = "30614-05.htm"
        st:set_cond(9, true)
        q:take_items(player, K.BELLADONNA, 1)
        q:give_items(player, K.ALDER_SKULL_1, 1)
      elseif cond > 8 then
        htmltext = "30614-06.htm"
      end
    elseif sw2 == K.IXIA then
      if cond == 5 then
        htmltext = "30463-01.htm"
        st:set_cond(6, true)
        q:take_items(player, K.HERBALISM_TEXTBOOK, 1)
        q:give_items(player, K.IXIA_LIST, 1)
      elseif cond == 6 then
        htmltext = "30463-02.htm"
      elseif cond == 7 then
        htmltext = "30463-03.htm"
        st:set_cond(8, true)
        q:take_items(player, K.IXIA_LIST, 1)
        q:take_items(player, K.DEAD_SEEKER_DUNG, -1)
        q:take_items(player, K.MARSH_SPIDER_FLUIDS, -1)
        q:take_items(player, K.MEDUSA_ICHOR, -1)
        q:take_items(player, K.NIGHTSHADE_ROOT, -1)
        q:take_items(player, K.TYRANT_BLOOD, -1)
        q:give_items(player, K.BELLADONNA, 1)
      elseif cond == 8 then
        htmltext = "30463-04.htm"
      elseif cond > 8 then
        htmltext = "30463-05.htm"
      end
    elseif sw2 == K.ALDER_SPIRIT then
      if cond == 10 then
        htmltext = "30613-01.htm"
        st:set_cond(11, true)
        q:give_items(player, K.ALDER_SKULL_2, 1)
        npc:delete()
      end
    elseif sw2 == K.ROA then
      if cond == 11 then
        htmltext = "30114-01.htm"
      elseif cond == 12 then
        htmltext = "30114-05.htm"
      elseif cond > 12 then
        htmltext = "30114-06.htm"
      end
    elseif sw2 == K.NORMAN then
      if cond == 12 then
        htmltext = "30210-01.htm"
        st:set_cond(13, true)
        q:take_items(player, K.ALDER_RECEIPT, 1)
        q:give_items(player, K.REVELATIONS_MANUSCRIPT, 1)
      elseif cond > 12 then
        htmltext = "30210-02.htm"
      end
    elseif sw2 == K.THIFIELL then
      if cond == 15 then
        htmltext = "30358-01.htm"
        st:set_cond(16, true)
        q:take_items(player, K.KAIRA_RECOMMENDATION, 1)
        q:give_items(player, K.PALUS_CHARM, 1)
        q:give_items(player, K.THIFIELL_LETTER, 1)
      elseif cond == 16 then
        htmltext = "30358-02.htm"
      elseif cond == 17 then
        htmltext = "30358-03.htm"
      elseif cond == 18 then
        htmltext = "30358-04.htm"
        q:take_items(player, K.PALUS_CHARM, 1)
        q:take_items(player, K.ARKENIA_LETTER, 1)
        q:give_items(player, K.MARK_OF_FATE, 1)
        q:add_exp_and_sp(player, 68183, 1750)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == K.ARKENIA then
      if cond == 16 then
        htmltext = "30419-01.htm"
      elseif cond == 17 then
        htmltext = q:has_quest_items(player, K.BLIGHT_TREANT_SAP) and q:has_quest_items(player, K.RED_FAIRY_DUST) and "30419-04.htm" or "30419-03.htm"
      elseif cond == 18 then
        htmltext = "30419-06.htm"
      end
    elseif sw2 == K.BLOODY_PIXY then
      if cond == 17 then
        if q:has_quest_items(player, K.PIXY_GARNET) then
          if q:get_quest_items_count(player, K.GRANDIS_SKULL) >= 10 and q:get_quest_items_count(player, K.KARUL_BUGBEAR_SKULL) >= 10 and q:get_quest_items_count(player, K.BREKA_OVERLORD_SKULL) >= 10 and q:get_quest_items_count(player, K.LETO_OVERLORD_SKULL) >= 10 then
            htmltext = "31845-04.htm"
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            q:take_items(player, K.BREKA_OVERLORD_SKULL, -1)
            q:take_items(player, K.GRANDIS_SKULL, -1)
            q:take_items(player, K.KARUL_BUGBEAR_SKULL, -1)
            q:take_items(player, K.LETO_OVERLORD_SKULL, -1)
            q:take_items(player, K.PIXY_GARNET, 1)
            q:give_items(player, K.RED_FAIRY_DUST, 1)
          else
            htmltext = "31845-03.htm"
          end
        elseif q:has_quest_items(player, K.RED_FAIRY_DUST) then
          htmltext = "31845-05.htm"
        else
          htmltext = "31845-01.htm"
        end
      elseif cond == 18 then
        htmltext = "31845-05.htm"
      end
    elseif sw2 == K.BLIGHT_TREANT then
      if cond == 17 then
        if q:has_quest_items(player, K.BLIGHT_TREANT_SEED) then
          if q:has_quest_items(player, K.BLACK_WILLOW_LEAF) then
            htmltext = "31850-04.htm"
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            q:take_items(player, K.BLACK_WILLOW_LEAF, 1)
            q:take_items(player, K.BLIGHT_TREANT_SEED, 1)
            q:give_items(player, K.BLIGHT_TREANT_SAP, 1)
          else
            htmltext = "31850-03.htm"
          end
        elseif q:has_quest_items(player, K.BLIGHT_TREANT_SAP) then
          htmltext = "31850-05.htm"
        else
          htmltext = "31850-01.htm"
        end
      elseif cond == 18 then
        htmltext = "31850-05.htm"
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
  local npcId = npc:id()
  if npcId == K.HANGMAN_TREE then
    if st:is_cond(2) then
      st:set_cond(3, true)
      q:take_items(player, K.METHEUS_FUNERAL_JAR, 1)
      q:give_items(player, K.KASANDRA_REMAINS, 1)
    end
  elseif npcId == K.DEAD_SEEKER then
    if st:is_cond(6) and q:get_quest_items_count(player, K.DEAD_SEEKER_DUNG) < 10 and rnd(1000000) < K.CHANCES[npcId] then
      q:give_items(player, K.DEAD_SEEKER_DUNG, 1)
      if q:get_quest_items_count(player, K.DEAD_SEEKER_DUNG) >= 10 and q:get_quest_items_count(player, K.TYRANT_BLOOD) >= 10 and q:get_quest_items_count(player, K.MEDUSA_ICHOR) >= 10 and q:get_quest_items_count(player, K.NIGHTSHADE_ROOT) >= 10 and q:get_quest_items_count(player, K.MARSH_SPIDER_FLUIDS) >= 10 then
        st:set_cond(7, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == K.TYRANT or npcId == K.TYRANT_KINGPIN then
    if st:is_cond(6) and q:get_quest_items_count(player, K.TYRANT_BLOOD) < 10 and rnd(1000000) < K.CHANCES[npcId] then
      q:give_items(player, K.TYRANT_BLOOD, 1)
      if q:get_quest_items_count(player, K.TYRANT_BLOOD) >= 10 and q:get_quest_items_count(player, K.DEAD_SEEKER_DUNG) >= 10 and q:get_quest_items_count(player, K.MEDUSA_ICHOR) >= 10 and q:get_quest_items_count(player, K.NIGHTSHADE_ROOT) >= 10 and q:get_quest_items_count(player, K.MARSH_SPIDER_FLUIDS) >= 10 then
        st:set_cond(7, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == K.MEDUSA then
    if st:is_cond(6) and q:get_quest_items_count(player, K.MEDUSA_ICHOR) < 10 and rnd(1000000) < K.CHANCES[npcId] then
      q:give_items(player, K.MEDUSA_ICHOR, 1)
      if q:get_quest_items_count(player, K.MEDUSA_ICHOR) >= 10 and q:get_quest_items_count(player, K.DEAD_SEEKER_DUNG) >= 10 and q:get_quest_items_count(player, K.TYRANT_BLOOD) >= 10 and q:get_quest_items_count(player, K.NIGHTSHADE_ROOT) >= 10 and q:get_quest_items_count(player, K.MARSH_SPIDER_FLUIDS) >= 10 then
        st:set_cond(7, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == K.MARSH_STAKATO or npcId == K.MARSH_STAKATO_WORKER or npcId == K.MARSH_STAKATO_SOLDIER or npcId == K.MARSH_STAKATO_DRONE then
    if st:is_cond(6) and q:get_quest_items_count(player, K.NIGHTSHADE_ROOT) < 10 and rnd(1000000) < K.CHANCES[npcId] then
      q:give_items(player, K.NIGHTSHADE_ROOT, 1)
      if q:get_quest_items_count(player, K.NIGHTSHADE_ROOT) >= 10 and q:get_quest_items_count(player, K.DEAD_SEEKER_DUNG) >= 10 and q:get_quest_items_count(player, K.TYRANT_BLOOD) >= 10 and q:get_quest_items_count(player, K.MEDUSA_ICHOR) >= 10 and q:get_quest_items_count(player, K.MARSH_SPIDER_FLUIDS) >= 10 then
        st:set_cond(7, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == K.MARSH_SPIDER then
    if st:is_cond(6) and q:get_quest_items_count(player, K.MARSH_SPIDER_FLUIDS) < 10 and rnd(1000000) < K.CHANCES[npcId] then
      q:give_items(player, K.MARSH_SPIDER_FLUIDS, 1)
      if q:get_quest_items_count(player, K.MARSH_SPIDER_FLUIDS) >= 10 and q:get_quest_items_count(player, K.DEAD_SEEKER_DUNG) >= 10 and q:get_quest_items_count(player, K.TYRANT_BLOOD) >= 10 and q:get_quest_items_count(player, K.MEDUSA_ICHOR) >= 10 and q:get_quest_items_count(player, K.NIGHTSHADE_ROOT) >= 10 then
        st:set_cond(7, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif npcId == K.GRANDIS then
    if q:has_quest_items(player, K.PIXY_GARNET) and q:get_quest_items_count(player, K.GRANDIS_SKULL) < 10 then
      q:give_items(player, K.GRANDIS_SKULL, 1)
    end
  elseif npcId == K.LETO_LIZARDMAN_OVERLORD then
    if q:has_quest_items(player, K.PIXY_GARNET) and q:get_quest_items_count(player, K.LETO_OVERLORD_SKULL) < 10 then
      q:give_items(player, K.LETO_OVERLORD_SKULL, 1)
    end
  elseif npcId == K.BREKA_ORC_OVERLORD then
    if q:has_quest_items(player, K.PIXY_GARNET) and q:get_quest_items_count(player, K.BREKA_OVERLORD_SKULL) < 10 then
      q:give_items(player, K.BREKA_OVERLORD_SKULL, 1)
    end
  elseif npcId == K.KARUL_BUGBEAR then
    if q:has_quest_items(player, K.PIXY_GARNET) and q:get_quest_items_count(player, K.KARUL_BUGBEAR_SKULL) < 10 then
      q:give_items(player, K.KARUL_BUGBEAR_SKULL, 1)
    end
  elseif npcId == K.BLACK_WILLOW_LURKER then
    if q:has_quest_items(player, K.BLIGHT_TREANT_SEED) and q:get_quest_items_count(player, K.BLACK_WILLOW_LEAF) < 1 then
      q:give_items(player, K.BLACK_WILLOW_LEAF, 1)
    end
  end
end)

-- Translated by questport from quests/Q00230_TestOfTheSummoner (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(230, "Q00230_TestOfTheSummoner", "Test of the Summoner")
q:requires("pets")
local K = {}

K.LARA = 30063
K.GALATEA = 30634
K.ALMORS = 30635
K.CAMONIELL = 30636
K.BELTHUS = 30637
K.BASILLA = 30638
K.CELESTIEL = 30639
K.BRYNTHEA = 30640
K.NOBLE_ANT = 20089
K.NOBLE_ANT_LEADER = 20090
K.WYRM = 20176
K.TYRANT = 20192
K.TYRANT_KINGPIN = 20193
K.BREKA_ORC = 20267
K.BREKA_ORC_ARCHER = 20268
K.BREKA_ORC_SHAMAN = 20269
K.BREKA_ORC_OVERLORD = 20270
K.BREKA_ORC_WARRIOR = 20271
K.FETTERED_SOUL = 20552
K.WINDSUS = 20553
K.GIANT_FUNGUS = 20555
K.MANASHEN_GARGOYLE = 20563
K.LETO_LIZARDMAN = 20577
K.LETO_LIZARDMAN_ARCHER = 20578
K.LETO_LIZARDMAN_SOLDIER = 20579
K.LETO_LIZARDMAN_WARRIOR = 20580
K.LETO_LIZARDMAN_SHAMAN = 20581
K.LETO_LIZARDMAN_OVERLORD = 20582
K.KARUL_BUGBEAR = 20600
K.PAKO_THE_CAT = 27102
K.UNICORN_RACER = 27103
K.SHADOW_TUREN = 27104
K.MIMI_THE_CAT = 27105
K.UNICORN_PHANTASM = 27106
K.SILHOUETTE_TILFO = 27107
K.LETO_LIZARDMAN_AMULET = 3337
K.SAC_OF_REDSPORES = 3338
K.KARUL_BUGBEAR_TOTEM = 3339
K.SHARDS_OF_MANASHEN = 3340
K.BREKA_ORC_TOTEM = 3341
K.CRIMSON_BLOODSTONE = 3342
K.TALONS_OF_TYRANT = 3343
K.WINGS_OF_DRONEANT = 3344
K.TUSK_OF_WINDSUS = 3345
K.FANGS_OF_WYRM = 3346
K.LARA_LIST_1 = 3347
K.LARA_LIST_2 = 3348
K.LARA_LIST_3 = 3349
K.LARA_LIST_4 = 3350
K.LARA_LIST_5 = 3351
K.GALATEA_LETTER = 3352
K.BEGINNER_ARCANA = 3353
K.ALMORS_ARCANA = 3354
K.CAMONIELL_ARCANA = 3355
K.BELTHUS_ARCANA = 3356
K.BASILLIA_ARCANA = 3357
K.CELESTIEL_ARCANA = 3358
K.BRYNTHEA_ARCANA = 3359
K.CRYSTAL_OF_PROGRESS_1 = 3360
K.CRYSTAL_OF_INPROGRESS_1 = 3361
K.CRYSTAL_OF_FOUL_1 = 3362
K.CRYSTAL_OF_DEFEAT_1 = 3363
K.CRYSTAL_OF_VICTORY_1 = 3364
K.CRYSTAL_OF_PROGRESS_2 = 3365
K.CRYSTAL_OF_INPROGRESS_2 = 3366
K.CRYSTAL_OF_FOUL_2 = 3367
K.CRYSTAL_OF_DEFEAT_2 = 3368
K.CRYSTAL_OF_VICTORY_2 = 3369
K.CRYSTAL_OF_PROGRESS_3 = 3370
K.CRYSTAL_OF_INPROGRESS_3 = 3371
K.CRYSTAL_OF_FOUL_3 = 3372
K.CRYSTAL_OF_DEFEAT_3 = 3373
K.CRYSTAL_OF_VICTORY_3 = 3374
K.CRYSTAL_OF_PROGRESS_4 = 3375
K.CRYSTAL_OF_INPROGRESS_4 = 3376
K.CRYSTAL_OF_FOUL_4 = 3377
K.CRYSTAL_OF_DEFEAT_4 = 3378
K.CRYSTAL_OF_VICTORY_4 = 3379
K.CRYSTAL_OF_PROGRESS_5 = 3380
K.CRYSTAL_OF_INPROGRESS_5 = 3381
K.CRYSTAL_OF_FOUL_5 = 3382
K.CRYSTAL_OF_DEFEAT_5 = 3383
K.CRYSTAL_OF_VICTORY_5 = 3384
K.CRYSTAL_OF_PROGRESS_6 = 3385
K.CRYSTAL_OF_INPROGRESS_6 = 3386
K.CRYSTAL_OF_FOUL_6 = 3387
K.CRYSTAL_OF_DEFEAT_6 = 3388
K.CRYSTAL_OF_VICTORY_6 = 3389
K.MARK_OF_SUMMONER = 3336
K.DIMENSIONAL_DIAMOND = 7562
K.LARA_LISTS = {
  {K.LARA_LIST_1, K.SAC_OF_REDSPORES, K.LETO_LIZARDMAN_AMULET},
  {K.LARA_LIST_2, K.KARUL_BUGBEAR_TOTEM, K.SHARDS_OF_MANASHEN},
  {K.LARA_LIST_3, K.CRIMSON_BLOODSTONE, K.BREKA_ORC_TOTEM},
  {K.LARA_LIST_4, K.TUSK_OF_WINDSUS, K.TALONS_OF_TYRANT},
  {K.LARA_LIST_5, K.WINGS_OF_DRONEANT, K.FANGS_OF_WYRM},
}
K._duelsInProgress = {}

local ProgressDuelMob = {}
ProgressDuelMob.__index = ProgressDuelMob

function ProgressDuelMob.new(attacker, pet)
  local self = setmetatable({}, ProgressDuelMob)
  self._attacker = attacker
  self._pet = pet
  return self
end

function ProgressDuelMob:getAttacker()
  return self._attacker
end

function ProgressDuelMob:getPet()
  return self._pet
end

q:register_quest_items(K.LETO_LIZARDMAN_AMULET, K.SAC_OF_REDSPORES, K.KARUL_BUGBEAR_TOTEM, K.SHARDS_OF_MANASHEN, K.BREKA_ORC_TOTEM, K.CRIMSON_BLOODSTONE, K.TALONS_OF_TYRANT, K.WINGS_OF_DRONEANT, K.TUSK_OF_WINDSUS, K.FANGS_OF_WYRM, K.LARA_LIST_1, K.LARA_LIST_2, K.LARA_LIST_3, K.LARA_LIST_4, K.LARA_LIST_5, K.GALATEA_LETTER, K.BEGINNER_ARCANA, K.ALMORS_ARCANA, K.CAMONIELL_ARCANA, K.BELTHUS_ARCANA, K.BASILLIA_ARCANA, K.CELESTIEL_ARCANA, K.BRYNTHEA_ARCANA, K.CRYSTAL_OF_PROGRESS_1, K.CRYSTAL_OF_INPROGRESS_1, K.CRYSTAL_OF_FOUL_1, K.CRYSTAL_OF_DEFEAT_1, K.CRYSTAL_OF_VICTORY_1, K.CRYSTAL_OF_PROGRESS_2, K.CRYSTAL_OF_INPROGRESS_2, K.CRYSTAL_OF_FOUL_2, K.CRYSTAL_OF_DEFEAT_2, K.CRYSTAL_OF_VICTORY_2, K.CRYSTAL_OF_PROGRESS_3, K.CRYSTAL_OF_INPROGRESS_3, K.CRYSTAL_OF_FOUL_3, K.CRYSTAL_OF_DEFEAT_3, K.CRYSTAL_OF_VICTORY_3, K.CRYSTAL_OF_PROGRESS_4, K.CRYSTAL_OF_INPROGRESS_4, K.CRYSTAL_OF_FOUL_4, K.CRYSTAL_OF_DEFEAT_4, K.CRYSTAL_OF_VICTORY_4, K.CRYSTAL_OF_PROGRESS_5, K.CRYSTAL_OF_INPROGRESS_5, K.CRYSTAL_OF_FOUL_5, K.CRYSTAL_OF_DEFEAT_5, K.CRYSTAL_OF_VICTORY_5, K.CRYSTAL_OF_PROGRESS_6, K.CRYSTAL_OF_INPROGRESS_6, K.CRYSTAL_OF_FOUL_6, K.CRYSTAL_OF_DEFEAT_6, K.CRYSTAL_OF_VICTORY_6)
q:add_start_npc(K.GALATEA)
q:add_talk_id(K.GALATEA, K.ALMORS, K.CAMONIELL, K.BELTHUS, K.BASILLA, K.CELESTIEL, K.BRYNTHEA, K.LARA)
q:add_kill_id(K.NOBLE_ANT, K.NOBLE_ANT_LEADER, K.WYRM, K.TYRANT, K.TYRANT_KINGPIN, K.BREKA_ORC, K.BREKA_ORC_ARCHER, K.BREKA_ORC_SHAMAN, K.BREKA_ORC_OVERLORD, K.BREKA_ORC_WARRIOR, K.FETTERED_SOUL, K.WINDSUS, K.GIANT_FUNGUS, K.MANASHEN_GARGOYLE, K.LETO_LIZARDMAN, K.LETO_LIZARDMAN_ARCHER, K.LETO_LIZARDMAN_SOLDIER, K.LETO_LIZARDMAN_WARRIOR, K.LETO_LIZARDMAN_SHAMAN, K.LETO_LIZARDMAN_OVERLORD, K.KARUL_BUGBEAR, K.PAKO_THE_CAT, K.UNICORN_RACER, K.SHADOW_TUREN, K.MIMI_THE_CAT, K.UNICORN_PHANTASM, K.SILHOUETTE_TILFO)
q:add_attack_id(K.PAKO_THE_CAT, K.UNICORN_RACER, K.SHADOW_TUREN, K.MIMI_THE_CAT, K.UNICORN_PHANTASM, K.SILHOUETTE_TILFO)

q:on_event(function(event, npc, player)
  local st = player:get_quest_state(q:name())
  if st == nil then
    return nil
  end
  local htmltext = event
  if event == "30634-08.htm" then
    st:start_quest()
    st:set("Belthus", "1")
    st:set("Brynthea", "1")
    st:set("Celestiel", "1")
    st:set("Camoniell", "1")
    st:set("Basilla", "1")
    st:set("Almors", "1")
    q:give_items(player, K.GALATEA_LETTER, 1)
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30634-08a.htm"
      q:give_items(player, K.DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30063-02.htm" then
    st:set_cond(2, true)
    q:take_items(player, K.GALATEA_LETTER, 1)
    local random = rnd(5)
    q:give_items(player, K.LARA_LISTS[random + 1][1], 1)
    st:set("Lara", tostring(random + 1)) -- avoid 0
  elseif event == "30063-04.htm" then
    local random = rnd(5)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.LARA_LISTS[random + 1][1], 1)
    st:set("Lara", tostring(random + 1))
  elseif event == "30635-02.htm" then
    if q:has_quest_items(player, K.BEGINNER_ARCANA) then
      htmltext = "30635-03.htm"
    end
  elseif event == "30635-04.htm" then
    st:set("Almors", "2") -- set state ready to fight
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.CRYSTAL_OF_FOUL_1, -1) -- just in case he cheated or lost
    q:take_items(player, K.CRYSTAL_OF_DEFEAT_1, -1)
    q:take_items(player, K.BEGINNER_ARCANA, 1)
    q:give_items(player, K.CRYSTAL_OF_PROGRESS_1, 1) -- give Starting Crystal
    npc:set_target(player)
    npc:cast(skills.get(4126, 1))
  elseif event == "30636-02.htm" then
    if q:has_quest_items(player, K.BEGINNER_ARCANA) then
      htmltext = "30636-03.htm"
    end
  elseif event == "30636-04.htm" then
    st:set("Camoniell", "2")
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.CRYSTAL_OF_FOUL_2, -1)
    q:take_items(player, K.CRYSTAL_OF_DEFEAT_2, -1)
    q:take_items(player, K.BEGINNER_ARCANA, 1)
    q:give_items(player, K.CRYSTAL_OF_PROGRESS_2, 1)
    npc:set_target(player)
    npc:cast(skills.get(4126, 1))
  elseif event == "30637-02.htm" then
    if q:has_quest_items(player, K.BEGINNER_ARCANA) then
      htmltext = "30637-03.htm"
    end
  elseif event == "30637-04.htm" then
    st:set("Belthus", "2")
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.CRYSTAL_OF_FOUL_3, -1)
    q:take_items(player, K.CRYSTAL_OF_DEFEAT_3, -1)
    q:take_items(player, K.BEGINNER_ARCANA, 1)
    q:give_items(player, K.CRYSTAL_OF_PROGRESS_3, 1)
    npc:set_target(player)
    npc:cast(skills.get(4126, 1))
  elseif event == "30638-02.htm" then
    if q:has_quest_items(player, K.BEGINNER_ARCANA) then
      htmltext = "30638-03.htm"
    end
  elseif event == "30638-04.htm" then
    st:set("Basilla", "2")
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.CRYSTAL_OF_FOUL_4, -1)
    q:take_items(player, K.CRYSTAL_OF_DEFEAT_4, -1)
    q:take_items(player, K.BEGINNER_ARCANA, 1)
    q:give_items(player, K.CRYSTAL_OF_PROGRESS_4, 1)
    npc:set_target(player)
    npc:cast(skills.get(4126, 1))
  elseif event == "30639-02.htm" then
    if q:has_quest_items(player, K.BEGINNER_ARCANA) then
      htmltext = "30639-03.htm"
    end
  elseif event == "30639-04.htm" then
    st:set("Celestiel", "2")
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.CRYSTAL_OF_FOUL_5, -1)
    q:take_items(player, K.CRYSTAL_OF_DEFEAT_5, -1)
    q:take_items(player, K.BEGINNER_ARCANA, 1)
    q:give_items(player, K.CRYSTAL_OF_PROGRESS_5, 1)
    npc:set_target(player)
    npc:cast(skills.get(4126, 1))
  elseif event == "30640-02.htm" then
    if q:has_quest_items(player, K.BEGINNER_ARCANA) then
      htmltext = "30640-03.htm"
    end
  elseif event == "30640-04.htm" then
    st:set("Brynthea", "2")
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.CRYSTAL_OF_FOUL_6, -1)
    q:take_items(player, K.CRYSTAL_OF_DEFEAT_6, -1)
    q:take_items(player, K.BEGINNER_ARCANA, 1)
    q:give_items(player, K.CRYSTAL_OF_PROGRESS_6, 1)
    npc:set_target(player)
    npc:cast(skills.get(4126, 1))
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local cond = st:get_cond()
  local npcId = npc:id()
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.WIZARD and player:class_id() ~= ClassId.ELVEN_WIZARD and player:class_id() ~= ClassId.DARK_WIZARD then
      htmltext = "30634-01.htm"
    elseif player:level() < 39 then
      htmltext = "30634-02.htm"
    else
      htmltext = "30634-03.htm"
    end
  elseif sw == State.STARTED then
    if npcId == K.LARA then
      if cond == 1 then
        htmltext = "30063-01.htm"
      else
        if st:get_int("Lara") == 0 then
          htmltext = "30063-03.htm"
        else
          local laraPart = K.LARA_LISTS[st:get_int("Lara")]
          if q:get_quest_items_count(player, laraPart[2]) < 30 or q:get_quest_items_count(player, laraPart[3]) < 30 then
            htmltext = "30063-05.htm"
          else
            htmltext = "30063-06.htm"
            st:set_cond(3, true)
            st:unset("Lara")
            q:take_items(player, laraPart[1], 1)
            q:take_items(player, laraPart[2], -1)
            q:take_items(player, laraPart[3], -1)
            q:give_items(player, K.BEGINNER_ARCANA, 2)
          end
        end
      end
    elseif npcId == K.GALATEA then
      if cond == 1 then
        htmltext = "30634-09.htm"
      elseif cond == 2 or cond == 3 then
        htmltext = not q:has_quest_items(player, K.BEGINNER_ARCANA) and "30634-10.htm" or "30634-11.htm"
      elseif cond == 4 then
        htmltext = "30634-12.htm"
        q:take_items(player, K.BEGINNER_ARCANA, -1)
        q:take_items(player, K.ALMORS_ARCANA, -1)
        q:take_items(player, K.BASILLIA_ARCANA, -1)
        q:take_items(player, K.BELTHUS_ARCANA, -1)
        q:take_items(player, K.BRYNTHEA_ARCANA, -1)
        q:take_items(player, K.CAMONIELL_ARCANA, -1)
        q:take_items(player, K.CELESTIEL_ARCANA, -1)
        q:take_items(player, K.LARA_LIST_1, -1)
        q:take_items(player, K.LARA_LIST_2, -1)
        q:take_items(player, K.LARA_LIST_3, -1)
        q:take_items(player, K.LARA_LIST_4, -1)
        q:take_items(player, K.LARA_LIST_5, -1)
        q:give_items(player, K.MARK_OF_SUMMONER, 1)
        q:add_exp_and_sp(player, 148409, 30000)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif npcId == K.ALMORS then
      local almorsStat = st:get_int("Almors")
      if almorsStat == 1 then
        htmltext = "30635-01.htm"
      elseif almorsStat == 2 then
        htmltext = "30635-08.htm"
      elseif almorsStat == 3 then
        htmltext = "30635-09.htm"
      elseif almorsStat == 4 then
        htmltext = "30635-05.htm"
      elseif almorsStat == 5 then
        htmltext = "30635-06.htm"
      elseif almorsStat == 6 then
        htmltext = "30635-07.htm"
        st:set("Almors", "7")
        q:take_items(player, K.CRYSTAL_OF_VICTORY_1, -1)
        q:give_items(player, K.ALMORS_ARCANA, 1)
        if q:has_quest_items(player, K.CAMONIELL_ARCANA, K.BELTHUS_ARCANA, K.BASILLIA_ARCANA, K.CELESTIEL_ARCANA, K.BRYNTHEA_ARCANA) then
          st:set_cond(4, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif almorsStat == 7 then
        htmltext = "30635-10.htm"
      end
    elseif npcId == K.CAMONIELL then
      local camoniellStat = st:get_int("Camoniell")
      if camoniellStat == 1 then
        htmltext = "30636-01.htm"
      elseif camoniellStat == 2 then
        htmltext = "30636-08.htm"
      elseif camoniellStat == 3 then
        htmltext = "30636-09.htm"
      elseif camoniellStat == 4 then
        htmltext = "30636-05.htm"
      elseif camoniellStat == 5 then
        htmltext = "30636-06.htm"
      elseif camoniellStat == 6 then
        htmltext = "30636-07.htm"
        st:set("Camoniell", "7")
        q:take_items(player, K.CRYSTAL_OF_VICTORY_2, -1)
        q:give_items(player, K.CAMONIELL_ARCANA, 1)
        if q:has_quest_items(player, K.ALMORS_ARCANA, K.BELTHUS_ARCANA, K.BASILLIA_ARCANA, K.CELESTIEL_ARCANA, K.BRYNTHEA_ARCANA) then
          st:set_cond(4, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif camoniellStat == 7 then
        htmltext = "30636-10.htm"
      end
    elseif npcId == K.BELTHUS then
      local belthusStat = st:get_int("Belthus")
      if belthusStat == 1 then
        htmltext = "30637-01.htm"
      elseif belthusStat == 2 then
        htmltext = "30637-08.htm"
      elseif belthusStat == 3 then
        htmltext = "30637-09.htm"
      elseif belthusStat == 4 then
        htmltext = "30637-05.htm"
      elseif belthusStat == 5 then
        htmltext = "30637-06.htm"
      elseif belthusStat == 6 then
        htmltext = "30637-07.htm"
        st:set("Belthus", "7")
        q:take_items(player, K.CRYSTAL_OF_VICTORY_3, -1)
        q:give_items(player, K.BELTHUS_ARCANA, 1)
        if q:has_quest_items(player, K.ALMORS_ARCANA, K.CAMONIELL_ARCANA, K.BASILLIA_ARCANA, K.CELESTIEL_ARCANA, K.BRYNTHEA_ARCANA) then
          st:set_cond(4, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif belthusStat == 7 then
        htmltext = "30637-10.htm"
      end
    elseif npcId == K.BASILLA then
      local basillaStat = st:get_int("Basilla")
      if basillaStat == 1 then
        htmltext = "30638-01.htm"
      elseif basillaStat == 2 then
        htmltext = "30638-08.htm"
      elseif basillaStat == 3 then
        htmltext = "30638-09.htm"
      elseif basillaStat == 4 then
        htmltext = "30638-05.htm"
      elseif basillaStat == 5 then
        htmltext = "30638-06.htm"
      elseif basillaStat == 6 then
        htmltext = "30638-07.htm"
        st:set("Basilla", "7")
        q:take_items(player, K.CRYSTAL_OF_VICTORY_4, -1)
        q:give_items(player, K.BASILLIA_ARCANA, 1)
        if q:has_quest_items(player, K.ALMORS_ARCANA, K.CAMONIELL_ARCANA, K.BELTHUS_ARCANA, K.CELESTIEL_ARCANA, K.BRYNTHEA_ARCANA) then
          st:set_cond(4, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif basillaStat == 7 then
        htmltext = "30638-10.htm"
      end
    elseif npcId == K.CELESTIEL then
      local celestielStat = st:get_int("Celestiel")
      if celestielStat == 1 then
        htmltext = "30639-01.htm"
      elseif celestielStat == 2 then
        htmltext = "30639-08.htm"
      elseif celestielStat == 3 then
        htmltext = "30639-09.htm"
      elseif celestielStat == 4 then
        htmltext = "30639-05.htm"
      elseif celestielStat == 5 then
        htmltext = "30639-06.htm"
      elseif celestielStat == 6 then
        htmltext = "30639-07.htm"
        st:set("Celestiel", "7")
        q:take_items(player, K.CRYSTAL_OF_VICTORY_5, -1)
        q:give_items(player, K.CELESTIEL_ARCANA, 1)
        if q:has_quest_items(player, K.ALMORS_ARCANA, K.CAMONIELL_ARCANA, K.BELTHUS_ARCANA, K.BASILLIA_ARCANA, K.BRYNTHEA_ARCANA) then
          st:set_cond(4, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif celestielStat == 7 then
        htmltext = "30639-10.htm"
      end
    elseif npcId == K.BRYNTHEA then
      local bryntheaStat = st:get_int("Brynthea")
      if bryntheaStat == 1 then
        htmltext = "30640-01.htm"
      elseif bryntheaStat == 2 then
        htmltext = "30640-08.htm"
      elseif bryntheaStat == 3 then
        htmltext = "30640-09.htm"
      elseif bryntheaStat == 4 then
        htmltext = "30640-05.htm"
      elseif bryntheaStat == 5 then
        htmltext = "30640-06.htm"
      elseif bryntheaStat == 6 then
        htmltext = "30640-07.htm"
        st:set("Brynthea", "7")
        q:take_items(player, K.CRYSTAL_OF_VICTORY_6, -1)
        q:give_items(player, K.BRYNTHEA_ARCANA, 1)
        if q:has_quest_items(player, K.ALMORS_ARCANA, K.CAMONIELL_ARCANA, K.BELTHUS_ARCANA, K.BASILLIA_ARCANA, K.CELESTIEL_ARCANA) then
          st:set_cond(4, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif bryntheaStat == 7 then
        htmltext = "30640-10.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_death(function(killer, victim, st)
  if st == nil then
    return
  end
  if not J.instance_of(killer, "Attackable") then
    return
  end
  local sw = killer:id()
  if sw == K.PAKO_THE_CAT then
    if st:get_int("Almors") == 3 then
      st:set("Almors", "4")
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(killer, K.CRYSTAL_OF_DEFEAT_1, 1)
    end
  elseif sw == K.UNICORN_RACER then
    if st:get_int("Camoniell") == 3 then
      st:set("Camoniell", "4")
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(killer, K.CRYSTAL_OF_DEFEAT_2, 1)
    end
  elseif sw == K.SHADOW_TUREN then
    if st:get_int("Belthus") == 3 then
      st:set("Belthus", "4")
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(killer, K.CRYSTAL_OF_DEFEAT_3, 1)
    end
  elseif sw == K.MIMI_THE_CAT then
    if st:get_int("Basilla") == 3 then
      st:set("Basilla", "4")
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(killer, K.CRYSTAL_OF_DEFEAT_4, 1)
    end
  elseif sw == K.UNICORN_PHANTASM then
    if st:get_int("Celestiel") == 3 then
      st:set("Celestiel", "4")
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(killer, K.CRYSTAL_OF_DEFEAT_5, 1)
    end
  elseif sw == K.SILHOUETTE_TILFO then
    if st:get_int("Brynthea") == 3 then
      st:set("Brynthea", "4")
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(killer, K.CRYSTAL_OF_DEFEAT_6, 1)
    end
  end
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local npcId = npc:id()
  if npcId == K.GIANT_FUNGUS then
    if st:get_int("Lara") == 1 and rnd(100) < 80 and q:get_quest_items_count(player, K.SAC_OF_REDSPORES) < 30 then
      q:give_items(player, K.SAC_OF_REDSPORES, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.LETO_LIZARDMAN or npcId == K.LETO_LIZARDMAN_ARCHER then
    if st:get_int("Lara") == 1 and rnd(100) < 25 and q:get_quest_items_count(player, K.LETO_LIZARDMAN_AMULET) < 30 then
      q:give_items(player, K.LETO_LIZARDMAN_AMULET, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.LETO_LIZARDMAN_SOLDIER or npcId == K.LETO_LIZARDMAN_WARRIOR then
    if st:get_int("Lara") == 1 and rnd(100) < 50 and q:get_quest_items_count(player, K.LETO_LIZARDMAN_AMULET) < 30 then
      q:give_items(player, K.LETO_LIZARDMAN_AMULET, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.LETO_LIZARDMAN_SHAMAN or npcId == K.LETO_LIZARDMAN_OVERLORD then
    if st:get_int("Lara") == 1 and rnd(100) < 75 and q:get_quest_items_count(player, K.LETO_LIZARDMAN_AMULET) < 30 then
      q:give_items(player, K.LETO_LIZARDMAN_AMULET, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.MANASHEN_GARGOYLE then
    if st:get_int("Lara") == 2 and rnd(100) < 80 and q:get_quest_items_count(player, K.SHARDS_OF_MANASHEN) < 30 then
      q:give_items(player, K.SHARDS_OF_MANASHEN, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.KARUL_BUGBEAR then
    if st:get_int("Lara") == 2 and rnd(100) < 80 and q:get_quest_items_count(player, K.KARUL_BUGBEAR_TOTEM) < 30 then
      q:give_items(player, K.KARUL_BUGBEAR_TOTEM, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.BREKA_ORC or npcId == K.BREKA_ORC_ARCHER or npcId == K.BREKA_ORC_WARRIOR then
    if st:get_int("Lara") == 3 and rnd(100) < 25 and q:get_quest_items_count(player, K.BREKA_ORC_TOTEM) < 30 then
      q:give_items(player, K.BREKA_ORC_TOTEM, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.BREKA_ORC_SHAMAN or npcId == K.BREKA_ORC_OVERLORD then
    if st:get_int("Lara") == 3 and rnd(100) < 50 and q:get_quest_items_count(player, K.BREKA_ORC_TOTEM) < 30 then
      q:give_items(player, K.BREKA_ORC_TOTEM, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.FETTERED_SOUL then
    if st:get_int("Lara") == 3 and rnd(100) < 60 and q:get_quest_items_count(player, K.CRIMSON_BLOODSTONE) < 30 then
      q:give_items(player, K.CRIMSON_BLOODSTONE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.WINDSUS then
    if st:get_int("Lara") == 4 and rnd(100) < 70 and q:get_quest_items_count(player, K.TUSK_OF_WINDSUS) < 30 then
      q:give_items(player, K.TUSK_OF_WINDSUS, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.TYRANT or npcId == K.TYRANT_KINGPIN then
    if st:get_int("Lara") == 4 and rnd(100) < 50 and q:get_quest_items_count(player, K.TALONS_OF_TYRANT) < 30 then
      q:give_items(player, K.TALONS_OF_TYRANT, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.NOBLE_ANT or npcId == K.NOBLE_ANT_LEADER then
    if st:get_int("Lara") == 5 and rnd(100) < 60 and q:get_quest_items_count(player, K.WINGS_OF_DRONEANT) < 30 then
      q:give_items(player, K.WINGS_OF_DRONEANT, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.WYRM then
    if st:get_int("Lara") == 5 and rnd(100) < 50 and q:get_quest_items_count(player, K.FANGS_OF_WYRM) < 30 then
      q:give_items(player, K.FANGS_OF_WYRM, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif npcId == K.PAKO_THE_CAT then
    if st:get_int("Almors") == 3 and K._duelsInProgress[npcId] ~= nil then
      st:set("Almors", "6")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:take_items(player, K.CRYSTAL_OF_INPROGRESS_1, -1)
      q:give_items(player, K.CRYSTAL_OF_VICTORY_1, 1)
      npc:broadcast_say(ChatType.GENERAL, "I'm sorry, Lord!")
      st:add_notify_of_death(st:get_player())
      K._duelsInProgress[npcId] = nil
    end
  elseif npcId == K.UNICORN_RACER then
    if st:get_int("Camoniell") == 3 and K._duelsInProgress[npcId] ~= nil then
      st:set("Camoniell", "6")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:take_items(player, K.CRYSTAL_OF_INPROGRESS_2, -1)
      q:give_items(player, K.CRYSTAL_OF_VICTORY_2, 1)
      npc:broadcast_say(ChatType.GENERAL, "I LOSE")
      st:add_notify_of_death(st:get_player())
      K._duelsInProgress[npcId] = nil
    end
  elseif npcId == K.SHADOW_TUREN then
    if st:get_int("Belthus") == 3 and K._duelsInProgress[npcId] ~= nil then
      st:set("Belthus", "6")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:take_items(player, K.CRYSTAL_OF_INPROGRESS_3, -1)
      q:give_items(player, K.CRYSTAL_OF_VICTORY_3, 1)
      npc:broadcast_say(ChatType.GENERAL, "Ugh! I lost...!")
      st:add_notify_of_death(st:get_player())
      K._duelsInProgress[npcId] = nil
    end
  elseif npcId == K.MIMI_THE_CAT then
    if st:get_int("Basilla") == 3 and K._duelsInProgress[npcId] ~= nil then
      st:set("Basilla", "6")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:take_items(player, K.CRYSTAL_OF_INPROGRESS_4, -1)
      q:give_items(player, K.CRYSTAL_OF_VICTORY_4, 1)
      npc:broadcast_say(ChatType.GENERAL, "Lost! Sorry, Lord!")
      st:add_notify_of_death(st:get_player())
      K._duelsInProgress[npcId] = nil
    end
  elseif npcId == K.UNICORN_PHANTASM then
    if st:get_int("Celestiel") == 3 and K._duelsInProgress[npcId] ~= nil then
      st:set("Celestiel", "6")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:take_items(player, K.CRYSTAL_OF_INPROGRESS_5, -1)
      q:give_items(player, K.CRYSTAL_OF_VICTORY_5, 1)
      npc:broadcast_say(ChatType.GENERAL, "I LOSE")
      st:add_notify_of_death(st:get_player())
      K._duelsInProgress[npcId] = nil
    end
  elseif npcId == K.SILHOUETTE_TILFO then
    if st:get_int("Brynthea") == 3 and K._duelsInProgress[npcId] ~= nil then
      st:set("Brynthea", "6")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:take_items(player, K.CRYSTAL_OF_INPROGRESS_6, -1)
      q:give_items(player, K.CRYSTAL_OF_VICTORY_6, 1)
      npc:broadcast_say(ChatType.GENERAL, "Ugh! Can this be happening?!")
      st:add_notify_of_death(st:get_player())
      K._duelsInProgress[npcId] = nil
    end
  end
end)

q:on_attack(function(npc, player, damage, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  st:add_notify_of_death(npc)
  local npcId = npc:id()
  if npcId == K.PAKO_THE_CAT then
    if st:get_int("Almors") == 2 and isPet and npc:hp() + damage >= npc:max_hp() then
      st:set("Almors", "3")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, K.CRYSTAL_OF_PROGRESS_1, -1)
      q:give_items(player, K.CRYSTAL_OF_INPROGRESS_1, 1)
      npc:broadcast_say(ChatType.GENERAL, "Whhiisshh!")
      K._duelsInProgress[npcId] = ProgressDuelMob.new(player, player:summon())
    elseif st:get_int("Almors") == 3 and K._duelsInProgress[npcId] ~= nil then
      local duel = K._duelsInProgress[npcId]
      -- check if the attacker is the same pet as the one that attacked before.
      if not isPet or player:summon() ~= duel:getPet() then
        local foulPlayer = duel:getAttacker()
        if foulPlayer ~= nil then
          st = foulPlayer:get_quest_state(q:name())
          if st ~= nil then
            st:set("Almors", "5")
            q:take_items(player, K.CRYSTAL_OF_PROGRESS_1, -1)
            q:take_items(player, K.CRYSTAL_OF_INPROGRESS_1, -1)
            q:give_items(player, K.CRYSTAL_OF_FOUL_1, 1)
            st:add_notify_of_death(st:get_player())
            npc:broadcast_say(ChatType.GENERAL, "Rule violation!")
            npc:die(npc)
          end
        end
      end
    end
  elseif npcId == K.UNICORN_RACER then
    if st:get_int("Camoniell") == 2 and isPet and npc:hp() + damage >= npc:max_hp() then
      st:set("Camoniell", "3")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, K.CRYSTAL_OF_PROGRESS_2, -1)
      q:give_items(player, K.CRYSTAL_OF_INPROGRESS_2, 1)
      npc:broadcast_say(ChatType.GENERAL, "START DUEL")
      K._duelsInProgress[npcId] = ProgressDuelMob.new(player, player:summon())
    elseif st:get_int("Camoniell") == 3 and K._duelsInProgress[npcId] ~= nil then
      local duel = K._duelsInProgress[npcId]
      if not isPet or player:summon() ~= duel:getPet() then
        local foulPlayer = duel:getAttacker()
        if foulPlayer ~= nil then
          st = foulPlayer:get_quest_state(q:name())
          if st ~= nil then
            st:set("Camoniell", "5")
            q:take_items(player, K.CRYSTAL_OF_PROGRESS_2, -1)
            q:take_items(player, K.CRYSTAL_OF_INPROGRESS_2, -1)
            q:give_items(player, K.CRYSTAL_OF_FOUL_2, 1)
            st:add_notify_of_death(st:get_player())
            npc:broadcast_say(ChatType.GENERAL, "RULE VIOLATION")
            npc:die(npc)
          end
        end
      end
    end
  elseif npcId == K.SHADOW_TUREN then
    if st:get_int("Belthus") == 2 and isPet and npc:hp() + damage >= npc:max_hp() then
      st:set("Belthus", "3")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, K.CRYSTAL_OF_PROGRESS_3, -1)
      q:give_items(player, K.CRYSTAL_OF_INPROGRESS_3, 1)
      npc:broadcast_say(ChatType.GENERAL, "So shall we start?!")
      K._duelsInProgress[npcId] = ProgressDuelMob.new(player, player:summon())
    elseif st:get_int("Belthus") == 3 and K._duelsInProgress[npcId] ~= nil then
      local duel = K._duelsInProgress[npcId]
      if not isPet or player:summon() ~= duel:getPet() then
        local foulPlayer = duel:getAttacker()
        if foulPlayer ~= nil then
          st = foulPlayer:get_quest_state(q:name())
          if st ~= nil then
            st:set("Belthus", "5")
            q:take_items(player, K.CRYSTAL_OF_PROGRESS_3, -1)
            q:take_items(player, K.CRYSTAL_OF_INPROGRESS_3, -1)
            q:give_items(player, K.CRYSTAL_OF_FOUL_3, 1)
            st:add_notify_of_death(st:get_player())
            npc:broadcast_say(ChatType.GENERAL, "Rule violation!!!")
            npc:die(npc)
          end
        end
      end
    end
  elseif npcId == K.MIMI_THE_CAT then
    if st:get_int("Basilla") == 2 and isPet and npc:hp() + damage >= npc:max_hp() then
      st:set("Basilla", "3")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, K.CRYSTAL_OF_PROGRESS_4, -1)
      q:give_items(player, K.CRYSTAL_OF_INPROGRESS_4, 1)
      npc:broadcast_say(ChatType.GENERAL, "Whish! Fight!")
      K._duelsInProgress[npcId] = ProgressDuelMob.new(player, player:summon())
    elseif st:get_int("Basilla") == 3 and K._duelsInProgress[npcId] ~= nil then
      local duel = K._duelsInProgress[npcId]
      if not isPet or player:summon() ~= duel:getPet() then
        local foulPlayer = duel:getAttacker()
        if foulPlayer ~= nil then
          st = foulPlayer:get_quest_state(q:name())
          if st ~= nil then
            st:set("Basilla", "5")
            q:take_items(player, K.CRYSTAL_OF_PROGRESS_4, -1)
            q:take_items(player, K.CRYSTAL_OF_INPROGRESS_4, -1)
            q:give_items(player, K.CRYSTAL_OF_FOUL_4, 1)
            st:add_notify_of_death(st:get_player())
            npc:broadcast_say(ChatType.GENERAL, "Rule violation!")
            npc:die(npc)
          end
        end
      end
    end
  elseif npcId == K.UNICORN_PHANTASM then
    if st:get_int("Celestiel") == 2 and isPet and npc:hp() + damage >= npc:max_hp() then
      st:set("Celestiel", "3")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, K.CRYSTAL_OF_PROGRESS_5, -1)
      q:give_items(player, K.CRYSTAL_OF_INPROGRESS_5, 1)
      npc:broadcast_say(ChatType.GENERAL, "START DUEL")
      K._duelsInProgress[npcId] = ProgressDuelMob.new(player, player:summon())
    elseif st:get_int("Celestiel") == 3 and K._duelsInProgress[npcId] ~= nil then
      local duel = K._duelsInProgress[npcId]
      if not isPet or player:summon() ~= duel:getPet() then
        local foulPlayer = duel:getAttacker()
        if foulPlayer ~= nil then
          st = foulPlayer:get_quest_state(q:name())
          if st ~= nil then
            st:set("Celestiel", "5")
            q:take_items(player, K.CRYSTAL_OF_PROGRESS_5, -1)
            q:take_items(player, K.CRYSTAL_OF_INPROGRESS_5, -1)
            q:give_items(player, K.CRYSTAL_OF_FOUL_5, 1)
            st:add_notify_of_death(st:get_player())
            npc:broadcast_say(ChatType.GENERAL, "RULE VIOLATION")
            npc:die(npc)
          end
        end
      end
    end
  elseif npcId == K.SILHOUETTE_TILFO then
    if st:get_int("Brynthea") == 2 and isPet and npc:hp() + damage >= npc:max_hp() then
      st:set("Brynthea", "3")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, K.CRYSTAL_OF_PROGRESS_6, -1)
      q:give_items(player, K.CRYSTAL_OF_INPROGRESS_6, 1)
      npc:broadcast_say(ChatType.GENERAL, "I'll walk all over you!")
      K._duelsInProgress[npcId] = ProgressDuelMob.new(player, player:summon())
    elseif st:get_int("Brynthea") == 3 and K._duelsInProgress[npcId] ~= nil then
      local duel = K._duelsInProgress[npcId]
      if not isPet or player:summon() ~= duel:getPet() then
        local foulPlayer = duel:getAttacker()
        if foulPlayer ~= nil then
          st = foulPlayer:get_quest_state(q:name())
          if st ~= nil then
            st:set("Brynthea", "5")
            q:take_items(player, K.CRYSTAL_OF_PROGRESS_6, -1)
            q:take_items(player, K.CRYSTAL_OF_INPROGRESS_6, -1)
            q:give_items(player, K.CRYSTAL_OF_FOUL_6, 1)
            st:add_notify_of_death(st:get_player())
            npc:broadcast_say(ChatType.GENERAL, "Rule violation!!!")
            npc:die(npc)
          end
        end
      end
    end
  end
end)

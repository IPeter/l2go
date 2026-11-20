-- Translated by questport from quests/Q00336_CoinsOfMagic (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Adapted from FirstTeam Interlude
--
-- The coin exchanges of the exchange NPCs are not in Mobius either: their
-- pages just show.
local J = require("lib.java")

local q = quest.new(336, "Q00336_CoinsOfMagic", "Coins of Magic")
local K = {}

K.COIN_DIAGRAM = 3811
K.KALDIS_COIN = 3812
K.MEMBERSHIP_1 = 3813
K.MEMBERSHIP_2 = 3814
K.MEMBERSHIP_3 = 3815
K.BLOOD_MEDUSA = 3472
K.BLOOD_WEREWOLF = 3473
K.BLOOD_BASILISK = 3474
K.BLOOD_DREVANUL = 3475
K.BLOOD_SUCCUBUS = 3476
K.GOLD_WYVERN = 3482
K.GOLD_KNIGHT = 3483
K.GOLD_GIANT = 3484
K.GOLD_DRAKE = 3485
K.GOLD_WYRM = 3486
K.SILVER_UNICORN = 3490
K.SILVER_FAIRY = 3491
K.SILVER_DRYAD = 3492
K.SILVER_GOLEM = 3494
K.SILVER_UNDINE = 3495
K.SORINT = 30232
K.BERNARD = 30702
K.PAGE = 30696
K.HAGGER = 30183
K.STAN = 30200
K.RALFORD = 30165
K.FERRIS = 30847
K.COLLOB = 30092
K.PANO = 30078
K.DUNING = 30688
K.LORAIN = 30673
K.TIMAK_ORC_ARCHER = 20584
K.TIMAK_ORC_SOLDIER = 20585
K.TIMAK_ORC_SHAMAN = 20587
K.LAKIN = 20604
K.TORTURED_UNDEAD = 20678
K.HATAR_HANISHEE = 20663
K.SHACKLE = 20235
K.TIMAK_ORC = 20583
K.HEADLESS_KNIGHT = 20146
K.ROYAL_CAVE_SERVANT = 20240
K.MALRUK_SUCCUBUS_TUREN = 20245
K.FORMOR = 20568
K.FORMOR_ELDER = 20569
K.VANOR_SILENOS_SHAMAN = 20685
K.TARLK_BUGBEAR_HIGH_WARRIOR = 20572
K.OEL_MAHUM = 20161
K.OEL_MAHUM_WARRIOR = 20575
K.HARIT_LIZARDMAN_MATRIARCH = 20645
K.HARIT_LIZARDMAN_SHAMAN = 20644
K.GRAVE_LICH = 21003
K.DOOM_SERVANT = 21006
K.DOOM_ARCHER = 21008
K.DOOM_KNIGHT = 20674
K.KOOKABURRA2 = 21276
K.KOOKABURRA3 = 21275
K.KOOKABURRA4 = 21274
K.ANTELOPE2 = 21278
K.ANTELOPE3 = 21279
K.ANTELOPE4 = 21280
K.BANDERSNATCH2 = 21282
K.BANDERSNATCH3 = 21284
K.BANDERSNATCH4 = 21283
K.BUFFALO2 = 21287
K.BUFFALO3 = 21288
K.BUFFALO4 = 21286
K.CLAWS_OF_SPLENDOR = 21521
K.WISDOM_OF_SPLENDOR = 21526
K.PUNISHMENT_OF_SPLENDOR = 21531
K.WAILING_OF_SPLENDOR = 21539
K.HUNGERED_CORPSE = 20954
K.BLOODY_GHOST = 20960
K.NIHIL_INVADER = 20957
K.DARK_GUARD = 20959
K.PROMOTE = {
  J.array(0, 0),
  J.array(0, 0),
  {K.SILVER_DRYAD, K.BLOOD_BASILISK, K.BLOOD_SUCCUBUS, K.SILVER_UNDINE, K.GOLD_GIANT, K.GOLD_WYRM},
  {K.BLOOD_WEREWOLF, K.GOLD_DRAKE, K.SILVER_FAIRY, K.BLOOD_DREVANUL, K.GOLD_KNIGHT, K.SILVER_GOLEM},
}
K.EXCHANGE_LEVEL = {
  {K.PAGE, 3},
  {K.LORAIN, 3},
  {K.HAGGER, 3},
  {K.RALFORD, 2},
  {K.STAN, 2},
  {K.DUNING, 2},
  {K.FERRIS, 1},
  {K.COLLOB, 1},
  {K.PANO, 1},
}
K.DROPLIST = {
  {K.TIMAK_ORC_ARCHER, K.BLOOD_MEDUSA},
  {K.TIMAK_ORC_SOLDIER, K.BLOOD_MEDUSA},
  {K.TIMAK_ORC_SHAMAN, 3472},
  {K.LAKIN, 3472},
  {K.TORTURED_UNDEAD, 3472},
  {K.HATAR_HANISHEE, 3472},
  {K.TIMAK_ORC, K.GOLD_WYVERN},
  {K.SHACKLE, K.GOLD_WYVERN},
  {K.HEADLESS_KNIGHT, K.GOLD_WYVERN},
  {K.ROYAL_CAVE_SERVANT, K.GOLD_WYVERN},
  {K.MALRUK_SUCCUBUS_TUREN, K.GOLD_WYVERN},
  {K.FORMOR, K.SILVER_UNICORN},
  {K.FORMOR_ELDER, K.SILVER_UNICORN},
  {K.VANOR_SILENOS_SHAMAN, K.SILVER_UNICORN},
  {K.TARLK_BUGBEAR_HIGH_WARRIOR, K.SILVER_UNICORN},
  {K.OEL_MAHUM, K.SILVER_UNICORN},
  {K.OEL_MAHUM_WARRIOR, K.SILVER_UNICORN},
}
K.MONSTERS = {
  K.GRAVE_LICH,
  K.DOOM_SERVANT,
  K.DOOM_ARCHER,
  K.DOOM_KNIGHT,
  K.KOOKABURRA2,
  K.KOOKABURRA3,
  K.KOOKABURRA4,
  K.ANTELOPE2,
  K.ANTELOPE3,
  K.ANTELOPE4,
  K.BANDERSNATCH2,
  K.BANDERSNATCH3,
  K.BANDERSNATCH4,
  K.BUFFALO2,
  K.BUFFALO3,
  K.BUFFALO4,
  K.CLAWS_OF_SPLENDOR,
  K.WISDOM_OF_SPLENDOR,
  K.PUNISHMENT_OF_SPLENDOR,
  K.WAILING_OF_SPLENDOR,
  K.HUNGERED_CORPSE,
  K.BLOODY_GHOST,
  K.NIHIL_INVADER,
  K.DARK_GUARD,
}

local function promote(st)
  local grade = st:get_int("grade")
  local html_
  if grade == 1 then
    html_ = "30232-15.htm"
  else
    local h = 0
    for _, i in ipairs(K.PROMOTE[grade + 1]) do
      if q:get_quest_items_count(st:get_player(), i) > 0 then
        h = h + 1
      end
    end
    if h == 6 then
      local player = st:get_player()
      for _, i in ipairs(K.PROMOTE[grade + 1]) do
        q:take_items(player, i, 1)
      end
      html_ = "30232-" .. (19 - grade) .. ".htm"
      q:take_items(player, K.KALDIS_COIN + grade, -1)
      q:give_items(player, K.COIN_DIAGRAM + grade, 1)
      st:set("grade", tostring(grade - 1))
      if grade == 3 then
        st:set_cond(9)
      elseif grade == 2 then
        st:set_cond(11)
      end
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_FANFARE_MIDDLE)
    else
      html_ = "30232-" .. (16 - grade) .. ".htm"
      if grade == 3 then
        st:set_cond(8)
      elseif grade == 2 then
        st:set_cond(9)
      end
    end
  end
  return html_
end

q:add_start_npc(K.SORINT)
q:add_talk_id(K.SORINT, K.BERNARD, K.PAGE, K.HAGGER, K.STAN, K.RALFORD, K.FERRIS, K.COLLOB, K.PANO, K.DUNING, K.LORAIN)
for _, mob in ipairs(K.DROPLIST) do
  q:add_kill_id(mob[1])
end
q:add_kill_id(K.MONSTERS)
q:add_kill_id(K.HARIT_LIZARDMAN_MATRIARCH)
q:add_kill_id(K.HARIT_LIZARDMAN_SHAMAN)
q:register_quest_items(K.COIN_DIAGRAM, K.KALDIS_COIN, K.MEMBERSHIP_1, K.MEMBERSHIP_2, K.MEMBERSHIP_3)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  local cond = st:get_cond()
  if "30702-06.htm" == event then
    if cond < 7 then
      st:set_cond(7)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
    end
  elseif "30232-22.htm" == event then
    if cond < 6 then
      st:set_cond(6)
    end
  elseif "30232-23.htm" == event then
    if cond < 5 then
      st:set_cond(5)
    end
  elseif "30702-02.htm" == event then
    st:set_cond(2)
  elseif "30232-05.htm" == event then
    st:start_quest()
    q:give_items(player, K.COIN_DIAGRAM, 1)
  elseif "30232-04.htm" == event or "30232-18a.htm" == event then
    st:exit_quest(true)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_GIVEUP)
  elseif "raise" == event then
    htmltext = promote(st)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local npcId = npc:id()
  local id = st:get_state()
  local grade = st:get_int("grade")
  if npcId == K.SORINT then
    if id == State.CREATED then
      if st:get_player():level() < 40 then
        htmltext = "30232-01.htm"
        st:exit_quest(true)
      else
        htmltext = "30232-02.htm"
      end
    elseif q:get_quest_items_count(player, K.COIN_DIAGRAM) > 0 then
      if q:get_quest_items_count(player, K.KALDIS_COIN) > 0 then
        q:take_items(player, K.KALDIS_COIN, -1)
        q:take_items(player, K.COIN_DIAGRAM, -1)
        q:give_items(player, K.MEMBERSHIP_3, 1)
        st:set("grade", "3")
        st:set_cond(4)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_FANFARE_MIDDLE)
        htmltext = "30232-07.htm"
      else
        htmltext = "30232-06.htm"
      end
    elseif grade == 3 then
      htmltext = "30232-12.htm"
    elseif grade == 2 then
      htmltext = "30232-11.htm"
    elseif grade == 1 then
      htmltext = "30232-10.htm"
    end
  elseif npcId == K.BERNARD then
    if q:get_quest_items_count(player, K.COIN_DIAGRAM) > 0 and grade == 0 then
      htmltext = "30702-01.htm"
    elseif grade == 3 then
      htmltext = "30702-05.htm"
    end
  else
    for _, e in ipairs(K.EXCHANGE_LEVEL) do
      if npcId == e[1] and grade <= e[2] then
        htmltext = npcId .. "-01.htm"
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
  local cond = st:get_cond()
  local npcId = npc:id()
  if npcId == K.HARIT_LIZARDMAN_MATRIARCH or npcId == K.HARIT_LIZARDMAN_SHAMAN then
    if cond == 2 and rnd(1000) < 63 then
      q:give_items(player, K.KALDIS_COIN, 1)
      st:set_cond(3)
    end
    return
  end
  local grade = st:get_int("grade")
  local chance = npc:level() + grade * 3 - 20
  for _, e in ipairs(K.DROPLIST) do
    if e[1] == npcId then
      if rnd(100) < chance then
        q:give_items(player, e[2], 1)
      end
      return
    end
  end
end)

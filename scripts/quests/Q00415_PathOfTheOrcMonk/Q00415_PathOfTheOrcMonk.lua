-- Translated by questport from quests/Q00415_PathOfTheOrcMonk (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(415, "Q00415_PathOfTheOrcMonk", "Path to a Monk")

-- NPCs
local GANTAKI = 30587
local ROSHEEK = 30590
local KASMAN = 30501
local TORUKU = 30591
local AREN = 32056
local MOIRA = 31979
-- Items
local POMEGRANATE = 1593
local LEATHER_POUCH_1 = 1594
local LEATHER_POUCH_2 = 1595
local LEATHER_POUCH_3 = 1596
local LEATHER_POUCH_FULL_1 = 1597
local LEATHER_POUCH_FULL_2 = 1598
local LEATHER_POUCH_FULL_3 = 1599
local KASHA_BEAR_CLAW = 1600
local KASHA_BLADE_SPIDER_TALON = 1601
local SCARLET_SALAMANDER_SCALE = 1602
local FIERY_SPIRIT_SCROLL = 1603
local ROSHEEK_LETTER = 1604
local GANTAKI_LETTER_OF_RECOMMENDATION = 1605
local FIG = 1606
local LEATHER_POUCH_4 = 1607
local LEATHER_POUCH_FULL_4 = 1608
local VUKU_ORC_TUSK = 1609
local RATMAN_FANG = 1610
local LANG_KLIZARDMAN_TOOTH = 1611
local FELIM_LIZARDMAN_TOOTH = 1612
local IRON_WILL_SCROLL = 1613
local TORUKU_LETTER = 1614
local KHAVATARI_TOTEM = 1615
local KASHA_SPIDER_TOOTH = 8545
local HORN_OF_BAAR_DRE_VANUL = 8546

q:register_quest_items(POMEGRANATE, LEATHER_POUCH_1, LEATHER_POUCH_2, LEATHER_POUCH_3, LEATHER_POUCH_FULL_1, LEATHER_POUCH_FULL_2, LEATHER_POUCH_FULL_3, KASHA_BEAR_CLAW, KASHA_BLADE_SPIDER_TALON, SCARLET_SALAMANDER_SCALE, FIERY_SPIRIT_SCROLL, ROSHEEK_LETTER, GANTAKI_LETTER_OF_RECOMMENDATION, FIG, LEATHER_POUCH_4, LEATHER_POUCH_FULL_4, VUKU_ORC_TUSK, RATMAN_FANG, LANG_KLIZARDMAN_TOOTH, FELIM_LIZARDMAN_TOOTH, IRON_WILL_SCROLL, TORUKU_LETTER, KASHA_SPIDER_TOOTH, HORN_OF_BAAR_DRE_VANUL)
q:add_start_npc(GANTAKI)
q:add_talk_id(GANTAKI, ROSHEEK, KASMAN, TORUKU, AREN, MOIRA)
q:add_kill_id(20014, 20017, 20024, 20359, 20415, 20476, 20478, 20479, 21118)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30587-05.htm" then
    if player:class_id() ~= ClassId.ORC_FIGHTER then
      htmltext = player:class_id() == ClassId.ORC_MONK and "30587-02a.htm" or "30587-02.htm"
    elseif player:level() < 19 then
      htmltext = "30587-03.htm"
    elseif q:has_quest_items(player, KHAVATARI_TOTEM) then
      htmltext = "30587-04.htm"
    end
  elseif event == "30587-06.htm" then
    st:start_quest()
    q:give_items(player, POMEGRANATE, 1)
  elseif event == "30587-09a.htm" then
    st:set_cond(9, true)
    q:take_items(player, ROSHEEK_LETTER, 1)
    q:give_items(player, GANTAKI_LETTER_OF_RECOMMENDATION, 1)
  elseif event == "30587-09b.htm" then
    st:set_cond(14, true)
    q:take_items(player, ROSHEEK_LETTER, 1)
  elseif event == "32056-03.htm" then
    st:set_cond(15, true)
  elseif event == "32056-08.htm" then
    st:set_cond(20, true)
  elseif event == "31979-03.htm" then
    q:take_items(player, FIERY_SPIRIT_SCROLL, 1)
    q:give_items(player, KHAVATARI_TOTEM, 1)
    q:add_exp_and_sp(player, 3200, 4230)
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
    htmltext = "30587-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == GANTAKI then
      if cond == 1 then
        htmltext = "30587-07.htm"
      elseif cond > 1 and cond < 8 then
        htmltext = "30587-08.htm"
      elseif cond == 8 then
        htmltext = "30587-09.htm"
      elseif cond == 9 then
        htmltext = "30587-10.htm"
      elseif cond > 9 then
        htmltext = "30587-11.htm"
      end
    elseif sw2 == ROSHEEK then
      if cond == 1 then
        htmltext = "30590-01.htm"
        st:set_cond(2, true)
        q:take_items(player, POMEGRANATE, 1)
        q:give_items(player, LEATHER_POUCH_1, 1)
      elseif cond == 2 then
        htmltext = "30590-02.htm"
      elseif cond == 3 then
        htmltext = "30590-03.htm"
        st:set_cond(4, true)
        q:take_items(player, LEATHER_POUCH_FULL_1, 1)
        q:give_items(player, LEATHER_POUCH_2, 1)
      elseif cond == 4 then
        htmltext = "30590-04.htm"
      elseif cond == 5 then
        htmltext = "30590-05.htm"
        st:set_cond(6, true)
        q:take_items(player, LEATHER_POUCH_FULL_2, 1)
        q:give_items(player, LEATHER_POUCH_3, 1)
      elseif cond == 6 then
        htmltext = "30590-06.htm"
      elseif cond == 7 then
        htmltext = "30590-07.htm"
        st:set_cond(8, true)
        q:take_items(player, LEATHER_POUCH_FULL_3, 1)
        q:give_items(player, FIERY_SPIRIT_SCROLL, 1)
        q:give_items(player, ROSHEEK_LETTER, 1)
      elseif cond == 8 then
        htmltext = "30590-08.htm"
      elseif cond > 8 then
        htmltext = "30590-09.htm"
      end
    elseif sw2 == KASMAN then
      if cond == 9 then
        htmltext = "30501-01.htm"
        st:set_cond(10, true)
        q:take_items(player, GANTAKI_LETTER_OF_RECOMMENDATION, 1)
        q:give_items(player, FIG, 1)
      elseif cond == 10 then
        htmltext = "30501-02.htm"
      elseif cond == 11 or cond == 12 then
        htmltext = "30501-03.htm"
      elseif cond == 13 then
        htmltext = "30501-04.htm"
        q:take_items(player, FIERY_SPIRIT_SCROLL, 1)
        q:take_items(player, IRON_WILL_SCROLL, 1)
        q:take_items(player, TORUKU_LETTER, 1)
        q:give_items(player, KHAVATARI_TOTEM, 1)
        q:add_exp_and_sp(player, 3200, 1500)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == TORUKU then
      if cond == 10 then
        htmltext = "30591-01.htm"
        st:set_cond(11, true)
        q:take_items(player, FIG, 1)
        q:give_items(player, LEATHER_POUCH_4, 1)
      elseif cond == 11 then
        htmltext = "30591-02.htm"
      elseif cond == 12 then
        htmltext = "30591-03.htm"
        st:set_cond(13, true)
        q:take_items(player, LEATHER_POUCH_FULL_4, 1)
        q:give_items(player, IRON_WILL_SCROLL, 1)
        q:give_items(player, TORUKU_LETTER, 1)
      elseif cond == 13 then
        htmltext = "30591-04.htm"
      end
    elseif sw2 == AREN then
      if cond == 14 then
        htmltext = "32056-01.htm"
      elseif cond == 15 then
        htmltext = "32056-04.htm"
      elseif cond == 16 then
        htmltext = "32056-05.htm"
        st:set_cond(17, true)
        q:take_items(player, KASHA_SPIDER_TOOTH, -1)
      elseif cond == 17 then
        htmltext = "32056-06.htm"
      elseif cond == 18 then
        htmltext = "32056-07.htm"
        st:set_cond(19, true)
        q:take_items(player, HORN_OF_BAAR_DRE_VANUL, -1)
      elseif cond == 20 then
        htmltext = "32056-09.htm"
      end
    elseif sw2 == MOIRA then
      if cond == 20 then
        htmltext = "31979-01.htm"
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
  -- L2Go: Mobius aborts the quest (the give-up sound, exitQuest(true)) when
  -- the kill was made with a weapon that is not a fist weapon
  -- (player.getActiveWeaponItem().getItemType() is neither DUALFIST nor
  -- FIST). The script API cannot read the active weapon's type yet, so that
  -- check is left out until it can.
  local sw = npc:id()
  if sw == 20479 then
    if st:is_cond(2) then
      q:give_items(player, KASHA_BEAR_CLAW, 1)
      if q:get_quest_items_count(player, KASHA_BEAR_CLAW) < 5 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        st:set_cond(3, true)
        q:take_items(player, KASHA_BEAR_CLAW, -1)
        q:take_items(player, LEATHER_POUCH_1, 1)
        q:give_items(player, LEATHER_POUCH_FULL_1, 1)
      end
    end
  elseif sw == 20478 then
    if st:is_cond(4) then
      q:give_items(player, KASHA_BLADE_SPIDER_TALON, 1)
      if q:get_quest_items_count(player, KASHA_BLADE_SPIDER_TALON) < 5 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        st:set_cond(5, true)
        q:take_items(player, KASHA_BLADE_SPIDER_TALON, -1)
        q:take_items(player, LEATHER_POUCH_2, 1)
        q:give_items(player, LEATHER_POUCH_FULL_2, 1)
      end
    elseif st:is_cond(15) and rnd.bool() then
      q:give_items(player, KASHA_SPIDER_TOOTH, 1)
      if q:get_quest_items_count(player, KASHA_SPIDER_TOOTH) < 6 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        st:set_cond(16, true)
      end
    end
  elseif sw == 20476 then
    if st:is_cond(15) and rnd.bool() then
      q:give_items(player, KASHA_SPIDER_TOOTH, 1)
      if q:get_quest_items_count(player, KASHA_SPIDER_TOOTH) < 6 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        st:set_cond(16, true)
      end
    end
  elseif sw == 20415 then
    if st:is_cond(6) then
      q:give_items(player, SCARLET_SALAMANDER_SCALE, 1)
      if q:get_quest_items_count(player, SCARLET_SALAMANDER_SCALE) < 5 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        st:set_cond(7, true)
        q:take_items(player, SCARLET_SALAMANDER_SCALE, -1)
        q:take_items(player, LEATHER_POUCH_3, 1)
        q:give_items(player, LEATHER_POUCH_FULL_3, 1)
      end
    end
  elseif sw == 20014 then
    if st:is_cond(11) then
      if q:get_quest_items_count(player, FELIM_LIZARDMAN_TOOTH) < 3 then
        q:give_items(player, FELIM_LIZARDMAN_TOOTH, 1)
      end
      if q:get_quest_items_count(player, FELIM_LIZARDMAN_TOOTH) >= 3 and q:get_quest_items_count(player, RATMAN_FANG) >= 3 and q:get_quest_items_count(player, LANG_KLIZARDMAN_TOOTH) >= 3 and q:get_quest_items_count(player, VUKU_ORC_TUSK) >= 3 then
        st:set_cond(12, true)
        q:take_items(player, VUKU_ORC_TUSK, -1)
        q:take_items(player, RATMAN_FANG, -1)
        q:take_items(player, LANG_KLIZARDMAN_TOOTH, -1)
        q:take_items(player, FELIM_LIZARDMAN_TOOTH, -1)
        q:take_items(player, LEATHER_POUCH_4, 1)
        q:give_items(player, LEATHER_POUCH_FULL_4, 1)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 20017 then
    if st:is_cond(11) then
      if q:get_quest_items_count(player, VUKU_ORC_TUSK) < 3 then
        q:give_items(player, VUKU_ORC_TUSK, 1)
      end
      if q:get_quest_items_count(player, VUKU_ORC_TUSK) >= 3 and q:get_quest_items_count(player, RATMAN_FANG) >= 3 and q:get_quest_items_count(player, LANG_KLIZARDMAN_TOOTH) >= 3 and q:get_quest_items_count(player, FELIM_LIZARDMAN_TOOTH) >= 3 then
        st:set_cond(12, true)
        q:take_items(player, VUKU_ORC_TUSK, -1)
        q:take_items(player, RATMAN_FANG, -1)
        q:take_items(player, LANG_KLIZARDMAN_TOOTH, -1)
        q:take_items(player, FELIM_LIZARDMAN_TOOTH, -1)
        q:take_items(player, LEATHER_POUCH_4, 1)
        q:give_items(player, LEATHER_POUCH_FULL_4, 1)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 20024 then
    if st:is_cond(11) then
      if q:get_quest_items_count(player, LANG_KLIZARDMAN_TOOTH) < 3 then
        q:give_items(player, LANG_KLIZARDMAN_TOOTH, 1)
      end
      if q:get_quest_items_count(player, LANG_KLIZARDMAN_TOOTH) >= 3 and q:get_quest_items_count(player, RATMAN_FANG) >= 3 and q:get_quest_items_count(player, FELIM_LIZARDMAN_TOOTH) >= 3 and q:get_quest_items_count(player, VUKU_ORC_TUSK) >= 3 then
        st:set_cond(12, true)
        q:take_items(player, VUKU_ORC_TUSK, -1)
        q:take_items(player, RATMAN_FANG, -1)
        q:take_items(player, LANG_KLIZARDMAN_TOOTH, -1)
        q:take_items(player, FELIM_LIZARDMAN_TOOTH, -1)
        q:take_items(player, LEATHER_POUCH_4, 1)
        q:give_items(player, LEATHER_POUCH_FULL_4, 1)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 20359 then
    if st:is_cond(11) then
      if q:get_quest_items_count(player, RATMAN_FANG) < 3 then
        q:give_items(player, RATMAN_FANG, 1)
      end
      if q:get_quest_items_count(player, RATMAN_FANG) >= 3 and q:get_quest_items_count(player, LANG_KLIZARDMAN_TOOTH) >= 3 and q:get_quest_items_count(player, FELIM_LIZARDMAN_TOOTH) >= 3 and q:get_quest_items_count(player, VUKU_ORC_TUSK) >= 3 then
        st:set_cond(12, true)
        q:take_items(player, VUKU_ORC_TUSK, -1)
        q:take_items(player, RATMAN_FANG, -1)
        q:take_items(player, LANG_KLIZARDMAN_TOOTH, -1)
        q:take_items(player, FELIM_LIZARDMAN_TOOTH, -1)
        q:take_items(player, LEATHER_POUCH_4, 1)
        q:give_items(player, LEATHER_POUCH_FULL_4, 1)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 21118 then
    if st:is_cond(17) then
      st:set_cond(18, true)
      q:give_items(player, HORN_OF_BAAR_DRE_VANUL, 1)
    end
  end
end)

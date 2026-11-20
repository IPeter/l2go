-- Translated by questport from quests/Q00408_PathOfTheElvenWizard (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(408, "Q00408_PathOfTheElvenWizard", "Path to an Elven Wizard")

-- NPCs
local ROSELLA = 30414
local GREENIS = 30157
local THALIA = 30371
local NORTHWIND = 30423
-- Items
local ROSELLA_LETTER = 1218
local RED_DOWN = 1219
local MAGICAL_POWERS_RUBY = 1220
local PURE_AQUAMARINE = 1221
local APPETIZING_APPLE = 1222
local GOLD_LEAVES = 1223
local IMMORTAL_LOVE = 1224
local AMETHYST = 1225
local NOBILITY_AMETHYST = 1226
local FERTILITY_PERIDOT = 1229
local ETERNITY_DIAMOND = 1230
local CHARM_OF_GRAIN = 1272
local SAP_OF_THE_MOTHER_TREE = 1273
local LUCKY_POTPOURRI = 1274

q:register_quest_items(ROSELLA_LETTER, RED_DOWN, MAGICAL_POWERS_RUBY, PURE_AQUAMARINE, APPETIZING_APPLE, GOLD_LEAVES, IMMORTAL_LOVE, AMETHYST, NOBILITY_AMETHYST, FERTILITY_PERIDOT, CHARM_OF_GRAIN, SAP_OF_THE_MOTHER_TREE, LUCKY_POTPOURRI)
q:add_start_npc(ROSELLA)
q:add_talk_id(ROSELLA, GREENIS, THALIA, NORTHWIND)
q:add_kill_id(20047, 20019, 20466)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30414-06.htm" then
    if player:class_id() ~= ClassId.ELVEN_MAGE then
      htmltext = player:class_id() == ClassId.ELVEN_WIZARD and "30414-02a.htm" or "30414-03.htm"
    elseif player:level() < 19 then
      htmltext = "30414-04.htm"
    elseif q:has_quest_items(player, ETERNITY_DIAMOND) then
      htmltext = "30414-05.htm"
    else
      st:start_quest()
      q:give_items(player, FERTILITY_PERIDOT, 1)
    end
  elseif event == "30414-07.htm" then
    if not q:has_quest_items(player, MAGICAL_POWERS_RUBY) then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:give_items(player, ROSELLA_LETTER, 1)
    else
      htmltext = "30414-10.htm"
    end
  elseif event == "30414-14.htm" then
    if not q:has_quest_items(player, PURE_AQUAMARINE) then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:give_items(player, APPETIZING_APPLE, 1)
    else
      htmltext = "30414-13.htm"
    end
  elseif event == "30414-18.htm" then
    if not q:has_quest_items(player, NOBILITY_AMETHYST) then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:give_items(player, IMMORTAL_LOVE, 1)
    else
      htmltext = "30414-17.htm"
    end
  elseif event == "30157-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:take_items(player, ROSELLA_LETTER, 1)
    q:give_items(player, CHARM_OF_GRAIN, 1)
  elseif event == "30371-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:take_items(player, APPETIZING_APPLE, 1)
    q:give_items(player, SAP_OF_THE_MOTHER_TREE, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30414-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == ROSELLA then
      if q:has_quest_items(player, MAGICAL_POWERS_RUBY, NOBILITY_AMETHYST, PURE_AQUAMARINE) then
        htmltext = "30414-24.htm"
        q:take_items(player, FERTILITY_PERIDOT, 1)
        q:take_items(player, MAGICAL_POWERS_RUBY, 1)
        q:take_items(player, NOBILITY_AMETHYST, 1)
        q:take_items(player, PURE_AQUAMARINE, 1)
        q:give_items(player, ETERNITY_DIAMOND, 1)
        q:add_exp_and_sp(player, 3200, 1890)
        player:social_action(3)
        st:exit_quest(true, true)
      elseif q:has_quest_items(player, ROSELLA_LETTER) then
        htmltext = "30414-08.htm"
      elseif q:has_quest_items(player, CHARM_OF_GRAIN) then
        if q:get_quest_items_count(player, RED_DOWN) == 5 then
          htmltext = "30414-25.htm"
        else
          htmltext = "30414-09.htm"
        end
      elseif q:has_quest_items(player, APPETIZING_APPLE) then
        htmltext = "30414-15.htm"
      elseif q:has_quest_items(player, SAP_OF_THE_MOTHER_TREE) then
        if q:get_quest_items_count(player, GOLD_LEAVES) == 5 then
          htmltext = "30414-26.htm"
        else
          htmltext = "30414-16.htm"
        end
      elseif q:has_quest_items(player, IMMORTAL_LOVE) then
        htmltext = "30414-19.htm"
      elseif q:has_quest_items(player, LUCKY_POTPOURRI) then
        if q:get_quest_items_count(player, AMETHYST) == 2 then
          htmltext = "30414-27.htm"
        else
          htmltext = "30414-20.htm"
        end
      else
        htmltext = "30414-11.htm"
      end
    elseif sw2 == GREENIS then
      if q:has_quest_items(player, ROSELLA_LETTER) then
        htmltext = "30157-01.htm"
      elseif q:get_quest_items_count(player, RED_DOWN) == 5 then
        htmltext = "30157-04.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        q:take_items(player, CHARM_OF_GRAIN, 1)
        q:take_items(player, RED_DOWN, -1)
        q:give_items(player, MAGICAL_POWERS_RUBY, 1)
      elseif q:has_quest_items(player, CHARM_OF_GRAIN) then
        htmltext = "30157-03.htm"
      end
    elseif sw2 == THALIA then
      if q:has_quest_items(player, APPETIZING_APPLE) then
        htmltext = "30371-01.htm"
      elseif q:get_quest_items_count(player, GOLD_LEAVES) == 5 then
        htmltext = "30371-04.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        q:take_items(player, GOLD_LEAVES, -1)
        q:take_items(player, SAP_OF_THE_MOTHER_TREE, 1)
        q:give_items(player, PURE_AQUAMARINE, 1)
      elseif q:has_quest_items(player, SAP_OF_THE_MOTHER_TREE) then
        htmltext = "30371-03.htm"
      end
    elseif sw2 == NORTHWIND then
      if q:has_quest_items(player, IMMORTAL_LOVE) then
        htmltext = "30423-01.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        q:take_items(player, IMMORTAL_LOVE, 1)
        q:give_items(player, LUCKY_POTPOURRI, 1)
      elseif q:get_quest_items_count(player, AMETHYST) == 2 then
        htmltext = "30423-03.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        q:take_items(player, AMETHYST, -1)
        q:take_items(player, LUCKY_POTPOURRI, 1)
        q:give_items(player, NOBILITY_AMETHYST, 1)
      elseif q:has_quest_items(player, LUCKY_POTPOURRI) then
        htmltext = "30423-02.htm"
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
  if sw == 20019 then
    if q:has_quest_items(player, SAP_OF_THE_MOTHER_TREE) and q:get_quest_items_count(player, GOLD_LEAVES) < 5 and rnd(10) < 4 then
      q:give_items(player, GOLD_LEAVES, 1)
      if q:get_quest_items_count(player, GOLD_LEAVES) < 5 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20047 then
    if q:has_quest_items(player, LUCKY_POTPOURRI) and q:get_quest_items_count(player, AMETHYST) < 2 and rnd(10) < 4 then
      q:give_items(player, AMETHYST, 1)
      if q:get_quest_items_count(player, AMETHYST) < 2 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20466 then
    if q:has_quest_items(player, CHARM_OF_GRAIN) and q:get_quest_items_count(player, RED_DOWN) < 5 and rnd(10) < 7 then
      q:give_items(player, RED_DOWN, 1)
      if q:get_quest_items_count(player, RED_DOWN) < 5 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  end
end)

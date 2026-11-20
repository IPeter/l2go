-- Translated by questport from quests/Q00404_PathOfTheHumanWizard (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(404, "Q00404_PathOfTheHumanWizard", "Path to a Human Wizard")

-- NPCs
local PARINA = 30391
local EARTH_SNAKE = 30409
local WASTELAND_LIZARDMAN = 30410
local FLAME_SALAMANDER = 30411
local WIND_SYLPH = 30412
local WATER_UNDINE = 30413
-- Items
local MAP_OF_LUSTER = 1280
local KEY_OF_FLAME = 1281
local FLAME_EARING = 1282
local BROKEN_BRONZE_MIRROR = 1283
local WIND_FEATHER = 1284
local WIND_BANGEL = 1285
local RAMA_DIARY = 1286
local SPARKLE_PEBBLE = 1287
local WATER_NECKLACE = 1288
local RUST_GOLD_COIN = 1289
local RED_SOIL = 1290
local EARTH_RING = 1291
local BEAD_OF_SEASON = 1292

q:register_quest_items(MAP_OF_LUSTER, KEY_OF_FLAME, FLAME_EARING, BROKEN_BRONZE_MIRROR, WIND_FEATHER, WIND_BANGEL, RAMA_DIARY, SPARKLE_PEBBLE, WATER_NECKLACE, RUST_GOLD_COIN, RED_SOIL, EARTH_RING)
q:add_start_npc(PARINA)
q:add_talk_id(PARINA, EARTH_SNAKE, WASTELAND_LIZARDMAN, FLAME_SALAMANDER, WIND_SYLPH, WATER_UNDINE)
q:add_kill_id(20021, 20359, 27030)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30391-08.htm" then
    st:start_quest()
  elseif event == "30410-03.htm" then
    st:set_cond(6, true)
    q:take_items(player, BROKEN_BRONZE_MIRROR, 1)
    q:give_items(player, WIND_FEATHER, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local cond = st:get_cond()
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.MAGE then
      htmltext = player:class_id() == ClassId.WIZARD and "30391-02a.htm" or "30391-01.htm"
    elseif player:level() < 19 then
      htmltext = "30391-02.htm"
    elseif q:has_quest_items(player, BEAD_OF_SEASON) then
      htmltext = "30391-03.htm"
    else
      htmltext = "30391-04.htm"
    end
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == PARINA then
      if cond < 13 then
        htmltext = "30391-05.htm"
      elseif cond == 13 then
        htmltext = "30391-06.htm"
        q:take_items(player, EARTH_RING, 1)
        q:take_items(player, FLAME_EARING, 1)
        q:take_items(player, WATER_NECKLACE, 1)
        q:take_items(player, WIND_BANGEL, 1)
        q:give_items(player, BEAD_OF_SEASON, 1)
        q:add_exp_and_sp(player, 3200, 2020)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == FLAME_SALAMANDER then
      if cond == 1 then
        htmltext = "30411-01.htm"
        st:set_cond(2, true)
        q:give_items(player, MAP_OF_LUSTER, 1)
      elseif cond == 2 then
        htmltext = "30411-02.htm"
      elseif cond == 3 then
        htmltext = "30411-03.htm"
        st:set_cond(4, true)
        q:take_items(player, KEY_OF_FLAME, 1)
        q:take_items(player, MAP_OF_LUSTER, 1)
        q:give_items(player, FLAME_EARING, 1)
      elseif cond > 3 then
        htmltext = "30411-04.htm"
      end
    elseif sw2 == WIND_SYLPH then
      if cond == 4 then
        htmltext = "30412-01.htm"
        st:set_cond(5, true)
        q:give_items(player, BROKEN_BRONZE_MIRROR, 1)
      elseif cond == 5 then
        htmltext = "30412-02.htm"
      elseif cond == 6 then
        htmltext = "30412-03.htm"
        st:set_cond(7, true)
        q:take_items(player, WIND_FEATHER, 1)
        q:give_items(player, WIND_BANGEL, 1)
      elseif cond > 6 then
        htmltext = "30412-04.htm"
      end
    elseif sw2 == WASTELAND_LIZARDMAN then
      if cond == 5 then
        htmltext = "30410-01.htm"
      elseif cond > 5 then
        htmltext = "30410-04.htm"
      end
    elseif sw2 == WATER_UNDINE then
      if cond == 7 then
        htmltext = "30413-01.htm"
        st:set_cond(8, true)
        q:give_items(player, RAMA_DIARY, 1)
      elseif cond == 8 then
        htmltext = "30413-02.htm"
      elseif cond == 9 then
        htmltext = "30413-03.htm"
        st:set_cond(10, true)
        q:take_items(player, RAMA_DIARY, 1)
        q:take_items(player, SPARKLE_PEBBLE, -1)
        q:give_items(player, WATER_NECKLACE, 1)
      elseif cond > 9 then
        htmltext = "30413-04.htm"
      end
    elseif sw2 == EARTH_SNAKE then
      if cond == 10 then
        htmltext = "30409-01.htm"
        st:set_cond(11, true)
        q:give_items(player, RUST_GOLD_COIN, 1)
      elseif cond == 11 then
        htmltext = "30409-02.htm"
      elseif cond == 12 then
        htmltext = "30409-03.htm"
        st:set_cond(13, true)
        q:take_items(player, RED_SOIL, 1)
        q:take_items(player, RUST_GOLD_COIN, 1)
        q:give_items(player, EARTH_RING, 1)
      elseif cond > 12 then
        htmltext = "30409-04.htm"
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
  if sw == 20359 then
    if st:is_cond(2) and rnd(10) < 8 then
      q:give_items(player, KEY_OF_FLAME, 1)
      st:set_cond(3, true)
    end
  elseif sw == 27030 then
    if st:is_cond(8) and rnd(10) < 8 then
      if q:get_quest_items_count(player, SPARKLE_PEBBLE) < 2 then
        q:give_items(player, SPARKLE_PEBBLE, 1)
        if q:get_quest_items_count(player, SPARKLE_PEBBLE) >= 2 then
          st:set_cond(9, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == 20021 then
    if st:is_cond(11) and rnd(10) < 2 then
      q:give_items(player, RED_SOIL, 1)
      st:set_cond(12, true)
    end
  end
end)

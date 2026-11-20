-- Translated by questport from quests/Q00420_LittleWing (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(420, "Q00420_LittleWing", "Little Wing")
-- The Dragonflutes summon hatchlings (Plan 11: pets 352/419-421). Mimyu's
-- npc:teleport does not exist yet either (Npc teleport, an API gap).
q:requires("pets")

-- NPCs
local MARIA = 30608
local CRONOS = 30610
local BYRON = 30711
local MIMYU = 30747
local EXARION = 30748
local ZWOV = 30749
local KALIBRAN = 30750
local SUZET = 30751
local SHAMHAI = 30752
local COOPER = 30829
-- Items
local FAIRY_DUST = 3499
local FAIRY_STONE = 3816
local DELUXE_FAIRY_STONE = 3817
local FAIRY_STONE_LIST = 3818
local DELUXE_FAIRY_STONE_LIST = 3819
local TOAD_LORD_BACK_SKIN = 3820
local JUICE_OF_MONKSHOOD = 3821
local SCALE_OF_DRAKE_EXARION = 3822
local EGG_OF_DRAKE_EXARION = 3823
local SCALE_OF_DRAKE_ZWOV = 3824
local EGG_OF_DRAKE_ZWOV = 3825
local SCALE_OF_DRAKE_KALIBRAN = 3826
local EGG_OF_DRAKE_KALIBRAN = 3827
local SCALE_OF_WYVERN_SUZET = 3828
local EGG_OF_WYVERN_SUZET = 3829
local SCALE_OF_WYVERN_SHAMHAI = 3830
local EGG_OF_WYVERN_SHAMHAI = 3831
-- Needed items
local COAL = 1870
local CHARCOAL = 1871
local SILVER_NUGGET = 1873
local STONE_OF_PURITY = 1875
local GEMSTONE_D = 2130
local GEMSTONE_C = 2131
-- Rewards
local DRAGONFLUTE_OF_WIND = 3500
local DRAGONFLUTE_OF_STAR = 3501
local DRAGONFLUTE_OF_TWILIGHT = 3502
local HATCHLING_SOFT_LEATHER = 3912
local FOOD_FOR_HATCHLING = 4038
-- Spawn Points
local LOCATIONS = {
  {x = 109816, y = 40854, z = -4640, heading = 0},
  {x = 108940, y = 41615, z = -4643, heading = 0},
  {x = 110395, y = 41625, z = -4642, heading = 0},
}
-- Misc
local _counter = 0

local function checkItems(player, isDeluxe)
  -- Conditions required for both cases.
  if q:get_quest_items_count(player, COAL) < 10 or q:get_quest_items_count(player, CHARCOAL) < 10 then
    return false
  end
  if isDeluxe then
    if q:get_quest_items_count(player, GEMSTONE_C) >= 1 and q:get_quest_items_count(player, SILVER_NUGGET) >= 5 and q:get_quest_items_count(player, STONE_OF_PURITY) >= 1 and q:get_quest_items_count(player, TOAD_LORD_BACK_SKIN) >= 20 then
      return true
    end
  else
    if q:get_quest_items_count(player, GEMSTONE_D) >= 1 and q:get_quest_items_count(player, SILVER_NUGGET) >= 3 and q:get_quest_items_count(player, TOAD_LORD_BACK_SKIN) >= 10 then
      return true
    end
  end
  return false
end

local function giveRandomPet(player, hasFairyDust)
  local pet = DRAGONFLUTE_OF_TWILIGHT
  local chance = rnd(100)
  if q:has_quest_items(player, EGG_OF_DRAKE_EXARION) then
    q:take_items(player, EGG_OF_DRAKE_EXARION, 1)
    if hasFairyDust then
      if chance < 45 then
        pet = DRAGONFLUTE_OF_WIND
      elseif chance < 75 then
        pet = DRAGONFLUTE_OF_STAR
      end
    elseif chance < 50 then
      pet = DRAGONFLUTE_OF_WIND
    elseif chance < 85 then
      pet = DRAGONFLUTE_OF_STAR
    end
  elseif q:has_quest_items(player, EGG_OF_WYVERN_SUZET) then
    q:take_items(player, EGG_OF_WYVERN_SUZET, 1)
    if hasFairyDust then
      if chance < 55 then
        pet = DRAGONFLUTE_OF_WIND
      elseif chance < 85 then
        pet = DRAGONFLUTE_OF_STAR
      end
    elseif chance < 65 then
      pet = DRAGONFLUTE_OF_WIND
    elseif chance < 95 then
      pet = DRAGONFLUTE_OF_STAR
    end
  elseif q:has_quest_items(player, EGG_OF_DRAKE_KALIBRAN) then
    q:take_items(player, EGG_OF_DRAKE_KALIBRAN, 1)
    if hasFairyDust then
      if chance < 60 then
        pet = DRAGONFLUTE_OF_WIND
      elseif chance < 90 then
        pet = DRAGONFLUTE_OF_STAR
      end
    elseif chance < 70 then
      pet = DRAGONFLUTE_OF_WIND
    else
      pet = DRAGONFLUTE_OF_STAR
    end
  elseif q:has_quest_items(player, EGG_OF_WYVERN_SHAMHAI) then
    q:take_items(player, EGG_OF_WYVERN_SHAMHAI, 1)
    if hasFairyDust then
      if chance < 70 then
        pet = DRAGONFLUTE_OF_WIND
      else
        pet = DRAGONFLUTE_OF_STAR
      end
    elseif chance < 85 then
      pet = DRAGONFLUTE_OF_WIND
    else
      pet = DRAGONFLUTE_OF_STAR
    end
  elseif q:has_quest_items(player, EGG_OF_DRAKE_ZWOV) then
    q:take_items(player, EGG_OF_DRAKE_ZWOV, 1)
    if hasFairyDust then
      if chance < 90 then
        pet = DRAGONFLUTE_OF_WIND
      else
        pet = DRAGONFLUTE_OF_STAR
      end
    else
      pet = DRAGONFLUTE_OF_WIND
    end
  end
  q:give_items(player, pet, 1)
end

q:register_quest_items(FAIRY_STONE, DELUXE_FAIRY_STONE, FAIRY_STONE_LIST, DELUXE_FAIRY_STONE_LIST, TOAD_LORD_BACK_SKIN, JUICE_OF_MONKSHOOD, SCALE_OF_DRAKE_EXARION, EGG_OF_DRAKE_EXARION, SCALE_OF_DRAKE_ZWOV, EGG_OF_DRAKE_ZWOV, SCALE_OF_DRAKE_KALIBRAN, EGG_OF_DRAKE_KALIBRAN, SCALE_OF_WYVERN_SUZET, EGG_OF_WYVERN_SUZET, SCALE_OF_WYVERN_SHAMHAI, EGG_OF_WYVERN_SHAMHAI)
q:add_start_npc(COOPER, MIMYU)
q:add_talk_id(MARIA, CRONOS, BYRON, MIMYU, EXARION, ZWOV, KALIBRAN, SUZET, SHAMHAI, COOPER)
q:add_kill_id(20202, 20231, 20233, 20270, 20551, 20580, 20589, 20590, 20591, 20592, 20593, 20594, 20595, 20596, 20597, 20598, 20599)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30829-02.htm" then
    st:start_quest()
  elseif event == "30610-05.htm" then
    st:set_cond(2, true)
    q:give_items(player, FAIRY_STONE_LIST, 1)
  elseif event == "30610-06.htm" then
    st:set_cond(2, true)
    q:give_items(player, DELUXE_FAIRY_STONE_LIST, 1)
  elseif event == "30610-12.htm" then
    st:set_cond(2, true)
    st:set("deluxestone", "1")
    q:give_items(player, FAIRY_STONE_LIST, 1)
  elseif event == "30610-13.htm" then
    st:set_cond(2, true)
    st:set("deluxestone", "1")
    q:give_items(player, DELUXE_FAIRY_STONE_LIST, 1)
  elseif event == "30608-03.htm" then
    if not checkItems(player, false) then
      htmltext = "30608-01.htm" -- Avoid to continue while trade or drop mats before clicking bypass
    else
      q:take_items(player, COAL, 10)
      q:take_items(player, CHARCOAL, 10)
      q:take_items(player, GEMSTONE_D, 1)
      q:take_items(player, SILVER_NUGGET, 3)
      q:take_items(player, TOAD_LORD_BACK_SKIN, -1)
      q:take_items(player, FAIRY_STONE_LIST, 1)
      q:give_items(player, FAIRY_STONE, 1)
    end
  elseif event == "30608-05.htm" then
    if not checkItems(player, true) then
      htmltext = "30608-01.htm" -- Avoid to continue while trade or drop mats before clicking bypass
    else
      q:take_items(player, COAL, 10)
      q:take_items(player, CHARCOAL, 10)
      q:take_items(player, GEMSTONE_C, 1)
      q:take_items(player, STONE_OF_PURITY, 1)
      q:take_items(player, SILVER_NUGGET, 5)
      q:take_items(player, TOAD_LORD_BACK_SKIN, -1)
      q:take_items(player, DELUXE_FAIRY_STONE_LIST, 1)
      q:give_items(player, DELUXE_FAIRY_STONE, 1)
    end
  elseif event == "30711-03.htm" then
    st:set_cond(4, true)
    if q:has_quest_items(player, DELUXE_FAIRY_STONE) then
      htmltext = "30711-04.htm"
    end
  elseif event == "30747-02.htm" then
    st:set("mimyu", "1")
    q:take_items(player, FAIRY_STONE, 1)
  elseif event == "30747-04.htm" then
    st:set("mimyu", "1")
    q:take_items(player, DELUXE_FAIRY_STONE, 1)
    q:give_items(player, FAIRY_DUST, 1)
  elseif event == "30747-07.htm" then
    st:set_cond(5, true)
    q:give_items(player, JUICE_OF_MONKSHOOD, 1)
  elseif event == "30747-12.htm" then
    if not q:has_quest_items(player, FAIRY_DUST) then
      htmltext = "30747-15.htm"
      giveRandomPet(player, false)
      st:exit_quest(true, true)
    end
  elseif event == "30747-13.htm" then
    giveRandomPet(player, q:has_quest_items(player, FAIRY_DUST))
    st:exit_quest(true, true)
  elseif event == "30747-14.htm" then
    if q:has_quest_items(player, FAIRY_DUST) then
      q:take_items(player, FAIRY_DUST, 1)
      giveRandomPet(player, true)
      if rnd(20) == 1 then
        q:give_items(player, HATCHLING_SOFT_LEATHER, 1)
      else
        htmltext = "30747-14t.htm"
        q:give_items(player, FOOD_FOR_HATCHLING, 20)
      end
      st:exit_quest(true, true)
    else
      htmltext = "30747-13.htm"
    end
  elseif event == "30748-02.htm" then
    st:set_cond(6, true)
    q:take_items(player, JUICE_OF_MONKSHOOD, 1)
    q:give_items(player, SCALE_OF_DRAKE_EXARION, 1)
  elseif event == "30749-02.htm" then
    st:set_cond(6, true)
    q:take_items(player, JUICE_OF_MONKSHOOD, 1)
    q:give_items(player, SCALE_OF_DRAKE_ZWOV, 1)
  elseif event == "30750-02.htm" then
    st:set_cond(6, true)
    q:take_items(player, JUICE_OF_MONKSHOOD, 1)
    q:give_items(player, SCALE_OF_DRAKE_KALIBRAN, 1)
  elseif event == "30750-05.htm" then
    st:set_cond(7, true)
    q:take_items(player, EGG_OF_DRAKE_KALIBRAN, 19)
    q:take_items(player, SCALE_OF_DRAKE_KALIBRAN, 1)
  elseif event == "30751-03.htm" then
    st:set_cond(6, true)
    q:take_items(player, JUICE_OF_MONKSHOOD, 1)
    q:give_items(player, SCALE_OF_WYVERN_SUZET, 1)
  elseif event == "30752-02.htm" then
    st:set_cond(6, true)
    q:take_items(player, JUICE_OF_MONKSHOOD, 1)
    q:give_items(player, SCALE_OF_WYVERN_SHAMHAI, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    local sw2 = npc:id()
    if sw2 == COOPER then
      htmltext = player:level() >= 35 and "30829-01.htm" or "30829-03.htm"
    elseif sw2 == MIMYU then
      _counter = _counter + 1
      local loc = LOCATIONS[math.fmod(_counter, 3) + 1]
      npc:teleport(loc.x, loc.y, loc.z)
      return nil
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw3 = npc:id()
    if sw3 == COOPER then
      htmltext = "30829-04.htm"
    elseif sw3 == CRONOS then
      if cond == 1 then
        htmltext = "30610-01.htm"
      elseif st:get_int("deluxestone") == 2 then
        htmltext = "30610-10.htm"
      elseif cond == 2 then
        if q:has_at_least_one_quest_item(player, FAIRY_STONE, DELUXE_FAIRY_STONE) then
          if st:get_int("deluxestone") == 1 then
            htmltext = "30610-14.htm"
          else
            htmltext = "30610-08.htm"
            st:set_cond(3, true)
          end
        else
          htmltext = "30610-07.htm"
        end
      elseif cond == 3 then
        htmltext = "30610-09.htm"
      elseif cond == 4 and q:has_at_least_one_quest_item(player, FAIRY_STONE, DELUXE_FAIRY_STONE) then
        htmltext = "30610-11.htm"
      end
    elseif sw3 == MARIA then
      if q:has_at_least_one_quest_item(player, FAIRY_STONE, DELUXE_FAIRY_STONE) then
        htmltext = "30608-06.htm"
      elseif cond == 2 then
        if q:has_quest_items(player, FAIRY_STONE_LIST) then
          htmltext = checkItems(player, false) and "30608-02.htm" or "30608-01.htm"
        elseif q:has_quest_items(player, DELUXE_FAIRY_STONE_LIST) then
          htmltext = checkItems(player, true) and "30608-04.htm" or "30608-01.htm"
        end
      end
    elseif sw3 == BYRON then
      local deluxestone = st:get_int("deluxestone")
      if deluxestone == 1 then
        if q:has_quest_items(player, FAIRY_STONE) then
          htmltext = "30711-05.htm"
          st:set_cond(4, true)
          st:unset("deluxestone")
        elseif q:has_quest_items(player, DELUXE_FAIRY_STONE) then
          htmltext = "30711-06.htm"
          st:set_cond(4, true)
          st:unset("deluxestone")
        else
          htmltext = "30711-10.htm"
        end
      elseif deluxestone == 2 then
        htmltext = "30711-09.htm"
      elseif cond == 3 then
        htmltext = "30711-01.htm"
      elseif cond == 4 then
        if q:has_quest_items(player, FAIRY_STONE) then
          htmltext = "30711-07.htm"
        elseif q:has_quest_items(player, DELUXE_FAIRY_STONE) then
          htmltext = "30711-08.htm"
        end
      end
    elseif sw3 == MIMYU then
      if cond == 4 then
        if st:get_int("mimyu") == 1 then
          htmltext = "30747-06.htm"
        elseif q:has_quest_items(player, FAIRY_STONE) then
          htmltext = "30747-01.htm"
        elseif q:has_quest_items(player, DELUXE_FAIRY_STONE) then
          htmltext = "30747-03.htm"
        end
      elseif cond == 5 then
        htmltext = "30747-08.htm"
      elseif cond == 6 then
        local eggs = q:get_quest_items_count(player, EGG_OF_DRAKE_EXARION) + q:get_quest_items_count(player, EGG_OF_DRAKE_ZWOV) + q:get_quest_items_count(player, EGG_OF_DRAKE_KALIBRAN) + q:get_quest_items_count(player, EGG_OF_WYVERN_SUZET) + q:get_quest_items_count(player, EGG_OF_WYVERN_SHAMHAI)
        if eggs < 20 then
          htmltext = "30747-09.htm"
        else
          htmltext = "30747-10.htm"
        end
      elseif cond == 7 then
        htmltext = "30747-11.htm"
      else
        _counter = _counter + 1
        local loc = LOCATIONS[math.fmod(_counter, 3) + 1]
        npc:teleport(loc.x, loc.y, loc.z)
        return nil
      end
    elseif sw3 == EXARION then
      if cond == 5 then
        htmltext = "30748-01.htm"
      elseif cond == 6 then
        if q:get_quest_items_count(player, EGG_OF_DRAKE_EXARION) < 20 then
          htmltext = "30748-03.htm"
        else
          htmltext = "30748-04.htm"
          st:set_cond(7, true)
          q:take_items(player, EGG_OF_DRAKE_EXARION, 19)
          q:take_items(player, SCALE_OF_DRAKE_EXARION, 1)
        end
      elseif cond == 7 then
        htmltext = "30748-05.htm"
      end
    elseif sw3 == ZWOV then
      if cond == 5 then
        htmltext = "30749-01.htm"
      elseif cond == 6 then
        if q:get_quest_items_count(player, EGG_OF_DRAKE_ZWOV) < 20 then
          htmltext = "30749-03.htm"
        else
          htmltext = "30749-04.htm"
          st:set_cond(7, true)
          q:take_items(player, EGG_OF_DRAKE_ZWOV, 19)
          q:take_items(player, SCALE_OF_DRAKE_ZWOV, 1)
        end
      elseif cond == 7 then
        htmltext = "30749-05.htm"
      end
    elseif sw3 == KALIBRAN then
      if cond == 5 then
        htmltext = "30750-01.htm"
      elseif cond == 6 then
        htmltext = q:get_quest_items_count(player, EGG_OF_DRAKE_KALIBRAN) < 20 and "30750-03.htm" or "30750-04.htm"
      elseif cond == 7 then
        htmltext = "30750-06.htm"
      end
    elseif sw3 == SUZET then
      if cond == 5 then
        htmltext = "30751-01.htm"
      elseif cond == 6 then
        if q:get_quest_items_count(player, EGG_OF_WYVERN_SUZET) < 20 then
          htmltext = "30751-04.htm"
        else
          htmltext = "30751-05.htm"
          st:set_cond(7, true)
          q:take_items(player, EGG_OF_WYVERN_SUZET, 19)
          q:take_items(player, SCALE_OF_WYVERN_SUZET, 1)
        end
      elseif cond == 7 then
        htmltext = "30751-06.htm"
      end
    elseif sw3 == SHAMHAI then
      if cond == 5 then
        htmltext = "30752-01.htm"
      elseif cond == 6 then
        if q:get_quest_items_count(player, EGG_OF_WYVERN_SHAMHAI) < 20 then
          htmltext = "30752-03.htm"
        else
          htmltext = "30752-04.htm"
          st:set_cond(7, true)
          q:take_items(player, EGG_OF_WYVERN_SHAMHAI, 19)
          q:take_items(player, SCALE_OF_WYVERN_SHAMHAI, 1)
        end
      elseif cond == 7 then
        htmltext = "30752-05.htm"
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
  if sw == 20231 then
    if q:has_quest_items(player, FAIRY_STONE_LIST) then
      if q:get_quest_items_count(player, TOAD_LORD_BACK_SKIN) < 10 and rnd(10) < 3 then
        q:give_items(player, TOAD_LORD_BACK_SKIN, 1)
        if q:get_quest_items_count(player, TOAD_LORD_BACK_SKIN) < 10 then
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        end
      end
    elseif q:has_quest_items(player, DELUXE_FAIRY_STONE_LIST) and q:get_quest_items_count(player, TOAD_LORD_BACK_SKIN) < 20 and rnd(10) < 3 then
      q:give_items(player, TOAD_LORD_BACK_SKIN, 1)
      if q:get_quest_items_count(player, TOAD_LORD_BACK_SKIN) < 20 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20580 then
    if q:has_quest_items(player, SCALE_OF_DRAKE_EXARION) and q:get_quest_items_count(player, EGG_OF_DRAKE_EXARION) < 20 then
      if rnd(10) < 5 then
        q:give_items(player, EGG_OF_DRAKE_EXARION, 1)
        if q:get_quest_items_count(player, EGG_OF_DRAKE_EXARION) < 20 then
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        end
      else
        npc:broadcast_say(ChatType.GENERAL, "If the eggs get taken, we're dead!")
      end
    end
  elseif sw == 20233 then
    if q:has_quest_items(player, SCALE_OF_DRAKE_ZWOV) and q:get_quest_items_count(player, EGG_OF_DRAKE_ZWOV) < 20 and rnd(10) < 5 then
      q:give_items(player, EGG_OF_DRAKE_ZWOV, 1)
      if q:get_quest_items_count(player, EGG_OF_DRAKE_ZWOV) < 20 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20551 then
    if q:has_quest_items(player, SCALE_OF_DRAKE_KALIBRAN) and q:get_quest_items_count(player, EGG_OF_DRAKE_KALIBRAN) < 20 then
      if rnd(10) < 5 then
        q:give_items(player, EGG_OF_DRAKE_KALIBRAN, 1)
        if q:get_quest_items_count(player, EGG_OF_DRAKE_KALIBRAN) < 20 then
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        end
      else
        npc:broadcast_say(ChatType.GENERAL, "Hey! Everybody watch the eggs!")
      end
    end
  elseif sw == 20270 then
    if q:has_quest_items(player, SCALE_OF_WYVERN_SUZET) and q:get_quest_items_count(player, EGG_OF_WYVERN_SUZET) < 20 then
      if rnd(10) < 5 then
        q:give_items(player, EGG_OF_WYVERN_SUZET, 1)
        if q:get_quest_items_count(player, EGG_OF_WYVERN_SUZET) < 20 then
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        end
      else
        npc:broadcast_say(ChatType.GENERAL, "I thought I'd caught one share... Whew!")
      end
    end
  elseif sw == 20202 then
    if q:has_quest_items(player, SCALE_OF_WYVERN_SHAMHAI) and q:get_quest_items_count(player, EGG_OF_WYVERN_SHAMHAI) < 20 then
      q:give_items(player, EGG_OF_WYVERN_SHAMHAI, 1)
      if q:get_quest_items_count(player, EGG_OF_WYVERN_SHAMHAI) < 20 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20589 or sw == 20590 or sw == 20591 or sw == 20592 or sw == 20593 or sw == 20594 or sw == 20595 or sw == 20596 or sw == 20597 or sw == 20598 or sw == 20599 then
    if q:has_quest_items(player, DELUXE_FAIRY_STONE) and rnd(100) < 30 then
      st:set("deluxestone", "2")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:take_items(player, DELUXE_FAIRY_STONE, 1)
      npc:broadcast_say(ChatType.GENERAL, "The stone... the Elven stone... broke...")
    end
  end
end)

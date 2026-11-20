-- Translated by questport from quests/Q00348_AnArrogantSearch (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(348, "Q00348_AnArrogantSearch", "An Arrogant Search")

-- NPCs
local HANELLIN = 30864
local CLAUDIA_ATHEBALDT = 31001
local MARTIEN = 30645
local HARNE = 30144
local ARK_GUARDIAN_CORPSE = 30980
local HOLY_ARK_OF_SECRECY_1 = 30977
local HOLY_ARK_OF_SECRECY_2 = 30978
local HOLY_ARK_OF_SECRECY_3 = 30979
local GUSTAV_ATHEBALDT = 30760
local HARDIN = 30832
local IASON_HEINE = 30969
-- Monsters
local LESSER_GIANT_MAGE = 20657
local LESSER_GIANT_ELDER = 20658
local PLATINUM_TRIBE_SHAMAN = 20828
local PLATINUM_TRIBE_OVERLORD = 20829
local GUARDIAN_ANGEL = 20859
local SEAL_ANGEL = 20860
-- Quest Monsters
local ANGEL_KILLER = 27184
local ARK_GUARDIAN_ELBEROTH = 27182
local ARK_GUARDIAN_SHADOW_FANG = 27183
-- Items
local TITAN_POWERSTONE = 4287
local HANELLIN_FIRST_LETTER = 4288
local HANELLIN_SECOND_LETTER = 4289
local HANELLIN_THIRD_LETTER = 4290
local FIRST_KEY_OF_ARK = 4291
local SECOND_KEY_OF_ARK = 4292
local THIRD_KEY_OF_ARK = 4293
local BOOK_OF_SAINT = 4397
local BLOOD_OF_SAINT = 4398
local BOUGH_OF_SAINT = 4399
local WHITE_FABRIC_TRIBE = 4294
local WHITE_FABRIC_ANGELS = 5232
local BLOODED_FABRIC = 4295
local ANTIDOTE = 1831
local HEALING_POTION = 1061
-- NPCs instances, in order to avoid infinite instances creation speaking to chests.
local _elberoth
local _shadowFang
local _angelKiller

q:register_quest_items(TITAN_POWERSTONE, HANELLIN_FIRST_LETTER, HANELLIN_SECOND_LETTER, HANELLIN_THIRD_LETTER, FIRST_KEY_OF_ARK, SECOND_KEY_OF_ARK, THIRD_KEY_OF_ARK, BOOK_OF_SAINT, BLOOD_OF_SAINT, BOUGH_OF_SAINT, WHITE_FABRIC_TRIBE, WHITE_FABRIC_ANGELS)
q:add_start_npc(HANELLIN)
q:add_talk_id(HANELLIN, CLAUDIA_ATHEBALDT, MARTIEN, HARNE, HOLY_ARK_OF_SECRECY_1, HOLY_ARK_OF_SECRECY_2, HOLY_ARK_OF_SECRECY_3, ARK_GUARDIAN_CORPSE, GUSTAV_ATHEBALDT, HARDIN, IASON_HEINE)
q:add_spawn_id(ARK_GUARDIAN_ELBEROTH, ARK_GUARDIAN_SHADOW_FANG, ANGEL_KILLER)
q:add_attack_id(ARK_GUARDIAN_ELBEROTH, ARK_GUARDIAN_SHADOW_FANG, ANGEL_KILLER, PLATINUM_TRIBE_SHAMAN, PLATINUM_TRIBE_OVERLORD)
q:add_kill_id(LESSER_GIANT_MAGE, LESSER_GIANT_ELDER, ARK_GUARDIAN_ELBEROTH, ARK_GUARDIAN_SHADOW_FANG, ANGEL_KILLER, PLATINUM_TRIBE_SHAMAN, PLATINUM_TRIBE_OVERLORD, GUARDIAN_ANGEL, SEAL_ANGEL)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30864-05.htm" then
    st:start_quest()
    st:set_cond(2)
    st:set("points", "0")
  elseif event == "30864-09.htm" then
    st:set_cond(4, true)
    q:take_items(player, TITAN_POWERSTONE, 1)
  elseif event == "30864-17.htm" then
    st:set_cond(5, true)
    q:give_items(player, HANELLIN_FIRST_LETTER, 1)
    q:give_items(player, HANELLIN_SECOND_LETTER, 1)
    q:give_items(player, HANELLIN_THIRD_LETTER, 1)
  elseif event == "30864-36.htm" then
    st:set_cond(24, true)
    q:give_adena(player, rnd(1, 2) * 12000, true)
  elseif event == "30864-37.htm" then
    st:set_cond(25, true)
  elseif event == "30864-51.htm" then
    st:set_cond(26, true)
    q:give_items(player, WHITE_FABRIC_ANGELS, q:has_quest_items(player, BLOODED_FABRIC) and 9 or 10)
  elseif event == "30864-58.htm" then
    st:set_cond(27, true)
  elseif event == "30864-57.htm" then
    st:exit_quest(true, true)
  elseif event == "30864-56.htm" then
    st:set_cond(29, true)
    st:set("gustav", "0") -- st.unset doesn't work.
    st:set("hardin", "0")
    st:set("iason", "0")
    q:give_items(player, WHITE_FABRIC_ANGELS, 10)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if q:has_quest_items(player, BLOODED_FABRIC) then
      htmltext = "30864-00.htm"
    elseif player:level() < 60 then
      htmltext = "30864-01.htm"
    else
      htmltext = "30864-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == HANELLIN then
      if cond == 1 then
        htmltext = "30864-02.htm"
      elseif cond == 2 then
        htmltext = not q:has_quest_items(player, TITAN_POWERSTONE) and "30864-06.htm" or "30864-07.htm"
      elseif cond == 4 then
        htmltext = "30864-09.htm"
      elseif cond > 4 and cond < 21 then
        htmltext = q:has_at_least_one_quest_item(player, BOOK_OF_SAINT, BLOOD_OF_SAINT, BOUGH_OF_SAINT) and "30864-28.htm" or "30864-24.htm"
      elseif cond == 21 then
        htmltext = "30864-29.htm"
        st:set_cond(22, true)
        q:take_items(player, BOOK_OF_SAINT, 1)
        q:take_items(player, BLOOD_OF_SAINT, 1)
        q:take_items(player, BOUGH_OF_SAINT, 1)
      elseif cond == 22 then
        if q:has_quest_items(player, WHITE_FABRIC_TRIBE) then
          htmltext = "30864-31.htm"
        elseif q:get_quest_items_count(player, ANTIDOTE) < 5 or not q:has_quest_items(player, HEALING_POTION) then
          htmltext = "30864-30.htm"
        else
          htmltext = "30864-31.htm"
          q:take_items(player, ANTIDOTE, 5)
          q:take_items(player, HEALING_POTION, 1)
          q:give_items(player, WHITE_FABRIC_TRIBE, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif cond == 24 then
        htmltext = "30864-38.htm"
      elseif cond == 25 then
        if q:has_quest_items(player, WHITE_FABRIC_TRIBE) then
          htmltext = "30864-39.htm"
        elseif q:has_quest_items(player, BLOODED_FABRIC) then
          htmltext = "30864-49.htm"
        else
          st:exit_quest(true, true)
        end
      elseif cond == 26 then
        local count = q:get_quest_items_count(player, BLOODED_FABRIC)
        if count + q:get_quest_items_count(player, WHITE_FABRIC_ANGELS) < 10 then
          htmltext = "30864-54.htm"
          q:take_items(player, BLOODED_FABRIC, -1)
          q:give_adena(player, 1000 * count + 4000, true)
          st:exit_quest(true)
        elseif count < 10 then
          htmltext = "30864-52.htm"
        elseif count >= 10 then
          htmltext = "30864-53.htm"
        end
      elseif cond == 27 then
        if st:get_int("gustav") + st:get_int("hardin") + st:get_int("iason") == 3 then
          htmltext = "30864-60.htm"
          st:set_cond(28, true)
          q:give_adena(player, 49000, true)
        elseif q:has_quest_items(player, BLOODED_FABRIC) and st:get_int("usedonbaium") ~= 1 then
          htmltext = "30864-59.htm"
        else
          htmltext = "30864-61.htm"
          st:exit_quest(true, true)
        end
      elseif cond == 28 then
        htmltext = "30864-55.htm"
      elseif cond == 29 then
        local count = q:get_quest_items_count(player, BLOODED_FABRIC)
        if count + q:get_quest_items_count(player, WHITE_FABRIC_ANGELS) < 10 then
          htmltext = "30864-54.htm"
          q:take_items(player, BLOODED_FABRIC, -1)
          q:give_adena(player, 5000 * count, true)
          st:exit_quest(true, true)
        elseif count < 10 then
          htmltext = "30864-52.htm"
        elseif count >= 10 then
          htmltext = "30864-53.htm"
        end
      end
    elseif sw2 == GUSTAV_ATHEBALDT then
      if cond == 27 then
        if q:get_quest_items_count(player, BLOODED_FABRIC) >= 3 and st:get_int("gustav") == 0 then
          st:set("gustav", "1")
          htmltext = "30760-01.htm"
          q:take_items(player, BLOODED_FABRIC, 3)
        elseif st:get_int("gustav") == 1 then
          htmltext = "30760-02.htm"
        else
          htmltext = "30760-03.htm"
          st:set("usedonbaium", "1")
        end
      end
    elseif sw2 == HARDIN then
      if cond == 27 then
        if q:has_quest_items(player, BLOODED_FABRIC) and st:get_int("hardin") == 0 then
          st:set("hardin", "1")
          htmltext = "30832-01.htm"
          q:take_items(player, BLOODED_FABRIC, 1)
        elseif st:get_int("hardin") == 1 then
          htmltext = "30832-02.htm"
        else
          htmltext = "30832-03.htm"
          st:set("usedonbaium", "1")
        end
      end
    elseif sw2 == IASON_HEINE then
      if cond == 27 then
        if q:get_quest_items_count(player, BLOODED_FABRIC) >= 6 and st:get_int("iason") == 0 then
          st:set("iason", "1")
          htmltext = "30969-01.htm"
          q:take_items(player, BLOODED_FABRIC, 6)
        elseif st:get_int("iason") == 1 then
          htmltext = "30969-02.htm"
        else
          htmltext = "30969-03.htm"
          st:set("usedonbaium", "1")
        end
      end
    elseif sw2 == HARNE then
      if cond >= 5 and cond <= 22 then
        if not q:has_quest_items(player, BLOOD_OF_SAINT) then
          if q:has_quest_items(player, HANELLIN_FIRST_LETTER) then
            htmltext = "30144-01.htm"
            st:set_cond(17, true)
            q:take_items(player, HANELLIN_FIRST_LETTER, 1)
            q:add_radar(player, -418, 44174, -3568)
          elseif not q:has_quest_items(player, FIRST_KEY_OF_ARK) then
            htmltext = "30144-03.htm"
            q:add_radar(player, -418, 44174, -3568)
          else
            htmltext = "30144-04.htm"
          end
        else
          htmltext = "30144-05.htm"
        end
      end
    elseif sw2 == CLAUDIA_ATHEBALDT then
      if cond >= 5 and cond <= 22 then
        if not q:has_quest_items(player, BOOK_OF_SAINT) then
          if q:has_quest_items(player, HANELLIN_SECOND_LETTER) then
            htmltext = "31001-01.htm"
            st:set_cond(9, true)
            q:take_items(player, HANELLIN_SECOND_LETTER, 1)
            q:add_radar(player, 181472, 7158, -2725)
          elseif not q:has_quest_items(player, SECOND_KEY_OF_ARK) then
            htmltext = "31001-03.htm"
            q:add_radar(player, 181472, 7158, -2725)
          else
            htmltext = "31001-04.htm"
          end
        else
          htmltext = "31001-05.htm"
        end
      end
    elseif sw2 == MARTIEN then
      if cond >= 5 and cond <= 22 then
        if not q:has_quest_items(player, BOUGH_OF_SAINT) then
          if q:has_quest_items(player, HANELLIN_THIRD_LETTER) then
            htmltext = "30645-01.htm"
            st:set_cond(13, true)
            q:take_items(player, HANELLIN_THIRD_LETTER, 1)
            q:add_radar(player, 50693, 158674, 376)
          elseif not q:has_quest_items(player, THIRD_KEY_OF_ARK) then
            htmltext = "30645-03.htm"
            q:add_radar(player, 50693, 158674, 376)
          else
            htmltext = "30645-04.htm"
          end
        else
          htmltext = "30645-05.htm"
        end
      end
    elseif sw2 == ARK_GUARDIAN_CORPSE then
      if not q:has_quest_items(player, HANELLIN_FIRST_LETTER) and cond >= 5 and cond <= 22 then
        if not q:has_quest_items(player, FIRST_KEY_OF_ARK) and not q:has_quest_items(player, BLOOD_OF_SAINT) then
          if st:get_int("angelkiller") == 0 then
            htmltext = "30980-01.htm"
            if _angelKiller == nil then
              _angelKiller = J.oid(q:add_spawn(ANGEL_KILLER, npc, false, 0))
            end
            if not st:is_cond(18) then
              st:set_cond(18, true)
            end
          else
            htmltext = "30980-02.htm"
            q:give_items(player, FIRST_KEY_OF_ARK, 1)
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            st:unset("angelkiller")
          end
        else
          htmltext = "30980-03.htm"
        end
      end
    elseif sw2 == HOLY_ARK_OF_SECRECY_1 then
      if not q:has_quest_items(player, HANELLIN_FIRST_LETTER) and cond >= 5 and cond <= 22 then
        if not q:has_quest_items(player, BLOOD_OF_SAINT) then
          if q:has_quest_items(player, FIRST_KEY_OF_ARK) then
            htmltext = "30977-02.htm"
            st:set_cond(20, true)
            q:take_items(player, FIRST_KEY_OF_ARK, 1)
            q:give_items(player, BLOOD_OF_SAINT, 1)
            if q:has_quest_items(player, BOOK_OF_SAINT, BOUGH_OF_SAINT) then
              st:set_cond(21)
            end
          else
            htmltext = "30977-04.htm"
          end
        else
          htmltext = "30977-03.htm"
        end
      end
    elseif sw2 == HOLY_ARK_OF_SECRECY_2 then
      if not q:has_quest_items(player, HANELLIN_SECOND_LETTER) and cond >= 5 and cond <= 22 then
        if not q:has_quest_items(player, BOOK_OF_SAINT) then
          if not q:has_quest_items(player, SECOND_KEY_OF_ARK) then
            htmltext = "30978-01.htm"
            if _elberoth == nil then
              _elberoth = J.oid(q:add_spawn(ARK_GUARDIAN_ELBEROTH, npc, false, 0))
            end
          else
            htmltext = "30978-02.htm"
            st:set_cond(12, true)
            q:take_items(player, SECOND_KEY_OF_ARK, 1)
            q:give_items(player, BOOK_OF_SAINT, 1)
            if q:has_quest_items(player, BLOOD_OF_SAINT, BOUGH_OF_SAINT) then
              st:set_cond(21)
            end
          end
        else
          htmltext = "30978-03.htm"
        end
      end
    elseif sw2 == HOLY_ARK_OF_SECRECY_3 then
      if not q:has_quest_items(player, HANELLIN_THIRD_LETTER) and cond >= 5 and cond <= 22 then
        if not q:has_quest_items(player, BOUGH_OF_SAINT) then
          if not q:has_quest_items(player, THIRD_KEY_OF_ARK) then
            htmltext = "30979-01.htm"
            if _shadowFang == nil then
              _shadowFang = J.oid(q:add_spawn(ARK_GUARDIAN_SHADOW_FANG, npc, false, 0))
            end
          else
            htmltext = "30979-02.htm"
            st:set_cond(16, true)
            q:take_items(player, THIRD_KEY_OF_ARK, 1)
            q:give_items(player, BOUGH_OF_SAINT, 1)
            if q:has_quest_items(player, BLOOD_OF_SAINT, BOOK_OF_SAINT) then
              st:set_cond(21)
            end
          end
        else
          htmltext = "30979-03.htm"
        end
      end
    end
  end
  return htmltext
end)

q:on_spawn(function(npc)
  local sw = npc:id()
  if sw == ARK_GUARDIAN_ELBEROTH then
    npc:broadcast_say(ChatType.GENERAL, "This does not belong to you. Take your hands out!")
  elseif sw == ARK_GUARDIAN_SHADOW_FANG then
    npc:broadcast_say(ChatType.GENERAL, "I don't believe it! Grrr!")
  elseif sw == ANGEL_KILLER then
    npc:broadcast_say(ChatType.GENERAL, "I have the key, do you wish to steal it?")
  end
end)

q:on_attack(function(npc, attacker, damage, isPet)
  local st = q:get_quest_state(attacker, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == ARK_GUARDIAN_ELBEROTH then
    if npc:is_script_value(0) then
      npc:broadcast_say(ChatType.GENERAL, "...I feel very sorry, but I have taken your life.")
      npc:set_script_value(1)
    end
  elseif sw == ARK_GUARDIAN_SHADOW_FANG then
    if npc:is_script_value(0) then
      npc:broadcast_say(ChatType.GENERAL, "I will cover this mountain with your blood!")
      npc:set_script_value(1)
    end
  elseif sw == ANGEL_KILLER then
    if npc:is_script_value(0) then
      npc:broadcast_say(ChatType.GENERAL, "Haha.. Really amusing! As for the key, search the corpse!")
      npc:set_script_value(1)
    end
    if npc:hp() / npc:max_hp() < 0.5 then
      npc:abort_attack()
      npc:broadcast_say(ChatType.GENERAL, "Can't get rid of you... Did you get the key from the corpse?")
      npc:delete()
      st:set_cond(19, true)
      st:set("angelkiller", "1")
      _angelKiller = nil
    end
  elseif sw == PLATINUM_TRIBE_OVERLORD or sw == PLATINUM_TRIBE_SHAMAN then
    local cond = st:get_cond()
    if (cond == 24 or cond == 25) and q:has_quest_items(attacker, WHITE_FABRIC_TRIBE) then
      local points = st:get_int("points") + (npc:id() == PLATINUM_TRIBE_SHAMAN and 60 or 70)
      if points > (cond == 24 and 80000 or 100000) then
        st:set("points", tostring(0))
        q:take_items(attacker, WHITE_FABRIC_TRIBE, 1)
        q:give_items(attacker, BLOODED_FABRIC, 1)
        if cond ~= 24 then
          q:play_sound(attacker, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          st:exit_quest(true, true)
        end
      else
        st:set("points", tostring(points))
      end
    end
  end
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local cond = st:get_cond()
  local sw = npc:id()
  if sw == LESSER_GIANT_ELDER or sw == LESSER_GIANT_MAGE then
    if cond == 2 and rnd(10) < 1 and not q:has_quest_items(player, TITAN_POWERSTONE) then
      q:give_items(player, TITAN_POWERSTONE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == ARK_GUARDIAN_ELBEROTH then
    if cond >= 5 and cond <= 22 and not q:has_quest_items(player, SECOND_KEY_OF_ARK) then
      st:set_cond(11, true)
      q:give_items(player, SECOND_KEY_OF_ARK, 1)
      npc:broadcast_say(ChatType.GENERAL, "Oh, dull-witted.. God, they...")
    end
    _elberoth = nil
  elseif sw == ARK_GUARDIAN_SHADOW_FANG then
    if cond >= 5 and cond <= 22 and not q:has_quest_items(player, THIRD_KEY_OF_ARK) then
      st:set_cond(15, true)
      q:give_items(player, THIRD_KEY_OF_ARK, 1)
      npc:broadcast_say(ChatType.GENERAL, "You do not know.. Seven seals are.. coughs")
    end
    _shadowFang = nil
  elseif sw == PLATINUM_TRIBE_OVERLORD or sw == PLATINUM_TRIBE_SHAMAN then
    if (cond == 24 or cond == 25) and q:has_quest_items(player, WHITE_FABRIC_TRIBE) then
      local points = st:get_int("points") + (npc:id() == PLATINUM_TRIBE_SHAMAN and 600 or 700)
      if points > (cond == 24 and 80000 or 100000) then
        st:set("points", tostring(0))
        q:take_items(player, WHITE_FABRIC_TRIBE, 1)
        q:give_items(player, BLOODED_FABRIC, 1)
        if cond ~= 24 then
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          st:exit_quest(true, true)
        end
      else
        st:set("points", tostring(points))
      end
    end
  elseif sw == SEAL_ANGEL or sw == GUARDIAN_ANGEL then
    if (cond == 26 or cond == 29) and rnd(4) < 1 and q:has_quest_items(player, WHITE_FABRIC_ANGELS) then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, WHITE_FABRIC_ANGELS, 1)
      q:give_items(player, BLOODED_FABRIC, 1)
    end
  elseif sw == ANGEL_KILLER then
    _angelKiller = nil
  end
end)

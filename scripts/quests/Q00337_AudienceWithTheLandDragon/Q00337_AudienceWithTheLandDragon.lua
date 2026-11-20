-- Translated by questport from quests/Q00337_AudienceWithTheLandDragon (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(337, "Q00337_AudienceWithTheLandDragon", "Audience with the Land Dragon")

-- NPCs
local GABRIELLE = 30753
local ORVEN = 30857 -- 1
local KENDRA = 30851 -- 2
local CHAKIRIS = 30705 -- 3
local KAIENA = 30720 -- 4
local MOKE = 30498 -- 1st abyssal
local HELTON = 30678 -- 2nd abyssal
local GILMORE = 30754 -- 3rd abyssal
local THEODRIC = 30755
-- Monsters
local BLOOD_QUEEN = 18001 -- 1
local SACRIFICE_OF_THE_SACRIFICED = 27171 -- 1
local HARIT_LIZARDMAN_SHAMAN = 20644 -- 2
local HARIT_LIZARDMAN_MATRIARCH = 20645 -- 2
local HARIT_LIZARDMAN_ZEALOT = 27172 -- 2
local KRANROT = 20650 -- 3
local HAMRUT = 20649 -- 3
local MARSH_DRAKE = 20680 -- 4
local MARSH_STALKER = 20679 -- 4
local ABYSSAL_JEWEL_1 = 27165 -- 1st abyssal
local JEWEL_GUARDIAN_MARA = 27168
local ABYSSAL_JEWEL_2 = 27166 -- 2nd abyssal
local JEWEL_GUARDIAN_MUSFEL = 27169
local CAVE_MAIDEN_1 = 20134 -- 3rd abyssal
local CAVE_MAIDEN_2 = 20287
local CAVE_KEEPER_1 = 20246
local CAVE_KEEPER_2 = 20277
local ABYSSAL_JEWEL_3 = 27167
local JEWEL_GUARDIAN_PYTON = 27170
-- Items
local FEATHER_OF_GABRIELLE = 3852
local MARK_OF_WATCHMAN = 3864
local REMAINS_OF_SACRIFIED = 3857 -- 1
local TOTEM_OF_LAND_DRAGON = 3858 -- 2
local KRANROT_SKIN = 3855 -- 3
local HAMRUT_LEG = 3856 -- 3
local MARSH_DRAKE_TALONS = 3854 -- 4
local MARSH_STALKER_HORN = 3853 -- 4
local FIRST_FRAGMENT_OF_ABYSS_JEWEL = 3859 -- 1st abyssal
local MARA_FANG = 3862
local SECOND_FRAGMENT_OF_ABYSS_JEWEL = 3860 -- 2nd abyssal
local MUSFEL_FANG = 3863
local HERALD_OF_SLAYER = 3890
local THIRD_FRAGMENT_OF_ABYSS_JEWEL = 3861 -- 3rd abyssal
local PORTAL_STONE = 3865
-- Variables
local _jewel1 = false
local _jewel2 = false
local _jewel3 = false
local DROPS_ON_KILL = {
  {SACRIFICE_OF_THE_SACRIFICED, 1, 1, REMAINS_OF_SACRIFIED},
  {HARIT_LIZARDMAN_ZEALOT, 1, 2, TOTEM_OF_LAND_DRAGON},
  {KRANROT, 1, 3, KRANROT_SKIN},
  {HAMRUT, 1, 3, HAMRUT_LEG},
  {MARSH_DRAKE, 1, 4, MARSH_DRAKE_TALONS},
  {MARSH_STALKER, 1, 4, MARSH_STALKER_HORN},
  {JEWEL_GUARDIAN_MARA, 2, 5, MARA_FANG},
  {JEWEL_GUARDIAN_MUSFEL, 2, 6, MUSFEL_FANG},
}
local DROP_ON_ATTACK = {
  {ABYSSAL_JEWEL_1, 2, 5, FIRST_FRAGMENT_OF_ABYSS_JEWEL, 20, JEWEL_GUARDIAN_MARA},
  {ABYSSAL_JEWEL_2, 2, 6, SECOND_FRAGMENT_OF_ABYSS_JEWEL, 20, JEWEL_GUARDIAN_MUSFEL},
  {ABYSSAL_JEWEL_3, 4, 7, THIRD_FRAGMENT_OF_ABYSS_JEWEL, 3, JEWEL_GUARDIAN_PYTON},
}

q:register_quest_items(FEATHER_OF_GABRIELLE, MARK_OF_WATCHMAN, REMAINS_OF_SACRIFIED, TOTEM_OF_LAND_DRAGON, KRANROT_SKIN, HAMRUT_LEG, MARSH_DRAKE_TALONS, MARSH_STALKER_HORN, FIRST_FRAGMENT_OF_ABYSS_JEWEL, MARA_FANG, SECOND_FRAGMENT_OF_ABYSS_JEWEL, MUSFEL_FANG, HERALD_OF_SLAYER, THIRD_FRAGMENT_OF_ABYSS_JEWEL)
q:add_start_npc(GABRIELLE)
q:add_talk_id(GABRIELLE, ORVEN, KENDRA, CHAKIRIS, KAIENA, MOKE, HELTON, GILMORE, THEODRIC)
q:add_attack_id(ABYSSAL_JEWEL_1, ABYSSAL_JEWEL_2, ABYSSAL_JEWEL_3)
q:add_kill_id(BLOOD_QUEEN, SACRIFICE_OF_THE_SACRIFICED, HARIT_LIZARDMAN_SHAMAN, HARIT_LIZARDMAN_MATRIARCH, HARIT_LIZARDMAN_ZEALOT, KRANROT, HAMRUT, MARSH_DRAKE, MARSH_STALKER, JEWEL_GUARDIAN_MARA, JEWEL_GUARDIAN_MUSFEL, CAVE_MAIDEN_1, CAVE_MAIDEN_2, CAVE_KEEPER_1, CAVE_KEEPER_2, JEWEL_GUARDIAN_PYTON)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30753-05.htm" then
    st:start_quest()
    st:set("drop1", "1")
    st:set("drop2", "1")
    st:set("drop3", "1")
    st:set("drop4", "1")
    q:give_items(player, FEATHER_OF_GABRIELLE, 1)
  elseif event == "30753-09.htm" then
    if q:get_quest_items_count(player, MARK_OF_WATCHMAN) >= 4 then
      st:set_cond(2)
      st:set("drop5", "2")
      st:set("drop6", "2")
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:take_items(player, MARK_OF_WATCHMAN, 4)
    else
      htmltext = nil
    end
  elseif event == "30755-05.htm" then
    if q:has_quest_items(player, THIRD_FRAGMENT_OF_ABYSS_JEWEL) then
      q:take_items(player, THIRD_FRAGMENT_OF_ABYSS_JEWEL, 1)
      q:take_items(player, HERALD_OF_SLAYER, 1)
      q:give_items(player, PORTAL_STONE, 1)
      st:exit_quest(true, true)
    else
      htmltext = nil
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 50 and "30753-02.htm" or "30753-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == GABRIELLE then
      if cond == 1 then
        htmltext = q:get_quest_items_count(player, MARK_OF_WATCHMAN) < 4 and "30753-06.htm" or "30753-08.htm"
      elseif cond == 2 then
        if q:get_quest_items_count(player, MARK_OF_WATCHMAN) < 2 then
          htmltext = "30753-10.htm"
        else
          htmltext = "30753-11.htm"
          st:set_cond(3, true)
          q:take_items(player, FEATHER_OF_GABRIELLE, 1)
          q:take_items(player, MARK_OF_WATCHMAN, 1)
          q:give_items(player, HERALD_OF_SLAYER, 1)
        end
      elseif cond == 3 then
        htmltext = "30753-12.htm"
      elseif cond == 4 then
        htmltext = "30753-13.htm"
      end
    elseif sw2 == ORVEN then
      if cond == 1 then
        if st:get_int("drop1") == 1 then
          if q:has_quest_items(player, REMAINS_OF_SACRIFIED) then
            htmltext = "30857-02.htm"
            st:unset("drop1")
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
            q:take_items(player, REMAINS_OF_SACRIFIED, 1)
            q:give_items(player, MARK_OF_WATCHMAN, 1)
          else
            htmltext = "30857-01.htm"
          end
        elseif q:get_quest_items_count(player, MARK_OF_WATCHMAN) < 4 then
          htmltext = "30857-03.htm"
        else
          htmltext = "30857-04.htm"
        end
      end
    elseif sw2 == KENDRA then
      if cond == 1 then
        if st:get_int("drop2") == 1 then
          if q:has_quest_items(player, TOTEM_OF_LAND_DRAGON) then
            htmltext = "30851-02.htm"
            st:unset("drop2")
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
            q:take_items(player, TOTEM_OF_LAND_DRAGON, 1)
            q:give_items(player, MARK_OF_WATCHMAN, 1)
          else
            htmltext = "30851-01.htm"
          end
        elseif q:get_quest_items_count(player, MARK_OF_WATCHMAN) < 4 then
          htmltext = "30851-03.htm"
        else
          htmltext = "30851-04.htm"
        end
      end
    elseif sw2 == CHAKIRIS then
      if cond == 1 then
        if st:get_int("drop3") == 1 then
          if q:has_quest_items(player, KRANROT_SKIN, HAMRUT_LEG) then
            htmltext = "30705-02.htm"
            st:unset("drop3")
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
            q:take_items(player, KRANROT_SKIN, 1)
            q:take_items(player, HAMRUT_LEG, 1)
            q:give_items(player, MARK_OF_WATCHMAN, 1)
          else
            htmltext = "30705-01.htm"
          end
        elseif q:get_quest_items_count(player, MARK_OF_WATCHMAN) < 4 then
          htmltext = "30705-03.htm"
        else
          htmltext = "30705-04.htm"
        end
      end
    elseif sw2 == KAIENA then
      if cond == 1 then
        if st:get_int("drop4") == 1 then
          if q:has_quest_items(player, MARSH_DRAKE_TALONS, MARSH_STALKER_HORN) then
            htmltext = "30720-02.htm"
            st:unset("drop4")
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
            q:take_items(player, MARSH_DRAKE_TALONS, 1)
            q:take_items(player, MARSH_STALKER_HORN, 1)
            q:give_items(player, MARK_OF_WATCHMAN, 1)
          else
            htmltext = "30720-01.htm"
          end
        elseif q:get_quest_items_count(player, MARK_OF_WATCHMAN) < 4 then
          htmltext = "30720-03.htm"
        else
          htmltext = "30720-04.htm"
        end
      end
    elseif sw2 == MOKE then
      if cond == 2 then
        local sw3 = st:get_int("drop5")
        if sw3 == 2 then
          htmltext = "30498-01.htm"
          st:set("drop5", "1")
        elseif sw3 == 1 then
          if q:has_quest_items(player, FIRST_FRAGMENT_OF_ABYSS_JEWEL, MARA_FANG) then
            htmltext = "30498-03.htm"
            st:unset("drop5")
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
            q:take_items(player, FIRST_FRAGMENT_OF_ABYSS_JEWEL, 1)
            q:take_items(player, MARA_FANG, 1)
            q:give_items(player, MARK_OF_WATCHMAN, 1)
          else
            htmltext = "30498-02.htm"
          end
        elseif sw3 == 0 then
          if q:get_quest_items_count(player, MARK_OF_WATCHMAN) < 2 then
            htmltext = "30498-04.htm"
          else
            htmltext = "30498-05.htm"
          end
        end
      end
    elseif sw2 == HELTON then
      if cond == 2 then
        local sw4 = st:get_int("drop6")
        if sw4 == 2 then
          htmltext = "30678-01.htm"
          st:set("drop6", "1")
        elseif sw4 == 1 then
          if q:has_quest_items(player, SECOND_FRAGMENT_OF_ABYSS_JEWEL, MUSFEL_FANG) then
            htmltext = "30678-03.htm"
            st:unset("drop6")
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
            q:take_items(player, SECOND_FRAGMENT_OF_ABYSS_JEWEL, 1)
            q:take_items(player, MUSFEL_FANG, 1)
            q:give_items(player, MARK_OF_WATCHMAN, 1)
          else
            htmltext = "30678-02.htm"
          end
        elseif sw4 == 0 then
          if q:get_quest_items_count(player, MARK_OF_WATCHMAN) < 2 then
            htmltext = "30678-04.htm"
          else
            htmltext = "30678-05.htm"
          end
        end
      end
    elseif sw2 == GILMORE then
      if cond == 1 or cond == 2 then
        htmltext = "30754-01.htm"
      elseif cond == 3 then
        htmltext = "30754-02.htm"
        st:set_cond(4, true)
        st:set("drop7", "1")
      elseif cond == 4 then
        htmltext = not q:has_quest_items(player, THIRD_FRAGMENT_OF_ABYSS_JEWEL) and "30754-04.htm" or "30754-05.htm"
      end
    elseif sw2 == THEODRIC then
      if cond == 1 or cond == 2 then
        htmltext = "30755-01.htm"
      elseif cond == 3 then
        htmltext = "30755-02.htm"
      elseif cond == 4 then
        htmltext = not q:has_quest_items(player, THIRD_FRAGMENT_OF_ABYSS_JEWEL) and "30755-03.htm" or "30755-04.htm"
      end
    end
  end
  return htmltext
end)

q:on_attack(function(npc, attacker, damage, isPet)
  local st = q:get_quest_state(attacker, false)
  if st == nil or not st:is_started() then
    return
  end
  local npcId = npc:id()
  for _, npcInfo in ipairs(DROP_ON_ATTACK) do
    if not (npcInfo[1] ~= npcId) then
      if npcInfo[2] ~= st:get_cond() then
        break
      end
      local percentHp = (npc:hp() + damage) * 100 / npc:max_hp()
      -- reward jewel fragment
      if percentHp < 33 then
        if rnd(100) < 33 and st:get_int("drop" .. npcInfo[3]) == 1 then
          local itemId = npcInfo[4]
          if not q:has_quest_items(attacker, itemId) then
            q:give_items(attacker, itemId, 1)
            q:play_sound(attacker, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        end
      elseif percentHp < 66 then
        if rnd(100) < 33 and st:get_int("drop" .. npcInfo[3]) == 1 then
          local spawn
          if npcId == ABYSSAL_JEWEL_3 then
            spawn = _jewel3
          elseif npcId == ABYSSAL_JEWEL_2 then
            spawn = _jewel2
          else
            spawn = _jewel1
          end
          if spawn then
            for i = 0, npcInfo[5] - 1 do
              local mob = q:add_spawn(npcInfo[6], {x = npc:x() + rnd(-150, 150), y = npc:y() + rnd(-150, 150), z = npc:z(), heading = npc:heading()}, true, 60000)
              mob:set_running()
              mob:add_damage_hate(attacker, 0, 500)
              mob:attack(attacker)
            end
            if npcId == ABYSSAL_JEWEL_3 then
              _jewel3 = false
            elseif npcId == ABYSSAL_JEWEL_2 then
              _jewel2 = false
            else
              _jewel1 = false
            end
          end
        end
      elseif percentHp > 90 then
        if npcId == ABYSSAL_JEWEL_3 then
          _jewel3 = true
        elseif npcId == ABYSSAL_JEWEL_2 then
          _jewel2 = true
        else
          _jewel1 = true
        end
      end
      break
    end
  end
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local cond = st:get_cond()
  local npcId = npc:id()
  if npcId == SACRIFICE_OF_THE_SACRIFICED or npcId == HARIT_LIZARDMAN_ZEALOT or npcId == KRANROT or npcId == HAMRUT or npcId == MARSH_DRAKE or npcId == MARSH_STALKER or npcId == JEWEL_GUARDIAN_MARA or npcId == JEWEL_GUARDIAN_MUSFEL then
    for _, npcInfo in ipairs(DROPS_ON_KILL) do
      if not (npcInfo[1] ~= npcId) then
        if npcInfo[2] == cond and st:get_int("drop" .. npcInfo[3]) == 1 then
          local itemId = npcInfo[4]
          if not q:has_quest_items(player, itemId) then
            q:give_items(player, itemId, 1)
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        end
        break
      end
    end
  elseif npcId == BLOOD_QUEEN then
    if cond == 1 and st:get_int("drop1") == 1 and not q:has_quest_items(player, REMAINS_OF_SACRIFIED) then
      for i = 0, 7 do
        q:add_spawn(SACRIFICE_OF_THE_SACRIFICED, {x = npc:x() + rnd(-100, 100), y = npc:y() + rnd(-100, 100), z = npc:z(), heading = npc:heading()}, true, 60000)
      end
    end
  elseif npcId == HARIT_LIZARDMAN_SHAMAN or npcId == HARIT_LIZARDMAN_MATRIARCH then
    if cond == 1 and rnd(5) == 0 and st:get_int("drop2") == 1 and not q:has_quest_items(player, TOTEM_OF_LAND_DRAGON) then
      for i = 0, 2 do
        q:add_spawn(HARIT_LIZARDMAN_ZEALOT, {x = npc:x() + rnd(-50, 50), y = npc:y() + rnd(-50, 50), z = npc:z(), heading = npc:heading()}, true, 60000)
      end
    end
  elseif npcId == CAVE_MAIDEN_1 or npcId == CAVE_MAIDEN_2 or npcId == CAVE_KEEPER_1 or npcId == CAVE_KEEPER_2 then
    if cond == 4 and rnd(5) == 0 and not q:has_quest_items(player, THIRD_FRAGMENT_OF_ABYSS_JEWEL) then
      q:add_spawn(ABYSSAL_JEWEL_3, {x = npc:x() + rnd(-50, 50), y = npc:y() + rnd(-50, 50), z = npc:z(), heading = npc:heading()}, true, 60000)
    end
  end
end)

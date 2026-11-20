-- Translated by questport from quests/Q00022_TragedyInVonHellmannForest (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(22, "Q00022_TragedyInVonHellmannForest", "Tragedy in von Hellmann Forest")

-- NPCs
local INNOCENTIN = 31328
local TIFAREN = 31334
local WELL = 31527
local GHOST_OF_PRIEST = 31528
local GHOST_OF_ADVENTURER = 31529
-- Monsters
local MOBS = {21553, 21554, 21555, 21556, 21561}
local SOUL_OF_WELL = 27217
-- Items
local CROSS_OF_EINHASAD = 7141
local LOST_SKULL_OF_ELF = 7142
local LETTER_OF_INNOCENTIN = 7143
local JEWEL_OF_ADVENTURER_1 = 7144
local JEWEL_OF_ADVENTURER_2 = 7145
local SEALED_REPORT_BOX = 7146
local REPORT_BOX = 7147
-- Misc
local MIN_LEVEL = 63
local PRIEST_LOC = {x = 38354, y = -49777, z = -1128}
local SOUL_WELL_LOC = {x = 34706, y = -54590, z = -2054}
local _tifarenOwner = 0
local _soulWellNpc = nil

q:add_kill_id(MOBS)
q:add_kill_id(SOUL_OF_WELL)
q:add_attack_id(SOUL_OF_WELL)
q:add_start_npc(TIFAREN)
q:add_talk_id(INNOCENTIN, TIFAREN, WELL, GHOST_OF_PRIEST, GHOST_OF_ADVENTURER)
q:register_quest_items(LOST_SKULL_OF_ELF, CROSS_OF_EINHASAD, REPORT_BOX, JEWEL_OF_ADVENTURER_1, JEWEL_OF_ADVENTURER_2, SEALED_REPORT_BOX)

q:on_event(function(event, npc, player)
  local qs = player:get_quest_state(q:name())
  local htmltext = nil
  if qs == nil then
    return htmltext
  end
  if event == "31529-02.htm" or event == "31529-04.htm" or event == "31529-05.htm" or event == "31529-06.htm" or event == "31529-07.htm" or event == "31529-09.htm" or event == "31529-13.htm" or event == "31529-13a.htm" or event == "31528-02.htm" or event == "31528-05.htm" or event == "31528-06.htm" or event == "31528-07.htm" or event == "31328-13.htm" or event == "31328-06.htm" or event == "31328-05.htm" or event == "31328-02.htm" or event == "31328-07.htm" or event == "31328-08.htm" or event == "31328-14.htm" or event == "31328-15.htm" or event == "31328-16.htm" or event == "31328-17.htm" or event == "31328-18.htm" or event == "31334-12.htm" then
    htmltext = event
  elseif event == "31334-02.htm" then
    if qs:is_created() then
      local qs2 = player:get_quest_state("Q00021_HiddenTruth")
      if player:level() >= MIN_LEVEL and qs2 ~= nil and qs2:is_completed() then
        htmltext = event
      else
        htmltext = "31334-03.htm"
      end
    end
  elseif event == "31334-04.htm" then
    if qs:is_created() then
      qs:start_quest()
      htmltext = event
    end
  elseif event == "31334-07.htm" then
    if not q:has_quest_items(player, CROSS_OF_EINHASAD) then
      qs:set_cond(2)
      htmltext = event
    else
      htmltext = "31334-06.htm"
      qs:set_cond(3)
    end
  elseif event == "31334-08.htm" then
    if qs:is_cond(3) then
      qs:set_cond(4, true)
      htmltext = event
    end
  elseif event == "31334-13.htm" then
    local cond = qs:get_cond()
    if 5 <= cond and cond <= 7 and q:has_quest_items(player, CROSS_OF_EINHASAD) then
      if _tifarenOwner == 0 then
        _tifarenOwner = player:object_id()
        local ghost2 = q:add_spawn(GHOST_OF_PRIEST, PRIEST_LOC, true, 0)
        ghost2:set_script_value(player:object_id())
        q:start_quest_timer("DESPAWN_GHOST2", 1000 * 120, ghost2, player)
        ghost2:broadcast_say(ChatType.GENERAL, "Did you call me, " .. player:name() .. "?")
        if (cond == 5 or cond == 6) and q:has_quest_items(player, LOST_SKULL_OF_ELF) then
          q:take_items(player, LOST_SKULL_OF_ELF, -1)
          qs:set_cond(7, true)
        end
        htmltext = event
      else
        qs:set_cond(6)
        htmltext = "31334-14.htm"
      end
    end
  elseif event == "31528-04.htm" then
    if npc:get_script_value() == player:object_id() then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
      htmltext = event
    end
  elseif event == "31528-08.htm" then
    local qt = q:get_quest_timer("DESPAWN_GHOST2", npc, player)
    if qt ~= nil and npc:get_script_value() == player:object_id() then
      qt:cancel()
      npc:set_script_value(0)
      q:start_quest_timer("DESPAWN_GHOST2", 1000 * 3, npc, player)
      qs:set_cond(8, true)
      htmltext = event
    end
  elseif event == "DESPAWN_GHOST2" then
    _tifarenOwner = 0
    if npc:get_script_value() ~= 0 then
      -- L2Go: Mobius says the NpcStringId name I_M_CONFUSED_MAYBE_IT_S_TIME_TO_GO_BACK
      -- as text; this is its line, as Q00021 says it.
      npc:broadcast_say(ChatType.GENERAL, "I'm confused! Maybe it's time to go back.")
    end
    npc:delete()
  elseif event == "31328-03.htm" then
    if qs:is_cond(8) then
      q:take_items(player, CROSS_OF_EINHASAD, -1)
      htmltext = event
    end
  elseif event == "31328-09.htm" then
    if qs:is_cond(8) then
      q:give_items(player, LETTER_OF_INNOCENTIN, 1)
      qs:set_cond(9, true)
      htmltext = event
    end
  elseif event == "31328-11.htm" then
    if qs:is_cond(14) and q:has_quest_items(player, REPORT_BOX) then
      q:take_items(player, REPORT_BOX, -1)
      qs:set_cond(15, true)
      htmltext = event
    end
  elseif event == "31328-19.htm" then
    if qs:is_cond(15) then
      qs:set_cond(16, true)
      htmltext = event
    end
  elseif event == "31527-02.htm" then
    if qs:is_cond(10) and _soulWellNpc == nil then
      _soulWellNpc = J.oid(q:add_spawn(SOUL_OF_WELL, SOUL_WELL_LOC, true, 0))
      q:start_quest_timer("activateSoulOfWell", 90000, J.obj(_soulWellNpc), player)
      q:start_quest_timer("despawnSoulOfWell", 120000, J.obj(_soulWellNpc), player)
      J.obj(_soulWellNpc):attack(player)
      htmltext = event
    else
      htmltext = "31527-03.htm"
    end
  elseif event == "activateSoulOfWell" then
    -- this enables onAttack ELSE IF block which allows the player to proceed the quest
    npc:set_script_value(1)
  elseif event == "despawnSoulOfWell" then
    -- if the player fails to proceed the quest in 2 minutes, the soul is unspawned
    if not npc:is_dead() then
      _soulWellNpc = nil
    end
    npc:delete()
  elseif event == "31529-03.htm" then
    if qs:is_cond(9) and q:has_quest_items(player, LETTER_OF_INNOCENTIN) then
      qs:set("memoState", "8")
      htmltext = event
    end
  elseif event == "31529-08.htm" then
    if qs:get_int("memoState") == 8 then
      qs:set("memoState", "9")
      htmltext = event
    end
  elseif event == "31529-11.htm" then
    if qs:get_int("memoState") == 9 then
      q:give_items(player, JEWEL_OF_ADVENTURER_1, 1)
      qs:set_cond(10, true)
      qs:set("memoState", "10")
      htmltext = event
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = npc:id()
  if sw == TIFAREN then
    local sw2 = qs:get_cond()
    if sw2 == 0 then
      if qs:is_created() then
        htmltext = "31334-01.htm"
      elseif qs:is_completed() then
        htmltext = q:get_already_completed_msg(player)
      end
    elseif sw2 == 1 or sw2 == 3 then
      htmltext = "31334-05.htm"
    elseif sw2 == 4 or sw2 == 5 then
      if q:has_quest_items(player, CROSS_OF_EINHASAD) then
        if not q:has_quest_items(player, LOST_SKULL_OF_ELF) then
          htmltext = "31334-09.htm"
        elseif _tifarenOwner == 0 then
          htmltext = "31334-10.htm"
        else
          htmltext = "31334-11.htm"
        end
      end
    elseif sw2 == 6 or sw2 == 7 then
      if q:has_quest_items(player, CROSS_OF_EINHASAD) then
        if _tifarenOwner == 0 then
          htmltext = "31334-17.htm"
        elseif _tifarenOwner == player:object_id() then
          htmltext = "31334-15.htm"
        else
          htmltext = "31334-16.htm"
          qs:set_cond(6)
        end
      end
    elseif sw2 == 8 then
      if q:has_quest_items(player, CROSS_OF_EINHASAD) then
        htmltext = "31334-18.htm"
      end
    end
  elseif sw == GHOST_OF_PRIEST then
    if npc:get_script_value() == player:object_id() then
      htmltext = "31528-01.htm"
    else
      htmltext = "31528-03.htm"
    end
  elseif sw == INNOCENTIN then
    local sw3 = qs:get_cond()
    if sw3 == 2 then
      if not q:has_quest_items(player, CROSS_OF_EINHASAD) then
        q:give_items(player, CROSS_OF_EINHASAD, 1)
        qs:set_cond(3)
        htmltext = "31328-01.htm"
      end
    elseif sw3 == 3 then
      if q:has_quest_items(player, CROSS_OF_EINHASAD) then
        htmltext = "31328-01b.htm"
      end
    elseif sw3 == 8 then
      if q:has_quest_items(player, CROSS_OF_EINHASAD) then
        htmltext = "31328-02.htm"
      else
        htmltext = "31328-04.htm"
      end
    elseif sw3 == 9 then
      htmltext = "31328-09a.htm"
    elseif sw3 == 14 then
      if q:has_quest_items(player, REPORT_BOX) then
        htmltext = "31328-10.htm"
      end
    elseif sw3 == 15 then
      htmltext = "31328-12.htm"
    elseif sw3 == 16 then
      q:add_exp_and_sp(player, 345966, 31578)
      qs:exit_quest(false)
      if player:level() >= MIN_LEVEL then
        htmltext = "31328-20.htm"
      else
        htmltext = "31328-21.htm"
      end
    end
  elseif sw == WELL then
    local sw4 = qs:get_cond()
    if sw4 == 10 then
      if q:has_quest_items(player, JEWEL_OF_ADVENTURER_1) then
        htmltext = "31527-01.htm"
      end
    elseif sw4 == 12 then
      if q:has_quest_items(player, JEWEL_OF_ADVENTURER_2) and not q:has_quest_items(player, SEALED_REPORT_BOX) then
        q:give_items(player, SEALED_REPORT_BOX, 1)
        qs:set_cond(13)
        htmltext = "31527-04.htm"
      end
    elseif sw4 == 13 or sw4 == 14 or sw4 == 15 or sw4 == 16 then
      htmltext = "31527-05.htm"
    end
  elseif sw == GHOST_OF_ADVENTURER then
    local sw5 = qs:get_cond()
    if sw5 == 9 then
      if q:has_quest_items(player, LETTER_OF_INNOCENTIN) then
        local sw6 = qs:get_int("memoState")
        if sw6 == 0 then
          htmltext = "31529-01.htm"
        elseif sw6 == 8 then
          htmltext = "31529-03a.htm"
        elseif sw6 == 9 then
          htmltext = "31529-10.htm"
        end
      end
    elseif sw5 == 10 then
      if q:has_quest_items(player, JEWEL_OF_ADVENTURER_1) then
        local id = qs:get_int("memoState")
        if id == 10 then
          htmltext = "31529-12.htm"
        elseif id == 11 then
          htmltext = "31529-14.htm"
        end
      end
    elseif sw5 == 11 then
      if q:has_quest_items(player, JEWEL_OF_ADVENTURER_2) and not q:has_quest_items(player, SEALED_REPORT_BOX) then
        htmltext = "31529-15.htm"
        qs:set_cond(12)
      end
    elseif sw5 == 13 then
      if q:has_quest_items(player, JEWEL_OF_ADVENTURER_2) and q:has_quest_items(player, SEALED_REPORT_BOX) then
        q:give_items(player, REPORT_BOX, 1)
        q:take_items(player, SEALED_REPORT_BOX, -1)
        q:take_items(player, JEWEL_OF_ADVENTURER_2, -1)
        qs:set_cond(14)
        htmltext = "31529-16.htm"
      end
    elseif sw5 == 14 then
      if q:has_quest_items(player, REPORT_BOX) then
        htmltext = "31529-17.htm"
      end
    end
  end
  return htmltext
end)

q:on_attack(function(npc, attacker, damage, isSummon)
  local qs = attacker:get_quest_state(q:name())
  if qs ~= nil and qs:is_cond(10) and q:has_quest_items(attacker, JEWEL_OF_ADVENTURER_1) then
    if qs:get_int("memoState") == 10 then
      qs:set("memoState", "11")
    elseif npc:is_script_value(1) then
      q:take_items(attacker, JEWEL_OF_ADVENTURER_1, -1)
      q:give_items(attacker, JEWEL_OF_ADVENTURER_2, 1)
      qs:set_cond(11)
    end
  end
end)

q:on_kill(function(npc, killer, isSummon)
  if util.in_range(Config.ALT_PARTY_RANGE, killer, npc, true) then
    if npc:id() == SOUL_OF_WELL then
      _soulWellNpc = nil
    else
      local qs = killer:get_quest_state(q:name())
      if qs ~= nil and qs:is_cond(4) and q:has_quest_items(killer, CROSS_OF_EINHASAD) and not q:has_quest_items(killer, LOST_SKULL_OF_ELF) and rnd(100) < 10 then
        q:give_items(killer, LOST_SKULL_OF_ELF, 1)
        qs:set_cond(5, true)
      end
    end
  end
end)

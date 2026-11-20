-- Translated by questport from quests/Q00226_TestOfTheHealer (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(226, "Q00226_TestOfTheHealer", "Test of the Healer")

-- NPCs
local BANDELLOS = 30473
local SORIUS = 30327
local ALLANA = 30424
local PERRIN = 30428
local GUPU = 30658
local ORPHAN_GIRL = 30659
local WINDY_SHAORING = 30660
local MYSTERIOUS_DARKELF = 30661
local PIPER_LONGBOW = 30662
local SLEIN_SHINING_BLADE = 30663
local KAIN_FLYING_KNIFE = 30664
local KRISTINA = 30665
local DAURIN_HAMMERCRUSH = 30674
-- Monsters
local LETO_LIZARDMAN_LEADER = 27123
local LETO_LIZARDMAN_ASSASSIN = 27124
local LETO_LIZARDMAN_SNIPER = 27125
local LETO_LIZARDMAN_WIZARD = 27126
local LETO_LIZARDMAN_LORD = 27127
local TATOMA = 27134
-- Items
local REPORT_OF_PERRIN = 2810
local KRISTINA_LETTER = 2811
local PICTURE_OF_WINDY = 2812
local GOLDEN_STATUE = 2813
local WINDY_PEBBLES = 2814
local ORDER_OF_SORIUS = 2815
local SECRET_LETTER_1 = 2816
local SECRET_LETTER_2 = 2817
local SECRET_LETTER_3 = 2818
local SECRET_LETTER_4 = 2819
-- Rewards
local MARK_OF_HEALER = 2820
local DIMENSIONAL_DIAMOND = 7562
-- Misc
local _tatoma
local _letoLeader

-- L2Go: Mobius counts the NPCs of npcId the player sees
-- (World.getVisibleObjects) and spawns until there are count of them. The
-- script API cannot list visible objects yet, so this counts the ones this
-- script spawned that are still alive (object IDs per NPC ID); these Leto
-- Lizardmen (27xxx) exist only through this quest.
local _spawned = {}

local function checkSpawn(npcId, count, player)
  local alive = {}
  for _, id in ipairs(_spawned[npcId] or {}) do
    local nearby = J.obj(id)
    if nearby ~= nil and not nearby:is_dead() then
      table.insert(alive, id)
    end
  end
  while #alive < count do
    local spawned = q:add_spawn(npcId, player, true, 300000)
    if spawned == nil then
      break -- the world could not spawn it
    end
    table.insert(alive, J.oid(spawned))
  end
  _spawned[npcId] = alive
end

-- L2Go: the Mysterious Dark Elf's first spawns count for checkSpawn too
-- (Mobius sees them as visible NPCs).
local function addCountedSpawn(npcId, player)
  local spawned = q:add_spawn(npcId, player, true, 300000)
  if spawned ~= nil then
    _spawned[npcId] = _spawned[npcId] or {}
    table.insert(_spawned[npcId], J.oid(spawned))
  end
end

q:register_quest_items(REPORT_OF_PERRIN, KRISTINA_LETTER, PICTURE_OF_WINDY, GOLDEN_STATUE, WINDY_PEBBLES, ORDER_OF_SORIUS, SECRET_LETTER_1, SECRET_LETTER_2, SECRET_LETTER_3, SECRET_LETTER_4)
q:add_start_npc(BANDELLOS)
q:add_talk_id(BANDELLOS, SORIUS, ALLANA, PERRIN, GUPU, ORPHAN_GIRL, WINDY_SHAORING, MYSTERIOUS_DARKELF, PIPER_LONGBOW, SLEIN_SHINING_BLADE, KAIN_FLYING_KNIFE, KRISTINA, DAURIN_HAMMERCRUSH)
q:add_kill_id(LETO_LIZARDMAN_LEADER, LETO_LIZARDMAN_ASSASSIN, LETO_LIZARDMAN_SNIPER, LETO_LIZARDMAN_WIZARD, LETO_LIZARDMAN_LORD, TATOMA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30473-04.htm" then
    st:start_quest()
    q:give_items(player, REPORT_OF_PERRIN, 1)
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30473-04a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30473-09.htm" then
    q:take_items(player, GOLDEN_STATUE, 1)
    q:give_items(player, MARK_OF_HEALER, 1)
    q:add_exp_and_sp(player, 134839, 50000)
    player:social_action(3)
    st:exit_quest(false, true)
  elseif event == "30428-02.htm" then
    st:set_cond(2, true)
    if _tatoma == nil then
      _tatoma = J.oid(q:add_spawn(TATOMA, {x = -93254, y = 147559, z = -2679, heading = 0}, false, 0))
      q:start_quest_timer("tatoma_despawn", 200000, nil, player, false)
    end
  elseif event == "30658-02.htm" then
    if q:get_quest_items_count(player, 57) >= 100000 then
      st:set_cond(7, true)
      q:take_items(player, 57, 100000)
      q:give_items(player, PICTURE_OF_WINDY, 1)
    else
      htmltext = "30658-05.htm"
    end
  elseif event == "30658-03.htm" then
    st:set("gupu", "1")
  elseif event == "30658-07.htm" then
    st:set_cond(9, true)
  elseif event == "30660-03.htm" then
    st:set_cond(8, true)
    q:take_items(player, PICTURE_OF_WINDY, 1)
    q:give_items(player, WINDY_PEBBLES, 1)
  elseif event == "30674-02.htm" then
    st:set_cond(11, true)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_BEFORE_BATTLE)
    q:take_items(player, ORDER_OF_SORIUS, 1)
    if _letoLeader == nil then
      _letoLeader = J.oid(q:add_spawn(LETO_LIZARDMAN_LEADER, {x = -97441, y = 106585, z = -3405, heading = 0}, false, 0))
      q:start_quest_timer("leto_leader_despawn", 200000, nil, player, false)
    end
  elseif event == "30665-02.htm" then
    st:set_cond(22, true)
    q:take_items(player, SECRET_LETTER_1, 1)
    q:take_items(player, SECRET_LETTER_2, 1)
    q:take_items(player, SECRET_LETTER_3, 1)
    q:take_items(player, SECRET_LETTER_4, 1)
    q:give_items(player, KRISTINA_LETTER, 1)
  elseif event == "tatoma_despawn" then
    -- L2Go: the NPC may be gone already (J.obj gives nil then); Java's
    -- deleteMe on a removed NPC does nothing.
    local tatoma = J.obj(_tatoma)
    if tatoma ~= nil then
      tatoma:delete()
    end
    _tatoma = nil
    return nil
  elseif event == "leto_leader_despawn" then
    local leader = J.obj(_letoLeader)
    if leader ~= nil then
      leader:delete()
    end
    _letoLeader = nil
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.KNIGHT and player:class_id() ~= ClassId.ELVEN_KNIGHT and player:class_id() ~= ClassId.CLERIC and player:class_id() ~= ClassId.ORACLE then
      htmltext = "30473-01.htm"
    elseif player:level() < 39 then
      htmltext = "30473-02.htm"
    else
      htmltext = "30473-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == BANDELLOS then
      if cond < 23 then
        htmltext = "30473-05.htm"
      else
        if not q:has_quest_items(player, GOLDEN_STATUE) then
          htmltext = "30473-06.htm"
          q:give_items(player, MARK_OF_HEALER, 1)
          q:add_exp_and_sp(player, 118304, 26250)
          player:social_action(3)
          st:exit_quest(false, true)
        else
          htmltext = "30473-07.htm"
        end
      end
    elseif sw2 == PERRIN then
      if cond < 3 then
        htmltext = "30428-01.htm"
      elseif cond == 3 then
        htmltext = "30428-03.htm"
        st:set_cond(4, true)
        q:take_items(player, REPORT_OF_PERRIN, 1)
      else
        htmltext = "30428-04.htm"
      end
    elseif sw2 == ORPHAN_GIRL then
      htmltext = "30659-0" .. rnd(1, 5) .. ".htm"
    elseif sw2 == ALLANA then
      if cond == 4 then
        htmltext = "30424-01.htm"
        st:set_cond(5, true)
      elseif cond > 4 then
        htmltext = "30424-02.htm"
      end
    elseif sw2 == GUPU then
      if st:get_int("gupu") == 1 and cond ~= 9 then
        htmltext = "30658-07.htm"
        st:set_cond(9, true)
      elseif cond == 5 then
        htmltext = "30658-01.htm"
        st:set_cond(6, true)
      elseif cond == 6 then
        htmltext = "30658-01.htm"
      elseif cond == 7 then
        htmltext = "30658-04.htm"
      elseif cond == 8 then
        htmltext = "30658-06.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:take_items(player, WINDY_PEBBLES, 1)
        q:give_items(player, GOLDEN_STATUE, 1)
      elseif cond > 8 then
        htmltext = "30658-07.htm"
      end
    elseif sw2 == WINDY_SHAORING then
      if cond == 7 then
        htmltext = "30660-01.htm"
      elseif q:has_quest_items(player, WINDY_PEBBLES) then
        htmltext = "30660-04.htm"
      end
    elseif sw2 == SORIUS then
      if cond == 9 then
        htmltext = "30327-01.htm"
        st:set_cond(10, true)
        q:give_items(player, ORDER_OF_SORIUS, 1)
      elseif cond > 9 and cond < 22 then
        htmltext = "30327-02.htm"
      elseif cond == 22 then
        htmltext = "30327-03.htm"
        st:set_cond(23, true)
        q:take_items(player, KRISTINA_LETTER, 1)
      elseif cond == 23 then
        htmltext = "30327-04.htm"
      end
    elseif sw2 == DAURIN_HAMMERCRUSH then
      if cond == 10 then
        htmltext = "30674-01.htm"
      elseif cond == 11 then
        htmltext = "30674-02a.htm"
        if _letoLeader == nil then
          _letoLeader = J.oid(q:add_spawn(LETO_LIZARDMAN_LEADER, {x = -97441, y = 106585, z = -3405, heading = 0}, false, 0))
          q:start_quest_timer("leto_leader_despawn", 200000, nil, player, false)
        end
      elseif cond == 12 then
        htmltext = "30674-03.htm"
        st:set_cond(13, true)
      elseif cond > 12 then
        htmltext = "30674-04.htm"
      end
    elseif sw2 == PIPER_LONGBOW or sw2 == SLEIN_SHINING_BLADE or sw2 == KAIN_FLYING_KNIFE then
      if cond == 13 or cond == 14 then
        htmltext = npc:id() .. "-01.htm"
      elseif cond > 14 and cond < 19 then
        htmltext = npc:id() .. "-02.htm"
      elseif cond > 18 and cond < 22 then
        htmltext = npc:id() .. "-03.htm"
        st:set_cond(21, true)
      end
    elseif sw2 == MYSTERIOUS_DARKELF then
      if cond == 13 then
        htmltext = "30661-01.htm"
        st:set_cond(14)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_BEFORE_BATTLE)
        addCountedSpawn(LETO_LIZARDMAN_ASSASSIN, player)
        addCountedSpawn(LETO_LIZARDMAN_ASSASSIN, player)
        addCountedSpawn(LETO_LIZARDMAN_ASSASSIN, player)
      elseif cond == 14 then
        htmltext = "30661-01.htm"
        checkSpawn(LETO_LIZARDMAN_ASSASSIN, 3, player)
      elseif cond == 15 then
        htmltext = "30661-02.htm"
        st:set_cond(16)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_BEFORE_BATTLE)
        addCountedSpawn(LETO_LIZARDMAN_SNIPER, player)
        addCountedSpawn(LETO_LIZARDMAN_SNIPER, player)
        addCountedSpawn(LETO_LIZARDMAN_SNIPER, player)
      elseif cond == 16 then
        htmltext = "30661-02.htm"
        checkSpawn(LETO_LIZARDMAN_SNIPER, 3, player)
      elseif cond == 17 then
        htmltext = "30661-03.htm"
        st:set_cond(18)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_BEFORE_BATTLE)
        addCountedSpawn(LETO_LIZARDMAN_WIZARD, player)
        addCountedSpawn(LETO_LIZARDMAN_WIZARD, player)
        addCountedSpawn(LETO_LIZARDMAN_LORD, player)
      elseif cond == 18 then
        htmltext = "30661-03.htm"
        checkSpawn(LETO_LIZARDMAN_WIZARD, 2, player)
        checkSpawn(LETO_LIZARDMAN_LORD, 1, player)
      elseif cond == 19 then
        htmltext = "30661-04.htm"
        st:set_cond(20, true)
      elseif cond == 20 or cond == 21 then
        htmltext = "30661-04.htm"
      end
    elseif sw2 == KRISTINA then
      if cond > 18 and cond < 22 then
        htmltext = "30665-01.htm"
      elseif cond > 21 then
        htmltext = "30665-04.htm"
      else
        htmltext = "30665-03.htm"
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
  local cond = st:get_cond()
  local sw = npc:id()
  if sw == TATOMA then
    if cond == 1 or cond == 2 then
      st:set_cond(3, true)
    end
    _tatoma = nil
    q:cancel_quest_timer("tatoma_despawn", nil, player)
  elseif sw == LETO_LIZARDMAN_LEADER then
    if cond == 10 or cond == 11 then
      st:set_cond(12, true)
      q:give_items(player, SECRET_LETTER_1, 1)
    end
    _letoLeader = nil
    q:cancel_quest_timer("leto_leader_despawn", nil, player)
  elseif sw == LETO_LIZARDMAN_ASSASSIN then
    if cond == 13 or cond == 14 then
      st:set_cond(15, true)
      q:give_items(player, SECRET_LETTER_2, 1)
    end
  elseif sw == LETO_LIZARDMAN_SNIPER then
    if cond == 15 or cond == 16 then
      st:set_cond(17, true)
      q:give_items(player, SECRET_LETTER_3, 1)
    end
  elseif sw == LETO_LIZARDMAN_LORD then
    if cond == 17 or cond == 18 then
      st:set_cond(19, true)
      q:give_items(player, SECRET_LETTER_4, 1)
    end
  end
end)

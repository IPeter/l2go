-- Translated by questport from quests/Q00242_PossessorOfAPreciousSoul2 (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(242, "Q00242_PossessorOfAPreciousSoul2", "Possessor of a Precious Soul - 2")
q:requires("subclass")

-- NPCs
local VIRGIL = 31742
local KASSANDRA = 31743
local OGMAR = 31744
local MYSTERIOUS_KNIGHT = 31751
local ANGEL_CORPSE = 31752
local KALIS = 30759
local MATILD = 30738
local CORNERSTONE = 31748
local FALLEN_UNICORN = 31746
local PURE_UNICORN = 31747
-- Monsters
local RESTRAINER_OF_GLORY = 27317
-- Items
local VIRGIL_LETTER = 7677
local GOLDEN_HAIR = 7590
local SORCERY_INGREDIENT = 7596
local ORB_OF_BINDING = 7595
local CARADINE_LETTER = 7678
-- Misc
local _unicorn = false

q:register_quest_items(GOLDEN_HAIR, SORCERY_INGREDIENT, ORB_OF_BINDING)
q:add_start_npc(VIRGIL)
q:add_talk_id(VIRGIL, KASSANDRA, OGMAR, MYSTERIOUS_KNIGHT, ANGEL_CORPSE, KALIS, MATILD, CORNERSTONE, FALLEN_UNICORN, PURE_UNICORN)
q:add_kill_id(RESTRAINER_OF_GLORY)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31743-05.htm" then
    st:set_cond(2, true)
  elseif event == "31744-02.htm" then
    st:set_cond(3, true)
  elseif event == "31751-02.htm" then
    st:set_cond(4, true)
    st:set("angel", "0")
  elseif event == "30759-02.htm" then
    st:set_cond(7, true)
  elseif event == "30759-05.htm" then
    if q:has_quest_items(player, SORCERY_INGREDIENT) then
      st:set("orb", "0")
      st:set("cornerstone", "0")
      st:set_cond(9, true)
      q:take_items(player, GOLDEN_HAIR, 1)
      q:take_items(player, SORCERY_INGREDIENT, 1)
    else
      st:set_cond(7)
      htmltext = "30759-02.htm"
    end
  elseif event == "30738-02.htm" then
    st:set_cond(8, true)
    q:give_items(player, SORCERY_INGREDIENT, 1)
  elseif event == "31748-03.htm" then
    if q:has_quest_items(player, ORB_OF_BINDING) then
      npc:die(npc)
      q:take_items(player, ORB_OF_BINDING, 1)
      local cornerstones = st:get_int("cornerstone")
      cornerstones = cornerstones + 1
      if cornerstones == 4 then
        st:unset("orb")
        st:unset("cornerstone")
        st:set_cond(10, true)
      else
        st:set("cornerstone", tostring(cornerstones))
      end
    else
      htmltext = nil
    end
  elseif event == "spu" then
    q:add_spawn(PURE_UNICORN, {x = 85884, y = -76588, z = -3470, heading = 0}, false, 0)
    return nil
  elseif event == "dspu" then
    npc:stop_respawn()
    npc:delete()
    q:start_quest_timer("sfu", 2000, nil, player, false)
    return nil
  elseif event == "sfu" then
    local unicorn = q:add_spawn(FALLEN_UNICORN, {x = 85884, y = -76588, z = -3470, heading = 0}, false, 0)
    unicorn:start_respawn()
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if q:has_quest_items(player, VIRGIL_LETTER) then
      if not player:is_sub_class_active() or player:level() < 60 then
        htmltext = "31742-02.htm"
      else
        st:start_quest()
        q:take_items(player, VIRGIL_LETTER, 1)
        htmltext = "31742-03.htm"
      end
    end
  elseif sw == State.STARTED then
    repeat
      if not player:is_sub_class_active() then
        break
      end
      local cond = st:get_cond()
      local sw2 = npc:id()
      if sw2 == VIRGIL then
        if cond == 1 then
          htmltext = "31742-04.htm"
        elseif cond == 2 then
          htmltext = "31742-05.htm"
        end
      elseif sw2 == KASSANDRA then
        if cond == 1 then
          htmltext = "31743-01.htm"
        elseif cond == 2 then
          htmltext = "31743-06.htm"
        elseif cond == 11 then
          htmltext = "31743-07.htm"
          q:give_items(player, CARADINE_LETTER, 1)
          q:add_exp_and_sp(player, 455764, 0)
          player:social_action(3)
          st:exit_quest(false, true)
        end
      elseif sw2 == OGMAR then
        if cond == 2 then
          htmltext = "31744-01.htm"
        elseif cond == 3 then
          htmltext = "31744-03.htm"
        end
      elseif sw2 == MYSTERIOUS_KNIGHT then
        if cond == 3 then
          htmltext = "31751-01.htm"
        elseif cond == 4 then
          htmltext = "31751-03.htm"
        elseif cond == 5 then
          if q:has_quest_items(player, GOLDEN_HAIR) then
            htmltext = "31751-04.htm"
            st:set_cond(6, true)
          else
            htmltext = "31751-03.htm"
            st:set_cond(4)
          end
        elseif cond == 6 then
          htmltext = "31751-05.htm"
        end
      elseif sw2 == ANGEL_CORPSE then
        if cond == 4 then
          npc:die(npc)
          local hair = st:get_int("angel")
          hair = hair + 1
          if hair == 4 then
            htmltext = "31752-02.htm"
            st:unset("angel")
            st:set_cond(5, true)
            q:give_items(player, GOLDEN_HAIR, 1)
          else
            st:set("angel", tostring(hair))
            htmltext = "31752-01.htm"
          end
        elseif cond == 5 then
          htmltext = "31752-01.htm"
        end
      elseif sw2 == KALIS then
        if cond == 6 then
          htmltext = "30759-01.htm"
        elseif cond == 7 then
          htmltext = "30759-03.htm"
        elseif cond == 8 then
          if q:has_quest_items(player, SORCERY_INGREDIENT) then
            htmltext = "30759-04.htm"
          else
            htmltext = "30759-03.htm"
            st:set_cond(7)
          end
        elseif cond == 9 then
          htmltext = "30759-06.htm"
        end
      elseif sw2 == MATILD then
        if cond == 7 then
          htmltext = "30738-01.htm"
        elseif cond == 8 then
          htmltext = "30738-03.htm"
        end
      elseif sw2 == CORNERSTONE then
        if cond == 9 then
          if q:has_quest_items(player, ORB_OF_BINDING) then
            htmltext = "31748-02.htm"
          else
            htmltext = "31748-01.htm"
          end
        end
      elseif sw2 == FALLEN_UNICORN then
        if cond == 9 then
          htmltext = "31746-01.htm"
        elseif cond == 10 then
          if not _unicorn then
            _unicorn = true
            npc:stop_respawn() -- Despawn fallen unicorn
            npc:delete()
            q:start_quest_timer("spu", 3000, npc, player, false)
          end
          htmltext = "31746-02.htm"
        end
      elseif sw2 == PURE_UNICORN then
        if cond == 10 then
          st:set_cond(11, true)
          if _unicorn then
            _unicorn = false
            q:start_quest_timer("dspu", 3000, npc, player, false)
          end
          htmltext = "31747-01.htm"
        elseif cond == 11 then
          htmltext = "31747-02.htm"
        end
      end
    until true
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(9) or not player:is_sub_class_active() then
    return
  end
  local orbs = st:get_int("orb") -- check orbs internally, because player can use them before he gets them all
  if orbs < 4 then
    orbs = orbs + 1
    st:set("orb", tostring(orbs))
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, ORB_OF_BINDING, 1)
  end
end)

-- Translated by questport from quests/Q00246_PossessorOfAPreciousSoul3 (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(246, "Q00246_PossessorOfAPreciousSoul3", "Possessor of a Precious Soul - 3")
q:requires("subclass")

-- NPCs
local CARADINE = 31740
local OSSIAN = 31741
local LADD = 30721
-- Items
local CARADINE_LETTER_2 = 7678
local WATERBINDER = 7591
local EVERGREEN = 7592
local RAIN_SONG = 7593
local RELIC_BOX = 7594
local CARADINE_LETTER_3 = 7679
-- Mobs
local PILGRIM_OF_SPLENDOR = 21541
local JUDGE_OF_SPLENDOR = 21544
local BARAKIEL = 25325

q:register_quest_items({WATERBINDER, EVERGREEN, RAIN_SONG, RELIC_BOX})
q:add_start_npc(CARADINE)
q:add_talk_id(CARADINE, OSSIAN, LADD)
q:add_kill_id(PILGRIM_OF_SPLENDOR, JUDGE_OF_SPLENDOR, BARAKIEL)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = player:get_quest_state(q:name())
  if st == nil then
    return htmltext
  end
  -- Caradine
  if J.equals_ignore_case(event, "31740-04.htm") then
    st:start_quest()
    q:take_items(player, CARADINE_LETTER_2, 1)
  elseif J.equals_ignore_case(event, "31741-02.htm") then
    st:set_cond(2)
  elseif J.equals_ignore_case(event, "31741-05.htm") then
    if q:has_quest_items(player, WATERBINDER) and q:has_quest_items(player, EVERGREEN) then
      st:set_cond(4)
      q:take_items(player, WATERBINDER, 1)
      q:take_items(player, EVERGREEN, 1)
    else
      htmltext = nil
    end
  elseif J.equals_ignore_case(event, "31741-08.htm") then
    if q:has_quest_items(player, RAIN_SONG) then
      st:set_cond(6)
      q:take_items(player, RAIN_SONG, 1)
      q:give_items(player, RELIC_BOX, 1)
    else
      htmltext = nil
    end
  elseif J.equals_ignore_case(event, "30721-02.htm") then
    if q:has_quest_items(player, RELIC_BOX) then
      q:take_items(player, RELIC_BOX, 1)
      q:give_items(player, CARADINE_LETTER_3, 1)
      q:add_exp_and_sp(player, 719843, 0)
      st:exit_quest(false)
    else
      htmltext = nil
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local st = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = st:get_state()
  if sw == State.CREATED then
    if q:has_quest_items(player, CARADINE_LETTER_2) then
      if not player:is_sub_class_active() or player:level() < 65 then
        htmltext = "31740-02.htm"
        st:exit_quest(true)
      else
        htmltext = "31740-01.htm"
      end
    end
  elseif sw == State.STARTED then
    repeat
      if not player:is_sub_class_active() then
        break
      end
      local sw2 = npc:id()
      if sw2 == CARADINE then
        if st:is_cond(1) then
          htmltext = "31740-05.htm"
        end
      elseif sw2 == OSSIAN then
        local sw3 = st:get_cond()
        if sw3 == 1 then
          htmltext = "31741-01.htm"
        elseif sw3 == 2 then
          htmltext = "31741-03.htm"
        elseif sw3 == 3 then
          if q:has_quest_items(player, WATERBINDER) and q:has_quest_items(player, EVERGREEN) then
            htmltext = "31741-04.htm"
          end
        elseif sw3 == 4 then
          htmltext = "31741-06.htm"
        elseif sw3 == 5 then
          if q:has_quest_items(player, RAIN_SONG) then
            htmltext = "31741-07.htm"
          end
        elseif sw3 == 6 then
          htmltext = "31741-09.htm"
        end
      elseif sw2 == LADD then
        if st:is_cond(6) and q:has_quest_items(player, RELIC_BOX) then
          htmltext = "30721-01.htm"
        end
      end
    until true
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local npcId = npc:id()
  if npcId == BARAKIEL then
    local pst
    if player:party() ~= nil and not (#player:party_members() == 0) then
      for _, member in ipairs(player:party_members()) do
        -- L2Go: Mobius skips members with calculateDistance3D(npc) > ALT_PARTY_RANGE;
        -- is_inside_radius_3d is the same test (distance <= range).
        if member:is_inside_radius_3d(npc, Config.ALT_PARTY_RANGE) then
          pst = member:get_quest_state(q:name())
          if pst ~= nil and pst:is_cond(4) and not q:has_quest_items(member, RAIN_SONG) then
            q:give_items(member, RAIN_SONG, 1)
            pst:set_cond(5, true)
          end
        end
      end
    else
      pst = player:get_quest_state(q:name())
      if pst ~= nil and pst:is_cond(4) and not q:has_quest_items(player, RAIN_SONG) then
        q:give_items(player, RAIN_SONG, 1)
        pst:set_cond(5, true)
      end
    end
  else
    if not player:is_sub_class_active() then
      return
    end
    local st = player:get_quest_state(q:name())
    if st ~= nil and not st:is_cond(2) then
      return
    end
    if rnd(10) < 2 then
      local neklaceOrRing = npcId == PILGRIM_OF_SPLENDOR and WATERBINDER or EVERGREEN
      if st ~= nil and not q:has_quest_items(player, neklaceOrRing) then
        q:give_items(player, neklaceOrRing, 1)
        if not q:has_quest_items(player, npcId == PILGRIM_OF_SPLENDOR and EVERGREEN or WATERBINDER) then
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          st:set_cond(3, true)
        end
      end
    end
  end
end)

-- Translated by questport from quests/Q00241_PossessorOfAPreciousSoul1 (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(241, "Q00241_PossessorOfAPreciousSoul1", "Possessor of a Precious Soul - 1")
q:requires("subclass")

-- NPCs
local TALIEN = 31739
local GABRIELLE = 30753
local GILMORE = 30754
local KANTABILON = 31042
local STEDMIEL = 30692
local VIRGIL = 31742
local OGMAR = 31744
local RAHORAKTI = 31336
local KASSANDRA = 31743
local CARADINE = 31740
local NOEL = 31272
-- Monsters
local BARAHAM = 27113
local MALRUK_SUCCUBUS = 20244
local MALRUK_SUCCUBUS_TUREN = 20245
local SPLINTER_STAKATO = 21508
local SPLINTER_STAKATO_WALKER = 21509
local SPLINTER_STAKATO_SOLDIER = 21510
local SPLINTER_STAKATO_DRONE1 = 21511
local SPLINTER_STAKATO_DRONE2 = 21512
-- Items
local LEGEND_OF_SEVENTEEN = 7587
local MALRUK_SUCCUBUS_CLAW = 7597
local ECHO_CRYSTAL = 7589
local POETRY_BOOK = 7588
local CRIMSON_MOSS = 7598
local RAHORAKTIS_MEDICINE = 7599
local LUNARGENT = 6029
local HELLFIRE_OIL = 6033
local VIRGILS_LETTER = 7677
-- Rewards
local CRIMSON_MOSS_CHANCE = 30
local MALRUK_SUCCUBUS_CLAW_CHANCE = 60

q:register_quest_items({
  LEGEND_OF_SEVENTEEN,
  MALRUK_SUCCUBUS_CLAW,
  ECHO_CRYSTAL,
  POETRY_BOOK,
  CRIMSON_MOSS,
  RAHORAKTIS_MEDICINE,
})
q:add_start_npc(TALIEN)
q:add_talk_id(TALIEN, GABRIELLE, GILMORE, KANTABILON, STEDMIEL, VIRGIL, OGMAR, RAHORAKTI, KASSANDRA, CARADINE, NOEL)
q:add_kill_id(BARAHAM, MALRUK_SUCCUBUS, MALRUK_SUCCUBUS_TUREN, SPLINTER_STAKATO, SPLINTER_STAKATO_WALKER, SPLINTER_STAKATO_SOLDIER, SPLINTER_STAKATO_DRONE1, SPLINTER_STAKATO_DRONE2)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = player:get_quest_state(q:name())
  if st == nil then
    return htmltext
  end
  -- Talien
  if J.equals_ignore_case(event, "31739-03.htm") then
    st:start_quest()
  elseif J.equals_ignore_case(event, "31739-07.htm") then
    st:set_cond(5)
    q:take_items(player, LEGEND_OF_SEVENTEEN, 1)
  elseif J.equals_ignore_case(event, "31739-10.htm") then
    st:set_cond(9)
    q:take_items(player, ECHO_CRYSTAL, 1)
  elseif J.equals_ignore_case(event, "31739-13.htm") then
    st:set_cond(11)
    q:take_items(player, POETRY_BOOK, 1)
  elseif J.equals_ignore_case(event, "30753-02.htm") then
    st:set_cond(2)
  elseif J.equals_ignore_case(event, "30754-02.htm") then
    st:set_cond(3)
  elseif J.equals_ignore_case(event, "31042-02.htm") then
    st:set_cond(6)
  elseif J.equals_ignore_case(event, "31042-05.htm") then
    st:set_cond(8)
    q:take_items(player, MALRUK_SUCCUBUS_CLAW, 10)
    q:give_items(player, ECHO_CRYSTAL, 1)
  elseif J.equals_ignore_case(event, "30692-02.htm") then
    st:set_cond(10)
    q:give_items(player, POETRY_BOOK, 1)
  elseif J.equals_ignore_case(event, "31742-02.htm") then
    st:set_cond(12)
  elseif J.equals_ignore_case(event, "31742-05.htm") then
    st:set_cond(18)
  elseif J.equals_ignore_case(event, "31744-02.htm") then
    st:set_cond(13)
  elseif J.equals_ignore_case(event, "31336-02.htm") then
    st:set_cond(14)
  elseif J.equals_ignore_case(event, "31336-05.htm") then
    st:set_cond(16)
    q:take_items(player, CRIMSON_MOSS, 5)
    q:give_items(player, RAHORAKTIS_MEDICINE, 1)
  elseif J.equals_ignore_case(event, "31743-02.htm") then
    st:set_cond(17)
    q:take_items(player, RAHORAKTIS_MEDICINE, 1)
  elseif J.equals_ignore_case(event, "31740-02.htm") then
    st:set_cond(19)
  elseif J.equals_ignore_case(event, "31740-05.htm") then
    q:give_items(player, VIRGILS_LETTER, 1)
    q:add_exp_and_sp(player, 263043, 0)
    st:exit_quest(false)
  elseif J.equals_ignore_case(event, "31272-02.htm") then
    st:set_cond(20)
  elseif J.equals_ignore_case(event, "31272-05.htm") then
    if q:has_quest_items(player, HELLFIRE_OIL) and q:get_quest_items_count(player, LUNARGENT) >= 5 then
      q:take_items(player, LUNARGENT, 5)
      q:take_items(player, HELLFIRE_OIL, 1)
      st:set_cond(21)
    else
      htmltext = "31272-07.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local st = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = st:get_state()
  if sw == State.CREATED then
    if not player:is_sub_class_active() or player:level() < 50 then
      htmltext = "31739-02.htm"
      st:exit_quest(true)
    else
      htmltext = "31739-01.htm"
    end
  elseif sw == State.STARTED then
    repeat
      if not player:is_sub_class_active() then
        break
      end
      local cond = st:get_cond()
      local sw2 = npc:id()
      if sw2 == TALIEN then
        if cond == 1 then
          htmltext = "31739-04.htm"
        elseif cond == 2 or cond == 3 then
          htmltext = "31739-05.htm"
        elseif cond == 4 then
          htmltext = "31739-06.htm"
        elseif cond == 5 then
          htmltext = "31739-08.htm"
        elseif cond == 8 then
          htmltext = "31739-09.htm"
        elseif cond == 9 then
          htmltext = "31739-11.htm"
        elseif cond == 10 then
          htmltext = "31739-12.htm"
        elseif cond == 11 then
          htmltext = "31739-14.htm"
        end
      elseif sw2 == GABRIELLE then
        if cond == 1 then
          htmltext = "30753-01.htm"
        elseif cond == 2 then
          htmltext = "30753-03.htm"
        end
      elseif sw2 == GILMORE then
        if cond == 2 then
          htmltext = "30754-01.htm"
        elseif cond == 3 then
          htmltext = "30754-03.htm"
        end
      elseif sw2 == KANTABILON then
        if cond == 5 then
          htmltext = "31042-01.htm"
        elseif cond == 6 then
          htmltext = "31042-03.htm"
        elseif cond == 7 then
          htmltext = "31042-04.htm"
        elseif cond == 8 then
          htmltext = "31042-06.htm"
        end
      elseif sw2 == STEDMIEL then
        if cond == 9 then
          htmltext = "30692-01.htm"
        elseif cond == 10 then
          htmltext = "30692-03.htm"
        end
      elseif sw2 == VIRGIL then
        if cond == 11 then
          htmltext = "31742-01.htm"
        elseif cond == 12 then
          htmltext = "31742-03.htm"
        elseif cond == 17 then
          htmltext = "31742-04.htm"
        elseif cond == 18 then
          htmltext = "31742-06.htm"
        end
      elseif sw2 == OGMAR then
        if cond == 12 then
          htmltext = "31744-01.htm"
        elseif cond == 13 then
          htmltext = "31744-03.htm"
        end
      elseif sw2 == RAHORAKTI then
        if cond == 13 then
          htmltext = "31336-01.htm"
        elseif cond == 14 then
          htmltext = "31336-03.htm"
        elseif cond == 15 then
          htmltext = "31336-04.htm"
        elseif cond == 16 then
          htmltext = "31336-06.htm"
        end
      elseif sw2 == KASSANDRA then
        if cond == 16 then
          htmltext = "31743-01.htm"
        elseif cond == 17 then
          htmltext = "31743-03.htm"
        end
      elseif sw2 == CARADINE then
        if cond == 18 then
          htmltext = "31740-01.htm"
        elseif cond == 19 then
          htmltext = "31740-03.htm"
        elseif cond == 21 then
          htmltext = "31740-04.htm"
        end
      elseif sw2 == NOEL then
        if cond == 19 then
          htmltext = "31272-01.htm"
        elseif cond == 20 then
          if q:has_quest_items(player, HELLFIRE_OIL) and q:get_quest_items_count(player, LUNARGENT) >= 5 then
            htmltext = "31272-04.htm"
          else
            htmltext = "31272-03.htm"
          end
        elseif cond == 21 then
          htmltext = "31272-06.htm"
        end
      end
    until true
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  if not player:is_sub_class_active() then
    return
  end
  local st
  local partyMember
  local sw = npc:id()
  if sw == BARAHAM then
    partyMember = q:get_random_party_member(player, 3)
    if partyMember == nil then
      return
    end
    st = partyMember:get_quest_state(q:name())
    q:give_items(partyMember, LEGEND_OF_SEVENTEEN, 1)
    st:set_cond(4, true)
  elseif sw == MALRUK_SUCCUBUS or sw == MALRUK_SUCCUBUS_TUREN then
    partyMember = q:get_random_party_member(player, 6)
    if partyMember == nil then
      return
    end
    st = partyMember:get_quest_state(q:name())
    if MALRUK_SUCCUBUS_CLAW_CHANCE >= rnd(100) and q:get_quest_items_count(partyMember, MALRUK_SUCCUBUS_CLAW) < 10 then
      q:give_items(partyMember, MALRUK_SUCCUBUS_CLAW, 1)
      if q:get_quest_items_count(partyMember, MALRUK_SUCCUBUS_CLAW) == 10 then
        st:set_cond(7, true)
      else
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == SPLINTER_STAKATO or sw == SPLINTER_STAKATO_WALKER or sw == SPLINTER_STAKATO_SOLDIER or sw == SPLINTER_STAKATO_DRONE1 or sw == SPLINTER_STAKATO_DRONE2 then
    partyMember = q:get_random_party_member(player, 14)
    if partyMember == nil then
      return
    end
    st = partyMember:get_quest_state(q:name())
    if CRIMSON_MOSS_CHANCE >= rnd(100) and q:get_quest_items_count(partyMember, CRIMSON_MOSS) < 5 then
      q:give_items(partyMember, CRIMSON_MOSS, 1)
      if q:get_quest_items_count(partyMember, CRIMSON_MOSS) == 5 then
        st:set_cond(15, true)
      else
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)

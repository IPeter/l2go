-- Translated by questport from quests/Q00111_ElrokianHuntersProof (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(111, "Q00111_ElrokianHuntersProof", "Elrokian Hunter's Proof")

-- NPCs
local MARQUEZ = 32113
local MUSHIKA = 32114
local ASAMAH = 32115
local KIRIKASHIN = 32116
-- Items
local DIARY_FRAGMENT = 8768
local EXPEDITION_MEMBERS_LETTER = 8769
local ORNITHOMINUS_CLAW = 8770
local DEINONYCHUS_BONE = 8771
local PACHYCEPHALOSAURUS_SKIN = 8772
local PRACTICE_ELROKIAN_TRAP = 8773
-- Mobs
local MOBS_DROP_CHANCES = {}

MOBS_DROP_CHANCES[22196] = {id = DIARY_FRAGMENT, chance = 0.51, count = 4} -- velociraptor_leader
MOBS_DROP_CHANCES[22197] = {id = DIARY_FRAGMENT, chance = 0.51, count = 4} -- velociraptor
MOBS_DROP_CHANCES[22198] = {id = DIARY_FRAGMENT, chance = 0.51, count = 4} -- velociraptor_s
MOBS_DROP_CHANCES[22218] = {id = DIARY_FRAGMENT, chance = 0.25, count = 4} -- velociraptor_n
MOBS_DROP_CHANCES[22200] = {id = ORNITHOMINUS_CLAW, chance = 0.66, count = 11} -- ornithomimus_leader
MOBS_DROP_CHANCES[22201] = {id = ORNITHOMINUS_CLAW, chance = 0.33, count = 11} -- ornithomimus
MOBS_DROP_CHANCES[22202] = {id = ORNITHOMINUS_CLAW, chance = 0.66, count = 11} -- ornithomimus_s
MOBS_DROP_CHANCES[22219] = {id = ORNITHOMINUS_CLAW, chance = 0.33, count = 11} -- ornithomimus_n
MOBS_DROP_CHANCES[22203] = {id = DEINONYCHUS_BONE, chance = 0.65, count = 11} -- deinonychus_leader
MOBS_DROP_CHANCES[22204] = {id = DEINONYCHUS_BONE, chance = 0.32, count = 11} -- deinonychus
MOBS_DROP_CHANCES[22205] = {id = DEINONYCHUS_BONE, chance = 0.66, count = 11} -- deinonychus_s
MOBS_DROP_CHANCES[22220] = {id = DEINONYCHUS_BONE, chance = 0.32, count = 11} -- deinonychus_n
MOBS_DROP_CHANCES[22208] = {id = PACHYCEPHALOSAURUS_SKIN, chance = 0.5, count = 11} -- pachycephalosaurus_ldr
MOBS_DROP_CHANCES[22209] = {id = PACHYCEPHALOSAURUS_SKIN, chance = 0.5, count = 11} -- pachycephalosaurus
MOBS_DROP_CHANCES[22210] = {id = PACHYCEPHALOSAURUS_SKIN, chance = 0.5, count = 11} -- pachycephalosaurus_s
MOBS_DROP_CHANCES[22221] = {id = PACHYCEPHALOSAURUS_SKIN, chance = 0.49, count = 11} -- pachycephalosaurus_n

q:register_quest_items(DIARY_FRAGMENT, EXPEDITION_MEMBERS_LETTER, ORNITHOMINUS_CLAW, DEINONYCHUS_BONE, PACHYCEPHALOSAURUS_SKIN, PRACTICE_ELROKIAN_TRAP)
q:add_start_npc(MARQUEZ)
q:add_talk_id(MARQUEZ, MUSHIKA, ASAMAH, KIRIKASHIN)
q:add_kill_id(J.keys(MOBS_DROP_CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32113-002.htm" then
    st:start_quest()
  elseif event == "32115-002.htm" then
    st:set_cond(3, true)
  elseif event == "32113-009.htm" then
    st:set_cond(4, true)
  elseif event == "32113-018.htm" then
    st:set_cond(6, true)
    q:take_items(player, DIARY_FRAGMENT, -1)
    q:give_items(player, EXPEDITION_MEMBERS_LETTER, 1)
  elseif event == "32116-003.htm" then
    st:set_cond(7)
    q:play_sound(player, QuestSound.ETCSOUND_ELROKI_SONG_FULL)
  elseif event == "32116-005.htm" then
    st:set_cond(8, true)
  elseif event == "32115-004.htm" then
    st:set_cond(9, true)
  elseif event == "32115-006.htm" then
    st:set_cond(10, true)
  elseif event == "32116-007.htm" then
    q:take_items(player, PRACTICE_ELROKIAN_TRAP, 1)
    q:give_items(player, 8763, 1)
    q:give_items(player, 8764, 100)
    q:give_adena(player, 1022636, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 75 and "32113-000.htm" or "32113-001.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == MARQUEZ then
      if cond == 1 or cond == 2 then
        htmltext = "32113-002.htm"
      elseif cond == 3 then
        htmltext = "32113-003.htm"
      elseif cond == 4 then
        htmltext = "32113-009.htm"
      elseif cond == 5 then
        htmltext = "32113-010.htm"
      end
    elseif sw2 == MUSHIKA then
      if cond == 1 then
        st:set_cond(2, true)
      end
      htmltext = "32114-001.htm"
    elseif sw2 == ASAMAH then
      if cond == 2 then
        htmltext = "32115-001.htm"
      elseif cond == 3 then
        htmltext = "32115-002.htm"
      elseif cond == 8 then
        htmltext = "32115-003.htm"
      elseif cond == 9 then
        htmltext = "32115-004.htm"
      elseif cond == 10 then
        htmltext = "32115-006.htm"
      elseif cond == 11 then
        htmltext = "32115-007.htm"
        st:set_cond(12, true)
        q:take_items(player, DEINONYCHUS_BONE, -1)
        q:take_items(player, ORNITHOMINUS_CLAW, -1)
        q:take_items(player, PACHYCEPHALOSAURUS_SKIN, -1)
        q:give_items(player, PRACTICE_ELROKIAN_TRAP, 1)
      end
    elseif sw2 == KIRIKASHIN then
      if cond < 6 then
        htmltext = "32116-008.htm"
      elseif cond == 6 then
        htmltext = "32116-001.htm"
        q:take_items(player, EXPEDITION_MEMBERS_LETTER, 1)
      elseif cond == 7 then
        htmltext = "32116-004.htm"
      elseif cond == 12 then
        htmltext = "32116-006.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

-- Mobius compares the drop's count (4 or 11) with the memo state, which
-- this quest never sets: as in Mobius, the monsters drop nothing.
q:on_kill(function(npc, player, isSummon)
  local qs = q:get_random_party_member_state(player, -1, 3, npc)
  if qs ~= nil then
    local item = MOBS_DROP_CHANCES[npc:id()]
    if item.count == qs:get_memo_state() then
      if qs:is_cond(4) and q:give_item_randomly(qs:get_player(), npc, item.id, 1, 1, 50, item.chance, true) then
        qs:set_cond(5)
      elseif qs:is_cond(10) and q:give_item_randomly(qs:get_player(), npc, item.id, 1, 1, 10, item.chance, true) and q:get_quest_items_count(qs:get_player(), ORNITHOMINUS_CLAW) >= 10 and q:get_quest_items_count(qs:get_player(), DEINONYCHUS_BONE) >= 10 and q:get_quest_items_count(qs:get_player(), PACHYCEPHALOSAURUS_SKIN) >= 10 then
        qs:set_cond(11)
      end
    end
  end
end)

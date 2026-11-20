-- Translated by questport from quests/Q00125_TheNameOfEvil1 (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(125, "Q00125_TheNameOfEvil1", "The Name of Evil - 1")

local MUSHIKA = 32114
local KARAKAWEI = 32117
local ULU_KAIMU = 32119
local BALU_KAIMU = 32120
local CHUTA_KAIMU = 32121
local ORNITHOMIMUS_CLAW = 8779
local DEINONYCHUS_BONE = 8780
local EPITAPH_OF_WISDOM = 8781
local GAZKH_FRAGMENT = 8782
local ORNITHOMIMUS = {}
local DEINONYCHUS = {}

ORNITHOMIMUS[22200] = 661
ORNITHOMIMUS[22201] = 330
ORNITHOMIMUS[22202] = 661
ORNITHOMIMUS[22219] = 327
ORNITHOMIMUS[22224] = 327
DEINONYCHUS[22203] = 651
DEINONYCHUS[22204] = 326
DEINONYCHUS[22205] = 651
DEINONYCHUS[22220] = 319
DEINONYCHUS[22225] = 319

q:register_quest_items(ORNITHOMIMUS_CLAW, DEINONYCHUS_BONE, EPITAPH_OF_WISDOM, GAZKH_FRAGMENT)
q:add_start_npc(MUSHIKA)
q:add_talk_id(MUSHIKA, KARAKAWEI, ULU_KAIMU, BALU_KAIMU, CHUTA_KAIMU)
q:add_kill_id(J.keys(ORNITHOMIMUS))
q:add_kill_id(J.keys(DEINONYCHUS))

-- Mobius has the stone statues cast skill 5089 (a visual) when the player
-- reaches them: npc.doCast(SkillData.getInstance().getSkill(5089, 1)). Left
-- out until the API has NPC casting (npc:cast).

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32114-05.htm" then
    st:start_quest()
  elseif event == "32114-09.htm" then
    st:set_cond(2, true)
    q:give_items(player, GAZKH_FRAGMENT, 1)
  elseif event == "32117-08.htm" then
    st:set_cond(3, true)
  elseif event == "32117-14.htm" then
    st:set_cond(5, true)
  elseif event == "32119-14.htm" then
    st:set_cond(6, true)
  elseif event == "32120-15.htm" then
    st:set_cond(7, true)
  elseif event == "32121-16.htm" then
    st:set_cond(8, true)
    q:take_items(player, GAZKH_FRAGMENT, -1)
    q:give_items(player, EPITAPH_OF_WISDOM, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    local first = player:get_quest_state("Q00124_MeetingTheElroki")
    if first ~= nil and first:is_completed() and player:level() >= 76 then
      htmltext = "32114-01.htm"
    else
      htmltext = "32114-00.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == MUSHIKA then
      if cond == 1 then
        htmltext = "32114-07.htm"
      elseif cond == 2 then
        htmltext = "32114-10.htm"
      elseif cond > 2 and cond < 8 then
        htmltext = "32114-11.htm"
      elseif cond == 8 then
        htmltext = "32114-12.htm"
        q:take_items(player, EPITAPH_OF_WISDOM, -1)
        st:exit_quest(false, true)
      end
    elseif sw2 == KARAKAWEI then
      if cond == 2 then
        htmltext = "32117-01.htm"
      elseif cond == 3 then
        htmltext = "32117-09.htm"
      elseif cond == 4 then
        if q:get_quest_items_count(player, ORNITHOMIMUS_CLAW) >= 2 and q:get_quest_items_count(player, DEINONYCHUS_BONE) >= 2 then
          htmltext = "32117-10.htm"
          q:take_items(player, ORNITHOMIMUS_CLAW, -1)
          q:take_items(player, DEINONYCHUS_BONE, -1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        else
          htmltext = "32117-09.htm"
          st:set_cond(3)
        end
      elseif cond == 5 then
        htmltext = "32117-15.htm"
      elseif cond == 6 or cond == 7 then
        htmltext = "32117-16.htm"
      elseif cond == 8 then
        htmltext = "32117-17.htm"
      end
    elseif sw2 == ULU_KAIMU then
      if cond == 5 then
        htmltext = "32119-01.htm"
      elseif cond == 6 then
        htmltext = "32119-14.htm"
      end
    elseif sw2 == BALU_KAIMU then
      if cond == 6 then
        htmltext = "32120-01.htm"
      elseif cond == 7 then
        htmltext = "32120-16.htm"
      end
    elseif sw2 == CHUTA_KAIMU then
      if cond == 7 then
        htmltext = "32121-01.htm"
      elseif cond == 8 then
        htmltext = "32121-17.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local partyMember = q:get_random_party_member(player, 3)
  if partyMember == nil then
    return
  end
  local qs = q:get_quest_state(partyMember, false)
  local npcId = npc:id()
  local ornithomimusChance = ORNITHOMIMUS[npcId]
  if ornithomimusChance ~= nil then
    if q:get_quest_items_count(player, ORNITHOMIMUS_CLAW) < 2 and rnd(1000) < ornithomimusChance then
      q:give_items(player, ORNITHOMIMUS_CLAW, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif DEINONYCHUS[npcId] ~= nil and q:get_quest_items_count(player, DEINONYCHUS_BONE) < 2 and rnd(1000) < DEINONYCHUS[npcId] then
    q:give_items(player, DEINONYCHUS_BONE, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
  if q:get_quest_items_count(player, ORNITHOMIMUS_CLAW) == 2 and q:get_quest_items_count(player, DEINONYCHUS_BONE) == 2 then
    qs:set_cond(4, true)
  end
end)

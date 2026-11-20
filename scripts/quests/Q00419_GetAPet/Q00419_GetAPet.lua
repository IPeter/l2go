-- Translated by questport from quests/Q00419_GetAPet (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(419, "Q00419_GetAPet", "Get a Pet")
q:requires("pets") -- the Wolf Collar summons a pet (Plan 11: pets 352/419-421)

-- NPCs
local MARTIN = 30731
local BELLA = 30256
local METTY = 30072
local ELLIE = 30091
-- Items
local ANIMAL_LOVER_LIST = 3417
local ANIMAL_SLAYER_LIST_1 = 3418
local ANIMAL_SLAYER_LIST_2 = 3419
local ANIMAL_SLAYER_LIST_3 = 3420
local ANIMAL_SLAYER_LIST_4 = 3421
local ANIMAL_SLAYER_LIST_5 = 3422
local BLOODY_FANG = 3423
local BLOODY_CLAW = 3424
local BLOODY_NAIL = 3425
local BLOODY_KASHA_FANG = 3426
local BLOODY_TARANTULA_NAIL = 3427
-- Reward
local WOLF_COLLAR = 2375
-- Droplist
local DROPLIST = {}

local function checkQuestions(st)
  local answers = st:get_int("correct") + st:get_int("wrong")
  if answers < 10 then
    local questions = J.split(st:get("quiz"), " ")
    local index = rnd(#questions - 1)
    local question = questions[index + 1]
    if #questions > 10 - answers then
      questions[index + 1] = questions[#questions]
      st:set("quiz", table.concat(J.copy_of(questions, #questions - 1), " "))
    end
    return "30731-" .. question .. ".htm"
  end
  if st:get_int("wrong") > 0 then
    st:unset("progress")
    st:unset("answers")
    st:unset("quiz")
    st:unset("wrong")
    st:unset("correct")
    return "30731-14.htm"
  end
  q:take_items(st:get_player(), ANIMAL_LOVER_LIST, 1)
  q:give_items(st:get_player(), WOLF_COLLAR, 1)
  st:exit_quest(true, true)
  return "30731-15.htm"
end

DROPLIST[20103] = {BLOODY_FANG, 600000}
DROPLIST[20106] = {BLOODY_FANG, 750000}
DROPLIST[20108] = {BLOODY_FANG, 1000000}
DROPLIST[20460] = {BLOODY_CLAW, 600000}
DROPLIST[20308] = {BLOODY_CLAW, 750000}
DROPLIST[20466] = {BLOODY_CLAW, 1000000}
DROPLIST[20025] = {BLOODY_NAIL, 600000}
DROPLIST[20105] = {BLOODY_NAIL, 750000}
DROPLIST[20034] = {BLOODY_NAIL, 1000000}
DROPLIST[20474] = {BLOODY_KASHA_FANG, 600000}
DROPLIST[20476] = {BLOODY_KASHA_FANG, 750000}
DROPLIST[20478] = {BLOODY_KASHA_FANG, 1000000}
DROPLIST[20403] = {BLOODY_TARANTULA_NAIL, 750000}
DROPLIST[20508] = {BLOODY_TARANTULA_NAIL, 1000000}

q:register_quest_items(ANIMAL_LOVER_LIST, ANIMAL_SLAYER_LIST_1, ANIMAL_SLAYER_LIST_2, ANIMAL_SLAYER_LIST_3, ANIMAL_SLAYER_LIST_4, ANIMAL_SLAYER_LIST_5, BLOODY_FANG, BLOODY_CLAW, BLOODY_NAIL, BLOODY_KASHA_FANG, BLOODY_TARANTULA_NAIL)
q:add_start_npc(MARTIN)
q:add_talk_id(MARTIN, BELLA, ELLIE, METTY)
q:add_kill_id(J.keys(DROPLIST))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "task" then
    local race = player:race()
    htmltext = "30731-0" .. (race + 4) .. ".htm"
    st:start_quest()
    q:give_items(player, ANIMAL_SLAYER_LIST_1 + race, 1)
  elseif event == "30731-12.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:take_items(player, ANIMAL_SLAYER_LIST_1, 1)
    q:take_items(player, ANIMAL_SLAYER_LIST_2, 1)
    q:take_items(player, ANIMAL_SLAYER_LIST_3, 1)
    q:take_items(player, ANIMAL_SLAYER_LIST_4, 1)
    q:take_items(player, ANIMAL_SLAYER_LIST_5, 1)
    q:take_items(player, BLOODY_FANG, -1)
    q:take_items(player, BLOODY_CLAW, -1)
    q:take_items(player, BLOODY_NAIL, -1)
    q:take_items(player, BLOODY_KASHA_FANG, -1)
    q:take_items(player, BLOODY_TARANTULA_NAIL, -1)
    q:give_items(player, ANIMAL_LOVER_LIST, 1)
  elseif event == "30256-03.htm" then
    st:set("progress", tostring(J.bor(st:get_int("progress"), 1)))
    if st:get_int("progress") == 7 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
  elseif event == "30072-02.htm" then
    st:set("progress", tostring(J.bor(st:get_int("progress"), 2)))
    if st:get_int("progress") == 7 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
  elseif event == "30091-02.htm" then
    st:set("progress", tostring(J.bor(st:get_int("progress"), 4)))
    if st:get_int("progress") == 7 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
  elseif event == "test" then
    st:set("answers", "0")
    st:set("quiz", "20 21 22 23 24 25 26 27 28 29 30 31 32 33")
    return checkQuestions(st)
  elseif event == "wrong" then
    st:set("wrong", tostring(st:get_int("wrong") + 1))
    return checkQuestions(st)
  elseif event == "right" then
    st:set("correct", tostring(st:get_int("correct") + 1))
    return checkQuestions(st)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 15 and "30731-01.htm" or "30731-02.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == MARTIN then
      if q:has_at_least_one_quest_item(player, ANIMAL_SLAYER_LIST_1, ANIMAL_SLAYER_LIST_2, ANIMAL_SLAYER_LIST_3, ANIMAL_SLAYER_LIST_4, ANIMAL_SLAYER_LIST_5) then
        local proofs = q:get_quest_items_count(player, BLOODY_FANG) + q:get_quest_items_count(player, BLOODY_CLAW) + q:get_quest_items_count(player, BLOODY_NAIL) + q:get_quest_items_count(player, BLOODY_KASHA_FANG) + q:get_quest_items_count(player, BLOODY_TARANTULA_NAIL)
        if proofs == 0 then
          htmltext = "30731-09.htm"
        elseif proofs < 50 then
          htmltext = "30731-10.htm"
        else
          htmltext = "30731-11.htm"
        end
      elseif st:get_int("progress") == 7 then
        htmltext = "30731-13.htm"
      else
        htmltext = "30731-16.htm"
      end
    elseif sw2 == BELLA then
      htmltext = "30256-01.htm"
    elseif sw2 == METTY then
      htmltext = "30072-01.htm"
    elseif sw2 == ELLIE then
      htmltext = "30091-01.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local drop = DROPLIST[npc:id()]
  if q:has_quest_items(player, drop[1] - 5) then
    if q:get_quest_items_count(player, drop[1]) < 50 then
      q:give_items(player, drop[1], 1)
      if q:get_quest_items_count(player, drop[1]) < 50 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  end
end)

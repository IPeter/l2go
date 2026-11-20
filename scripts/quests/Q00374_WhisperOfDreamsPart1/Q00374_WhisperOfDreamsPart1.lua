-- Translated by questport from quests/Q00374_WhisperOfDreamsPart1 (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(374, "Q00374_WhisperOfDreamsPart1", "Whisper of Dreams, Part 1")

-- NPCs
local MANAKIA = 30515
local TORAI = 30557
-- Monsters
local CAVE_BEAST = 20620
local DEATH_WAVE = 20621
-- Items
local CAVE_BEAST_TOOTH = 5884
local DEATH_WAVE_LIGHT = 5885
local SEALED_MYSTERIOUS_STONE = 5886
local MYSTERIOUS_STONE = 5887
-- Rewards
local REWARDS = {{5486, 3, 2950}, {5487, 2, 18050}, {5488, 2, 18050}, {5485, 4, 10450}, {5489, 6, 15550}}

q:register_quest_items(DEATH_WAVE_LIGHT, CAVE_BEAST_TOOTH, SEALED_MYSTERIOUS_STONE, MYSTERIOUS_STONE)
q:add_start_npc(MANAKIA)
q:add_talk_id(MANAKIA, TORAI)
q:add_kill_id(CAVE_BEAST, DEATH_WAVE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  -- Manakia
  if event == "30515-03.htm" then
    st:start_quest()
    st:set("condStone", "1")
  elseif J.starts_with(event, "30515-06-") then
    if q:get_quest_items_count(player, CAVE_BEAST_TOOTH) >= 65 and q:get_quest_items_count(player, DEATH_WAVE_LIGHT) >= 65 then
      htmltext = "30515-06.htm"
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      local reward = REWARDS[J.parse_int(J.substring(event, 9, 10)) + 1]
      q:take_items(player, CAVE_BEAST_TOOTH, -1)
      q:take_items(player, DEATH_WAVE_LIGHT, -1)
      q:give_adena(player, reward[3], true)
      q:give_items(player, reward[1], reward[2])
    else
      htmltext = "30515-07.htm"
    end
  elseif event == "30515-08.htm" then
    st:exit_quest(true, true)
  elseif event == "30557-02.htm" then
    if st:is_cond(2) and q:has_quest_items(player, SEALED_MYSTERIOUS_STONE) then
      st:set_cond(3, true)
      q:take_items(player, SEALED_MYSTERIOUS_STONE, -1)
      q:give_items(player, MYSTERIOUS_STONE, 1)
    else
      htmltext = "30557-03.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 56 and "30515-01.htm" or "30515-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == MANAKIA then
      if not q:has_quest_items(player, SEALED_MYSTERIOUS_STONE) then
        if q:get_quest_items_count(player, CAVE_BEAST_TOOTH) >= 65 and q:get_quest_items_count(player, DEATH_WAVE_LIGHT) >= 65 then
          htmltext = "30515-05.htm"
        else
          htmltext = "30515-04.htm"
        end
      else
        if cond == 1 then
          htmltext = "30515-09.htm"
          st:set_cond(2, true)
        else
          htmltext = "30515-10.htm"
        end
      end
    elseif sw2 == TORAI then
      if cond == 2 and q:has_quest_items(player, SEALED_MYSTERIOUS_STONE) then
        htmltext = "30557-01.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  -- Drop tooth or light to anyone.
  local partyMember = q:get_random_party_member_state(player, State.STARTED)
  if partyMember == nil then
    return
  end
  local st = partyMember:get_quest_state(q:name())
  if st == nil then
    return
  end
  if rnd.bool() then
    q:give_items(st:get_player(), npc:id() == CAVE_BEAST and CAVE_BEAST_TOOTH or DEATH_WAVE_LIGHT, rnd(1, 65))
  end
  -- Drop sealed mysterious stone to party member who still need it.
  partyMember = q:get_random_party_member(player, "condStone", "1")
  if partyMember == nil then
    return
  end
  st = partyMember:get_quest_state(q:name())
  if st == nil then
    return
  end
  if rnd(100) < 1 then
    q:give_items(st:get_player(), SEALED_MYSTERIOUS_STONE, 1)
    st:unset("condStone")
  end
end)

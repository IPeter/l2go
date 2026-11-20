-- Translated by questport from quests/Q00602_ShadowOfLight (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(602, "Q00602_ShadowOfLight", "Shadow of Light")

local EYE_OF_DARKNESS = 7189
local REWARDS = {
  {6699, 40000, 120000, 20000, 20},
  {6698, 60000, 110000, 15000, 40},
  {6700, 40000, 150000, 10000, 50},
  {0, 100000, 140000, 11250, 100},
}

q:register_quest_items(EYE_OF_DARKNESS)
q:add_start_npc(31683) -- Eye of Argos
q:add_talk_id(31683)
q:add_kill_id(21299, 21304)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31683-02.htm" then
    if player:level() < 68 then
      htmltext = "31683-02a.htm"
    else
      st:start_quest()
    end
  elseif event == "31683-05.htm" then
    q:take_items(player, EYE_OF_DARKNESS, -1)
    local random = rnd(100)
    for _, element in ipairs(REWARDS) do
      if random < element[5] then
        q:give_adena(player, element[2], true)
        if element[1] ~= 0 then
          q:give_items(player, element[1], 3)
        end
        q:add_exp_and_sp(player, element[3], element[4])
        break
      end
    end
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "31683-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "31683-03.htm"
    elseif cond == 2 then
      htmltext = "31683-04.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local qs = q:get_random_party_member_state(player, 1, 3, npc)
  if qs == nil then
    return
  end
  local partyMember = qs:get_player()
  local st = q:get_quest_state(partyMember, false)
  if st == nil then
    return
  end
  if rnd(100) < (npc:id() == 21299 and 45 or 50) then
    q:give_items(partyMember, EYE_OF_DARKNESS, 1)
    if q:get_quest_items_count(partyMember, EYE_OF_DARKNESS) < 100 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

-- Translated by questport from quests/Q00632_NecromancersRequest (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(632, "Q00632_NecromancersRequest", "Necromancer's Request")

-- Monsters
local VAMPIRES = {21568, 21573, 21582, 21585, 21586, 21587, 21588, 21589, 21590, 21591, 21592, 21593, 21594, 21595}
local UNDEADS = {21547, 21548, 21549, 21551, 21552, 21555, 21556, 21562, 21571, 21576, 21577, 21579}
-- Items
local VAMPIRE_HEART = 7542
local ZOMBIE_BRAIN = 7543

q:register_quest_items(VAMPIRE_HEART, ZOMBIE_BRAIN)
q:add_start_npc(31522) -- Mysterious Wizard
q:add_talk_id(31522)
q:add_kill_id(VAMPIRES)
q:add_kill_id(UNDEADS)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31522-03.htm" then
    st:start_quest()
  elseif event == "31522-06.htm" then
    if q:get_quest_items_count(player, VAMPIRE_HEART) >= 200 then
      st:set_cond(1, true)
      q:take_items(player, VAMPIRE_HEART, -1)
      q:give_adena(player, 120000, true)
    else
      htmltext = "31522-09.htm"
    end
  elseif event == "31522-08.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 63 and "31522-01.htm" or "31522-02.htm"
  elseif sw == State.STARTED then
    htmltext = q:get_quest_items_count(player, VAMPIRE_HEART) >= 200 and "31522-05.htm" or "31522-04.htm"
  end
  return htmltext
end)

-- As in Mobius, an undead that drops no brain still rolls for a heart.
q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil or not st:is_started() then
    return
  end
  local partyMember = st:get_player()
  for _, undead in ipairs(UNDEADS) do
    if undead == npc:id() and rnd(100) < 33 then
      q:give_items(partyMember, ZOMBIE_BRAIN, 1)
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      return
    end
  end
  if st:is_cond(1) and rnd.bool() then
    q:give_items(partyMember, VAMPIRE_HEART, 1)
    if q:get_quest_items_count(partyMember, VAMPIRE_HEART) < 200 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)

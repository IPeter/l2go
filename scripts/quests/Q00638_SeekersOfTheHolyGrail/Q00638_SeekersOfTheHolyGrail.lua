-- Translated by questport from quests/Q00638_SeekersOfTheHolyGrail (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(638, "Q00638_SeekersOfTheHolyGrail", "Seekers of the Holy Grail")

-- NPC
local INNOCENTIN = 31328
-- Item
local PAGAN_TOTEM = 8068

q:register_quest_items(PAGAN_TOTEM)
q:add_start_npc(INNOCENTIN)
q:add_talk_id(INNOCENTIN)
for i = 22138, 22174 do
  q:add_kill_id(i)
end

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31328-02.htm" then
    st:start_quest()
  elseif event == "31328-06.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 73 and "31328-00.htm" or "31328-01.htm"
  elseif sw == State.STARTED then
    if q:get_quest_items_count(player, PAGAN_TOTEM) >= 2000 then
      htmltext = "31328-03.htm"
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:take_items(player, PAGAN_TOTEM, 2000)
      local chance = rnd(3)
      if chance == 0 then
        q:reward_items(player, 959, 1)
      elseif chance == 1 then
        q:reward_items(player, 960, 1)
      else
        q:give_adena(player, 3576000, true)
      end
    else
      htmltext = "31328-04.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_random_party_member_state(killer, -1, 3, npc)
  if qs ~= nil then
    q:give_item_randomly(qs:get_player(), npc, PAGAN_TOTEM, 1, 1, 0, 1, true)
  end
end)

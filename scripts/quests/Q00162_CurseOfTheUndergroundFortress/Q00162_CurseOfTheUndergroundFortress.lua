-- Translated by questport from quests/Q00162_CurseOfTheUndergroundFortress (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(162, "Q00162_CurseOfTheUndergroundFortress", "Curse of the Underground Fortress")

-- Items
local BONE_FRAGMENT = 1158
local ELF_SKULL = 1159
-- Rewards
local BONE_SHIELD = 625
-- Drop chances
local MONSTERS_SKULLS = {}
local MONSTERS_BONES = {}

MONSTERS_SKULLS[20033] = 25 -- Shade Horror
MONSTERS_SKULLS[20345] = 26 -- Dark Terror
MONSTERS_SKULLS[20371] = 23 -- Mist Terror
MONSTERS_BONES[20463] = 25 -- Dungeon Skeleton Archer
MONSTERS_BONES[20464] = 23 -- Dungeon Skeleton
MONSTERS_BONES[20504] = 26 -- Dread Soldier

q:register_quest_items(BONE_FRAGMENT, ELF_SKULL)
q:add_start_npc(30147) -- Unoren
q:add_talk_id(30147)
q:add_kill_id(J.keys(MONSTERS_SKULLS))
q:add_kill_id(J.keys(MONSTERS_BONES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30147-04.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() == Race.DARK_ELF then
      htmltext = "30147-00.htm"
    elseif player:level() < 12 then
      htmltext = "30147-01.htm"
    else
      htmltext = "30147-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "30147-05.htm"
    elseif cond == 2 then
      htmltext = "30147-06.htm"
      q:take_items(player, ELF_SKULL, -1)
      q:take_items(player, BONE_FRAGMENT, -1)
      q:give_items(player, BONE_SHIELD, 1)
      q:give_adena(player, 24000, true)
      st:exit_quest(false, true)
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_cond(1) then
    if MONSTERS_SKULLS[npc:id()] ~= nil then
      if rnd(100) < MONSTERS_SKULLS[npc:id()] then
        local skulls = q:get_quest_items_count(killer, ELF_SKULL)
        if skulls < 3 then
          q:give_items(killer, ELF_SKULL, 1)
          skulls = skulls + 1
          if skulls >= 3 and q:get_quest_items_count(killer, BONE_FRAGMENT) >= 10 then
            qs:set_cond(2, true)
          else
            q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        end
      end
    elseif MONSTERS_BONES[npc:id()] ~= nil and rnd(100) < MONSTERS_BONES[npc:id()] then
      local bones = q:get_quest_items_count(killer, BONE_FRAGMENT)
      if bones < 10 then
        q:give_items(killer, BONE_FRAGMENT, 1)
        bones = bones + 1
        if bones >= 10 and q:get_quest_items_count(killer, ELF_SKULL) >= 3 then
          qs:set_cond(2, true)
        else
          q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  end
end)

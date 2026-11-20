-- Translated by questport from quests/Q00365_DevilsLegacy (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(365, "Q00365_DevilsLegacy", "Devil's Legacy")

-- NPCs
local RANDOLF = 30095
local COLLOB = 30092
-- Item
local PIRATE_TREASURE_CHEST = 5873

q:register_quest_items(PIRATE_TREASURE_CHEST)
q:add_start_npc(RANDOLF)
q:add_talk_id(RANDOLF, COLLOB)
q:add_kill_id(20836, 20845, 21629, 21630) -- Pirate Zombie && Pirate Zombie Captain.

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30095-02.htm" then
    st:start_quest()
  elseif event == "30095-06.htm" then
    st:exit_quest(true, true)
  elseif event == "30092-05.htm" then
    if not q:has_quest_items(player, PIRATE_TREASURE_CHEST) then
      htmltext = "30092-02.htm"
    elseif q:get_quest_items_count(player, 57) < 600 then
      htmltext = "30092-03.htm"
    else
      q:take_items(player, PIRATE_TREASURE_CHEST, 1)
      q:take_items(player, 57, 600)
      local i0
      if rnd(100) < 80 then
        i0 = rnd(100)
        if i0 < 1 then
          q:give_items(player, 955, 1)
        elseif i0 < 4 then
          q:give_items(player, 956, 1)
        elseif i0 < 36 then
          q:give_items(player, 1868, 1)
        elseif i0 < 68 then
          q:give_items(player, 1884, 1)
        else
          q:give_items(player, 1872, 1)
        end
        htmltext = "30092-05.htm"
      else
        i0 = rnd(1000)
        if i0 < 10 then
          q:give_items(player, 951, 1)
        elseif i0 < 40 then
          q:give_items(player, 952, 1)
        elseif i0 < 60 then
          q:give_items(player, 955, 1)
        elseif i0 < 260 then
          q:give_items(player, 956, 1)
        elseif i0 < 445 then
          q:give_items(player, 1879, 1)
        elseif i0 < 630 then
          q:give_items(player, 1880, 1)
        elseif i0 < 815 then
          q:give_items(player, 1882, 1)
        else
          q:give_items(player, 1881, 1)
        end
        htmltext = "30092-06.htm"
        -- Curse effect !
        local skill = skills.get(4082, 1)
        if skill ~= nil and not player:is_affected_by_skill(skill:id()) then
          skills.apply_effects(skill, npc, player)
        end
      end
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 39 and "30095-00.htm" or "30095-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == RANDOLF then
      if not q:has_quest_items(player, PIRATE_TREASURE_CHEST) then
        htmltext = "30095-03.htm"
      else
        htmltext = "30095-05.htm"
        local reward = q:get_quest_items_count(player, PIRATE_TREASURE_CHEST) * 1600
        q:take_items(player, PIRATE_TREASURE_CHEST, -1)
        q:give_adena(player, reward, true)
      end
    elseif sw2 == COLLOB then
      htmltext = "30092-01.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local qs = q:get_random_party_member_state(player, -1, 3, npc)
  if qs ~= nil then
    q:give_item_randomly(qs:get_player(), npc, PIRATE_TREASURE_CHEST, 1, 1, 0, npc:id() == 20836 and 0.36 or 0.52, true)
  end
end)

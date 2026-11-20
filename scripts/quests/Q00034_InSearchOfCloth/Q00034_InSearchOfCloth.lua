-- Translated by questport from quests/Q00034_InSearchOfCloth (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(34, "Q00034_InSearchOfCloth", "In Search of Cloth")

-- NPCs
local RADIA = 30088
local RALFORD = 30165
local VARAN = 30294
-- Monsters
local TRISALIM_SPIDER = 20560
local TRISALIM_TARANTULA = 20561
-- Items
local SPINNERET = 7528
local SUEDE = 1866
local THREAD = 1868
local SPIDERSILK = 7161
-- Rewards
local MYSTERIOUS_CLOTH = 7076

q:register_quest_items(SPINNERET, SPIDERSILK)
q:add_start_npc(RADIA)
q:add_talk_id(RADIA, RALFORD, VARAN)
q:add_kill_id(TRISALIM_SPIDER, TRISALIM_TARANTULA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30088-1.htm" then
    st:start_quest()
  elseif event == "30294-1.htm" then
    st:set_cond(2, true)
  elseif event == "30088-3.htm" then
    st:set_cond(3, true)
  elseif event == "30165-1.htm" then
    st:set_cond(4, true)
  elseif event == "30165-3.htm" then
    st:set_cond(6, true)
    q:take_items(player, SPINNERET, 10)
    q:give_items(player, SPIDERSILK, 1)
  elseif event == "30088-5.htm" then
    if q:get_quest_items_count(player, SUEDE) >= 3000 and q:get_quest_items_count(player, THREAD) >= 5000 and q:has_quest_items(player, SPIDERSILK) then
      q:take_items(player, SPIDERSILK, 1)
      q:take_items(player, SUEDE, 3000)
      q:take_items(player, THREAD, 5000)
      q:give_items(player, MYSTERIOUS_CLOTH, 1)
      st:exit_quest(false, true)
    else
      htmltext = "30088-4a.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() >= 60 then
      local fwear = player:get_quest_state("Q00037_MakeFormalWear")
      if fwear ~= nil and fwear:is_cond(6) then
        htmltext = "30088-0.htm"
      else
        htmltext = "30088-0a.htm"
      end
    else
      htmltext = "30088-0b.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == RADIA then
      if cond == 1 then
        htmltext = "30088-1a.htm"
      elseif cond == 2 then
        htmltext = "30088-2.htm"
      elseif cond == 3 then
        htmltext = "30088-3a.htm"
      elseif cond == 6 then
        if q:get_quest_items_count(player, SUEDE) < 3000 or q:get_quest_items_count(player, THREAD) < 5000 or not q:has_quest_items(player, SPIDERSILK) then
          htmltext = "30088-4a.htm"
        else
          htmltext = "30088-4.htm"
        end
      end
    elseif sw2 == VARAN then
      if cond == 1 then
        htmltext = "30294-0.htm"
      elseif cond > 1 then
        htmltext = "30294-1a.htm"
      end
    elseif sw2 == RALFORD then
      if cond == 3 then
        htmltext = "30165-0.htm"
      elseif cond == 4 and q:get_quest_items_count(player, SPINNERET) < 10 then
        htmltext = "30165-1a.htm"
      elseif cond == 5 then
        htmltext = "30165-2.htm"
      elseif cond > 5 then
        htmltext = "30165-3a.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local member = q:get_random_party_member(player, 4)
  if member ~= nil and rnd.bool() then
    local qs = q:get_quest_state(member, false)
    q:give_items(member, SPINNERET, 1)
    if q:get_quest_items_count(member, SPINNERET) >= 10 then
      qs:set_cond(5, true)
    else
      q:play_sound(member, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)

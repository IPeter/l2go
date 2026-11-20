-- Translated by questport from quests/Q00036_MakeASewingKit (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(36, "Q00036_MakeASewingKit", "Make a Sewing Kit")

-- Items
local REINFORCED_STEEL = 7163
local ARTISANS_FRAME = 1891
local ORIHARUKON = 1893
-- Reward
local SEWING_KIT = 7078

q:register_quest_items(REINFORCED_STEEL)
q:add_start_npc(30847) -- Ferris
q:add_talk_id(30847)
q:add_kill_id(20566) -- Iron Golem

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30847-1.htm" then
    st:start_quest()
  elseif event == "30847-3.htm" then
    st:set_cond(3, true)
    q:take_items(player, REINFORCED_STEEL, 5)
  elseif event == "30847-5.htm" then
    if q:get_quest_items_count(player, ORIHARUKON) >= 10 and q:get_quest_items_count(player, ARTISANS_FRAME) >= 10 then
      q:take_items(player, ARTISANS_FRAME, 10)
      q:take_items(player, ORIHARUKON, 10)
      q:give_items(player, SEWING_KIT, 1)
      st:exit_quest(false, true)
    else
      htmltext = "30847-4a.htm"
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
        htmltext = "30847-0.htm"
      else
        htmltext = "30847-0a.htm"
      end
    else
      htmltext = "30847-0b.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "30847-1a.htm"
    elseif cond == 2 then
      htmltext = "30847-2.htm"
    elseif cond == 3 then
      htmltext = (q:get_quest_items_count(player, ORIHARUKON) < 10 or q:get_quest_items_count(player, ARTISANS_FRAME) < 10) and "30847-4a.htm" or "30847-4.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local member = q:get_random_party_member(player, 1)
  if member ~= nil and rnd.bool() then
    q:give_items(member, REINFORCED_STEEL, 1)
    if q:get_quest_items_count(member, REINFORCED_STEEL) >= 5 then
      q:get_quest_state(member, false):set_cond(2, true)
    else
      q:play_sound(member, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)

-- Translated by questport from quests/Q00035_FindGlitteringJewelry (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(35, "Q00035_FindGlitteringJewelry", "Find Glittering Jewelry")

-- NPCs
local ELLIE = 30091
local FELTON = 30879
-- Items
local ROUGH_JEWEL = 7162
local ORIHARUKON = 1893
local SILVER_NUGGET = 1873
local THONS = 4044
-- Reward
local JEWEL_BOX = 7077

q:register_quest_items(ROUGH_JEWEL)
q:add_start_npc(ELLIE)
q:add_talk_id(ELLIE, FELTON)
q:add_kill_id(20135) -- Alligator

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30091-1.htm" then
    st:start_quest()
  elseif event == "30879-1.htm" then
    st:set_cond(2, true)
  elseif event == "30091-3.htm" then
    st:set_cond(4, true)
    q:take_items(player, ROUGH_JEWEL, 10)
  elseif event == "30091-5.htm" then
    if q:get_quest_items_count(player, ORIHARUKON) >= 5 and q:get_quest_items_count(player, SILVER_NUGGET) >= 500 and q:get_quest_items_count(player, THONS) >= 150 then
      q:take_items(player, ORIHARUKON, 5)
      q:take_items(player, SILVER_NUGGET, 500)
      q:take_items(player, THONS, 150)
      q:give_items(player, JEWEL_BOX, 1)
      st:exit_quest(false, true)
    else
      htmltext = "30091-4a.htm"
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
        htmltext = "30091-0.htm"
      else
        htmltext = "30091-0a.htm"
      end
    else
      htmltext = "30091-0b.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ELLIE then
      if cond == 1 or cond == 2 then
        htmltext = "30091-1a.htm"
      elseif cond == 3 then
        htmltext = "30091-2.htm"
      elseif cond == 4 then
        htmltext = q:get_quest_items_count(player, ORIHARUKON) >= 5 and q:get_quest_items_count(player, SILVER_NUGGET) >= 500 and q:get_quest_items_count(player, THONS) >= 150 and "30091-4.htm" or "30091-4a.htm"
      end
    elseif sw2 == FELTON then
      if cond == 1 then
        htmltext = "30879-0.htm"
      elseif cond > 1 then
        htmltext = "30879-1a.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local member = q:get_random_party_member(player, 2)
  if member ~= nil and rnd.bool() then
    q:give_items(member, ROUGH_JEWEL, 1)
    if q:get_quest_items_count(member, ROUGH_JEWEL) >= 10 then
      q:get_quest_state(member, false):set_cond(3, true)
    else
      q:play_sound(member, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)

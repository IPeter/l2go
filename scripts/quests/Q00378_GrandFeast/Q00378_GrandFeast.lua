-- Translated by questport from quests/Q00378_GrandFeast (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(378, "Q00378_GrandFeast", "Magnificent Feast")

-- NPC
local RANSPO = 30594
-- Items
local WINE_15 = 5956
local WINE_30 = 5957
local WINE_60 = 5958
local MUSICAL_SCORE = 4421
local SALAD_RECIPE = 1455
local SAUCE_RECIPE = 1456
local STEAK_RECIPE = 1457
local RITRON_DESSERT = 5959
-- Rewards
local REWARDS = {}

REWARDS["9"] = {847, 1, 5700}
REWARDS["10"] = {846, 2, 0}
REWARDS["12"] = {909, 1, 25400}
REWARDS["17"] = {846, 2, 1200}
REWARDS["18"] = {879, 1, 6900}
REWARDS["20"] = {890, 2, 8500}
REWARDS["33"] = {879, 1, 8100}
REWARDS["34"] = {910, 1, 0}
REWARDS["36"] = {848, 1, 2200}

q:add_start_npc(RANSPO)
q:add_talk_id(RANSPO)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30594-2.htm" then
    st:start_quest()
  elseif event == "30594-4a.htm" then
    if q:has_quest_items(player, WINE_15) then
      st:set_cond(2, true)
      st:set("score", "1")
      q:take_items(player, WINE_15, 1)
    else
      htmltext = "30594-4.htm"
    end
  elseif event == "30594-4b.htm" then
    if q:has_quest_items(player, WINE_30) then
      st:set_cond(2, true)
      st:set("score", "2")
      q:take_items(player, WINE_30, 1)
    else
      htmltext = "30594-4.htm"
    end
  elseif event == "30594-4c.htm" then
    if q:has_quest_items(player, WINE_60) then
      st:set_cond(2, true)
      st:set("score", "4")
      q:take_items(player, WINE_60, 1)
    else
      htmltext = "30594-4.htm"
    end
  elseif event == "30594-6.htm" then
    if q:has_quest_items(player, MUSICAL_SCORE) then
      st:set_cond(3, true)
      q:take_items(player, MUSICAL_SCORE, 1)
    else
      htmltext = "30594-5.htm"
    end
  elseif event == "30594-8a.htm" then
    if q:has_quest_items(player, SALAD_RECIPE) then
      st:set_cond(4, true)
      local score = st:get_int("score")
      st:set("score", tostring(score + 8))
      q:take_items(player, SALAD_RECIPE, 1)
    else
      htmltext = "30594-8.htm"
    end
  elseif event == "30594-8b.htm" then
    if q:has_quest_items(player, SAUCE_RECIPE) then
      st:set_cond(4, true)
      local score = st:get_int("score")
      st:set("score", tostring(score + 16))
      q:take_items(player, SAUCE_RECIPE, 1)
    else
      htmltext = "30594-8.htm"
    end
  elseif event == "30594-8c.htm" then
    if q:has_quest_items(player, STEAK_RECIPE) then
      st:set_cond(4, true)
      local score = st:get_int("score")
      st:set("score", tostring(score + 32))
      q:take_items(player, STEAK_RECIPE, 1)
    else
      htmltext = "30594-8.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 20 and "30594-0.htm" or "30594-1.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "30594-3.htm"
    elseif cond == 2 then
      htmltext = not q:has_quest_items(player, MUSICAL_SCORE) and "30594-5.htm" or "30594-5a.htm"
    elseif cond == 3 then
      htmltext = "30594-7.htm"
    elseif cond == 4 then
      local score = st:get("score")
      if REWARDS[score] ~= nil and q:has_quest_items(player, RITRON_DESSERT) then
        htmltext = "30594-10.htm"
        q:take_items(player, RITRON_DESSERT, 1)
        q:give_items(player, REWARDS[score][1], REWARDS[score][2])
        local adena = REWARDS[score][3]
        if adena > 0 then
          q:give_adena(player, adena, true)
        end
        st:exit_quest(true, true)
      else
        htmltext = "30594-9.htm"
      end
    end
  end
  return htmltext
end)

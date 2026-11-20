-- Translated by questport from quests/Q00344_1000YearsTheEndOfLamentation (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(344, "Q00344_1000YearsTheEndOfLamentation", "1000 years, the End of Lamentation")

-- NPCs
local GILMORE = 30754
local RODEMAI = 30756
local ORVEN = 30857
local KAIEN = 30623
local GARVARENTZ = 30704
-- Items
local ARTICLE_DEAD_HERO = 4269
local OLD_KEY = 4270
local OLD_HILT = 4271
local OLD_TOTEM = 4272
local CRUCIFIX = 4273
-- Drop chances
local CHANCES = {}

local function rewards(st, npcId)
  local player = st:get_player()
  if npcId == ORVEN then
    if q:has_quest_items(player, CRUCIFIX) then
      st:set("success", "1")
      q:take_items(player, CRUCIFIX, -1)
      local chance = rnd(100)
      if chance < 80 then
        q:give_items(player, 1875, 19)
      elseif chance < 95 then
        q:give_items(player, 952, 5)
      else
        q:give_items(player, 2437, 1)
      end
    end
  elseif npcId == GARVARENTZ then
    if q:has_quest_items(player, OLD_TOTEM) then
      st:set("success", "1")
      q:take_items(player, OLD_TOTEM, -1)
      local chance = rnd(100)
      if chance < 55 then
        q:give_items(player, 1882, 70)
      elseif chance < 99 then
        q:give_items(player, 1881, 50)
      else
        q:give_items(player, 191, 1)
      end
    end
  elseif npcId == KAIEN then
    if q:has_quest_items(player, OLD_HILT) then
      st:set("success", "1")
      q:take_items(player, OLD_HILT, -1)
      local chance = rnd(100)
      if chance < 60 then
        q:give_items(player, 1874, 25)
      elseif chance < 85 then
        q:give_items(player, 1887, 10)
      elseif chance < 99 then
        q:give_items(player, 951, 1)
      else
        q:give_items(player, 133, 1)
      end
    end
  elseif npcId == RODEMAI then
    if q:has_quest_items(player, OLD_KEY) then
      st:set("success", "1")
      q:take_items(player, OLD_KEY, -1)
      local chance = rnd(100)
      if chance < 80 then
        q:give_items(player, 1879, 55)
      elseif chance < 95 then
        q:give_items(player, 951, 1)
      else
        q:give_items(player, 885, 1)
      end
    end
  end
end

CHANCES[20236] = 380000
CHANCES[20237] = 490000
CHANCES[20238] = 460000
CHANCES[20239] = 490000
CHANCES[20240] = 530000
CHANCES[20272] = 380000
CHANCES[20273] = 490000
CHANCES[20274] = 460000
CHANCES[20275] = 490000
CHANCES[20276] = 530000

q:register_quest_items(ARTICLE_DEAD_HERO, OLD_KEY, OLD_HILT, OLD_TOTEM, CRUCIFIX)
q:add_start_npc(GILMORE)
q:add_talk_id(GILMORE, RODEMAI, ORVEN, GARVARENTZ, KAIEN)
q:add_kill_id(20236, 20237, 20238, 20239, 20240, 20272, 20273, 20274, 20275, 20276)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30754-04.htm" then
    st:start_quest()
  elseif event == "30754-07.htm" then
    if st:get("success") ~= nil then
      st:set_cond(1, true)
      st:unset("success")
    end
  elseif event == "30754-08.htm" then
    st:exit_quest(true, true)
  elseif event == "30754-06.htm" then
    if not q:has_quest_items(player, ARTICLE_DEAD_HERO) then
      htmltext = "30754-06a.htm"
    else
      local amount = q:get_quest_items_count(player, ARTICLE_DEAD_HERO)
      q:take_items(player, ARTICLE_DEAD_HERO, -1)
      q:give_adena(player, amount * 60, true)
      -- Special item, % based on actual number of qItems.
      if rnd(1000) < math.min(10, math.max(1, J.idiv(amount, 10))) then
        htmltext = "30754-10.htm"
      end
    end
  elseif event == "30754-11.htm" then
    local random = rnd(4)
    if random < 1 then
      htmltext = "30754-12.htm"
      q:give_items(player, OLD_KEY, 1)
    elseif random < 2 then
      htmltext = "30754-13.htm"
      q:give_items(player, OLD_HILT, 1)
    elseif random < 3 then
      htmltext = "30754-14.htm"
      q:give_items(player, OLD_TOTEM, 1)
    else
      q:give_items(player, CRUCIFIX, 1)
    end
    st:set_cond(2, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 48 and "30754-01.htm" or "30754-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == GILMORE then
      if cond == 1 then
        htmltext = q:has_quest_items(player, ARTICLE_DEAD_HERO) and "30754-05.htm" or "30754-09.htm"
      elseif cond == 2 then
        htmltext = st:get("success") ~= nil and "30754-16.htm" or "30754-15.htm"
      end
    else
      if cond == 2 then
        if st:get("success") ~= nil then
          htmltext = npc:id() .. "-02.htm"
        else
          rewards(st, npc:id())
          htmltext = npc:id() .. "-01.htm"
        end
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if rnd(1000000) < CHANCES[npc:id()] then
    q:give_items(player, ARTICLE_DEAD_HERO, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)

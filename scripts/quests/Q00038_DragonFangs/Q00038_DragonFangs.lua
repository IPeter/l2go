-- Translated by questport from quests/Q00038_DragonFangs (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(38, "Q00038_DragonFangs", "Dragon Fangs")

-- NPCs
local LUIS = 30386
local IRIS = 30034
local ROHMER = 30344
-- Items
local FEATHER_ORNAMENT = 7173
local TOOTH_OF_TOTEM = 7174
local TOOTH_OF_DRAGON = 7175
local LETTER_OF_IRIS = 7176
local LETTER_OF_ROHMER = 7177
-- Reward { item, adena }
local REWARD = {{45, 5200}, {627, 1500}, {1123, 3200}, {605, 3200}}
-- Droplist
local DROPLIST = {}

DROPLIST[21100] = {1, FEATHER_ORNAMENT, 100, 1000000}
DROPLIST[20357] = {1, FEATHER_ORNAMENT, 100, 1000000}
DROPLIST[21101] = {6, TOOTH_OF_DRAGON, 50, 500000}
DROPLIST[20356] = {6, TOOTH_OF_DRAGON, 50, 500000}

q:register_quest_items(FEATHER_ORNAMENT, TOOTH_OF_TOTEM, TOOTH_OF_DRAGON, LETTER_OF_IRIS, LETTER_OF_ROHMER)
q:add_start_npc(LUIS)
q:add_talk_id(LUIS, IRIS, ROHMER)
q:add_kill_id(J.keys(DROPLIST))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30386-02.htm" then
    st:start_quest()
  elseif event == "30386-04.htm" then
    st:set_cond(3, true)
    q:take_items(player, FEATHER_ORNAMENT, 100)
    q:give_items(player, TOOTH_OF_TOTEM, 1)
  elseif event == "30034-02a.htm" then
    if q:has_quest_items(player, TOOTH_OF_TOTEM) then
      htmltext = "30034-02.htm"
      st:set_cond(4, true)
      q:take_items(player, TOOTH_OF_TOTEM, 1)
      q:give_items(player, LETTER_OF_IRIS, 1)
    end
  elseif event == "30344-02a.htm" then
    if q:has_quest_items(player, LETTER_OF_IRIS) then
      htmltext = "30344-02.htm"
      st:set_cond(5, true)
      q:take_items(player, LETTER_OF_IRIS, 1)
      q:give_items(player, LETTER_OF_ROHMER, 1)
    end
  elseif event == "30034-04a.htm" then
    if q:has_quest_items(player, LETTER_OF_ROHMER) then
      htmltext = "30034-04.htm"
      st:set_cond(6, true)
      q:take_items(player, LETTER_OF_ROHMER, 1)
    end
  elseif event == "30034-06a.htm" then
    if q:get_quest_items_count(player, TOOTH_OF_DRAGON) >= 50 then
      local position = rnd(#REWARD)
      htmltext = "30034-06.htm"
      q:take_items(player, TOOTH_OF_DRAGON, 50)
      q:give_items(player, REWARD[position + 1][1], 1)
      q:give_adena(player, REWARD[position + 1][2], true)
      st:exit_quest(false, true)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 19 and "30386-01a.htm" or "30386-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == LUIS then
      if cond == 1 then
        htmltext = "30386-02a.htm"
      elseif cond == 2 then
        htmltext = "30386-03.htm"
      elseif cond > 2 then
        htmltext = "30386-03a.htm"
      end
    elseif sw2 == IRIS then
      if cond == 3 then
        htmltext = "30034-01.htm"
      elseif cond == 4 then
        htmltext = "30034-02b.htm"
      elseif cond == 5 then
        htmltext = "30034-03.htm"
      elseif cond == 6 then
        htmltext = "30034-05a.htm"
      elseif cond == 7 then
        htmltext = "30034-05.htm"
      end
    elseif sw2 == ROHMER then
      if cond == 4 then
        htmltext = "30344-01.htm"
      elseif cond > 4 then
        htmltext = "30344-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local droplist = DROPLIST[npc:id()]
  if droplist ~= nil and st:is_cond(droplist[1]) then
    local itemId = droplist[2]
    local amount = droplist[3]
    local chance = droplist[4]
    if rnd(1000000) < chance then
      q:give_items(player, itemId, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      if q:get_quest_items_count(player, itemId) >= amount then
        st:set_cond(droplist[1] + 1, true)
      end
    end
  end
end)

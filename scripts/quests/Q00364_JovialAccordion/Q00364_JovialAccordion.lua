-- Translated by questport from quests/Q00364_JovialAccordion (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(364, "Q00364_JovialAccordion", "Jovial Accordion")

-- NPCs
local BARBADO = 30959
local SWAN = 30957
local SABRIN = 30060
local XABER = 30075
local CLOTH_CHEST = 30961
local BEER_CHEST = 30960
-- Items
local KEY_1 = 4323
local KEY_2 = 4324
local STOLEN_BEER = 4321
local STOLEN_CLOTHES = 4322
local ECHO = 4421

q:register_quest_items(KEY_1, KEY_2, STOLEN_BEER, STOLEN_CLOTHES)
q:add_start_npc(BARBADO)
q:add_talk_id(BARBADO, SWAN, SABRIN, XABER, CLOTH_CHEST, BEER_CHEST)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30959-02.htm" then
    st:start_quest()
    st:set("items", "0")
  elseif event == "30957-02.htm" then
    st:set_cond(2, true)
    q:give_items(player, KEY_1, 1)
    q:give_items(player, KEY_2, 1)
  elseif event == "30960-04.htm" then
    if q:has_quest_items(player, KEY_2) then
      q:take_items(player, KEY_2, 1)
      if rnd.bool() then
        htmltext = "30960-02.htm"
        q:give_items(player, STOLEN_BEER, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif event == "30961-04.htm" then
    if q:has_quest_items(player, KEY_1) then
      q:take_items(player, KEY_1, 1)
      if rnd.bool() then
        htmltext = "30961-02.htm"
        q:give_items(player, STOLEN_CLOTHES, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
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
    htmltext = player:level() < 15 and "30959-00.htm" or "30959-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local stolenItems = st:get_int("items")
    local sw2 = npc:id()
    if sw2 == BARBADO then
      if cond == 1 or cond == 2 then
        htmltext = "30959-03.htm"
      elseif cond == 3 then
        htmltext = "30959-04.htm"
        q:give_items(player, ECHO, 1)
        st:exit_quest(true, true)
      end
    elseif sw2 == SWAN then
      if cond == 1 then
        htmltext = "30957-01.htm"
      elseif cond == 2 then
        if stolenItems > 0 then
          st:set_cond(3, true)
          if stolenItems == 2 then
            htmltext = "30957-04.htm"
            q:give_adena(player, 100, true)
          else
            htmltext = "30957-05.htm"
          end
        else
          if not q:has_quest_items(player, KEY_1) and not q:has_quest_items(player, KEY_2) then
            htmltext = "30957-06.htm"
            st:exit_quest(true, true)
          else
            htmltext = "30957-03.htm"
          end
        end
      elseif cond == 3 then
        htmltext = "30957-07.htm"
      end
    elseif sw2 == BEER_CHEST then
      htmltext = "30960-03.htm"
      if cond == 2 and q:has_quest_items(player, KEY_2) then
        htmltext = "30960-01.htm"
      end
    elseif sw2 == CLOTH_CHEST then
      htmltext = "30961-03.htm"
      if cond == 2 and q:has_quest_items(player, KEY_1) then
        htmltext = "30961-01.htm"
      end
    elseif sw2 == SABRIN then
      if q:has_quest_items(player, STOLEN_BEER) then
        htmltext = "30060-01.htm"
        st:set("items", tostring(stolenItems + 1))
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:take_items(player, STOLEN_BEER, 1)
      else
        htmltext = "30060-02.htm"
      end
    elseif sw2 == XABER then
      if q:has_quest_items(player, STOLEN_CLOTHES) then
        htmltext = "30075-01.htm"
        st:set("items", tostring(stolenItems + 1))
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:take_items(player, STOLEN_CLOTHES, 1)
      else
        htmltext = "30075-02.htm"
      end
    end
  end
  return htmltext
end)

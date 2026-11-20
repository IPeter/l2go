-- Translated by questport from quests/Q00381_LetsBecomeARoyalMember (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(381, "Q00381_LetsBecomeARoyalMember", "Let's Become a Royal Member!")

-- NPCs
local SORINT = 30232
local SANDRA = 30090
-- Items
local KAIL_COIN = 5899
local COIN_ALBUM = 5900
local GOLDEN_CLOVER_COIN = 7569
local COIN_COLLECTOR_MEMBERSHIP = 3813
-- Reward
local ROYAL_MEMBERSHIP = 5898

q:register_quest_items(KAIL_COIN, GOLDEN_CLOVER_COIN)
q:add_start_npc(SORINT)
q:add_talk_id(SORINT, SANDRA)
q:add_kill_id(21018, 27316)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30090-02.htm" then
    st:set("aCond", "1") -- Alternative cond used for Sandra.
  elseif event == "30232-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = (player:level() < 55 or not q:has_quest_items(player, COIN_COLLECTOR_MEMBERSHIP)) and "30232-02.htm" or "30232-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == SORINT then
      if not q:has_quest_items(player, KAIL_COIN) then
        htmltext = "30232-04.htm"
      elseif not q:has_quest_items(player, COIN_ALBUM) then
        htmltext = "30232-05.htm"
      else
        htmltext = "30232-06.htm"
        q:take_items(player, KAIL_COIN, -1)
        q:take_items(player, COIN_ALBUM, -1)
        q:give_items(player, ROYAL_MEMBERSHIP, 1)
        st:exit_quest(true, true)
      end
    elseif sw2 == SANDRA then
      if not q:has_quest_items(player, COIN_ALBUM) then
        if st:get_int("aCond") == 0 then
          htmltext = "30090-01.htm"
        else
          if not q:has_quest_items(player, GOLDEN_CLOVER_COIN) then
            htmltext = "30090-03.htm"
          else
            htmltext = "30090-04.htm"
            q:take_items(player, GOLDEN_CLOVER_COIN, -1)
            q:give_items(player, COIN_ALBUM, 1)
          end
        end
      else
        htmltext = "30090-05.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if npc:id() == 21018 then
    if not q:has_quest_items(player, KAIL_COIN) and rnd(100) < 5 then
      q:give_items(player, KAIL_COIN, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif st:get_int("aCond") == 1 then
    if not q:has_quest_items(player, GOLDEN_CLOVER_COIN) then
      q:give_items(player, GOLDEN_CLOVER_COIN, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)

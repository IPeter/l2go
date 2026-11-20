-- Translated by questport from quests/Q00345_MethodToRaiseTheDead (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(345, "Q00345_MethodToRaiseTheDead", "Method to Raise the Dead")

-- NPCs
local XENOVIA = 30912
local DOROTHY = 30970
local ORPHEUS = 30971
local MEDIUM_JAR = 30973
-- Items
local VICTIM_ARM_BONE = 4274
local VICTIM_THIGH_BONE = 4275
local VICTIM_SKULL = 4276
local VICTIM_RIB_BONE = 4277
local VICTIM_SPINE = 4278
local USELESS_BONE_PIECES = 4280
local POWDER_TO_SUMMON_DEAD_SOULS = 4281
-- Rewards
local BILL_OF_IASON_HEINE = 4407
local IMPERIAL_DIAMOND = 3456

q:register_quest_items(VICTIM_ARM_BONE, VICTIM_THIGH_BONE, VICTIM_SKULL, VICTIM_RIB_BONE, VICTIM_SPINE, POWDER_TO_SUMMON_DEAD_SOULS, USELESS_BONE_PIECES)
q:add_start_npc(DOROTHY)
q:add_talk_id(DOROTHY, XENOVIA, MEDIUM_JAR, ORPHEUS)
q:add_kill_id(20789, 20791)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30970-03.htm" then
    st:start_quest()
  elseif event == "30970-06.htm" then
    st:set_cond(2, true)
  elseif event == "30912-04.htm" then
    if player:adena() >= 1000 then
      htmltext = "30912-03.htm"
      st:set_cond(3, true)
      q:take_items(player, 57, 1000)
      q:give_items(player, POWDER_TO_SUMMON_DEAD_SOULS, 1)
    end
  elseif event == "30973-04.htm" then
    if st:is_cond(3) then
      local chance = rnd(3)
      if chance == 0 then
        st:set_cond(6, true)
        htmltext = "30973-02a.htm"
      elseif chance == 1 then
        st:set_cond(6, true)
        htmltext = "30973-02b.htm"
      else
        st:set_cond(7, true)
        htmltext = "30973-02c.htm"
      end
      q:take_items(player, POWDER_TO_SUMMON_DEAD_SOULS, -1)
      q:take_items(player, VICTIM_ARM_BONE, -1)
      q:take_items(player, VICTIM_THIGH_BONE, -1)
      q:take_items(player, VICTIM_SKULL, -1)
      q:take_items(player, VICTIM_RIB_BONE, -1)
      q:take_items(player, VICTIM_SPINE, -1)
    end
  elseif event == "30971-02a.htm" then
    if q:has_quest_items(player, USELESS_BONE_PIECES) then
      htmltext = "30971-02.htm"
    end
  elseif event == "30971-03.htm" then
    if q:has_quest_items(player, USELESS_BONE_PIECES) then
      local amount = q:get_quest_items_count(player, USELESS_BONE_PIECES) * 104
      q:take_items(player, USELESS_BONE_PIECES, -1)
      q:give_adena(player, amount, true)
    else
      htmltext = "30971-02a.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 35 and "30970-00.htm" or "30970-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == DOROTHY then
      if cond == 1 then
        htmltext = not q:has_quest_items(player, VICTIM_ARM_BONE, VICTIM_THIGH_BONE, VICTIM_SKULL, VICTIM_RIB_BONE, VICTIM_SPINE) and "30970-04.htm" or "30970-05.htm"
      elseif cond == 2 then
        htmltext = "30970-07.htm"
      elseif cond > 2 and cond < 6 then
        htmltext = "30970-08.htm"
      else
        -- Shared part between cond 6 and 7.
        local amount = q:get_quest_items_count(player, USELESS_BONE_PIECES) * 70
        q:take_items(player, USELESS_BONE_PIECES, -1)
        -- Scaried little girl
        if cond == 7 then
          htmltext = "30970-10.htm"
          q:give_adena(player, 3040 + amount, true)
          -- Reward can be either an Imperial Diamond or bills.
          if rnd(100) < 10 then
            q:give_items(player, IMPERIAL_DIAMOND, 1)
          else
            q:give_items(player, BILL_OF_IASON_HEINE, 5)
          end
        else
          htmltext = "30970-09.htm"
          q:give_adena(player, 5390 + amount, true)
          q:give_items(player, BILL_OF_IASON_HEINE, 3)
        end
        st:exit_quest(true, true)
      end
    elseif sw2 == XENOVIA then
      if cond == 2 then
        htmltext = "30912-01.htm"
      elseif cond > 2 then
        htmltext = "30912-06.htm"
      end
    elseif sw2 == MEDIUM_JAR then
      htmltext = "30973-01.htm"
    elseif sw2 == ORPHEUS then
      htmltext = "30971-01.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if rnd(4) == 0 then
    local randomPart = rnd(VICTIM_ARM_BONE, VICTIM_SPINE)
    if not q:has_quest_items(player, randomPart) then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(player, randomPart, 1)
      return
    end
  end
  q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  q:give_items(player, USELESS_BONE_PIECES, 1)
end)

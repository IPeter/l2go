-- Translated by questport from quests/Q00166_MassOfDarkness (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(166, "Q00166_MassOfDarkness", "Mass of Darkness")

-- NPCs
local UNDRIAS = 30130
local IRIA = 30135
local DORANKUS = 30139
local TRUDY = 30143
-- Items
local UNDRIAS_LETTER = 1088
local CEREMONIAL_DAGGER = 1089
local DREVIANT_WINE = 1090
local GARMIEL_SCRIPTURE = 1091

q:register_quest_items(UNDRIAS_LETTER, CEREMONIAL_DAGGER, DREVIANT_WINE, GARMIEL_SCRIPTURE)
q:add_start_npc(UNDRIAS)
q:add_talk_id(UNDRIAS, IRIA, DORANKUS, TRUDY)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30130-04.htm" then
    st:start_quest()
    q:give_items(player, UNDRIAS_LETTER, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DARK_ELF then
      htmltext = "30130-00.htm"
    elseif player:level() < 2 then
      htmltext = "30130-02.htm"
    else
      htmltext = "30130-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == UNDRIAS then
      if cond == 1 then
        htmltext = "30130-05.htm"
      elseif cond == 2 then
        htmltext = "30130-06.htm"
        q:take_items(player, CEREMONIAL_DAGGER, 1)
        q:take_items(player, DREVIANT_WINE, 1)
        q:take_items(player, GARMIEL_SCRIPTURE, 1)
        q:take_items(player, UNDRIAS_LETTER, 1)
        q:give_adena(player, 500, true)
        q:add_exp_and_sp(player, 500, 0)
        st:exit_quest(false, true)
      end
    elseif sw2 == IRIA then
      if cond == 1 and not q:has_quest_items(player, CEREMONIAL_DAGGER) then
        htmltext = "30135-01.htm"
        q:give_items(player, CEREMONIAL_DAGGER, 1)
        if q:has_quest_items(player, DREVIANT_WINE, GARMIEL_SCRIPTURE) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif cond == 2 then
        htmltext = "30135-02.htm"
      end
    elseif sw2 == DORANKUS then
      if cond == 1 and not q:has_quest_items(player, DREVIANT_WINE) then
        htmltext = "30139-01.htm"
        q:give_items(player, DREVIANT_WINE, 1)
        if q:has_quest_items(player, CEREMONIAL_DAGGER, GARMIEL_SCRIPTURE) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif cond == 2 then
        htmltext = "30139-02.htm"
      end
    elseif sw2 == TRUDY then
      if cond == 1 and not q:has_quest_items(player, GARMIEL_SCRIPTURE) then
        htmltext = "30143-01.htm"
        q:give_items(player, GARMIEL_SCRIPTURE, 1)
        if q:has_quest_items(player, CEREMONIAL_DAGGER, DREVIANT_WINE) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif cond == 2 then
        htmltext = "30143-02.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

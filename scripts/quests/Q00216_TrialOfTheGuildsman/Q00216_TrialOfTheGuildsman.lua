-- Translated by questport from quests/Q00216_TrialOfTheGuildsman (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(216, "Q00216_TrialOfTheGuildsman", "Trial of the Guildsman")

-- NPCs
local VALKON = 30103
local NORMAN = 30210
local ALTRAN = 30283
local PINTER = 30298
local DUNING = 30688
-- Monsters
local ANT = 20079
local ANT_CAPTAIN = 20080
local GRANITE_GOLEM = 20083
local MANDRAGORA_SPROUT = 20154
local MANDRAGORA_SAPLING = 20155
local MANDRAGORA_BLOSSOM = 20156
local SILENOS = 20168
local STRAIN = 20200
local GHOUL = 20201
local DEAD_SEEKER = 20202
local BREKA_ORC = 20267
local BREKA_ORC_ARCHER = 20268
local BREKA_ORC_SHAMAN = 20269
local BREKA_ORC_OVERLORD = 20270
local BREKA_ORC_WARRIOR = 20271
-- Items
local RECIPE_JOURNEYMAN_RING = 3024
local RECIPE_AMBER_BEAD = 3025
local VALKON_RECOMMENDATION = 3120
local MANDRAGORA_BERRY = 3121
local ALTRAN_INSTRUCTIONS = 3122
local ALTRAN_RECOMMENDATION_1 = 3123
local ALTRAN_RECOMMENDATION_2 = 3124
local NORMAN_INSTRUCTIONS = 3125
local NORMAN_RECEIPT = 3126
local DUNING_INSTRUCTIONS = 3127
local DUNING_KEY = 3128
local NORMAN_LIST = 3129
local GRAY_BONE_POWDER = 3130
local GRANITE_WHETSTONE = 3131
local RED_PIGMENT = 3132
local BRAIDED_YARN = 3133
local JOURNEYMAN_GEM = 3134
local PINTER_INSTRUCTIONS = 3135
local AMBER_BEAD = 3136
local AMBER_LUMP = 3137
local JOURNEYMAN_DECO_BEADS = 3138
local JOURNEYMAN_RING = 3139
-- Rewards
local MARK_OF_GUILDSMAN = 3119
local DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(RECIPE_JOURNEYMAN_RING, RECIPE_AMBER_BEAD, VALKON_RECOMMENDATION, MANDRAGORA_BERRY, ALTRAN_INSTRUCTIONS, ALTRAN_RECOMMENDATION_1, ALTRAN_RECOMMENDATION_2, NORMAN_INSTRUCTIONS, NORMAN_RECEIPT, DUNING_INSTRUCTIONS, DUNING_KEY, NORMAN_LIST, GRAY_BONE_POWDER, GRANITE_WHETSTONE, RED_PIGMENT, BRAIDED_YARN, JOURNEYMAN_GEM, PINTER_INSTRUCTIONS, AMBER_BEAD, AMBER_LUMP, JOURNEYMAN_DECO_BEADS, JOURNEYMAN_RING)
q:add_start_npc(VALKON)
q:add_talk_id(VALKON, NORMAN, ALTRAN, PINTER, DUNING)
q:add_kill_id(ANT, ANT_CAPTAIN, GRANITE_GOLEM, MANDRAGORA_SPROUT, MANDRAGORA_SAPLING, MANDRAGORA_BLOSSOM, SILENOS, STRAIN, GHOUL, DEAD_SEEKER, BREKA_ORC, BREKA_ORC_ARCHER, BREKA_ORC_SHAMAN, BREKA_ORC_OVERLORD, BREKA_ORC_WARRIOR)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30103-06.htm" then
    if q:get_quest_items_count(player, 57) >= 2000 then
      st:start_quest()
      q:take_items(player, 57, 2000)
      q:give_items(player, VALKON_RECOMMENDATION, 1)
      if not player:var_bool("secondClassChange35", false) then
        htmltext = "30103-06d.htm"
        q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_35[player:class_id()])
        player:set_var("secondClassChange35", true)
      end
    else
      htmltext = "30103-05a.htm"
    end
  elseif event == "30103-06c.htm" or event == "30103-07c.htm" then
    if st:get_cond() < 3 then
      st:set_cond(3, true)
    end
  elseif event == "30103-09a.htm" or event == "30103-09b.htm" then
    q:take_items(player, ALTRAN_INSTRUCTIONS, 1)
    q:take_items(player, JOURNEYMAN_RING, -1)
    q:give_items(player, MARK_OF_GUILDSMAN, 1)
    q:add_exp_and_sp(player, 80993, 12250)
    player:social_action(3)
    st:exit_quest(false, true)
  elseif event == "30210-04.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, ALTRAN_RECOMMENDATION_1, 1)
    q:give_items(player, NORMAN_INSTRUCTIONS, 1)
    q:give_items(player, NORMAN_RECEIPT, 1)
  elseif event == "30210-10.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, NORMAN_LIST, 1)
  elseif event == "30283-03.htm" then
    st:set_cond(5, true)
    q:take_items(player, MANDRAGORA_BERRY, 1)
    q:take_items(player, VALKON_RECOMMENDATION, 1)
    q:give_items(player, ALTRAN_INSTRUCTIONS, 1)
    q:give_items(player, ALTRAN_RECOMMENDATION_1, 1)
    q:give_items(player, ALTRAN_RECOMMENDATION_2, 1)
    q:give_items(player, RECIPE_JOURNEYMAN_RING, 1)
  elseif event == "30298-04.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, ALTRAN_RECOMMENDATION_2, 1)
    q:give_items(player, PINTER_INSTRUCTIONS, 1)
    -- Artisan receives a recipe to craft Amber Beads, while spoiler case is handled in onKill section.
    if player:class_id() == ClassId.ARTISAN then
      htmltext = "30298-05.htm"
      q:give_items(player, RECIPE_AMBER_BEAD, 1)
    end
  elseif event == "30688-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, NORMAN_RECEIPT, 1)
    q:give_items(player, DUNING_INSTRUCTIONS, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.SCAVENGER and player:class_id() ~= ClassId.ARTISAN then
      htmltext = "30103-01.htm"
    elseif player:level() < 35 then
      htmltext = "30103-02.htm"
    else
      htmltext = "30103-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == VALKON then
      if cond == 1 then
        htmltext = "30103-06c.htm"
      elseif cond < 5 then
        htmltext = "30103-07.htm"
      elseif cond == 5 then
        htmltext = "30103-08.htm"
      elseif cond == 6 then
        htmltext = q:get_quest_items_count(player, JOURNEYMAN_RING) == 7 and "30103-09.htm" or "30103-08.htm"
      end
    elseif sw2 == ALTRAN then
      if cond < 4 then
        htmltext = "30283-01.htm"
        if cond == 1 then
          st:set_cond(2, true)
        end
      elseif cond == 4 then
        htmltext = "30283-02.htm"
      elseif cond > 4 then
        htmltext = "30283-04.htm"
      end
    elseif sw2 == NORMAN then
      if cond == 5 then
        if q:has_quest_items(player, ALTRAN_RECOMMENDATION_1) then
          htmltext = "30210-01.htm"
        elseif q:has_quest_items(player, NORMAN_RECEIPT) then
          htmltext = "30210-05.htm"
        elseif q:has_quest_items(player, DUNING_INSTRUCTIONS) then
          htmltext = "30210-06.htm"
        elseif q:get_quest_items_count(player, DUNING_KEY) == 30 then
          htmltext = "30210-07.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, DUNING_KEY, -1)
        elseif q:has_quest_items(player, NORMAN_LIST) then
          if q:get_quest_items_count(player, GRAY_BONE_POWDER) == 70 and q:get_quest_items_count(player, GRANITE_WHETSTONE) == 70 and q:get_quest_items_count(player, RED_PIGMENT) == 70 and q:get_quest_items_count(player, BRAIDED_YARN) == 70 then
            htmltext = "30210-12.htm"
            q:take_items(player, NORMAN_INSTRUCTIONS, 1)
            q:take_items(player, NORMAN_LIST, 1)
            q:take_items(player, BRAIDED_YARN, -1)
            q:take_items(player, GRANITE_WHETSTONE, -1)
            q:take_items(player, GRAY_BONE_POWDER, -1)
            q:take_items(player, RED_PIGMENT, -1)
            q:give_items(player, JOURNEYMAN_GEM, 7)
            if q:get_quest_items_count(player, JOURNEYMAN_DECO_BEADS) == 7 then
              st:set_cond(6, true)
            else
              q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            end
          else
            htmltext = "30210-11.htm"
          end
        end
      end
    elseif sw2 == DUNING then
      if cond == 5 then
        if q:has_quest_items(player, NORMAN_RECEIPT) then
          htmltext = "30688-01.htm"
        elseif q:has_quest_items(player, DUNING_INSTRUCTIONS) then
          if q:get_quest_items_count(player, DUNING_KEY) < 30 then
            htmltext = "30688-03.htm"
          else
            htmltext = "30688-04.htm"
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            q:take_items(player, DUNING_INSTRUCTIONS, 1)
          end
        else
          htmltext = "30688-05.htm"
        end
      end
    elseif sw2 == PINTER then
      if cond == 5 then
        if q:has_quest_items(player, ALTRAN_RECOMMENDATION_2) then
          htmltext = player:level() < 36 and "30298-01.htm" or "30298-02.htm"
        elseif q:has_quest_items(player, PINTER_INSTRUCTIONS) then
          if q:get_quest_items_count(player, AMBER_BEAD) < 70 then
            htmltext = "30298-06.htm"
          else
            htmltext = "30298-07.htm"
            q:take_items(player, AMBER_BEAD, -1)
            q:take_items(player, PINTER_INSTRUCTIONS, 1)
            q:give_items(player, JOURNEYMAN_DECO_BEADS, 7)
            if q:get_quest_items_count(player, JOURNEYMAN_GEM) == 7 then
              st:set_cond(6, true)
            else
              q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            end
          end
        end
      elseif q:has_quest_items(player, JOURNEYMAN_DECO_BEADS) then
        htmltext = "30298-08.htm"
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
  local sw = npc:id()
  if sw == MANDRAGORA_SPROUT or sw == MANDRAGORA_SAPLING or sw == MANDRAGORA_BLOSSOM then
    if st:is_cond(3) then
      q:give_items(player, MANDRAGORA_BERRY, 1)
      st:set_cond(4, true)
    end
  elseif sw == BREKA_ORC or sw == BREKA_ORC_ARCHER or sw == BREKA_ORC_WARRIOR or sw == BREKA_ORC_OVERLORD or sw == BREKA_ORC_SHAMAN then
    if q:has_quest_items(player, DUNING_INSTRUCTIONS) and q:get_quest_items_count(player, DUNING_KEY) < 30 then
      q:give_items(player, DUNING_KEY, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
  elseif sw == GHOUL or sw == STRAIN then
    if q:has_quest_items(player, NORMAN_LIST) and q:get_quest_items_count(player, GRAY_BONE_POWDER) < 70 then
      q:give_items(player, GRAY_BONE_POWDER, 5)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
  elseif sw == GRANITE_GOLEM then
    if q:has_quest_items(player, NORMAN_LIST) and q:get_quest_items_count(player, GRANITE_WHETSTONE) < 70 then
      q:give_items(player, GRANITE_WHETSTONE, 7)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
  elseif sw == DEAD_SEEKER then
    if q:has_quest_items(player, NORMAN_LIST) and q:get_quest_items_count(player, RED_PIGMENT) < 70 then
      q:give_items(player, RED_PIGMENT, 7)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
  elseif sw == SILENOS then
    if q:has_quest_items(player, NORMAN_LIST) and q:get_quest_items_count(player, BRAIDED_YARN) < 70 then
      q:give_items(player, BRAIDED_YARN, 10)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
  elseif sw == ANT or sw == ANT_CAPTAIN then
    local count = 0
    -- L2Go: Mobius adds 5 beads for the Scavenger who spoiled the ant
    -- (npc.isSweepActive() and npc.getSpoilerObjectId()). The script API has
    -- no spoil state yet, so that bonus is left out until it does.
    if rnd.bool() and player:class_id() == ClassId.ARTISAN then
      q:give_items(player, AMBER_LUMP, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    end
    if q:get_quest_items_count(player, AMBER_BEAD) + count < 70 then
      count = count + 5
    end
    if count > 0 then
      q:give_item_randomly(player, npc, AMBER_BEAD, count, count, 70, 1, true)
    end
  end
end)

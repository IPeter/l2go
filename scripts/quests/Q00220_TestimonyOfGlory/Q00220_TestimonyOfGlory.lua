-- Translated by questport from quests/Q00220_TestimonyOfGlory (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(220, "Q00220_TestimonyOfGlory", "Testimony of Glory")
local K = {}

K.KASMAN = 30501
K.VOKIAN = 30514
K.MANAKIA = 30515
K.KAKAI = 30565
K.TANAPI = 30571
K.VOLTAR = 30615
K.KEPRA = 30616
K.BURAI = 30617
K.HARAK = 30618
K.DRIKO = 30619
K.CHIANTA = 30642
K.TYRANT = 20192
K.MARSH_STAKATO_DRONE = 20234
K.GUARDIAN_BASILISK = 20550
K.MANASHEN_GARGOYLE = 20563
K.TIMAK_ORC = 20583
K.TIMAK_ORC_ARCHER = 20584
K.TIMAK_ORC_SOLDIER = 20585
K.TIMAK_ORC_WARRIOR = 20586
K.TIMAK_ORC_SHAMAN = 20587
K.TIMAK_ORC_OVERLORD = 20588
K.TAMLIN_ORC = 20601
K.TAMLIN_ORC_ARCHER = 20602
K.RAGNA_ORC_OVERLORD = 20778
K.RAGNA_ORC_SEER = 20779
K.PASHIKA_SON_OF_VOLTAR = 27080
K.VULTUS_SON_OF_VOLTAR = 27081
K.ENKU_ORC_OVERLORD = 27082
K.MAKUM_BUGBEAR_THUG = 27083
K.REVENANT_OF_TANTOS_CHIEF = 27086
K.VOKIAN_ORDER_1 = 3204
K.MANASHEN_SHARD = 3205
K.TYRANT_TALON = 3206
K.GUARDIAN_BASILISK_FANG = 3207
K.VOKIAN_ORDER_2 = 3208
K.NECKLACE_OF_AUTHORITY = 3209
K.CHIANTA_ORDER_1 = 3210
K.SCEPTER_OF_BREKA = 3211
K.SCEPTER_OF_ENKU = 3212
K.SCEPTER_OF_VUKU = 3213
K.SCEPTER_OF_TUREK = 3214
K.SCEPTER_OF_TUNATH = 3215
K.CHIANTA_ORDER_2 = 3216
K.CHIANTA_ORDER_3 = 3217
K.TAMLIN_ORC_SKULL = 3218
K.TIMAK_ORC_HEAD = 3219
K.SCEPTER_BOX = 3220
K.PASHIKA_HEAD = 3221
K.VULTUS_HEAD = 3222
K.GLOVE_OF_VOLTAR = 3223
K.ENKU_OVERLORD_HEAD = 3224
K.GLOVE_OF_KEPRA = 3225
K.MAKUM_BUGBEAR_HEAD = 3226
K.GLOVE_OF_BURAI = 3227
K.MANAKIA_LETTER_1 = 3228
K.MANAKIA_LETTER_2 = 3229
K.KASMAN_LETTER_1 = 3230
K.KASMAN_LETTER_2 = 3231
K.KASMAN_LETTER_3 = 3232
K.DRIKO_CONTRACT = 3233
K.STAKATO_DRONE_HUSK = 3234
K.TANAPI_ORDER = 3235
K.SCEPTER_OF_TANTOS = 3236
K.RITUAL_BOX = 3237
K.MARK_OF_GLORY = 3203
K.DIMENSIONAL_DIAMOND = 7562
-- L2Go: the groups keep object IDs (J.oid), since NPC values expire when a
-- handler ends; an NPC no longer in the world is no longer spawned.
K._sonsOfVoltar = {}
K._enkuOrcOverlords = {}
K._makumBugbearThugs = {}

-- @param group the monsters spawned for a quest step
-- @return {@code true} while any of them is still alive, a new group is spawned only after the previous one is gone
local function isGroupAlive(group)
  for _, id in ipairs(group) do
    local npc = J.obj(id)
    if npc ~= nil and not npc:is_dead() then
      return true
    end
  end
  return false
end

q:register_quest_items(K.VOKIAN_ORDER_1, K.MANASHEN_SHARD, K.TYRANT_TALON, K.GUARDIAN_BASILISK_FANG, K.VOKIAN_ORDER_2, K.NECKLACE_OF_AUTHORITY, K.CHIANTA_ORDER_1, K.SCEPTER_OF_BREKA, K.SCEPTER_OF_ENKU, K.SCEPTER_OF_VUKU, K.SCEPTER_OF_TUREK, K.SCEPTER_OF_TUNATH, K.CHIANTA_ORDER_2, K.CHIANTA_ORDER_3, K.TAMLIN_ORC_SKULL, K.TIMAK_ORC_HEAD, K.SCEPTER_BOX, K.PASHIKA_HEAD, K.VULTUS_HEAD, K.GLOVE_OF_VOLTAR, K.ENKU_OVERLORD_HEAD, K.GLOVE_OF_KEPRA, K.MAKUM_BUGBEAR_HEAD, K.GLOVE_OF_BURAI, K.MANAKIA_LETTER_1, K.MANAKIA_LETTER_2, K.KASMAN_LETTER_1, K.KASMAN_LETTER_2, K.KASMAN_LETTER_3, K.DRIKO_CONTRACT, K.STAKATO_DRONE_HUSK, K.TANAPI_ORDER, K.SCEPTER_OF_TANTOS, K.RITUAL_BOX)
q:add_start_npc(K.VOKIAN)
q:add_talk_id(K.KASMAN, K.VOKIAN, K.MANAKIA, K.KAKAI, K.TANAPI, K.VOLTAR, K.KEPRA, K.BURAI, K.HARAK, K.DRIKO, K.CHIANTA)
q:add_attack_id(K.RAGNA_ORC_OVERLORD, K.RAGNA_ORC_SEER, K.REVENANT_OF_TANTOS_CHIEF)
q:add_kill_id(K.TYRANT, K.MARSH_STAKATO_DRONE, K.GUARDIAN_BASILISK, K.MANASHEN_GARGOYLE, K.TIMAK_ORC, K.TIMAK_ORC_ARCHER, K.TIMAK_ORC_SOLDIER, K.TIMAK_ORC_WARRIOR, K.TIMAK_ORC_SHAMAN, K.TIMAK_ORC_OVERLORD, K.TAMLIN_ORC, K.TAMLIN_ORC_ARCHER, K.RAGNA_ORC_OVERLORD, K.RAGNA_ORC_SEER, K.PASHIKA_SON_OF_VOLTAR, K.VULTUS_SON_OF_VOLTAR, K.ENKU_ORC_OVERLORD, K.MAKUM_BUGBEAR_THUG, K.REVENANT_OF_TANTOS_CHIEF)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30514-05.htm" then
    st:start_quest()
    q:give_items(player, K.VOKIAN_ORDER_1, 1)
    if not player:var_bool("secondClassChange37", false) then
      htmltext = "30514-05a.htm"
      q:give_items(player, K.DIMENSIONAL_DIAMOND, quest.DF_REWARD_37[player:race()])
      player:set_var("secondClassChange37", true)
    end
  elseif event == "30642-03.htm" then
    st:set_cond(4, true)
    q:take_items(player, K.VOKIAN_ORDER_2, 1)
    q:give_items(player, K.CHIANTA_ORDER_1, 1)
  elseif event == "30642-07.htm" then
    q:take_items(player, K.CHIANTA_ORDER_1, 1)
    q:take_items(player, K.KASMAN_LETTER_1, 1)
    q:take_items(player, K.MANAKIA_LETTER_1, 1)
    q:take_items(player, K.MANAKIA_LETTER_2, 1)
    q:take_items(player, K.SCEPTER_OF_BREKA, 1)
    q:take_items(player, K.SCEPTER_OF_ENKU, 1)
    q:take_items(player, K.SCEPTER_OF_TUNATH, 1)
    q:take_items(player, K.SCEPTER_OF_TUREK, 1)
    q:take_items(player, K.SCEPTER_OF_VUKU, 1)
    if player:level() >= 37 then
      st:set_cond(6, true)
      q:give_items(player, K.CHIANTA_ORDER_3, 1)
    else
      htmltext = "30642-06.htm"
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(player, K.CHIANTA_ORDER_2, 1)
    end
  elseif event == "30501-02.htm" then
    if not q:has_quest_items(player, K.SCEPTER_OF_VUKU) then
      if q:has_quest_items(player, K.KASMAN_LETTER_1) then
        htmltext = "30501-04.htm"
      else
        htmltext = "30501-03.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, K.KASMAN_LETTER_1, 1)
      end
      q:add_radar(player, -2150, 124443, -3724)
    end
  elseif event == "30501-05.htm" then
    if not q:has_quest_items(player, K.SCEPTER_OF_TUREK) then
      if q:has_quest_items(player, K.KASMAN_LETTER_2) then
        htmltext = "30501-07.htm"
      else
        htmltext = "30501-06.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, K.KASMAN_LETTER_2, 1)
      end
      q:add_radar(player, -94294, 110818, -3563)
    end
  elseif event == "30501-08.htm" then
    if not q:has_quest_items(player, K.SCEPTER_OF_TUNATH) then
      if q:has_quest_items(player, K.KASMAN_LETTER_3) then
        htmltext = "30501-10.htm"
      else
        htmltext = "30501-09.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, K.KASMAN_LETTER_3, 1)
      end
      q:add_radar(player, -55217, 200628, -3724)
    end
  elseif event == "30515-02.htm" then
    if not q:has_quest_items(player, K.SCEPTER_OF_BREKA) then
      if q:has_quest_items(player, K.MANAKIA_LETTER_1) then
        htmltext = "30515-04.htm"
      else
        htmltext = "30515-03.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, K.MANAKIA_LETTER_1, 1)
      end
      q:add_radar(player, 80100, 119991, -2264)
    end
  elseif event == "30515-05.htm" then
    if not q:has_quest_items(player, K.SCEPTER_OF_ENKU) then
      if q:has_quest_items(player, K.MANAKIA_LETTER_2) then
        htmltext = "30515-07.htm"
      else
        htmltext = "30515-06.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, K.MANAKIA_LETTER_2, 1)
      end
      q:add_radar(player, 19815, 189703, -3032)
    end
  elseif event == "30615-04.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.MANAKIA_LETTER_1, 1)
    q:give_items(player, K.GLOVE_OF_VOLTAR, 1)
    if not isGroupAlive(K._sonsOfVoltar) then
      J.clear(K._sonsOfVoltar)
      table.insert(K._sonsOfVoltar, J.oid(q:add_spawn(K.PASHIKA_SON_OF_VOLTAR, {x = 80117, y = 120039, z = -2259, heading = 0}, false, 200000)))
      table.insert(K._sonsOfVoltar, J.oid(q:add_spawn(K.VULTUS_SON_OF_VOLTAR, {x = 80058, y = 120038, z = -2259, heading = 0}, false, 200000)))
    end
  elseif event == "30616-05.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.MANAKIA_LETTER_2, 1)
    q:give_items(player, K.GLOVE_OF_KEPRA, 1)
    if not isGroupAlive(K._enkuOrcOverlords) then
      J.clear(K._enkuOrcOverlords)
      table.insert(K._enkuOrcOverlords, J.oid(q:add_spawn(K.ENKU_ORC_OVERLORD, {x = 19894, y = 189743, z = -3074, heading = 0}, false, 200000)))
      table.insert(K._enkuOrcOverlords, J.oid(q:add_spawn(K.ENKU_ORC_OVERLORD, {x = 19869, y = 189800, z = -3059, heading = 0}, false, 200000)))
      table.insert(K._enkuOrcOverlords, J.oid(q:add_spawn(K.ENKU_ORC_OVERLORD, {x = 19818, y = 189818, z = -3047, heading = 0}, false, 200000)))
      table.insert(K._enkuOrcOverlords, J.oid(q:add_spawn(K.ENKU_ORC_OVERLORD, {x = 19753, y = 189837, z = -3027, heading = 0}, false, 200000)))
    end
  elseif event == "30617-04.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.KASMAN_LETTER_2, 1)
    q:give_items(player, K.GLOVE_OF_BURAI, 1)
    if not isGroupAlive(K._makumBugbearThugs) then
      J.clear(K._makumBugbearThugs)
      table.insert(K._makumBugbearThugs, J.oid(q:add_spawn(K.MAKUM_BUGBEAR_THUG, {x = -94292, y = 110781, z = -3701, heading = 0}, false, 200000)))
      table.insert(K._makumBugbearThugs, J.oid(q:add_spawn(K.MAKUM_BUGBEAR_THUG, {x = -94293, y = 110861, z = -3701, heading = 0}, false, 200000)))
    end
  elseif event == "30618-03.htm" then
    q:take_items(player, K.KASMAN_LETTER_3, 1)
    q:give_items(player, K.SCEPTER_OF_TUNATH, 1)
    if q:has_quest_items(player, K.SCEPTER_OF_BREKA, K.SCEPTER_OF_ENKU, K.SCEPTER_OF_VUKU, K.SCEPTER_OF_TUREK) then
      st:set_cond(5, true)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30619-03.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, K.KASMAN_LETTER_1, 1)
    q:give_items(player, K.DRIKO_CONTRACT, 1)
  elseif event == "30571-03.htm" then
    st:set_cond(9, true)
    q:take_items(player, K.SCEPTER_BOX, 1)
    q:give_items(player, K.TANAPI_ORDER, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ORC then
      htmltext = "30514-01.htm"
    elseif player:level() < 37 then
      htmltext = "30514-02.htm"
    elseif ClassId.level(player:class_id()) ~= 1 then
      htmltext = "30514-01a.htm"
    else
      htmltext = "30514-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == K.VOKIAN then
      if cond == 1 then
        htmltext = "30514-06.htm"
      elseif cond == 2 then
        htmltext = "30514-08.htm"
        st:set_cond(3, true)
        q:take_items(player, K.GUARDIAN_BASILISK_FANG, 10)
        q:take_items(player, K.MANASHEN_SHARD, 10)
        q:take_items(player, K.TYRANT_TALON, 10)
        q:take_items(player, K.VOKIAN_ORDER_1, 1)
        q:give_items(player, K.NECKLACE_OF_AUTHORITY, 1)
        q:give_items(player, K.VOKIAN_ORDER_2, 1)
      elseif cond == 3 then
        htmltext = "30514-09.htm"
      elseif cond == 8 then
        htmltext = "30514-10.htm"
      end
    elseif sw2 == K.CHIANTA then
      if cond == 3 then
        htmltext = "30642-01.htm"
      elseif cond == 4 then
        htmltext = "30642-04.htm"
      elseif cond == 5 then
        if q:has_quest_items(player, K.CHIANTA_ORDER_2) then
          if player:level() >= 37 then
            htmltext = "30642-09.htm"
            st:set_cond(6, true)
            q:take_items(player, K.CHIANTA_ORDER_2, 1)
            q:give_items(player, K.CHIANTA_ORDER_3, 1)
          else
            htmltext = "30642-08.htm"
          end
        else
          htmltext = "30642-05.htm"
        end
      elseif cond == 6 then
        htmltext = "30642-10.htm"
      elseif cond == 7 then
        htmltext = "30642-11.htm"
        st:set_cond(8, true)
        q:take_items(player, K.CHIANTA_ORDER_3, 1)
        q:take_items(player, K.NECKLACE_OF_AUTHORITY, 1)
        q:take_items(player, K.TAMLIN_ORC_SKULL, 20)
        q:take_items(player, K.TIMAK_ORC_HEAD, 20)
        q:give_items(player, K.SCEPTER_BOX, 1)
      elseif cond == 8 then
        htmltext = "30642-12.htm"
      elseif cond > 8 then
        htmltext = "30642-13.htm"
      end
    elseif sw2 == K.KASMAN then
      if q:has_quest_items(player, K.CHIANTA_ORDER_1) then
        htmltext = "30501-01.htm"
      elseif cond > 4 then
        htmltext = "30501-11.htm"
      end
    elseif sw2 == K.MANAKIA then
      if q:has_quest_items(player, K.CHIANTA_ORDER_1) then
        htmltext = "30515-01.htm"
      elseif cond > 4 then
        htmltext = "30515-08.htm"
      end
    elseif sw2 == K.VOLTAR then
      if cond > 3 then
        if q:has_quest_items(player, K.MANAKIA_LETTER_1) then
          htmltext = "30615-02.htm"
          q:remove_radar(player, 80100, 119991, -2264)
        elseif q:has_quest_items(player, K.GLOVE_OF_VOLTAR) then
          htmltext = "30615-05.htm"
          if not isGroupAlive(K._sonsOfVoltar) then
            J.clear(K._sonsOfVoltar)
            table.insert(K._sonsOfVoltar, J.oid(q:add_spawn(K.PASHIKA_SON_OF_VOLTAR, {x = 80117, y = 120039, z = -2259, heading = 0}, false, 200000)))
            table.insert(K._sonsOfVoltar, J.oid(q:add_spawn(K.VULTUS_SON_OF_VOLTAR, {x = 80058, y = 120038, z = -2259, heading = 0}, false, 200000)))
          end
        elseif q:has_quest_items(player, K.PASHIKA_HEAD, K.VULTUS_HEAD) then
          htmltext = "30615-06.htm"
          q:take_items(player, K.PASHIKA_HEAD, 1)
          q:take_items(player, K.VULTUS_HEAD, 1)
          q:give_items(player, K.SCEPTER_OF_BREKA, 1)
          if q:has_quest_items(player, K.SCEPTER_OF_ENKU, K.SCEPTER_OF_VUKU, K.SCEPTER_OF_TUREK, K.SCEPTER_OF_TUNATH) then
            st:set_cond(5, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        elseif q:has_quest_items(player, K.SCEPTER_OF_BREKA) then
          htmltext = "30615-07.htm"
        elseif q:has_quest_items(player, K.CHIANTA_ORDER_1) then
          htmltext = "30615-01.htm"
        elseif cond < 9 then
          htmltext = "30615-08.htm"
        end
      end
    elseif sw2 == K.KEPRA then
      if cond > 3 then
        if q:has_quest_items(player, K.MANAKIA_LETTER_2) then
          htmltext = "30616-02.htm"
          q:remove_radar(player, 19815, 189703, -3032)
        elseif q:has_quest_items(player, K.GLOVE_OF_KEPRA) then
          htmltext = "30616-05.htm"
          if not isGroupAlive(K._enkuOrcOverlords) then
            J.clear(K._enkuOrcOverlords)
            table.insert(K._enkuOrcOverlords, J.oid(q:add_spawn(K.ENKU_ORC_OVERLORD, {x = 19894, y = 189743, z = -3074, heading = 0}, false, 200000)))
            table.insert(K._enkuOrcOverlords, J.oid(q:add_spawn(K.ENKU_ORC_OVERLORD, {x = 19869, y = 189800, z = -3059, heading = 0}, false, 200000)))
            table.insert(K._enkuOrcOverlords, J.oid(q:add_spawn(K.ENKU_ORC_OVERLORD, {x = 19818, y = 189818, z = -3047, heading = 0}, false, 200000)))
            table.insert(K._enkuOrcOverlords, J.oid(q:add_spawn(K.ENKU_ORC_OVERLORD, {x = 19753, y = 189837, z = -3027, heading = 0}, false, 200000)))
          end
        elseif q:get_quest_items_count(player, K.ENKU_OVERLORD_HEAD) >= 4 then
          htmltext = "30616-06.htm"
          q:take_items(player, K.ENKU_OVERLORD_HEAD, -1)
          q:give_items(player, K.SCEPTER_OF_ENKU, 1)
          if q:has_quest_items(player, K.SCEPTER_OF_BREKA, K.SCEPTER_OF_VUKU, K.SCEPTER_OF_TUREK, K.SCEPTER_OF_TUNATH) then
            st:set_cond(5, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        elseif q:has_quest_items(player, K.SCEPTER_OF_ENKU) then
          htmltext = "30616-07.htm"
        elseif q:has_quest_items(player, K.CHIANTA_ORDER_1) then
          htmltext = "30616-01.htm"
        elseif cond < 9 then
          htmltext = "30616-08.htm"
        end
      end
    elseif sw2 == K.BURAI then
      if cond > 3 then
        if q:has_quest_items(player, K.KASMAN_LETTER_2) then
          htmltext = "30617-02.htm"
          q:remove_radar(player, -94294, 110818, -3563)
        elseif q:has_quest_items(player, K.GLOVE_OF_BURAI) then
          htmltext = "30617-04.htm"
          if not isGroupAlive(K._makumBugbearThugs) then
            J.clear(K._makumBugbearThugs)
            table.insert(K._makumBugbearThugs, J.oid(q:add_spawn(K.MAKUM_BUGBEAR_THUG, {x = -94292, y = 110781, z = -3701, heading = 0}, false, 200000)))
            table.insert(K._makumBugbearThugs, J.oid(q:add_spawn(K.MAKUM_BUGBEAR_THUG, {x = -94293, y = 110861, z = -3701, heading = 0}, false, 200000)))
          end
        elseif q:get_quest_items_count(player, K.MAKUM_BUGBEAR_HEAD) == 2 then
          htmltext = "30617-05.htm"
          q:take_items(player, K.MAKUM_BUGBEAR_HEAD, 2)
          q:give_items(player, K.SCEPTER_OF_TUREK, 1)
          if q:has_quest_items(player, K.SCEPTER_OF_BREKA, K.SCEPTER_OF_VUKU, K.SCEPTER_OF_ENKU, K.SCEPTER_OF_TUNATH) then
            st:set_cond(5, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        elseif q:has_quest_items(player, K.SCEPTER_OF_TUREK) then
          htmltext = "30617-06.htm"
        elseif q:has_quest_items(player, K.CHIANTA_ORDER_1) then
          htmltext = "30617-01.htm"
        elseif cond < 8 then
          htmltext = "30617-07.htm"
        end
      end
    elseif sw2 == K.HARAK then
      if cond > 3 then
        if q:has_quest_items(player, K.KASMAN_LETTER_3) then
          htmltext = "30618-02.htm"
          q:remove_radar(player, -55217, 200628, -3724)
        elseif q:has_quest_items(player, K.SCEPTER_OF_TUNATH) then
          htmltext = "30618-04.htm"
        elseif q:has_quest_items(player, K.CHIANTA_ORDER_1) then
          htmltext = "30618-01.htm"
        elseif cond < 9 then
          htmltext = "30618-05.htm"
        end
      end
    elseif sw2 == K.DRIKO then
      if cond > 3 then
        if q:has_quest_items(player, K.KASMAN_LETTER_1) then
          htmltext = "30619-02.htm"
          q:remove_radar(player, -2150, 124443, -3724)
        elseif q:has_quest_items(player, K.DRIKO_CONTRACT) then
          if q:get_quest_items_count(player, K.STAKATO_DRONE_HUSK) == 30 then
            htmltext = "30619-05.htm"
            q:take_items(player, K.DRIKO_CONTRACT, 1)
            q:take_items(player, K.STAKATO_DRONE_HUSK, 30)
            q:give_items(player, K.SCEPTER_OF_VUKU, 1)
            if q:has_quest_items(player, K.SCEPTER_OF_BREKA, K.SCEPTER_OF_TUREK, K.SCEPTER_OF_ENKU, K.SCEPTER_OF_TUNATH) then
              st:set_cond(5, true)
            else
              q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            end
          else
            htmltext = "30619-04.htm"
          end
        elseif q:has_quest_items(player, K.SCEPTER_OF_VUKU) then
          htmltext = "30619-06.htm"
        elseif q:has_quest_items(player, K.CHIANTA_ORDER_1) then
          htmltext = "30619-01.htm"
        elseif cond < 8 then
          htmltext = "30619-07.htm"
        end
      end
    elseif sw2 == K.TANAPI then
      if cond == 8 then
        htmltext = "30571-01.htm"
      elseif cond == 9 then
        htmltext = "30571-04.htm"
      elseif cond == 10 then
        htmltext = "30571-05.htm"
        st:set_cond(11, true)
        q:take_items(player, K.SCEPTER_OF_TANTOS, 1)
        q:take_items(player, K.TANAPI_ORDER, 1)
        q:give_items(player, K.RITUAL_BOX, 1)
      elseif cond == 11 then
        htmltext = "30571-06.htm"
      end
    elseif sw2 == K.KAKAI then
      if cond > 7 and cond < 11 then
        htmltext = "30565-01.htm"
      elseif cond == 11 then
        htmltext = "30565-02.htm"
        q:take_items(player, K.RITUAL_BOX, 1)
        q:give_items(player, K.MARK_OF_GLORY, 1)
        q:add_exp_and_sp(player, 91457, 2500)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_attack(function(npc, attacker, damage, isPet)
  local st = q:get_quest_state(attacker, false)
  if st == nil then
    return
  end
  local sw = npc:id()
  if sw == K.RAGNA_ORC_OVERLORD or sw == K.RAGNA_ORC_SEER then
    if st:is_cond(9) and npc:is_script_value(0) then
      npc:broadcast_say(ChatType.GENERAL, "Is it a lackey of Kakai?!")
      npc:set_script_value(1)
    end
  elseif sw == K.REVENANT_OF_TANTOS_CHIEF then
    if st:is_cond(9) then
      if npc:is_script_value(0) then
        npc:broadcast_say(ChatType.GENERAL, "How regretful! Unjust dishonor!")
        npc:set_script_value(1)
      end
      -- L2Go: Mobius also shouts "Indignant and unfair death!" (script
      -- value 2) once the chief is below a third of its HP. The script API
      -- cannot read an NPC's HP yet (npc.getCurrentHp/getMaxHp), so that
      -- line is left out; nothing else depends on it.
    end
  end
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == K.TYRANT then
    if st:is_cond(1) and q:get_quest_items_count(player, K.TYRANT_TALON) < 10 and rnd(10) < 5 then
      q:give_items(player, K.TYRANT_TALON, 1)
      if q:get_quest_items_count(player, K.TYRANT_TALON) >= 10 and q:get_quest_items_count(player, K.GUARDIAN_BASILISK_FANG) + q:get_quest_items_count(player, K.MANASHEN_SHARD) == 20 then
        st:set_cond(2, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.GUARDIAN_BASILISK then
    if st:is_cond(1) and q:get_quest_items_count(player, K.GUARDIAN_BASILISK_FANG) < 10 and rnd(10) < 5 then
      q:give_items(player, K.GUARDIAN_BASILISK_FANG, 1)
      if q:get_quest_items_count(player, K.GUARDIAN_BASILISK_FANG) >= 10 and q:get_quest_items_count(player, K.TYRANT_TALON) + q:get_quest_items_count(player, K.MANASHEN_SHARD) == 20 then
        st:set_cond(2, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.MANASHEN_GARGOYLE then
    if st:is_cond(1) and q:get_quest_items_count(player, K.MANASHEN_SHARD) < 10 and rnd(100) < 75 then
      q:give_items(player, K.MANASHEN_SHARD, 1)
      if q:get_quest_items_count(player, K.MANASHEN_SHARD) >= 10 and q:get_quest_items_count(player, K.TYRANT_TALON) + q:get_quest_items_count(player, K.GUARDIAN_BASILISK_FANG) == 20 then
        st:set_cond(2, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.MARSH_STAKATO_DRONE then
    if q:has_quest_items(player, K.DRIKO_CONTRACT) and q:get_quest_items_count(player, K.STAKATO_DRONE_HUSK) < 30 and rnd(100) < 75 then
      q:give_items(player, K.STAKATO_DRONE_HUSK, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.PASHIKA_SON_OF_VOLTAR then
    if q:has_quest_items(player, K.GLOVE_OF_VOLTAR) and not q:has_quest_items(player, K.PASHIKA_HEAD) then
      q:give_items(player, K.PASHIKA_HEAD, 1)
      if q:has_quest_items(player, K.VULTUS_HEAD) then
        q:take_items(player, K.GLOVE_OF_VOLTAR, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.VULTUS_SON_OF_VOLTAR then
    if q:has_quest_items(player, K.GLOVE_OF_VOLTAR) and not q:has_quest_items(player, K.VULTUS_HEAD) then
      q:give_items(player, K.VULTUS_HEAD, 1)
      if q:has_quest_items(player, K.PASHIKA_HEAD) then
        q:take_items(player, K.GLOVE_OF_VOLTAR, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.ENKU_ORC_OVERLORD then
    if q:has_quest_items(player, K.GLOVE_OF_KEPRA) then
      q:give_items(player, K.ENKU_OVERLORD_HEAD, 1)
      if q:get_quest_items_count(player, K.ENKU_OVERLORD_HEAD) >= 4 then
        q:take_items(player, K.GLOVE_OF_KEPRA, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.MAKUM_BUGBEAR_THUG then
    if q:has_quest_items(player, K.GLOVE_OF_BURAI) then
      q:give_items(player, K.MAKUM_BUGBEAR_HEAD, 1)
      if q:get_quest_items_count(player, K.MAKUM_BUGBEAR_HEAD) >= 2 then
        q:take_items(player, K.GLOVE_OF_BURAI, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.TIMAK_ORC or sw == K.TIMAK_ORC_ARCHER or sw == K.TIMAK_ORC_SOLDIER or sw == K.TIMAK_ORC_WARRIOR or sw == K.TIMAK_ORC_SHAMAN or sw == K.TIMAK_ORC_OVERLORD then
    if st:is_cond(6) and q:get_quest_items_count(player, K.TIMAK_ORC_HEAD) < 20 and rnd(1000000) < 500000 + (npc:id() - 20583) * 100000 then
      q:give_items(player, K.TIMAK_ORC_HEAD, 1)
      if q:get_quest_items_count(player, K.TIMAK_ORC_HEAD) >= 20 and q:get_quest_items_count(player, K.TAMLIN_ORC_SKULL) >= 20 then
        st:set_cond(7, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.TAMLIN_ORC then
    if st:is_cond(6) and q:get_quest_items_count(player, K.TAMLIN_ORC_SKULL) < 20 and rnd(1000000) < 500000 then
      q:give_items(player, K.TAMLIN_ORC_SKULL, 1)
      if q:get_quest_items_count(player, K.TAMLIN_ORC_SKULL) >= 20 and q:get_quest_items_count(player, K.TIMAK_ORC_HEAD) >= 20 then
        st:set_cond(7, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.TAMLIN_ORC_ARCHER then
    if st:is_cond(6) and q:get_quest_items_count(player, K.TAMLIN_ORC_SKULL) < 20 and rnd(1000000) < 600000 then
      q:give_items(player, K.TAMLIN_ORC_SKULL, 1)
      if q:get_quest_items_count(player, K.TAMLIN_ORC_SKULL) >= 20 and q:get_quest_items_count(player, K.TIMAK_ORC_HEAD) >= 20 then
        st:set_cond(7, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.RAGNA_ORC_OVERLORD or sw == K.RAGNA_ORC_SEER then
    if st:is_cond(9) then
      npc:broadcast_say(ChatType.GENERAL, "Too late!")
      q:add_spawn(K.REVENANT_OF_TANTOS_CHIEF, npc, true, 200000)
    end
  elseif sw == K.REVENANT_OF_TANTOS_CHIEF then
    if st:is_cond(9) then
      st:set_cond(10, true)
      q:give_items(player, K.SCEPTER_OF_TANTOS, 1)
      npc:broadcast_say(ChatType.GENERAL, "I'll get revenge someday!!")
    end
  end
end)

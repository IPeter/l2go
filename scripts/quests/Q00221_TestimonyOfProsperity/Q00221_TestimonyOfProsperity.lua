-- Translated by questport from quests/Q00221_TestimonyOfProsperity (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(221, "Q00221_TestimonyOfProsperity", "Testimony of Prosperity")
local K = {}

K.WILFORD = 30005
K.PARMAN = 30104
K.LILITH = 30368
K.BRIGHT = 30466
K.SHARI = 30517
K.MION = 30519
K.LOCKIRIN = 30531
K.SPIRON = 30532
K.BALANKI = 30533
K.KEEF = 30534
K.FILAUR = 30535
K.ARIN = 30536
K.MARYSE_REDBONNET = 30553
K.BOLTER = 30554
K.TOROCCO = 30555
K.TOMA = 30556
K.PIOTUR = 30597
K.EMILY = 30620
K.NIKOLA = 30621
K.BOX_OF_TITAN = 30622
K.MANDRAGORA_SPROUT_1 = 20223
K.MANDRAGORA_SPROUT_2 = 20154
K.MANDRAGORA_SAPLING = 20155
K.MANDRAGORA_BLOSSOM = 20156
K.MARSH_STAKATO = 20157
K.GIANT_CRIMSON_ANT = 20228
K.MARSH_STAKATO_WORKER = 20230
K.TOAD_LORD = 20231
K.MARSH_STAKATO_SOLDIER = 20232
K.MARSH_SPIDER = 20233
K.MARSH_STAKATO_DRONE = 20234
K.ADENA = 57
K.ANIMAL_SKIN = 1867
K.RECIPE_TITAN_KEY = 3023
K.KEY_OF_TITAN = 3030
K.RING_OF_TESTIMONY_1 = 3239
K.RING_OF_TESTIMONY_2 = 3240
K.OLD_ACCOUNT_BOOK = 3241
K.BLESSED_SEED = 3242
K.EMILY_RECIPE = 3243
K.LILITH_ELVEN_WAFER = 3244
K.MAPHR_TABLET_FRAGMENT = 3245
K.COLLECTION_LICENSE = 3246
K.LOCKIRIN_NOTICE_1 = 3247
K.LOCKIRIN_NOTICE_2 = 3248
K.LOCKIRIN_NOTICE_3 = 3249
K.LOCKIRIN_NOTICE_4 = 3250
K.LOCKIRIN_NOTICE_5 = 3251
K.CONTRIBUTION_OF_SHARI = 3252
K.CONTRIBUTION_OF_MION = 3253
K.CONTRIBUTION_OF_MARYSE = 3254
K.MARYSE_REQUEST = 3255
K.CONTRIBUTION_OF_TOMA = 3256
K.RECEIPT_OF_BOLTER = 3257
K.RECEIPT_OF_CONTRIBUTION_1 = 3258
K.RECEIPT_OF_CONTRIBUTION_2 = 3259
K.RECEIPT_OF_CONTRIBUTION_3 = 3260
K.RECEIPT_OF_CONTRIBUTION_4 = 3261
K.RECEIPT_OF_CONTRIBUTION_5 = 3262
K.PROCURATION_OF_TOROCCO = 3263
K.BRIGHT_LIST = 3264
K.MANDRAGORA_PETAL = 3265
K.CRIMSON_MOSS = 3266
K.MANDRAGORA_BOUQUET = 3267
K.PARMAN_INSTRUCTIONS = 3268
K.PARMAN_LETTER = 3269
K.CLAY_DOUGH = 3270
K.PATTERN_OF_KEYHOLE = 3271
K.NIKOLAS_LIST = 3272
K.STAKATO_SHELL = 3273
K.TOAD_LORD_SAC = 3274
K.SPIDER_THORN = 3275
K.CRYSTAL_BROOCH = 3428
K.MARK_OF_PROSPERITY = 3238
K.DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(K.RING_OF_TESTIMONY_1, K.RING_OF_TESTIMONY_2, K.OLD_ACCOUNT_BOOK, K.BLESSED_SEED, K.EMILY_RECIPE, K.LILITH_ELVEN_WAFER, K.MAPHR_TABLET_FRAGMENT, K.COLLECTION_LICENSE, K.LOCKIRIN_NOTICE_1, K.LOCKIRIN_NOTICE_2, K.LOCKIRIN_NOTICE_3, K.LOCKIRIN_NOTICE_4, K.LOCKIRIN_NOTICE_5, K.CONTRIBUTION_OF_SHARI, K.CONTRIBUTION_OF_MION, K.CONTRIBUTION_OF_MARYSE, K.MARYSE_REQUEST, K.CONTRIBUTION_OF_TOMA, K.RECEIPT_OF_BOLTER, K.RECEIPT_OF_CONTRIBUTION_1, K.RECEIPT_OF_CONTRIBUTION_2, K.RECEIPT_OF_CONTRIBUTION_3, K.RECEIPT_OF_CONTRIBUTION_4, K.RECEIPT_OF_CONTRIBUTION_5, K.PROCURATION_OF_TOROCCO, K.BRIGHT_LIST, K.MANDRAGORA_PETAL, K.CRIMSON_MOSS, K.MANDRAGORA_BOUQUET, K.PARMAN_INSTRUCTIONS, K.PARMAN_LETTER, K.CLAY_DOUGH, K.PATTERN_OF_KEYHOLE, K.NIKOLAS_LIST, K.STAKATO_SHELL, K.TOAD_LORD_SAC, K.SPIDER_THORN, K.CRYSTAL_BROOCH)
q:add_start_npc(K.PARMAN)
q:add_talk_id(K.WILFORD, K.PARMAN, K.LILITH, K.BRIGHT, K.SHARI, K.MION, K.LOCKIRIN, K.SPIRON, K.BALANKI, K.KEEF, K.FILAUR, K.ARIN, K.MARYSE_REDBONNET, K.BOLTER, K.TOROCCO, K.TOMA, K.PIOTUR, K.EMILY, K.NIKOLA, K.BOX_OF_TITAN)
q:add_kill_id(K.MANDRAGORA_SPROUT_1, K.MANDRAGORA_SAPLING, K.MANDRAGORA_BLOSSOM, K.MARSH_STAKATO, K.MANDRAGORA_SPROUT_2, K.GIANT_CRIMSON_ANT, K.MARSH_STAKATO_WORKER, K.TOAD_LORD, K.MARSH_STAKATO_SOLDIER, K.MARSH_SPIDER, K.MARSH_STAKATO_DRONE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30104-04.htm" then
    st:start_quest()
    q:give_items(player, K.RING_OF_TESTIMONY_1, 1)
    if not player:var_bool("secondClassChange37", false) then
      htmltext = "30104-04e.htm"
      q:give_items(player, K.DIMENSIONAL_DIAMOND, quest.DF_REWARD_37[player:race()])
      player:set_var("secondClassChange37", true)
    end
  elseif event == "30104-07.htm" then
    q:take_items(player, K.BLESSED_SEED, 1)
    q:take_items(player, K.EMILY_RECIPE, 1)
    q:take_items(player, K.LILITH_ELVEN_WAFER, 1)
    q:take_items(player, K.OLD_ACCOUNT_BOOK, 1)
    q:take_items(player, K.RING_OF_TESTIMONY_1, 1)
    if player:level() < 38 then
      st:set_cond(3, true)
      q:give_items(player, K.PARMAN_INSTRUCTIONS, 1)
    else
      htmltext = "30104-08.htm"
      st:set_cond(4, true)
      q:give_items(player, K.PARMAN_LETTER, 1)
      q:give_items(player, K.RING_OF_TESTIMONY_2, 1)
    end
  elseif event == "30531-02.htm" then
    if q:has_quest_items(player, K.COLLECTION_LICENSE) then
      htmltext = "30531-04.htm"
    end
  elseif event == "30531-03.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.COLLECTION_LICENSE, 1)
    q:give_items(player, K.LOCKIRIN_NOTICE_1, 1)
    q:give_items(player, K.LOCKIRIN_NOTICE_2, 1)
    q:give_items(player, K.LOCKIRIN_NOTICE_3, 1)
    q:give_items(player, K.LOCKIRIN_NOTICE_4, 1)
    q:give_items(player, K.LOCKIRIN_NOTICE_5, 1)
  elseif event == "30534-03a.htm" then
    if q:get_quest_items_count(player, K.ADENA) >= 5000 then
      htmltext = "30534-03b.htm"
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:take_items(player, K.ADENA, 5000)
      q:take_items(player, K.PROCURATION_OF_TOROCCO, 1)
      q:give_items(player, K.RECEIPT_OF_CONTRIBUTION_3, 1)
    end
  elseif event == "30005-04.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.CRYSTAL_BROOCH, 1)
  elseif event == "30466-03.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.BRIGHT_LIST, 1)
  elseif event == "30555-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, K.PROCURATION_OF_TOROCCO, 1)
  elseif event == "30368-03.htm" then
    q:take_items(player, K.CRYSTAL_BROOCH, 1)
    q:give_items(player, K.LILITH_ELVEN_WAFER, 1)
    if q:has_quest_items(player, K.BLESSED_SEED, K.OLD_ACCOUNT_BOOK, K.EMILY_RECIPE) then
      st:set_cond(2, true)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30597-02.htm" then
    q:give_items(player, K.BLESSED_SEED, 1)
    if q:has_quest_items(player, K.OLD_ACCOUNT_BOOK, K.EMILY_RECIPE, K.LILITH_ELVEN_WAFER) then
      st:set_cond(2, true)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30620-03.htm" then
    q:take_items(player, K.MANDRAGORA_BOUQUET, 1)
    q:give_items(player, K.EMILY_RECIPE, 1)
    if q:has_quest_items(player, K.BLESSED_SEED, K.OLD_ACCOUNT_BOOK, K.LILITH_ELVEN_WAFER) then
      st:set_cond(2, true)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif event == "30621-04.htm" then
    st:set_cond(5, true)
    q:give_items(player, K.CLAY_DOUGH, 1)
  elseif event == "30622-02.htm" then
    st:set_cond(6, true)
    q:take_items(player, K.CLAY_DOUGH, 1)
    q:give_items(player, K.PATTERN_OF_KEYHOLE, 1)
  elseif event == "30622-04.htm" then
    st:set_cond(9, true)
    q:take_items(player, K.KEY_OF_TITAN, 1)
    q:take_items(player, K.NIKOLAS_LIST, 1)
    q:take_items(player, K.RECIPE_TITAN_KEY, 1)
    q:take_items(player, K.STAKATO_SHELL, 20)
    q:take_items(player, K.SPIDER_THORN, 10)
    q:take_items(player, K.TOAD_LORD_SAC, 10)
    q:give_items(player, K.MAPHR_TABLET_FRAGMENT, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DWARF then
      htmltext = "30104-01.htm"
    elseif player:level() < 37 then
      htmltext = "30104-02.htm"
    elseif ClassId.level(player:class_id()) ~= 1 then
      htmltext = "30104-01a.htm"
    else
      htmltext = "30104-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == K.PARMAN then
      if cond == 1 then
        htmltext = "30104-05.htm"
      elseif cond == 2 then
        htmltext = "30104-06.htm"
      elseif cond == 3 then
        if player:level() < 38 then
          htmltext = "30104-09.htm"
        else
          htmltext = "30104-10.htm"
          st:set_cond(4, true)
          q:take_items(player, K.PARMAN_INSTRUCTIONS, 1)
          q:give_items(player, K.PARMAN_LETTER, 1)
          q:give_items(player, K.RING_OF_TESTIMONY_2, 1)
        end
      elseif cond > 3 and cond < 7 then
        htmltext = "30104-11.htm"
      elseif cond == 7 or cond == 8 then
        htmltext = "30104-12.htm"
      elseif cond == 9 then
        htmltext = "30104-13.htm"
        q:take_items(player, K.MAPHR_TABLET_FRAGMENT, 1)
        q:take_items(player, K.RING_OF_TESTIMONY_2, 1)
        q:give_items(player, K.MARK_OF_PROSPERITY, 1)
        q:add_exp_and_sp(player, 12969, 1000)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == K.LOCKIRIN then
      if cond == 1 or cond == 2 then
        if q:has_quest_items(player, K.COLLECTION_LICENSE) then
          if q:has_quest_items(player, K.RECEIPT_OF_CONTRIBUTION_1, K.RECEIPT_OF_CONTRIBUTION_2, K.RECEIPT_OF_CONTRIBUTION_3, K.RECEIPT_OF_CONTRIBUTION_4, K.RECEIPT_OF_CONTRIBUTION_5) then
            htmltext = "30531-05.htm"
            q:take_items(player, K.COLLECTION_LICENSE, 1)
            q:take_items(player, K.RECEIPT_OF_CONTRIBUTION_1, 1)
            q:take_items(player, K.RECEIPT_OF_CONTRIBUTION_2, 1)
            q:take_items(player, K.RECEIPT_OF_CONTRIBUTION_3, 1)
            q:take_items(player, K.RECEIPT_OF_CONTRIBUTION_4, 1)
            q:take_items(player, K.RECEIPT_OF_CONTRIBUTION_5, 1)
            q:give_items(player, K.OLD_ACCOUNT_BOOK, 1)
            if q:has_quest_items(player, K.BLESSED_SEED, K.EMILY_RECIPE, K.LILITH_ELVEN_WAFER) then
              st:set_cond(2, true)
            else
              q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            end
          else
            htmltext = "30531-04.htm"
          end
        else
          htmltext = q:has_quest_items(player, K.OLD_ACCOUNT_BOOK) and "30531-06.htm" or "30531-01.htm"
        end
      elseif cond >= 4 then
        htmltext = "30531-07.htm"
      end
    elseif sw2 == K.SPIRON then
      if cond == 1 and q:has_quest_items(player, K.COLLECTION_LICENSE) then
        if q:has_quest_items(player, K.LOCKIRIN_NOTICE_1) then
          htmltext = "30532-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.LOCKIRIN_NOTICE_1, 1)
        elseif q:has_quest_items(player, K.CONTRIBUTION_OF_SHARI) then
          htmltext = "30532-03.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.CONTRIBUTION_OF_SHARI, 1)
          q:give_items(player, K.RECEIPT_OF_CONTRIBUTION_1, 1)
        else
          htmltext = q:has_quest_items(player, K.RECEIPT_OF_CONTRIBUTION_1) and "30532-04.htm" or "30532-02.htm"
        end
      end
    elseif sw2 == K.BALANKI then
      if cond == 1 and q:has_quest_items(player, K.COLLECTION_LICENSE) then
        if q:has_quest_items(player, K.LOCKIRIN_NOTICE_2) then
          htmltext = "30533-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.LOCKIRIN_NOTICE_2, 1)
        elseif q:has_quest_items(player, K.CONTRIBUTION_OF_MARYSE, K.CONTRIBUTION_OF_MION) then
          htmltext = "30533-03.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.CONTRIBUTION_OF_MARYSE, 1)
          q:take_items(player, K.CONTRIBUTION_OF_MION, 1)
          q:give_items(player, K.RECEIPT_OF_CONTRIBUTION_2, 1)
        else
          htmltext = q:has_quest_items(player, K.RECEIPT_OF_CONTRIBUTION_2) and "30533-04.htm" or "30533-02.htm"
        end
      end
    elseif sw2 == K.KEEF then
      if cond == 1 and q:has_quest_items(player, K.COLLECTION_LICENSE) then
        if q:has_quest_items(player, K.LOCKIRIN_NOTICE_3) then
          htmltext = "30534-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.LOCKIRIN_NOTICE_3, 1)
        elseif q:has_quest_items(player, K.PROCURATION_OF_TOROCCO) then
          htmltext = "30534-03.htm"
        else
          htmltext = q:has_quest_items(player, K.RECEIPT_OF_CONTRIBUTION_3) and "30534-04.htm" or "30534-02.htm"
        end
      end
    elseif sw2 == K.FILAUR then
      if cond == 1 and q:has_quest_items(player, K.COLLECTION_LICENSE) then
        if q:has_quest_items(player, K.LOCKIRIN_NOTICE_4) then
          htmltext = "30535-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.LOCKIRIN_NOTICE_4, 1)
        elseif q:has_quest_items(player, K.RECEIPT_OF_BOLTER) then
          htmltext = "30535-03.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.RECEIPT_OF_BOLTER, 1)
          q:give_items(player, K.RECEIPT_OF_CONTRIBUTION_4, 1)
        else
          htmltext = q:has_quest_items(player, K.RECEIPT_OF_CONTRIBUTION_4) and "30535-04.htm" or "30535-02.htm"
        end
      end
    elseif sw2 == K.ARIN then
      if cond == 1 and q:has_quest_items(player, K.COLLECTION_LICENSE) then
        if q:has_quest_items(player, K.LOCKIRIN_NOTICE_5) then
          htmltext = "30536-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.LOCKIRIN_NOTICE_5, 1)
        elseif q:has_quest_items(player, K.CONTRIBUTION_OF_TOMA) then
          htmltext = "30536-03.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, K.CONTRIBUTION_OF_TOMA, 1)
          q:give_items(player, K.RECEIPT_OF_CONTRIBUTION_5, 1)
        else
          htmltext = q:has_quest_items(player, K.RECEIPT_OF_CONTRIBUTION_5) and "30536-04.htm" or "30536-02.htm"
        end
      end
    elseif sw2 == K.SHARI then
      if cond == 1 and q:has_quest_items(player, K.COLLECTION_LICENSE) then
        if q:has_quest_items(player, K.CONTRIBUTION_OF_SHARI) then
          htmltext = "30517-02.htm"
        elseif not q:has_at_least_one_quest_item(player, K.LOCKIRIN_NOTICE_1, K.RECEIPT_OF_CONTRIBUTION_1) then
          htmltext = "30517-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:give_items(player, K.CONTRIBUTION_OF_SHARI, 1)
        end
      end
    elseif sw2 == K.MION then
      if cond == 1 and q:has_quest_items(player, K.COLLECTION_LICENSE) then
        if q:has_quest_items(player, K.CONTRIBUTION_OF_MION) then
          htmltext = "30519-02.htm"
        elseif not q:has_at_least_one_quest_item(player, K.LOCKIRIN_NOTICE_2, K.RECEIPT_OF_CONTRIBUTION_2) then
          htmltext = "30519-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:give_items(player, K.CONTRIBUTION_OF_MION, 1)
        end
      end
    elseif sw2 == K.MARYSE_REDBONNET then
      if cond == 1 and q:has_quest_items(player, K.COLLECTION_LICENSE) then
        if q:has_quest_items(player, K.MARYSE_REQUEST) then
          if q:get_quest_items_count(player, K.ANIMAL_SKIN) < 100 then
            htmltext = "30553-02.htm"
          else
            htmltext = "30553-03.htm"
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            q:take_items(player, K.ANIMAL_SKIN, 100)
            q:take_items(player, K.MARYSE_REQUEST, 1)
            q:give_items(player, K.CONTRIBUTION_OF_MARYSE, 1)
          end
        elseif q:has_quest_items(player, K.CONTRIBUTION_OF_MARYSE) then
          htmltext = "30553-04.htm"
        elseif not q:has_at_least_one_quest_item(player, K.LOCKIRIN_NOTICE_2, K.RECEIPT_OF_CONTRIBUTION_2) then
          htmltext = "30553-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:give_items(player, K.MARYSE_REQUEST, 1)
        end
      end
    elseif sw2 == K.TOROCCO then
      if cond == 1 and q:has_quest_items(player, K.COLLECTION_LICENSE) then
        if q:has_quest_items(player, K.PROCURATION_OF_TOROCCO) then
          htmltext = "30555-03.htm"
        elseif not q:has_at_least_one_quest_item(player, K.LOCKIRIN_NOTICE_3, K.RECEIPT_OF_CONTRIBUTION_3) then
          htmltext = "30555-01.htm"
        end
      end
    elseif sw2 == K.BOLTER then
      if cond == 1 and q:has_quest_items(player, K.COLLECTION_LICENSE) then
        if q:has_quest_items(player, K.RECEIPT_OF_BOLTER) then
          htmltext = "30554-02.htm"
        elseif not q:has_at_least_one_quest_item(player, K.LOCKIRIN_NOTICE_4, K.RECEIPT_OF_CONTRIBUTION_4) then
          htmltext = "30554-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:give_items(player, K.RECEIPT_OF_BOLTER, 1)
        end
      end
    elseif sw2 == K.TOMA then
      if cond == 1 and q:has_quest_items(player, K.COLLECTION_LICENSE) then
        if q:has_quest_items(player, K.CONTRIBUTION_OF_TOMA) then
          htmltext = "30556-02.htm"
        elseif not q:has_at_least_one_quest_item(player, K.LOCKIRIN_NOTICE_5, K.RECEIPT_OF_CONTRIBUTION_5) then
          htmltext = "30556-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:give_items(player, K.CONTRIBUTION_OF_TOMA, 1)
        end
      end
    elseif sw2 == K.PIOTUR then
      if cond == 1 or cond == 2 then
        htmltext = q:has_quest_items(player, K.BLESSED_SEED) and "30597-03.htm" or "30597-01.htm"
      elseif cond >= 4 then
        htmltext = "30597-04.htm"
      end
    elseif sw2 == K.WILFORD then
      if cond == 1 or cond == 2 then
        if q:has_quest_items(player, K.LILITH_ELVEN_WAFER) then
          htmltext = "30005-06.htm"
        else
          htmltext = q:has_quest_items(player, K.CRYSTAL_BROOCH) and "30005-05.htm" or "30005-01.htm"
        end
      elseif cond >= 4 then
        htmltext = "30005-07.htm"
      end
    elseif sw2 == K.LILITH then
      if cond == 1 or cond == 2 then
        if q:has_quest_items(player, K.CRYSTAL_BROOCH) then
          htmltext = "30368-01.htm"
        elseif q:has_quest_items(player, K.LILITH_ELVEN_WAFER) then
          htmltext = "30368-04.htm"
        end
      elseif cond >= 4 then
        htmltext = "30368-05.htm"
      end
    elseif sw2 == K.BRIGHT then
      if cond == 1 or cond == 2 then
        if q:has_quest_items(player, K.EMILY_RECIPE) then
          htmltext = "30466-07.htm"
        elseif q:has_quest_items(player, K.MANDRAGORA_BOUQUET) then
          htmltext = "30466-06.htm"
        elseif q:has_quest_items(player, K.BRIGHT_LIST) then
          if q:get_quest_items_count(player, K.CRIMSON_MOSS) + q:get_quest_items_count(player, K.MANDRAGORA_PETAL) < 30 then
            htmltext = "30466-04.htm"
          else
            htmltext = "30466-05.htm"
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            q:take_items(player, K.BRIGHT_LIST, 1)
            q:take_items(player, K.CRIMSON_MOSS, 10)
            q:take_items(player, K.MANDRAGORA_PETAL, 20)
            q:give_items(player, K.MANDRAGORA_BOUQUET, 1)
          end
        else
          htmltext = "30466-01.htm"
        end
      elseif cond >= 4 then
        htmltext = "30466-08.htm"
      end
    elseif sw2 == K.EMILY then
      if cond == 1 or cond == 2 then
        if q:has_quest_items(player, K.EMILY_RECIPE) then
          htmltext = "30620-04.htm"
        elseif q:has_quest_items(player, K.MANDRAGORA_BOUQUET) then
          htmltext = "30620-01.htm"
        end
      elseif cond >= 4 then
        htmltext = "30620-05.htm"
      end
    elseif sw2 == K.NIKOLA then
      if cond == 4 then
        htmltext = "30621-01.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:take_items(player, K.PARMAN_LETTER, 1)
      elseif cond == 5 then
        htmltext = "30621-05.htm"
      elseif cond == 6 then
        htmltext = "30621-06.htm"
        st:set_cond(7, true)
        q:take_items(player, K.PATTERN_OF_KEYHOLE, 1)
        q:give_items(player, K.NIKOLAS_LIST, 1)
        q:give_items(player, K.RECIPE_TITAN_KEY, 1)
      elseif cond == 7 or cond == 8 then
        htmltext = q:has_quest_items(player, K.KEY_OF_TITAN) and "30621-08.htm" or "30621-07.htm"
      elseif cond == 9 then
        htmltext = "30621-09.htm"
      end
    elseif sw2 == K.BOX_OF_TITAN then
      if cond == 5 then
        htmltext = "30622-01.htm"
      elseif cond == 8 and q:has_quest_items(player, K.KEY_OF_TITAN) then
        htmltext = "30622-03.htm"
      else
        htmltext = "30622-05.htm"
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
  if sw == K.MANDRAGORA_SPROUT_1 then
    if q:has_quest_items(player, K.BRIGHT_LIST) and q:get_quest_items_count(player, K.MANDRAGORA_PETAL) < 20 and rnd(10) < 3 then
      q:give_items(player, K.MANDRAGORA_PETAL, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.MANDRAGORA_SPROUT_2 then
    if q:has_quest_items(player, K.BRIGHT_LIST) and q:get_quest_items_count(player, K.MANDRAGORA_PETAL) < 20 and rnd(10) < 6 then
      q:give_items(player, K.MANDRAGORA_PETAL, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.MANDRAGORA_SAPLING then
    if q:has_quest_items(player, K.BRIGHT_LIST) and q:get_quest_items_count(player, K.MANDRAGORA_PETAL) < 20 and rnd(10) < 8 then
      q:give_items(player, K.MANDRAGORA_PETAL, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.MANDRAGORA_BLOSSOM then
    if q:has_quest_items(player, K.BRIGHT_LIST) and q:get_quest_items_count(player, K.MANDRAGORA_PETAL) < 20 then
      q:give_items(player, K.MANDRAGORA_PETAL, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.GIANT_CRIMSON_ANT then
    if q:has_quest_items(player, K.BRIGHT_LIST) and q:get_quest_items_count(player, K.CRIMSON_MOSS) < 10 then
      q:give_items(player, K.CRIMSON_MOSS, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == K.MARSH_STAKATO then
    if st:is_cond(7) and q:get_quest_items_count(player, K.STAKATO_SHELL) < 20 and rnd(10) < 2 then
      q:give_items(player, K.STAKATO_SHELL, 1)
      if q:get_quest_items_count(player, K.STAKATO_SHELL) >= 20 and q:get_quest_items_count(player, K.TOAD_LORD_SAC) + q:get_quest_items_count(player, K.SPIDER_THORN) == 20 then
        st:set_cond(8, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.MARSH_STAKATO_WORKER then
    if st:is_cond(7) and q:get_quest_items_count(player, K.STAKATO_SHELL) < 20 and rnd(10) < 3 then
      q:give_items(player, K.STAKATO_SHELL, 1)
      if q:get_quest_items_count(player, K.STAKATO_SHELL) >= 20 and q:get_quest_items_count(player, K.TOAD_LORD_SAC) + q:get_quest_items_count(player, K.SPIDER_THORN) == 20 then
        st:set_cond(8, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.MARSH_STAKATO_SOLDIER then
    if st:is_cond(7) and q:get_quest_items_count(player, K.STAKATO_SHELL) < 20 and rnd(10) < 5 then
      q:give_items(player, K.STAKATO_SHELL, 1)
      if q:get_quest_items_count(player, K.STAKATO_SHELL) >= 20 and q:get_quest_items_count(player, K.TOAD_LORD_SAC) + q:get_quest_items_count(player, K.SPIDER_THORN) == 20 then
        st:set_cond(8, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.MARSH_STAKATO_DRONE then
    if st:is_cond(7) and q:get_quest_items_count(player, K.STAKATO_SHELL) < 20 and rnd(10) < 6 then
      q:give_items(player, K.STAKATO_SHELL, 1)
      if q:get_quest_items_count(player, K.STAKATO_SHELL) >= 20 and q:get_quest_items_count(player, K.TOAD_LORD_SAC) + q:get_quest_items_count(player, K.SPIDER_THORN) == 20 then
        st:set_cond(8, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.TOAD_LORD then
    if st:is_cond(7) and q:get_quest_items_count(player, K.TOAD_LORD_SAC) < 10 and rnd(10) < 2 then
      q:give_items(player, K.TOAD_LORD_SAC, 1)
      if q:get_quest_items_count(player, K.TOAD_LORD_SAC) >= 10 and q:get_quest_items_count(player, K.STAKATO_SHELL) + q:get_quest_items_count(player, K.SPIDER_THORN) == 30 then
        st:set_cond(8, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == K.MARSH_SPIDER then
    if st:is_cond(7) and q:get_quest_items_count(player, K.SPIDER_THORN) < 10 and rnd(10) < 2 then
      q:give_items(player, K.SPIDER_THORN, 1)
      if q:get_quest_items_count(player, K.SPIDER_THORN) >= 10 and q:get_quest_items_count(player, K.STAKATO_SHELL) + q:get_quest_items_count(player, K.TOAD_LORD_SAC) == 30 then
        st:set_cond(8, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)

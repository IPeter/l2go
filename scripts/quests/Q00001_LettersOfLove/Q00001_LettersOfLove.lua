-- Letters of Love (Q001), ported by hand from Mobius CT_0 Interlude
-- quests/Q00001_LettersOfLove. Darin (Talking Island) sends a letter to
-- Roxxy and a receipt to Baulro; the reward is a Necklace of Knowledge.
--
-- The pages (30048-01.htm, ...) are Mobius's, served from
-- data/html/scripts/quests/Q00001_LettersOfLove/.

local q = quest.new(1, "Q00001_LettersOfLove", "Letters of Love")

-- NPCs
local DARIN = 30048
local ROXXY = 30006
local BAULRO = 30033

-- Items
local DARIN_LETTER = 687
local ROXXY_KERCHIEF = 688
local DARIN_RECEIPT = 1079
local BAULRO_POTION = 1080

-- Reward
local NECKLACE = 906

q:register_quest_items(DARIN_LETTER, ROXXY_KERCHIEF, DARIN_RECEIPT, BAULRO_POTION)
q:add_start_npc(DARIN)
q:add_talk_id(DARIN, ROXXY, BAULRO)

q:on_event(function(event, npc, player)
  local st = q:get_quest_state(player, false)
  if st == nil then
    return event
  end
  if event == "30048-06.htm" then
    st:start_quest()
    q:give_items(player, DARIN_LETTER, 1)
  end
  return event
end)

local function talk_darin(st, player, cond)
  if cond == 1 then
    return "30048-07.htm"
  elseif cond == 2 then
    st:set_cond(3, true)
    q:take_items(player, ROXXY_KERCHIEF, 1)
    q:give_items(player, DARIN_RECEIPT, 1)
    return "30048-08.htm"
  elseif cond == 3 then
    return "30048-09.htm"
  elseif cond == 4 then
    q:take_items(player, BAULRO_POTION, 1)
    q:give_items(player, NECKLACE, 1)
    st:exit_quest(false, true)
    return "30048-10.htm"
  end
end

local function talk_roxxy(st, player, cond)
  if cond == 1 then
    st:set_cond(2, true)
    q:take_items(player, DARIN_LETTER, 1)
    q:give_items(player, ROXXY_KERCHIEF, 1)
    return "30006-01.htm"
  elseif cond == 2 then
    return "30006-02.htm"
  elseif cond > 2 then
    return "30006-03.htm"
  end
end

local function talk_baulro(st, player, cond)
  if cond == 3 then
    st:set_cond(4, true)
    q:take_items(player, DARIN_RECEIPT, 1)
    q:give_items(player, BAULRO_POTION, 1)
    return "30033-01.htm"
  elseif cond == 4 then
    return "30033-02.htm"
  end
end

local talkers = { [DARIN] = talk_darin, [ROXXY] = talk_roxxy, [BAULRO] = talk_baulro }

q:on_talk(function(npc, player)
  local st = q:get_quest_state(player, true)
  local state = st:get_state()
  if state == State.CREATED then
    return player:level() < 2 and "30048-01.htm" or "30048-02.htm"
  elseif state == State.STARTED then
    local talk = talkers[npc:id()]
    if talk ~= nil then
      local page = talk(st, player, st:get_cond())
      if page ~= nil then
        return page
      end
    end
  elseif state == State.COMPLETED then
    return q:get_already_completed_msg(player)
  end
  return q:get_no_quest_msg(player)
end)

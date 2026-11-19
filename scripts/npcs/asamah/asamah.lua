-- Asamah, chief of the Elroki, ported from Mobius ai/others/Asamah.
--
-- The first talk shows 32115-01.htm to players who finished the quest
-- "Elrokian Hunter's Proof" (Q00111) and 32115-02.htm to everyone else.
-- Quests come with the quest API; until then nobody has finished it. The
-- pages' "Script Asamah <page>" links show the capture skill explanations.

local asamah = script.new("Asamah")

local ASAMAH = 32115

-- The pages the links may ask for.
local PAGES = {
  ["32115-03.htm"] = true,
  ["32115-04.htm"] = true,
}

-- finished_elrokian_hunters_proof reports whether the player finished
-- Q00111_ElrokianHuntersProof. Quest states do not exist yet.
local function finished_elrokian_hunters_proof(player)
  return false
end

asamah:on_first_talk(ASAMAH, function(npc, player)
  if finished_elrokian_hunters_proof(player) then
    return "32115-01.htm"
  end
  return "32115-02.htm"
end)

asamah:on_event(function(event, npc, player)
  if PAGES[event] then
    return event
  end
  return nil
end)

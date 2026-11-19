-- Kierre's statue, ported from Mobius ai/others/Kier.
--
-- The statue on the way of the quest "The Other Side of Truth" (Q00115)
-- shows 32022-02.html to players who never took the quest and 32022-01.html
-- to those who are on it; for players who finished it, it has nothing to
-- say (nil: the NPC's own page). Quests come with the quest API; until then
-- nobody has taken it.

local kier = script.new("Kier")

local KIER = 32022

-- quest_state returns the player's state of Q00115_TheOtherSideOfTruth:
-- nil (never taken), "started" or "completed". Quest states do not exist
-- yet.
local function quest_state(player)
  return nil
end

kier:on_first_talk(KIER, function(npc, player)
  local state = quest_state(player)
  if state == nil then
    return "32022-02.html"
  elseif state ~= "completed" then
    return "32022-01.html"
  end
  return nil
end)

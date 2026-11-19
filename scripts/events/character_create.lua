-- Character creation rules.
--
-- Runs before the game server accepts a new character. Rejecting shows the
-- player one of the client's fixed messages, picked by the reason name
-- (see events.reject in types/l2go.lua).

-- Names that contain any of these words are refused, ignoring case. The
-- defaults come from L2J Mobius (Player.ini, ForbiddenNames). Add or remove
-- words, save, and reload the scripts.
local forbidden_words = {
  "annou",
  "ammou",
  "amnou",
  "anmou",
  "anou",
  "amou",
}

events.on_character_create(function(ev)
  local name = ev.name:lower()
  for _, word in ipairs(forbidden_words) do
    if name:find(word:lower(), 1, true) then
      log.info("character name refused", "account", ev.account, "name", ev.name, "word", word)
      return events.reject("incorrect_name")
    end
  end
end)

-- Helpers of the class change village masters (scripts/village_master/),
-- shared by the ports of Mobius's ElfHuman*, Orc* and Dwarf* Change1/Change2
-- scripts. Each of those has a ClassChangeRequested method made of one
-- branch per class; transfer() is that branch, the same for every class.
--
--   local cm = require("lib.class_master")

local M = {}

-- numbered(set, npc_ids, from, to, ext) adds the pages "<npc>-<NN><ext>"
-- for every NPC and every NN from..to (two digits) to set and returns it:
-- the page names Mobius's onEvent lets through.
function M.numbered(set, npc_ids, from, to, ext)
  for _, id in ipairs(npc_ids) do
    for n = from, to do
      set[string.format("%d-%02d%s", id, n, ext)] = true
    end
  end
  return set
end

-- page(prefix, n, ext) is "<prefix>-<NN><ext>", e.g. page(30373, 23, ".htm")
-- is "30373-23.htm".
function M.page(prefix, n, ext)
  return string.format("%s-%02d%s", tostring(prefix), n, ext or ".htm")
end

-- set_class is Mobius's player.setPlayerClass(id); player.setBaseClass(id);
-- player.broadcastUserInfo(). The class change effect, system message and
-- skills come with set_class_id.
function M.set_class(player, class_id)
  player:set_class_id(class_id)
  player:set_base_class(class_id)
  player:broadcast_user_info()
end

-- take_all takes every one of the items (Mobius takeItems(player, -1, ids)).
local function take_all(q, player, items)
  if #items == 1 then
    q:take_items(player, items[1], -1)
  else
    q:take_items_ids(player, -1, unpack(items))
  end
end

-- transfer is one class's branch of Mobius's ClassChangeRequested, for a
-- player whose class may change into t.class:
--   * below min_level: page t.low with the proof items, t.low_no without;
--   * with every item of t.items: the items are taken, the class changes,
--     15 coupons are given, page t.done;
--   * otherwise page t.no_proof.
-- Pages are page(prefix, n, ext).
function M.transfer(q, player, t, min_level, coupon, prefix, ext)
  if player:level() < min_level then
    if q:has_quest_items(player, unpack(t.items)) then
      return M.page(prefix, t.low, ext)
    end
    return M.page(prefix, t.low_no, ext)
  elseif q:has_quest_items(player, unpack(t.items)) then
    take_all(q, player, t.items)
    M.set_class(player, t.class)
    q:give_items(player, coupon, 15)
    return M.page(prefix, t.done, ext)
  end
  return M.page(prefix, t.no_proof, ext)
end

-- The Shadow Item Exchange Coupons a class change gives (15 of them).
M.COUPON_D_GRADE = 8869
M.COUPON_C_GRADE = 8870

return M

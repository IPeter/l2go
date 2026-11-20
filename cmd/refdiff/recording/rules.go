package recording

import (
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/testclient"
)

// The value rules of the Milestone 5–7 packets: adena, loot, HP bars and
// damage.

// itemAdena is the item ID of adena.
const itemAdena = 57

// Placeholders of the value rules.
const (
	phCurrent = "<cur>"
	phDamage  = "<damage>"
)

// curMax pairs the current and maximum attributes of StatusUpdate.
var curMax = map[int32]int32{
	testclient.StatusCurHP: testclient.StatusMaxHP,
	testclient.StatusCurMP: testclient.StatusMaxMP,
	testclient.StatusCurCP: testclient.StatusMaxCP,
}

// userInfoCurMax pairs the current and maximum fields of UserInfo.
var userInfoCurMax = map[string]string{"HP": "MaxHP", "MP": "MaxMP", "CP": "MaxCP"}

// damageParams are the system messages that carry a random damage, with the
// index of the damage parameter: 35 "You hit for $s1 damage.", 36 and 37
// "$s1 hit you for $s2 damage.".
var damageParams = map[string]int{"35": 0, "36": 1, "37": 1}

// adenaPackets carry the player's adena in a top-level Adena field.
var adenaPackets = map[string]bool{
	"BuyList": true, "SellList": true, "WareHouseDepositList": true, "WareHouseWithdrawalList": true,
}

// inventoryPackets list the player's own items, adena included.
var inventoryPackets = map[string]bool{"ItemList": true, "InventoryUpdate": true, "WareHouseDepositList": true}

// randomValues applies the value rules to the leaves of one packet:
//   - The player's adena (Adena of the shop and warehouse windows, the count
//     of the adena item of ItemList, InventoryUpdate and the deposit window)
//     becomes relative to what the role had in its first ItemList: adena0,
//     adena0+900, adena0-30. Prices and fees still show as the differences;
//     the starting amount is the server's config.
//   - A monster's loot (a DropItem whose dropper is no scenario character)
//     shows <drop> for its item, count and stack flag: drops are rolled.
//   - Current HP, MP and CP (StatusUpdate, UserInfo) below the maximum of
//     the same packet become <cur>: after a fight or a regeneration tick
//     they depend on random damage and timing.
//   - The damage of the "You hit for $s1 damage." messages becomes
//     <damage>.
func (n *normalizer) randomValues(name string, ls []leaf) {
	if n.hasAdena {
		n.relativeAdena(name, ls)
	}
	switch name {
	case "DropItem":
		n.monsterLoot(ls)
	case "StatusUpdate":
		statusCurrent(ls)
	case "UserInfo":
		userInfoCurrent(ls)
	case "SystemMessage":
		damageMessage(ls)
	}
}

// leafIndex returns the index of the leaf at path, -1 when there is none.
func leafIndex(ls []leaf, path string) int {
	for i := range ls {
		if ls[i].path == path {
			return i
		}
	}
	return -1
}

func (n *normalizer) relativeAdena(name string, ls []leaf) {
	if adenaPackets[name] {
		if i := leafIndex(ls, "Adena"); i >= 0 {
			ls[i].val = n.adenaValue(ls[i].val)
		}
	}
	if !inventoryPackets[name] {
		return
	}
	adena := strconv.Itoa(itemAdena)
	for i := range ls {
		p := ls[i].path
		if !strings.HasPrefix(p, "Items[") || !strings.HasSuffix(p, "].ItemID") || ls[i].val != adena {
			continue
		}
		if c := leafIndex(ls, strings.TrimSuffix(p, "ItemID")+"Count"); c >= 0 {
			ls[c].val = n.adenaValue(ls[c].val)
		}
	}
}

// adenaValue renders an adena amount relative to the start.
func (n *normalizer) adenaValue(v string) string {
	a, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return v
	}
	d := a - n.startAdena
	switch {
	case d == 0:
		return "adena0"
	case d > 0:
		return "adena0+" + strconv.FormatInt(d, 10)
	default:
		return "adena0" + strconv.FormatInt(d, 10)
	}
}

func (n *normalizer) monsterLoot(ls []leaf) {
	i := leafIndex(ls, "DropperID")
	if i < 0 || ls[i].kind == "" {
		return
	}
	if _, player := n.anchors[ls[i].id]; player {
		return
	}
	for j := range ls {
		switch ls[j].path {
		case "ItemID", "Count", "Stackable":
			ls[j].val = phDrop
		}
	}
}

func statusCurrent(ls []leaf) {
	// value is the index of each attribute's Value leaf.
	value := make(map[int32]int)
	for i := range ls {
		p := ls[i].path
		if !strings.HasPrefix(p, "Attributes[") || !strings.HasSuffix(p, "].ID") {
			continue
		}
		id, err := strconv.ParseInt(ls[i].val, 10, 32)
		if err != nil {
			continue
		}
		if v := leafIndex(ls, strings.TrimSuffix(p, "ID")+"Value"); v >= 0 {
			value[int32(id)] = v
		}
	}
	for cur, maxID := range curMax {
		c, ok := value[cur]
		if !ok {
			continue
		}
		if m, ok := value[maxID]; !ok || ls[m].val != ls[c].val {
			ls[c].val = phCurrent
		}
	}
}

func userInfoCurrent(ls []leaf) {
	for cur, maxPath := range userInfoCurMax {
		c, m := leafIndex(ls, cur), leafIndex(ls, maxPath)
		if c >= 0 && m >= 0 && ls[c].val != ls[m].val {
			ls[c].val = phCurrent
		}
	}
}

func damageMessage(ls []leaf) {
	i := leafIndex(ls, "ID")
	if i < 0 {
		return
	}
	param, ok := damageParams[ls[i].val]
	if !ok {
		return
	}
	if d := leafIndex(ls, "Params["+strconv.Itoa(param)+"].Number"); d >= 0 {
		ls[d].val = phDamage
	}
}

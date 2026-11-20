package conditions

import (
	"errors"
	"fmt"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// Item is the <conditions> of an item template (Mobius ItemTemplate
// preconditions, DocumentItem): a condition and the message a player gets
// when it fails.
type Item struct {
	// Cond is the condition; nil means the item has none that can fail
	// (an empty <conditions>).
	Cond *Condition
	// Msg is the text of the msg attribute; when set, MsgID and AddName
	// are not used.
	Msg string
	// MsgID is the system message of the msgId attribute, 0 if none.
	MsgID int
	// AddName is set when the message takes the item name (addName with a
	// positive msgId).
	AddName bool
}

// Result is the outcome of an item condition check. When OK is false the
// player is told Msg as text when it is set, otherwise the system message
// MsgID (if not 0), with the item's name as its parameter when AddName is
// set (ItemTemplate.checkCondition).
type Result struct {
	OK      bool
	Msg     string
	MsgID   int
	AddName bool
}

// ParseItem parses an item's <conditions> element. Like Mobius it uses the
// first child element that is not a <value>.
func ParseItem(e *data.XMLElement) (*Item, error) {
	if e == nil {
		return nil, nil
	}
	if !strings.EqualFold(e.Name, "conditions") {
		return nil, fmt.Errorf("conditions: <%s>, want <conditions>", e.Name)
	}
	it := &Item{}
	for i := range e.Children {
		if strings.EqualFold(e.Children[i].Name, "value") {
			continue
		}
		c, err := Parse(&e.Children[i])
		if err != nil {
			return nil, err
		}
		it.Cond = &c
		break
	}
	if msg, ok := e.Attrs["msg"]; ok && it.Cond != nil {
		it.Msg = msg
	} else if id, ok := e.Attrs["msgId"]; ok && it.Cond != nil {
		n, err := decodeInt(id)
		if err != nil {
			return nil, fmt.Errorf("conditions: msgId: %w", err)
		}
		it.MsgID = n
		_, addName := e.Attrs["addName"]
		it.AddName = addName && n > 0
	}
	return it, nil
}

// Check tests the condition for s. A nil Item, or one without a condition,
// passes.
func (it *Item) Check(s Subject) Result {
	if it == nil || it.Cond == nil || it.Cond.Test(s) {
		return Result{OK: true}
	}
	return Result{Msg: it.Msg, MsgID: it.MsgID, AddName: it.AddName}
}

// Table holds the parsed conditions of item templates by item ID. Items
// without <conditions> are not in it. It is read-only after NewTable.
type Table map[int]*Item

// NewTable parses the conditions of every item. It reports every item whose
// conditions it cannot parse and leaves those out.
func NewTable(items []data.Item) (Table, error) {
	t := make(Table)
	var errs []error
	for i := range items {
		if items[i].Conditions == nil {
			continue
		}
		c, err := ParseItem(items[i].Conditions)
		if err != nil {
			errs = append(errs, fmt.Errorf("item %d: %w", items[i].ID, err))
			continue
		}
		t[items[i].ID] = c
	}
	return t, errors.Join(errs...)
}

// Check tests the conditions of the item template itemID for s (Mobius
// ItemTemplate.checkCondition, without its GM and Olympiad checks, which
// are the caller's). An item without conditions passes.
func (t Table) Check(itemID int, s Subject) Result {
	return t[itemID].Check(s)
}

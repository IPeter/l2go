package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// The client packets of a direct trade between two players.

// Opcodes of the trade packets (Mobius ClientPackets).
const (
	OpTradeRequest       = 0x15
	OpAddTradeItem       = 0x16
	OpTradeDone          = 0x17
	OpAnswerTradeRequest = 0x44
)

// TradeRequest asks the target player to trade. Mobius answers the target
// with SendTradeRequest.
type TradeRequest struct {
	ObjectID int32 // the player asked
}

// ParseTradeRequest parses a TradeRequest body (Mobius
// TradeRequest.readImpl): d object ID.
func ParseTradeRequest(data []byte) (TradeRequest, error) {
	r := packets.NewReader(data)
	p := TradeRequest{ObjectID: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return TradeRequest{}, fmt.Errorf("TradeRequest: %w", err)
	}
	return p, nil
}

// AddTradeItem puts an inventory item into the open trade.
type AddTradeItem struct {
	// TradeID is a word the client sends; Mobius only logs it.
	TradeID  int32
	ObjectID int32
	// Count is not checked here: Mobius ignores a count below 1 (with a
	// warning in its log).
	Count int32
}

// ParseAddTradeItem parses an AddTradeItem body (Mobius
// AddTradeItem.readImpl): d trade ID, d object ID, d count.
func ParseAddTradeItem(data []byte) (AddTradeItem, error) {
	r := packets.NewReader(data)
	p := AddTradeItem{TradeID: r.ReadInt32(), ObjectID: r.ReadInt32(), Count: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return AddTradeItem{}, fmt.Errorf("AddTradeItem: %w", err)
	}
	return p, nil
}

// TradeDone is the OK or Cancel button of the trade window.
type TradeDone struct {
	// Confirm is set when the int the client sends is exactly 1; Mobius
	// cancels the trade on any other value.
	Confirm bool
}

// ParseTradeDone parses a TradeDone body (Mobius TradeDone.readImpl): d
// response.
func ParseTradeDone(data []byte) (TradeDone, error) {
	r := packets.NewReader(data)
	p := TradeDone{Confirm: r.ReadInt32() == 1}
	if err := r.Err(); err != nil {
		return TradeDone{}, fmt.Errorf("TradeDone: %w", err)
	}
	return p, nil
}

// AnswerTradeRequest is the answer to SendTradeRequest.
type AnswerTradeRequest struct {
	// Accept is set when the int the client sends is exactly 1; Mobius
	// takes any other value as a refusal.
	Accept bool
}

// ParseAnswerTradeRequest parses an AnswerTradeRequest body (Mobius
// AnswerTradeRequest.readImpl): d response.
func ParseAnswerTradeRequest(data []byte) (AnswerTradeRequest, error) {
	r := packets.NewReader(data)
	p := AnswerTradeRequest{Accept: r.ReadInt32() == 1}
	if err := r.Err(); err != nil {
		return AnswerTradeRequest{}, fmt.Errorf("AnswerTradeRequest: %w", err)
	}
	return p, nil
}

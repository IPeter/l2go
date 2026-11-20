package game

import (
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/world"
)

// Buylist stock (decision I8). A product with a limited stock starts full.
// The first sale starts its restock timer, which later sales do not move
// (Mobius BuyListTaskManager.add is putIfAbsent); when it runs out the
// stock is full again. The stock lives on the loop and is not saved.

// stockKey names a product of a buylist.
type stockKey struct{ list, item int }

// productStock is what is left of a limited product since its first sale,
// and the timer that fills it again.
type productStock struct {
	// left may go below 0 when one request lists the product twice (Mobius
	// lets the count go negative and shows 0).
	left  int64
	timer world.TimerID
}

// stockLeft returns how many of the limited product pr of list listID are
// left (Mobius Product.getCount).
func (s *Server) stockLeft(listID int, pr *data.BuyListProduct) int64 {
	if st := s.shop.stock[stockKey{listID, pr.ItemID}]; st != nil {
		return max(st.left, 0)
	}
	return int64(pr.Count)
}

// takeStock takes count of the limited product pr of list listID (Mobius
// Product.decreaseCount) and reports whether there were that many. The
// first sale since the last restock starts the restock timer.
func (s *Server) takeStock(listID int, pr *data.BuyListProduct, count int64) bool {
	key := stockKey{listID, pr.ItemID}
	st := s.shop.stock[key]
	if st == nil {
		st = &productStock{left: int64(pr.Count)}
		s.shop.stock[key] = st
	}
	if st.timer == 0 {
		delay := time.Duration(pr.RestockDelay) * time.Minute
		st.timer = s.loop.After(delay, func() { s.restock(key, st) })
	}
	st.left -= count
	return st.left >= 0
}

// restock fills a product again (Mobius Product.restock).
func (s *Server) restock(key stockKey, st *productStock) {
	if s.shop.stock[key] == st {
		delete(s.shop.stock, key)
	}
}

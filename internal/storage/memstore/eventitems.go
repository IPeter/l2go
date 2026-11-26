package memstore

import (
	"context"
	"slices"
)

// DeleteItemsByItemID deletes every stored item of these templates,
// whoever owns them (the mongostore method of the same name). It returns
// how many were deleted.
func (s *Characters) DeleteItemsByItemID(ctx context.Context, itemIDs []int32) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.takeFault(); err != nil {
		return 0, err
	}
	var doomed []int32
	for id, e := range s.items {
		if slices.Contains(itemIDs, e.item.ItemID) {
			doomed = append(doomed, id)
		}
	}
	for _, id := range doomed {
		s.deleteItem(id)
	}
	return int64(len(doomed)), nil
}

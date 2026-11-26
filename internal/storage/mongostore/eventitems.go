package mongostore

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// DeleteItemsByItemID deletes every stored item of these templates,
// whoever owns them and wherever they are kept (Mobius LongTimeEvent's
// "DELETE FROM items WHERE item_id=?" for the items of an event that
// ended). It returns how many were deleted.
func (c *Characters) DeleteItemsByItemID(ctx context.Context, itemIDs []int32) (int64, error) {
	if len(itemIDs) == 0 {
		return 0, nil
	}
	res, err := c.items.DeleteMany(ctx, bson.D{{Key: "item_id", Value: bson.D{{Key: "$in", Value: itemIDs}}}})
	if err != nil {
		return 0, fmt.Errorf("mongostore: delete items %v: %w", itemIDs, err)
	}
	return res.DeletedCount, nil
}

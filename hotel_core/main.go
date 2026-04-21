package main

import (
	nib_facility "github.com/tiket/TIX-HOTEL-INVENTORY-SCRIPT/hotel_core/nib_facility"
	room_grouping_sync_queue_non_unique "github.com/tiket/TIX-HOTEL-INVENTORY-SCRIPT/hotel_core/room-grouping-sync-queue-non-unique"
)

func main() {
	// bulk_merge_vendor.FindSlowHotelMerge()
	// master_query.FindRoomGroupingKey()
	// room_grouping_sync_queue_non_unique.RunScript()
	_ = room_grouping_sync_queue_non_unique.RunScript
	nib_facility.RunScript()
}

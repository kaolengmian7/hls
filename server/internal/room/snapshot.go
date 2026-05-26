package room

import "hls.dev/server/internal/protocol"

// buildSnapshot 构造该房间在当前时刻对外可见的完整快照。
//
// 注意：返回的 Snapshot 内的切片是新分配的拷贝，避免外部修改污染 Room 内部状态。
func (r *Room) buildSnapshot() protocol.RoomSnapshot {
	players := make([]protocol.Player, 0, len(r.playerOrder))
	for _, pid := range r.playerOrder {
		if p, ok := r.players[pid]; ok {
			players = append(players, *p)
		}
	}
	records := make([]protocol.LotteryRecord, len(r.lotteryRecords))
	copy(records, r.lotteryRecords)
	settlements := make([]protocol.SettlementRecord, len(r.settlementRecords))
	copy(settlements, r.settlementRecords)
	feed := make([]protocol.FeedEvent, len(r.feed))
	copy(feed, r.feed)

	return protocol.RoomSnapshot{
		Code:              r.code,
		Status:            r.status,
		HostID:            r.hostID,
		Players:           players,
		GuaranteedAmount:  r.guaranteedAmount,
		LotteryRecords:    records,
		SettlementRecords: settlements,
		Feed:              feed,
	}
}

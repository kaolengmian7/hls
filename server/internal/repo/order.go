package repo

import (
	"database/sql"
	"math"
	"time"
)

// Order mirrors a row in the orders table.
type Order struct {
	ID              int64          `db:"id"               json:"id"`
	Code            string         `db:"code"             json:"code"`
	Title           string         `db:"title"            json:"title"`
	OrderType       string         `db:"order_type"       json:"orderType"`
	Price           float64        `db:"price"            json:"price"`
	CommissionRate  float64        `db:"commission_rate"  json:"commissionRate"`
	BonusRate       float64        `db:"bonus_rate"       json:"bonusRate"`
	PlayerCount     int            `db:"player_count"     json:"playerCount"`
	DispatcherID    int64          `db:"dispatcher_id"    json:"dispatcherId"`
	DispatcherName  string         `db:"dispatcher_name"  json:"dispatcherName"`
	EnforcerAID     sql.NullInt64  `db:"enforcer_a_id"    json:"-"`
	EnforcerAName   sql.NullString `db:"enforcer_a_name"  json:"-"`
	EnforcerBID     sql.NullInt64  `db:"enforcer_b_id"    json:"-"`
	EnforcerBName   sql.NullString `db:"enforcer_b_name"  json:"-"`
	Remark          sql.NullString `db:"remark"           json:"-"`
	Status          string         `db:"status"           json:"status"`
	ComplaintStatus string         `db:"complaint_status" json:"complaintStatus"`
	CreatedAt       time.Time      `db:"created_at"       json:"createdAt"`
	UpdatedAt       time.Time      `db:"updated_at"       json:"updatedAt"`
}

// Earnings returns per-player earnings, rounded to two decimals.
// Formula: (price*(1-commission%/100) + price*bonus%/100) / playerCount.
func (o *Order) Earnings() float64 {
	if o.PlayerCount <= 0 {
		return 0
	}
	net := o.Price * (1 - o.CommissionRate/100)
	bonus := o.Price * o.BonusRate / 100
	per := (net + bonus) / float64(o.PlayerCount)
	return math.Round(per*100) / 100
}

package ledger

import "time"

type HolidayEntity struct {
	NID         int       `json:"nid"`
	Tahun       int       `json:"tahun"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
}

type ParameterEntity struct {
	NID               int       `json:"nid"`
	Update            time.Time `json:"update"`
	Description       string    `json:"description"`
	FlatFee           float64   `json:"flat_fee"`
	LendingFee        float64   `json:"lending_fee"`
	BorrowingFee      float64   `json:"borrowing_fee"`
	MaxQuantity       float64   `json:"max_quantity"` // Max
	BorrowMaxOpenDay  int       `json:"borrow_max_open_day"`
	DenominationLimit int       `json:"denomination_limit"` // Min 100
	LastUpdate        time.Time `json:"last_update"`
}

type SessionTimeEntity struct {
	NID           int       `json:"nid"`
	Description   string    `json:"description"`
	Update        time.Time `json:"update"`
	Session1Start time.Time `json:"session1_start"`
	Session1End   time.Time `json:"session1_end"`
	Session2Start time.Time `json:"session2_start"`
	Session2End   time.Time `json:"session2_end"`
	LastUpdate    time.Time `json:"last_update"`
}

type InstrumentEntity struct {
	NID        int       `json:"nid"`
	Code       string    `json:"code"` // KPEI-012345
	Name       string    `json:"name"` // stok Name
	Type       string    `json:"type"`
	Status     bool      `json:"status"` // Eligible
	LastUpdate time.Time `json:"last_update"`
}

type ParticipantEntity struct {
	NID             int       `json:"nid"`
	Code            string    `json:"code"` // YU
	Name            string    `json:"name"`
	BorrEligibility bool      `json:"borr_eligibility"`
	LendEligibility bool      `json:"lend_eligibility"`
	LastUpdate      time.Time `json:"last_update"`
}

type AccountEntity struct {
	NID             int       `json:"nid"`
	Code            string    `json:"code"` // "YU-012345"-01/02/04/05
	SID             string    `json:"sid"`
	Name            string    `json:"name"`
	ParticipantNID  int       `json:"participant_nid"`
	ParticipantCode string    `json:"participant_code"`
	TradeLimit      float64   `json:"trade_limit"`
	PoolLimit       float64   `json:"pool_limit"`
	LastUpdate      time.Time `json:"last_update"`
}

type OrderEntity struct {
	NID               int       `json:"nid"`
	PrevNID           int       `json:"prev_nid"`
	ReffRequestID     string    `json:"reff_request_id"`
	AccountNID        int       `json:"account_nid"`
	AccountCode       string    `json:"account_code"`
	ParticipantNID    int       `json:"participant_nid"`
	ParticipantCode   string    `json:"participant_code"`
	InstrumentNID     int       `json:"instrument_nid"`
	InstrumentCode    string    `json:"instrument_code"`
	Side              string    `json:"side"`
	Quantity          float64   `json:"quantity"`
	DoneQuantity      float64   `json:"done_quantity"`
	SettlementDate    time.Time `json:"settlement_date"`
	ReimbursementDate time.Time `json:"reimbursement_date"`
	Periode           int       `json:"periode"`
	State             string    `json:"state"`
	MarketPrice       float64   `json:"market_price"`
	Rate              float64   `json:"rate"`
	Instruction       string    `json:"instruction"`
	ARO               bool      `json:"aro"`
	WReffRequestID    string    `json:"w_reff_request_id"`
	Message           string    `json:"message"`
	EntryAt           time.Time `json:"entry_at"`
	PendingAt         time.Time `json:"pending_at"`
	OpenAt            time.Time `json:"open_at"`
	RejectAt          time.Time `json:"reject_at"`
	AmmendAt          time.Time `json:"ammend_at"`
	WithdrawAt        time.Time `json:"withdraw_at"`
}

type TradeEntity struct {
	NID            int       `json:"nid"`
	KpeiReff       string    `json:"kpei_reff"`
	InstrumentNID  int       `json:"instrument_nid"`
	InstrumentCode string    `json:"instrument_code"`
	Quantity       float64   `json:"quantity"`
	Periode        int       `json:"periode"`
	State          string    `json:"state"`
	FeeFlatRate    float64   `json:"fee_flat_rate"`
	FeeBorrRate    float64   `json:"fee_borr_rate"`
	FeeLendRate    float64   `json:"fee_lend_rate"`
	MatchedAt      time.Time `json:"matched_at"`
	SettledAt      time.Time `json:"settled_at"`
	ReimburseAt    time.Time `json:"reimburse_at"`
	Lender         []int     `json:"lender"`
	Borrower       []int     `json:"borrower"`
}

type ContractEntity struct {
	NID                    int       `json:"nid"`
	TradeNID               int       `json:"trade_nid"`
	KpeiReff               string    `json:"kpei_reff"`
	Side                   string    `json:"side"`
	AccountNID             int       `json:"account_nid"`
	AccountCode            string    `json:"account_code"`
	AccountSID             string    `json:"account_sid"`
	AccountParticipantNID  int       `json:"account_participant_nid"`
	AccountParticipantCode string    `json:"account_participant_code"`
	OrderNID               int       `json:"order_nid"`
	InstrumentNID          int       `json:"instrument_nid"`
	InstrumentCode         string    `json:"instrument_code"`
	Quantity               float64   `json:"quantity"`
	Periode                int       `json:"periode"`
	State                  string    `json:"state"`
	FeeFlatVal             float64   `json:"fee_flat_val"`
	FeeValDaily            float64   `json:"fee_val_daily"`
	FeeValAccumulated      float64   `json:"fee_val_accumulated"`
	MatchedAt              time.Time `json:"matched_at"`
	ReimburseAt            time.Time `json:"reimburse_at"`
}

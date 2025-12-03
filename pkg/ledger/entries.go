package ledger

import "time"

type ServiceStart struct {
	Timestamp time.Time `json:"timestamp"`
	ID        string    `json:"id"`
	StartID   string    `json:"start_id"`
	StartTime time.Time `json:"start_time"`
}

type Holiday struct {
	Timestamp   time.Time `json:"timestamp"`
	NID         int       `json:"nid"`
	Tahun       int       `json:"tahun"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
}

type Parameter struct {
	Timestamp         time.Time `json:"timestamp"`
	NID               int       `json:"nid"`
	Update            time.Time `json:"update"`
	Description       string    `json:"description"`
	FlatFee           float64   `json:"flat_fee"`
	LendingFee        float64   `json:"lending_fee"`
	BorrowingFee      float64   `json:"borrowing_fee"`
	MaxQuantity       float64   `json:"max_quantity"` // Max
	BorrowMaxOpenDay  int       `json:"borrow_max_open_day"`
	DenominationLimit int       `json:"denomination_limit"` // Min 100
}

type SessionTime struct {
	Timestamp     time.Time `json:"timestamp"`
	NID           int       `json:"nid"`
	Description   string    `json:"description"`
	Update        time.Time `json:"update"`
	Session1Start time.Time `json:"session1_start"`
	Session1End   time.Time `json:"session1_end"`
	Session2Start time.Time `json:"session2_start"`
	Session2End   time.Time `json:"session2_end"`
}

type Instrument struct {
	Timestamp time.Time `json:"timestamp"`
	NID       int       `json:"nid"`
	Code      string    `json:"code"` // KPEI-012345
	Name      string    `json:"name"` // stok Name
	Type      string    `json:"type"`
	Status    bool      `json:"status"` // Eligible
}

type Participant struct {
	Timestamp       time.Time `json:"timestamp"`
	NID             int       `json:"nid"`
	Code            string    `json:"code"` // YU
	Name            string    `json:"name"`
	BorrEligibility bool      `json:"borr_eligibility"`
	LendEligibility bool      `json:"lend_eligibility"`
}

type Account struct {
	Timestamp       time.Time `json:"timestamp"`
	NID             int       `json:"nid"`
	Code            string    `json:"code"` // "YU-012345"-01/02/04/05
	SID             string    `json:"sid"`
	Name            string    `json:"name"`
	ParticipantNID  int       `json:"participant_nid"`
	ParticipantCode string    `json:"participant_code"`
}

type AccountLimit struct {
	Timestamp  time.Time `json:"timestamp"`
	NID        int       `json:"nid"`
	Code       string    `json:"code"` // "YU-012345"-01/02/04/05
	AccountNID int       `json:"account_nid"`
	TradeLimit float64   `json:"trade_limit"`
	PoolLimit  float64   `json:"pool_limit"`
}

type Order struct {
	Timestamp         time.Time `json:"timestamp"`
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
	SettlementDate    time.Time `json:"settlement_date"`
	ReimbursementDate time.Time `json:"reimbursement_date"`
	Periode           int       `json:"periode"`
	State             string    `json:"state"`
	MarketPrice       float64   `json:"market_price"`
	Rate              float64   `json:"rate"`
	Instruction       string    `json:"instruction"`
	ARO               bool      `json:"aro"`
}

type OrderAck struct {
	Timestamp time.Time `json:"timestamp"`
	OrderNID  int       `json:"order_nid"`
}

type OrderNak struct {
	Timestamp time.Time `json:"timestamp"`
	OrderNID  int       `json:"order_nid"`
	Message   string    `json:"message"`
}

type OrderPending struct {
	Timestamp time.Time `json:"timestamp"`
	OrderNID  int       `json:"order_nid"`
}

type OrderWithdraw struct {
	Timestamp     time.Time `json:"timestamp"`
	OrderNID      int       `json:"order_nid"`
	ReffRequestID string    `json:"reff_request_id"`
}

type OrderWithdrawAck struct {
	Timestamp time.Time `json:"timestamp"`
	OrderNID  int       `json:"order_nid"`
}

type OrderWithdrawNak struct {
	Timestamp time.Time `json:"timestamp"`
	OrderNID  int       `json:"order_nid"`
	Message   string    `json:"message"`
}

type Trade struct {
	Timestamp      time.Time `json:"timestamp"`
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
	ReimburseAt    time.Time `json:"reimburse_at"`
	Lender         []Contract
	Borrower       []Contract
}

type Contract struct {
	Timestamp              time.Time `json:"timestamp"`
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

type TradeWait struct {
	Timestamp time.Time `json:"timestamp"`
	TradeNID  int       `json:"trade_nid"`
}

type TradeAck struct {
	Timestamp time.Time `json:"timestamp"`
	TradeNID  int       `json:"trade_nid"`
}

type TradeNak struct {
	Timestamp time.Time `json:"timestamp"`
	TradeNID  int       `json:"trade_nid"`
	Message   string    `json:"message"`
}

type TradeReimburse struct {
	Timestamp time.Time `json:"timestamp"`
	TradeNID  int       `json:"trade_nid"`
}

type Sod struct {
	Timestamp time.Time `json:"timestamp"`
	Date      time.Time `json:"date"`
}

type Eod struct {
	Timestamp time.Time `json:"timestamp"`
	Date      time.Time `json:"date"`
}

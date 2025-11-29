package ledger

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type LedgerPoint struct {
	// Private entity collections with individual mutexes
	participants  map[string]ParticipantEntity
	participantMu sync.RWMutex

	accounts  map[string]AccountEntity
	accountMu sync.RWMutex

	instruments  map[string]InstrumentEntity
	instrumentMu sync.RWMutex

	parameter   ParameterEntity
	parameterMu sync.RWMutex

	sessionTime   SessionTimeEntity
	sessionTimeMu sync.RWMutex

	holidays  map[int]HolidayEntity
	holidayMu sync.RWMutex

	orders   map[int]OrderEntity
	ordersMu sync.RWMutex

	trades   map[int]TradeEntity
	tradesMu sync.RWMutex

	contracts   map[int]ContractEntity
	contractsMu sync.RWMutex

	// Public fields (channels, config)
	Commit  chan any
	IsReady bool

	// Private fields
	allSync      []LedgerPointInterface
	rx           chan kafka.Message
	url          string
	topic        string
	id           string
	startid      string
	lastOrderNID int
}

type LedgerPointInterface interface {
	SyncServiceStart(a ServiceStart)
	SyncParameter(a Parameter)
	SyncSessionTime(a SessionTime)
	SyncHoliday(a Holiday)
	SyncAccount(a Account)
	SyncAccountLimit(a AccountLimit)
	SyncParticipant(a Participant)
	SyncInstrument(a Instrument)
	SyncOrder(a Order)
	SyncOrderAck(a OrderAck)
	SyncOrderNak(a OrderNak)
	SyncOrderPending(a OrderPending)
	SyncOrderWithdraw(a OrderWithdraw)
	SyncOrderWithdrawAck(a OrderWithdrawAck)
	SyncOrderWithdrawNak(a OrderWithdrawNak)
	SyncTrade(a Trade)
	SyncTradeWait(a TradeWait)
	SyncTradeAck(a TradeAck)
	SyncTradeNak(a TradeNak)
	SyncTradeReimburse(a TradeReimburse)
	SyncContract(a Contract)
	SyncSod(a Sod)
	SyncEod(a Eod)
}

// ============================================================================
// Thread-Safe Getter Methods (return copies)
// ============================================================================

// GetOrder returns a copy of the order by NID
func (lp *LedgerPoint) GetOrder(nid int) (OrderEntity, bool) {
	lp.ordersMu.RLock()
	defer lp.ordersMu.RUnlock()
	order, exists := lp.orders[nid]
	return order, exists
}

// GetAccount returns a copy of the account by code
func (lp *LedgerPoint) GetAccount(code string) (AccountEntity, bool) {
	lp.accountMu.RLock()
	defer lp.accountMu.RUnlock()
	account, exists := lp.accounts[code]
	return account, exists
}

// GetParticipant returns a copy of the participant by code
func (lp *LedgerPoint) GetParticipant(code string) (ParticipantEntity, bool) {
	lp.participantMu.RLock()
	defer lp.participantMu.RUnlock()
	participant, exists := lp.participants[code]
	return participant, exists
}

// GetInstrument returns a copy of the instrument by code
func (lp *LedgerPoint) GetInstrument(code string) (InstrumentEntity, bool) {
	lp.instrumentMu.RLock()
	defer lp.instrumentMu.RUnlock()
	instrument, exists := lp.instruments[code]
	return instrument, exists
}

// GetTrade returns a copy of the trade by NID
func (lp *LedgerPoint) GetTrade(nid int) (TradeEntity, bool) {
	lp.tradesMu.RLock()
	defer lp.tradesMu.RUnlock()
	trade, exists := lp.trades[nid]
	return trade, exists
}

// GetContract returns a copy of the contract by NID
func (lp *LedgerPoint) GetContract(nid int) (ContractEntity, bool) {
	lp.contractsMu.RLock()
	defer lp.contractsMu.RUnlock()
	contract, exists := lp.contracts[nid]
	return contract, exists
}

// GetHoliday returns a copy of the holiday by NID
func (lp *LedgerPoint) GetHoliday(nid int) (HolidayEntity, bool) {
	lp.holidayMu.RLock()
	defer lp.holidayMu.RUnlock()
	holiday, exists := lp.holidays[nid]
	return holiday, exists
}

// GetParameter returns a copy of the current parameter
func (lp *LedgerPoint) GetParameter() ParameterEntity {
	lp.parameterMu.RLock()
	defer lp.parameterMu.RUnlock()
	return lp.parameter
}

// GetSessionTime returns a copy of the current session time
func (lp *LedgerPoint) GetSessionTime() SessionTimeEntity {
	lp.sessionTimeMu.RLock()
	defer lp.sessionTimeMu.RUnlock()
	return lp.sessionTime
}

// ============================================================================
// Thread-Safe Iterator Methods (with lambda callbacks)
// ============================================================================

// ForEachOrder iterates over all orders with a callback function
// Return false from callback to stop iteration
func (lp *LedgerPoint) ForEachOrder(fn func(OrderEntity) bool) {
	lp.ordersMu.RLock()
	defer lp.ordersMu.RUnlock()
	for _, order := range lp.orders {
		if !fn(order) { // Pass copy
			break
		}
	}
}

// ForEachAccount iterates over all accounts with a callback function
func (lp *LedgerPoint) ForEachAccount(fn func(AccountEntity) bool) {
	lp.accountMu.RLock()
	defer lp.accountMu.RUnlock()
	for _, account := range lp.accounts {
		if !fn(account) {
			break
		}
	}
}

// ForEachParticipant iterates over all participants with a callback function
func (lp *LedgerPoint) ForEachParticipant(fn func(ParticipantEntity) bool) {
	lp.participantMu.RLock()
	defer lp.participantMu.RUnlock()
	for _, participant := range lp.participants {
		if !fn(participant) {
			break
		}
	}
}

// ForEachInstrument iterates over all instruments with a callback function
func (lp *LedgerPoint) ForEachInstrument(fn func(InstrumentEntity) bool) {
	lp.instrumentMu.RLock()
	defer lp.instrumentMu.RUnlock()
	for _, instrument := range lp.instruments {
		if !fn(instrument) {
			break
		}
	}
}

// ForEachTrade iterates over all trades with a callback function
func (lp *LedgerPoint) ForEachTrade(fn func(TradeEntity) bool) {
	lp.tradesMu.RLock()
	defer lp.tradesMu.RUnlock()
	for _, trade := range lp.trades {
		if !fn(trade) {
			break
		}
	}
}

// ForEachContract iterates over all contracts with a callback function
func (lp *LedgerPoint) ForEachContract(fn func(ContractEntity) bool) {
	lp.contractsMu.RLock()
	defer lp.contractsMu.RUnlock()
	for _, contract := range lp.contracts {
		if !fn(contract) {
			break
		}
	}
}

// ForEachHoliday iterates over all holidays with a callback function
func (lp *LedgerPoint) ForEachHoliday(fn func(HolidayEntity) bool) {
	lp.holidayMu.RLock()
	defer lp.holidayMu.RUnlock()
	for _, holiday := range lp.holidays {
		if !fn(holiday) {
			break
		}
	}
}

func CreateLedgerPoint(url string, topic string, id string) *LedgerPoint {

	point := LedgerPoint{
		// Initialize private entity maps
		holidays:     make(map[int]HolidayEntity),
		orders:       make(map[int]OrderEntity),
		trades:       make(map[int]TradeEntity),
		contracts:    make(map[int]ContractEntity),
		participants: make(map[string]ParticipantEntity),
		accounts:     make(map[string]AccountEntity),
		instruments:  make(map[string]InstrumentEntity),

		// Initialize public channels
		Commit:  make(chan any, 1000),
		IsReady: false,

		// Initialize private fields
		url:          url,
		topic:        topic,
		id:           id,
		startid:      id + "_" + time.Now().Format("20060102150405"),
		rx:           make(chan kafka.Message, 1000), // Buffer 1000 messages
		lastOrderNID: 0,
	}

	return &point
}

func (obj *LedgerPoint) Start(subscriber []LedgerPointInterface, ctx context.Context) {
	log.Println("🚀 Starting LedgerPoint processing...")
	if subscriber != nil {
		obj.allSync = subscriber
	}

	go obj.go_process(ctx)
}

func (obj *LedgerPoint) go_process(ctx context.Context) {

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{obj.url},
		Topic:     obj.topic,
		Partition: 0, // required when no GroupID
		MinBytes:  1,
		MaxBytes:  10e6,
		MaxWait:   100 * time.Millisecond, // Don't wait too long for batches
	})
	r.SetOffset(kafka.FirstOffset) // Start from beginning

	go obj.go_receive(r)

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{obj.url},
		Topic:        obj.topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 1 * time.Millisecond, // Flush immediately (default is 1 second!)
		Async:        false,                // Synchronous writes
		RequiredAcks: 1,                    // Wait for leader acknowledgment
	})
	defer writer.Close()

	// Commit ServiceStart event to mark the beginning of this LedgerPoint instance
	start := ServiceStart{
		ID:        obj.id,
		StartID:   obj.startid,
		StartTime: time.Now(),
	}
	obj.Commit <- start

	for {
		select {
		case trx := <-obj.Commit:

			val, _ := json.Marshal(trx)
			msg := kafka.Message{
				Key:   []byte("ledgerpoint"),
				Value: val,
			}

			switch trx.(type) {
			case ServiceStart:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("ServiceStart")}}
			case Holiday:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("Holiday")}}
			case Parameter:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("Parameter")}}
			case SessionTime:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("SessionTime")}}
			case Instrument:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("Instrument")}}
			case Participant:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("Participant")}}
			case Account:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("Account")}}
			case AccountLimit:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("AccountLimit")}}
			case Order:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("Order")}}
			case OrderAck:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("OrderAck")}}
			case OrderNak:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("OrderNak")}}
			case OrderPending:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("OrderPending")}}
			case OrderWithdraw:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("OrderWithdraw")}}
			case Trade:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("Trade")}}
			case TradeAck:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("TradeAck")}}
			case TradeNak:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("TradeNak")}}
			case TradeReimburse:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("TradeReimburse")}}
			case Contract:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("Contract")}}
			case Sod:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("Sod")}}
			case Eod:
				msg.Headers = []kafka.Header{{Key: "event-type", Value: []byte("Eod")}}
			}

			writeCtx, writeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := writer.WriteMessages(writeCtx, msg)
			writeCancel() // Cancel immediately after write, don't defer in loop!
			if err != nil {
				log.Fatalf("failed to write message: %v", err)
			}

		case msg := <-obj.rx:
			// Process incoming message
			// Check if message has headers
			if len(msg.Headers) == 0 {
				log.Printf("⚠️  Received message without headers, skipping")
				continue
			}

			switch string(msg.Headers[0].Value) {
			case "ServiceStart":
				var serviceStart ServiceStart
				json.Unmarshal(msg.Value, &serviceStart)
				obj.SyncServiceStart(serviceStart)
			case "Holiday":
				var holiday Holiday
				json.Unmarshal(msg.Value, &holiday)
				obj.SyncHoliday(holiday)
			case "Parameter":
				var parameter Parameter
				json.Unmarshal(msg.Value, &parameter)
				obj.SyncParameter(parameter)
			case "SessionTime":
				var sessionTime SessionTime
				json.Unmarshal(msg.Value, &sessionTime)
				obj.SyncSessionTime(sessionTime)
			case "Account":
				var account Account
				json.Unmarshal(msg.Value, &account)
				obj.SyncAccount(account)
			case "AccountLimit":
				var accountLimit AccountLimit
				json.Unmarshal(msg.Value, &accountLimit)
				obj.SyncAccountLimit(accountLimit)
			case "Instrument":
				var instrument Instrument
				json.Unmarshal(msg.Value, &instrument)
				obj.SyncInstrument(instrument)
			case "Participant":
				var participant Participant
				json.Unmarshal(msg.Value, &participant)
				obj.SyncParticipant(participant)
			case "Order":
				var order Order
				json.Unmarshal(msg.Value, &order)
				obj.SyncOrder(order)
			case "OrderAck":
				var orderAck OrderAck
				json.Unmarshal(msg.Value, &orderAck)
				obj.SyncOrderAck(orderAck)
			case "OrderNak":
				var orderNak OrderNak
				json.Unmarshal(msg.Value, &orderNak)
				obj.SyncOrderNak(orderNak)
			case "OrderPending":
				var orderPending OrderPending
				json.Unmarshal(msg.Value, &orderPending)
				obj.SyncOrderPending(orderPending)
			case "OrderWithdraw":
				var orderWithdraw OrderWithdraw
				json.Unmarshal(msg.Value, &orderWithdraw)
				obj.SyncOrderWithdraw(orderWithdraw)
			case "Trade":
				var trade Trade
				json.Unmarshal(msg.Value, &trade)
				obj.SyncTrade(trade)
			case "TradeAck":
				var tradeAck TradeAck
				json.Unmarshal(msg.Value, &tradeAck)
				obj.SyncTradeAck(tradeAck)
			case "TradeNak":
				var tradeNak TradeNak
				json.Unmarshal(msg.Value, &tradeNak)
				obj.SyncTradeNak(tradeNak)
			case "TradeReimburse":
				var tradeReimburse TradeReimburse
				json.Unmarshal(msg.Value, &tradeReimburse)
				obj.SyncTradeReimburse(tradeReimburse)
			case "Contract":
				var contract Contract
				json.Unmarshal(msg.Value, &contract)
				obj.SyncContract(contract)
			case "Sod":
				var sod Sod
				json.Unmarshal(msg.Value, &sod)
				obj.SyncSod(sod)
			case "Eod":
				var eod Eod
				json.Unmarshal(msg.Value, &eod)
				obj.SyncEod(eod)
			}

		case <-ctx.Done():
			r.Close()
			log.Println("⏰ Context cancelled, stopping processing...")
			return
		}
	}
}

func (obj *LedgerPoint) go_receive(r *kafka.Reader) {
	log.Println("📥 Waiting for messages...")

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Fatalf("❌ could not read message: %v", err)
		}

		// log.Printf("🔔 <-- key=%s value=%s offset=%d\n",
		// 	string(m.Key), string(m.Value), m.Offset)

		obj.rx <- m
	}
}

func (obj *LedgerPoint) SyncServiceStart(a ServiceStart) {
	if a.StartID == obj.startid {
		obj.IsReady = true
		for _, sync := range obj.allSync {
			sync.SyncServiceStart(a)
		}
	}
}

func (obj *LedgerPoint) SyncHoliday(a Holiday) {
	obj.holidayMu.Lock()
	obj.holidays[a.NID] = HolidayEntity{
		NID:         a.NID,
		Date:        a.Date,
		Description: a.Description,
	}
	obj.holidayMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncHoliday(a)
	}
}

func (obj *LedgerPoint) SyncParameter(a Parameter) {
	obj.parameterMu.Lock()
	obj.parameter = ParameterEntity{
		NID:               a.NID,
		Update:            a.Update,
		Description:       a.Description,
		FlatFee:           a.FlatFee,
		LendingFee:        a.LendingFee,
		BorrowingFee:      a.BorrowingFee,
		MaxQuantity:       a.MaxQuantity,
		BorrowMaxOpenDay:  a.BorrowMaxOpenDay,
		DenominationLimit: a.DenominationLimit,
		LastUpdate:        time.Now(),
	}
	obj.parameterMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncParameter(a)
	}
}

func (obj *LedgerPoint) SyncSessionTime(a SessionTime) {
	obj.sessionTimeMu.Lock()
	obj.sessionTime = SessionTimeEntity{
		NID:           a.NID,
		Description:   a.Description,
		Update:        a.Update,
		Session1Start: a.Session1Start,
		Session1End:   a.Session1End,
		Session2Start: a.Session2Start,
		Session2End:   a.Session2End,
		LastUpdate:    time.Now(),
	}
	obj.sessionTimeMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncSessionTime(a)
	}
}

func (obj *LedgerPoint) SyncAccount(a Account) {
	obj.accountMu.Lock()
	obj.accounts[a.Code] = AccountEntity{
		NID:             a.NID,
		Code:            a.Code,
		SID:             a.SID,
		Name:            a.Name,
		ParticipantNID:  a.ParticipantNID,
		ParticipantCode: a.ParticipantCode,
		TradeLimit:      0,
		PoolLimit:       0,
		LastUpdate:      time.Now(),
	}
	obj.accountMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncAccount(a)
	}
}

func (obj *LedgerPoint) SyncAccountLimit(a AccountLimit) {
	obj.accountMu.Lock()
	if account, exists := obj.accounts[a.Code]; exists {
		account.TradeLimit = a.TradeLimit
		account.PoolLimit = a.PoolLimit
		account.LastUpdate = time.Now()
		obj.accounts[a.Code] = account
	}
	obj.accountMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncAccountLimit(a)
	}
}

func (obj *LedgerPoint) SyncInstrument(a Instrument) {
	obj.instrumentMu.Lock()
	obj.instruments[a.Code] = InstrumentEntity{
		NID:        a.NID,
		Code:       a.Code,
		Name:       a.Name,
		Type:       a.Type,
		Status:     a.Status,
		LastUpdate: time.Now(),
	}
	obj.instrumentMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncInstrument(a)
	}
}

func (obj *LedgerPoint) SyncParticipant(a Participant) {
	obj.participantMu.Lock()
	obj.participants[a.Code] = ParticipantEntity{
		NID:             a.NID,
		Code:            a.Code,
		Name:            a.Name,
		BorrEligibility: a.BorrEligibility,
		LendEligibility: a.LendEligibility,
		LastUpdate:      time.Now(),
	}
	obj.participantMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncParticipant(a)
	}
}

func (obj *LedgerPoint) SyncOrder(a Order) {
	obj.ordersMu.Lock()
	obj.orders[a.NID] = OrderEntity{
		NID:               a.NID,
		PrevNID:           a.PrevNID,
		ReffRequestID:     a.ReffRequestID,
		AccountNID:        a.AccountNID,
		AccountCode:       a.AccountCode,
		ParticipantNID:    a.ParticipantNID,
		ParticipantCode:   a.ParticipantCode,
		InstrumentNID:     a.InstrumentNID,
		InstrumentCode:    a.InstrumentCode,
		Side:              a.Side,
		Quantity:          a.Quantity,
		DoneQuantity:      0,
		SettlementDate:    a.SettlementDate,
		ReimbursementDate: a.ReimbursementDate,
		Periode:           a.Periode,
		State:             "S",
		MarketPrice:       a.MarketPrice,
		Rate:              a.Rate,
		Instruction:       a.Instruction,
		ARO:               a.ARO,
		WReffRequestID:    "",
		Message:           "",
		EntryAt:           time.Now(),
		OpenAt:            time.Now(),
		RejectAt:          time.Now(),
		AmmendAt:          time.Now(),
		WithdrawAt:        time.Now(),
	}
	obj.ordersMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncOrder(a)
	}
}

func (obj *LedgerPoint) SyncOrderAck(a OrderAck) {
	obj.ordersMu.Lock()
	if order, exists := obj.orders[a.OrderNID]; exists {
		order.OpenAt = time.Now()
		order.State = "O"
		obj.orders[a.OrderNID] = order
		if order.PrevNID != 0 {
			if prevOrder, exists := obj.orders[order.PrevNID]; exists {
				prevOrder.AmmendAt = time.Now()
				prevOrder.State = "A"
				obj.orders[order.PrevNID] = prevOrder
			}
		}
	}
	obj.ordersMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncOrderAck(a)
	}
}

func (obj *LedgerPoint) SyncOrderNak(a OrderNak) {
	obj.ordersMu.Lock()
	if order, exists := obj.orders[a.OrderNID]; exists {
		order.RejectAt = time.Now()
		order.State = "R"
		order.Message = a.Message
		obj.orders[a.OrderNID] = order
	}
	obj.ordersMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncOrderNak(a)
	}
}

func (obj *LedgerPoint) SyncOrderPending(a OrderPending) {
	obj.ordersMu.Lock()
	if order, exists := obj.orders[a.OrderNID]; exists {
		order.WithdrawAt = time.Now()
		order.State = "G"
		obj.orders[a.OrderNID] = order
	}
	obj.ordersMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncOrderPending(a)
	}
}

func (obj *LedgerPoint) SyncOrderWithdraw(a OrderWithdraw) {
	obj.ordersMu.Lock()
	if order, exists := obj.orders[a.OrderNID]; exists {
		order.WReffRequestID = a.ReffRequestID
		obj.orders[a.OrderNID] = order
	}
	obj.ordersMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncOrderWithdraw(a)
	}
}

func (obj *LedgerPoint) SyncOrderWithdrawAck(a OrderWithdrawAck) {
	obj.ordersMu.Lock()
	if order, exists := obj.orders[a.OrderNID]; exists {
		order.WithdrawAt = time.Now()
		order.State = "W"
		obj.orders[a.OrderNID] = order
	}
	obj.ordersMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncOrderWithdrawAck(a)
	}
}

func (obj *LedgerPoint) SyncOrderWithdrawNak(a OrderWithdrawNak) {
	obj.ordersMu.Lock()
	if order, exists := obj.orders[a.OrderNID]; exists {
		order.WReffRequestID = ""
		obj.orders[a.OrderNID] = order
	}
	obj.ordersMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncOrderWithdrawNak(a)
	}
}

func (obj *LedgerPoint) SyncTrade(a Trade) {
	// Lock multiple mutexes in consistent order to prevent deadlock
	obj.ordersMu.Lock()
	obj.contractsMu.Lock()
	obj.tradesMu.Lock()

	borrContract := make([]int, 0)
	for _, borr := range a.Borrower {
		contract := ContractEntity{
			NID:                    borr.NID,
			TradeNID:               borr.TradeNID,
			KpeiReff:               borr.KpeiReff,
			Side:                   borr.Side,
			AccountParticipantNID:  borr.AccountParticipantNID,
			AccountParticipantCode: borr.AccountParticipantCode,
			AccountNID:             borr.AccountNID,
			AccountCode:            borr.AccountCode,
			OrderNID:               borr.OrderNID,
			InstrumentNID:          borr.InstrumentNID,
			InstrumentCode:         borr.InstrumentCode,
			Quantity:               borr.Quantity,
			Periode:                borr.Periode,
			FeeFlatVal:             borr.FeeFlatVal,
			FeeValDaily:            borr.FeeValDaily,
			FeeValAccumulated:      borr.FeeValAccumulated,
			State:                  borr.State,
			MatchedAt:              borr.MatchedAt,
			ReimburseAt:            borr.ReimburseAt,
		}
		borrContract = append(borrContract, borr.NID)
		obj.contracts[borr.NID] = contract

		if order, exists := obj.orders[borr.OrderNID]; exists {
			order.DoneQuantity += borr.Quantity
			if order.DoneQuantity >= order.Quantity {
				order.State = "M"
			} else {
				order.State = "P"
			}
			obj.orders[borr.OrderNID] = order
		}
	}

	lendContract := make([]int, 0)
	for _, lend := range a.Lender {
		contract := ContractEntity{
			NID:                    lend.NID,
			TradeNID:               lend.TradeNID,
			KpeiReff:               lend.KpeiReff,
			Side:                   lend.Side,
			AccountParticipantNID:  lend.AccountParticipantNID,
			AccountParticipantCode: lend.AccountParticipantCode,
			AccountNID:             lend.AccountNID,
			AccountCode:            lend.AccountCode,
			OrderNID:               lend.OrderNID,
			InstrumentNID:          lend.InstrumentNID,
			InstrumentCode:         lend.InstrumentCode,
			Quantity:               lend.Quantity,
			Periode:                lend.Periode,
			FeeFlatVal:             lend.FeeFlatVal,
			FeeValDaily:            lend.FeeValDaily,
			FeeValAccumulated:      lend.FeeValAccumulated,
			State:                  lend.State,
			MatchedAt:              lend.MatchedAt,
			ReimburseAt:            lend.ReimburseAt,
		}
		lendContract = append(lendContract, lend.NID)
		obj.contracts[lend.NID] = contract

		if order, exists := obj.orders[lend.OrderNID]; exists {
			order.DoneQuantity += lend.Quantity
			if order.DoneQuantity >= order.Quantity {
				order.State = "M"
			} else {
				order.State = "P"
			}
			obj.orders[lend.OrderNID] = order
		}
	}

	obj.trades[a.NID] = TradeEntity{
		NID:            a.NID,
		KpeiReff:       a.KpeiReff,
		InstrumentNID:  a.InstrumentNID,
		InstrumentCode: a.InstrumentCode,
		Quantity:       a.Quantity,
		Periode:        a.Periode,
		FeeLendRate:    a.FeeLendRate,
		MatchedAt:      a.MatchedAt,
		ReimburseAt:    a.ReimburseAt,
		Borrower:       borrContract,
		Lender:         lendContract,
	}

	// Unlock in reverse order
	obj.tradesMu.Unlock()
	obj.contractsMu.Unlock()
	obj.ordersMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncTrade(a)
	}
}

func (obj *LedgerPoint) SyncTradeWait(a TradeWait) {
	obj.tradesMu.Lock()
	obj.contractsMu.Lock()

	if trade, exists := obj.trades[a.TradeNID]; exists {
		trade.State = "E"
		obj.trades[a.TradeNID] = trade

		for _, contractNID := range trade.Borrower {
			if contract, exists := obj.contracts[contractNID]; exists {
				contract.State = "E"
				obj.contracts[contractNID] = contract
			}
		}

		for _, contractNID := range trade.Lender {
			if contract, exists := obj.contracts[contractNID]; exists {
				contract.State = "E"
				obj.contracts[contractNID] = contract
			}
		}
	}
	obj.contractsMu.Unlock()
	obj.tradesMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncTradeWait(a)
	}
}

func (obj *LedgerPoint) SyncTradeAck(a TradeAck) {
	obj.tradesMu.Lock()
	obj.contractsMu.Lock()

	if trade, exists := obj.trades[a.TradeNID]; exists {
		trade.State = "O"
		obj.trades[a.TradeNID] = trade
		for _, contractNID := range trade.Borrower {
			if contract, exists := obj.contracts[contractNID]; exists {
				contract.State = "O"
				obj.contracts[contractNID] = contract
			}
		}
		for _, contractNID := range trade.Lender {
			if contract, exists := obj.contracts[contractNID]; exists {
				contract.State = "O"
				obj.contracts[contractNID] = contract
			}
		}
	}
	obj.contractsMu.Unlock()
	obj.tradesMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncTradeAck(a)
	}
}

func (obj *LedgerPoint) SyncTradeNak(a TradeNak) {
	obj.ordersMu.Lock()
	obj.tradesMu.Lock()
	obj.contractsMu.Lock()

	if trade, exists := obj.trades[a.TradeNID]; exists {
		trade.State = "R"
		obj.trades[a.TradeNID] = trade
		for _, contractNID := range trade.Borrower {
			if contract, exists := obj.contracts[contractNID]; exists {
				contract.State = "R"
				obj.contracts[contractNID] = contract
				if order, exists := obj.orders[contract.OrderNID]; exists {
					order.DoneQuantity -= contract.Quantity
					if order.DoneQuantity > 0 {
						order.State = "P"
					} else {
						order.State = "O"
					}

					obj.orders[contract.OrderNID] = order
				}
			}
		}
		for _, contractNID := range trade.Lender {
			if contract, exists := obj.contracts[contractNID]; exists {
				contract.State = "R"
				obj.contracts[contractNID] = contract
				if order, exists := obj.orders[contract.OrderNID]; exists {
					order.DoneQuantity -= contract.Quantity
					if order.DoneQuantity > 0 {
						order.State = "P"
					} else {
						order.State = "O"
					}
					obj.orders[contract.OrderNID] = order
				}
			}
		}
	}
	obj.contractsMu.Unlock()
	obj.tradesMu.Unlock()
	obj.ordersMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncTradeNak(a)
	}
}

func (obj *LedgerPoint) SyncTradeReimburse(a TradeReimburse) {
	obj.tradesMu.Lock()
	obj.contractsMu.Lock()

	if trade, exists := obj.trades[a.TradeNID]; exists {
		trade.State = "C"
		trade.ReimburseAt = time.Now()
		obj.trades[a.TradeNID] = trade
		for _, contractNID := range trade.Borrower {
			if contract, exists := obj.contracts[contractNID]; exists {
				contract.State = "C"
				contract.ReimburseAt = time.Now()
				obj.contracts[contractNID] = contract
			}
		}
		for _, contractNID := range trade.Lender {
			if contract, exists := obj.contracts[contractNID]; exists {
				contract.State = "C"
				contract.ReimburseAt = time.Now()
				obj.contracts[contractNID] = contract
			}
		}
	}
	obj.contractsMu.Unlock()
	obj.tradesMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncTradeReimburse(a)
	}
}

func (obj *LedgerPoint) SyncContract(a Contract) {
	obj.contractsMu.Lock()

	obj.contracts[a.NID] = ContractEntity{
		NID:                    a.NID,
		TradeNID:               a.TradeNID,
		KpeiReff:               a.KpeiReff,
		Side:                   a.Side,
		AccountParticipantNID:  a.AccountParticipantNID,
		AccountParticipantCode: a.AccountParticipantCode,
		AccountNID:             a.AccountNID,
		AccountCode:            a.AccountCode,
		OrderNID:               a.OrderNID,
		InstrumentNID:          a.InstrumentNID,
		InstrumentCode:         a.InstrumentCode,
		Quantity:               a.Quantity,
		Periode:                a.Periode,
		FeeFlatVal:             a.FeeFlatVal,
		FeeValDaily:            a.FeeValDaily,
		FeeValAccumulated:      a.FeeValAccumulated,
		State:                  a.State,
		MatchedAt:              a.MatchedAt,
		ReimburseAt:            a.ReimburseAt,
	}
	obj.contractsMu.Unlock()

	for _, sync := range obj.allSync {
		sync.SyncContract(a)
	}
}

// GetCurrentTimeMillis returns current time in milliseconds
func GetCurrentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func (obj *LedgerPoint) SyncSod(a Sod) {
	for _, sync := range obj.allSync {
		sync.SyncSod(a)
	}
}

func (obj *LedgerPoint) SyncEod(a Eod) {
	for _, sync := range obj.allSync {
		sync.SyncEod(a)
	}
}

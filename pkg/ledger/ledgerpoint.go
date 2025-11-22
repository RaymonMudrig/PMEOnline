package ledger

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type LedgerPoint struct {
	Participant map[string]ParticipantEntity `json:"participant"`
	Account     map[string]AccountEntity     `json:"account"`
	Instrument  map[string]InstrumentEntity  `json:"instrument"`
	Parameter   ParameterEntity              `json:"parameter"`
	SessionTime SessionTimeEntity            `json:"session_time"`
	Holiday     map[int]HolidayEntity        `json:"holiday"`
	Orders      map[int]OrderEntity          `json:"orders"`
	Trades      map[int]TradeEntity          `json:"trades"`
	Contracts   map[int]ContractEntity       `json:"contracts"`
	Commit      chan any
	Sync        chan LedgerPointInterface
	allSync     []LedgerPointInterface
	IsReady     bool
	rx          chan kafka.Message
	url         string
	topic       string
	id          string
	startid     string
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
	SyncOrderWithdraw(a OrderWithdraw)
	SyncOrderWithdrawAck(a OrderWithdrawAck)
	SyncOrderWithdrawNak(a OrderWithdrawNak)
	SyncTrade(a Trade)
	SyncTradeWait(a TradeWait)
	SyncTradeAck(a TradeAck)
	SyncTradeNak(a TradeNak)
	SyncTradeReimburse(a TradeReimburse)
	SyncContract(a Contract)
}

func CreateLedgerPoint(url string, topic string, id string, ctx context.Context) *LedgerPoint {

	point := LedgerPoint{
		Holiday:     make(map[int]HolidayEntity),
		Orders:      make(map[int]OrderEntity),
		Trades:      make(map[int]TradeEntity),
		Contracts:   make(map[int]ContractEntity),
		Participant: make(map[string]ParticipantEntity),
		Account:     make(map[string]AccountEntity),
		Instrument:  make(map[string]InstrumentEntity),
		Commit:      make(chan any, 1000),
		Sync:        make(chan LedgerPointInterface, 1000),
		IsReady:     false,
		url:         url,
		topic:       topic,
		id:          id,
		startid:     id + "_" + time.Now().Format("20060102150405"),
		rx:          make(chan kafka.Message, 1000), // Buffer 1000 messages
	}

	go point.go_process(ctx)

	return &point
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
			}

			writeCtx, writeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := writer.WriteMessages(writeCtx, msg)
			writeCancel() // Cancel immediately after write, don't defer in loop!
			if err != nil {
				log.Fatalf("failed to write message: %v", err)
			}

		case msg := <-obj.rx:
			// Process incoming message
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
			}

		case sync := <-obj.Sync:
			obj.allSync = append(obj.allSync, sync)

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
	obj.Holiday[a.NID] = HolidayEntity{
		NID:         a.NID,
		Date:        a.Date,
		Description: a.Description,
	}
	for _, sync := range obj.allSync {
		sync.SyncHoliday(a)
	}
}
func (obj *LedgerPoint) SyncParameter(a Parameter) {
	obj.Parameter = ParameterEntity{
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
	for _, sync := range obj.allSync {
		sync.SyncParameter(a)
	}
}
func (obj *LedgerPoint) SyncSessionTime(a SessionTime) {
	obj.SessionTime = SessionTimeEntity{
		NID:           a.NID,
		Description:   a.Description,
		Update:        a.Update,
		Session1Start: a.Session1Start,
		Session1End:   a.Session1End,
		Session2Start: a.Session2Start,
		Session2End:   a.Session2End,
		LastUpdate:    time.Now(),
	}
	for _, sync := range obj.allSync {
		sync.SyncSessionTime(a)
	}
}
func (obj *LedgerPoint) SyncAccount(a Account) {
	obj.Account[a.Code] = AccountEntity{
		NID:             a.NID,
		Code:            a.Code,
		SID:             a.SID,
		Name:            a.Name,
		Address:         a.Address,
		ParticipantNID:  a.ParticipantNID,
		ParticipantCode: a.ParticipantCode,
		TradeLimit:      0,
		PoolLimit:       0,
		LastUpdate:      time.Now(),
	}
	for _, sync := range obj.allSync {
		sync.SyncAccount(a)
	}
}
func (obj *LedgerPoint) SyncAccountLimit(a AccountLimit) {
	if account, exists := obj.Account[a.Code]; exists {
		account.TradeLimit = a.TradeLimit
		account.PoolLimit = a.PoolLimit
		account.LastUpdate = time.Now()
		obj.Account[a.Code] = account
	}
	for _, sync := range obj.allSync {
		sync.SyncAccountLimit(a)
	}
}
func (obj *LedgerPoint) SyncInstrument(a Instrument) {
	obj.Instrument[a.Code] = InstrumentEntity{
		NID:        a.NID,
		Code:       a.Code,
		Name:       a.Name,
		Type:       a.Type,
		Status:     a.Status,
		LastUpdate: time.Now(),
	}
	for _, sync := range obj.allSync {
		sync.SyncInstrument(a)
	}
}
func (obj *LedgerPoint) SyncParticipant(a Participant) {
	obj.Participant[a.Code] = ParticipantEntity{
		NID:             a.NID,
		Code:            a.Code,
		Name:            a.Name,
		BorrEligibility: a.BorrEligibility,
		LendEligibility: a.LendEligibility,
		LastUpdate:      time.Now(),
	}
	for _, sync := range obj.allSync {
		sync.SyncParticipant(a)
	}
}
func (obj *LedgerPoint) SyncOrder(a Order) {
	obj.Orders[a.NID] = OrderEntity{
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
		State:             "S",
		EntryAt:           time.Now(),
		OpenAt:            time.Now(),
		RejectAt:          time.Now(),
		AmmendAt:          time.Now(),
		WithdrawAt:        time.Now(),
	}
	for _, sync := range obj.allSync {
		sync.SyncOrder(a)
	}
}
func (obj *LedgerPoint) SyncOrderAck(a OrderAck) {
	if order, exists := obj.Orders[a.OrderNID]; exists {
		order.OpenAt = time.Now()
		order.State = "O"
		obj.Orders[a.OrderNID] = order
		if order.PrevNID != 0 {
			if prevOrder, exists := obj.Orders[order.PrevNID]; exists {
				prevOrder.AmmendAt = time.Now()
				prevOrder.State = "A"
				obj.Orders[order.PrevNID] = prevOrder
			}
		}
	}
	for _, sync := range obj.allSync {
		sync.SyncOrderAck(a)
	}
}
func (obj *LedgerPoint) SyncOrderNak(a OrderNak) {
	if order, exists := obj.Orders[a.OrderNID]; exists {
		order.RejectAt = time.Now()
		order.State = "R"
		obj.Orders[a.OrderNID] = order
	}
	for _, sync := range obj.allSync {
		sync.SyncOrderNak(a)
	}
}
func (obj *LedgerPoint) SyncOrderWithdraw(a OrderWithdraw) {
	if order, exists := obj.Orders[a.OrderNID]; exists {
		order.WReffRequestID = a.ReffRequestID
		obj.Orders[a.OrderNID] = order
	}
	for _, sync := range obj.allSync {
		sync.SyncOrderWithdraw(a)
	}
}
func (obj *LedgerPoint) SyncOrderWithdrawAck(a OrderWithdrawAck) {
	if order, exists := obj.Orders[a.OrderNID]; exists {
		order.WithdrawAt = time.Now()
		order.State = "W"
		obj.Orders[a.OrderNID] = order
	}
	for _, sync := range obj.allSync {
		sync.SyncOrderWithdrawAck(a)
	}
}
func (obj *LedgerPoint) SyncOrderWithdrawNak(a OrderWithdrawNak) {
	if order, exists := obj.Orders[a.OrderNID]; exists {
		order.WReffRequestID = ""
		obj.Orders[a.OrderNID] = order
	}
	for _, sync := range obj.allSync {
		sync.SyncOrderWithdrawNak(a)
	}
}
func (obj *LedgerPoint) SyncTrade(a Trade) {
	obj.Trades[a.NID] = TradeEntity{
		NID:            a.NID,
		KpeiReff:       a.KpeiReff,
		InstrumentNID:  a.InstrumentNID,
		InstrumentCode: a.InstrumentCode,
		Quantity:       a.Quantity,
		Periode:        a.Periode,
		FeeLendRate:    a.FeeLendRate,
		MatchedAt:      a.MatchedAt,
		ReimburseAt:    a.ReimburseAt,
		Borrower:       make(map[int]ContractEntity),
		Lender:         make(map[int]ContractEntity),
	}
	for _, borr := range a.Borrower {
		obj.Trades[a.NID].Borrower[borr.NID] = ContractEntity{
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
		if order, exists := obj.Orders[borr.OrderNID]; exists {
			order.DoneQuantity += borr.Quantity
			if order.DoneQuantity >= order.Quantity {
				order.State = "M"
			} else {
				order.State = "P"
			}
			obj.Orders[borr.OrderNID] = order
		}
	}
	for _, lend := range a.Lender {
		obj.Trades[a.NID].Lender[lend.NID] = ContractEntity{
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
		if order, exists := obj.Orders[lend.OrderNID]; exists {
			order.DoneQuantity += lend.Quantity
			if order.DoneQuantity >= order.Quantity {
				order.State = "M"
			} else {
				order.State = "P"
			}
			obj.Orders[lend.OrderNID] = order
		}
	}

	for _, sync := range obj.allSync {
		sync.SyncTrade(a)
	}
}
func (obj *LedgerPoint) SyncTradeWait(a TradeWait) {
	if trade, exists := obj.Trades[a.TradeNID]; exists {
		trade.State = "E"
		for _, contract := range trade.Borrower {
			contract.State = "E"
			trade.Borrower[contract.NID] = contract
		}
		for _, contract := range trade.Lender {
			contract.State = "E"
			trade.Lender[contract.NID] = contract
		}
		obj.Trades[a.TradeNID] = trade
	}
	for _, sync := range obj.allSync {
		sync.SyncTradeWait(a)
	}
}
func (obj *LedgerPoint) SyncTradeAck(a TradeAck) {
	if trade, exists := obj.Trades[a.TradeNID]; exists {
		trade.State = "O"
		for _, contract := range trade.Borrower {
			contract.State = "O"
			trade.Borrower[contract.NID] = contract
		}
		for _, contract := range trade.Lender {
			contract.State = "O"
			trade.Lender[contract.NID] = contract
		}
		obj.Trades[a.TradeNID] = trade
	}
	for _, sync := range obj.allSync {
		sync.SyncTradeAck(a)
	}
}
func (obj *LedgerPoint) SyncTradeNak(a TradeNak) {
	if trade, exists := obj.Trades[a.TradeNID]; exists {
		trade.State = "R"
		for _, contract := range trade.Borrower {
			contract.State = "R"
			trade.Borrower[contract.NID] = contract
			if order, exists := obj.Orders[contract.OrderNID]; exists {
				order.DoneQuantity -= contract.Quantity
				if order.DoneQuantity > 0 {
					order.State = "P"
				} else {
					order.State = "O"
				}
				obj.Orders[contract.OrderNID] = order
			}
		}
		for _, contract := range trade.Lender {
			contract.State = "R"
			trade.Lender[contract.NID] = contract
			if order, exists := obj.Orders[contract.OrderNID]; exists {
				order.DoneQuantity -= contract.Quantity
				if order.DoneQuantity > 0 {
					order.State = "P"
				} else {
					order.State = "O"
				}
				obj.Orders[contract.OrderNID] = order
			}
		}
		obj.Trades[a.TradeNID] = trade
	}
	for _, sync := range obj.allSync {
		sync.SyncTradeNak(a)
	}
}
func (obj *LedgerPoint) SyncTradeReimburse(a TradeReimburse) {
}
func (obj *LedgerPoint) SyncContract(a Contract) {
	obj.Contracts[a.NID] = ContractEntity{
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
	for _, sync := range obj.allSync {
		sync.SyncContract(a)
	}
}

// GetCurrentTimeMillis returns current time in milliseconds
func GetCurrentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

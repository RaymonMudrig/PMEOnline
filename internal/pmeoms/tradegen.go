package pmeoms

import (
	"fmt"
	"time"

	"pmeonline/pkg/ledger"
	"pmeonline/pkg/ledger/risk"
)

// TradeGenerator handles trade and contract generation
type TradeGenerator struct {
	calculator *risk.Calculator
	tradeIDSeq int
}

// NewTradeGenerator creates a new trade generator
func NewTradeGenerator(calc *risk.Calculator) *TradeGenerator {
	return &TradeGenerator{
		calculator: calc,
		tradeIDSeq: 1,
	}
}

// GenerateTrade creates a Trade and Contracts from a match
func (tg *TradeGenerator) GenerateTrade(match Match) ledger.Trade {
	// Generate unique trade reference
	tradeNID := int(ledger.GetCurrentTimeMillis()) + tg.tradeIDSeq
	tg.tradeIDSeq++

	kpeiReff := fmt.Sprintf("PME-%s-%d", time.Now().Format("20060102"), tradeNID)

	// Get fee rates
	flatFeeRate, borrowFeeRate, lendFeeRate := tg.calculator.GetFeeRates()

	// Determine market price (use borrower's market price)
	marketPrice := match.BorrowerOrder.MarketPrice
	if marketPrice <= 0 {
		marketPrice = match.LenderOrder.MarketPrice
	}

	// Calculate fees
	flatFee := tg.calculator.CalculateFlatFee(marketPrice, match.Quantity)
	borrowDailyFee := tg.calculator.CalculateBorrowingDailyFee(marketPrice, match.Quantity)
	lendDailyFee := tg.calculator.CalculateLendingDailyFee(marketPrice, match.Quantity)

	// Determine settlement and reimbursement dates
	// Use the later settlement date and earlier reimbursement date
	settlementDate := match.BorrowerOrder.SettlementDate
	if match.LenderOrder.SettlementDate.After(settlementDate) {
		settlementDate = match.LenderOrder.SettlementDate
	}

	reimbursementDate := match.BorrowerOrder.ReimbursementDate
	if match.LenderOrder.ReimbursementDate.Before(reimbursementDate) {
		reimbursementDate = match.LenderOrder.ReimbursementDate
	}

	// Calculate periode
	periode := int(reimbursementDate.Sub(settlementDate).Hours() / 24)

	// Create borrower contract
	borrowerContract := ledger.Contract{
		NID:                    tradeNID*10 + 1,
		TradeNID:               tradeNID,
		KpeiReff:               kpeiReff + "-BORR",
		Side:                   "BORR",
		AccountNID:             match.BorrowerOrder.AccountNID,
		AccountCode:            match.BorrowerOrder.AccountCode,
		AccountSID:             "", // Will be filled by eClear API
		AccountParticipantNID:  match.BorrowerOrder.ParticipantNID,
		AccountParticipantCode: match.BorrowerOrder.ParticipantCode,
		OrderNID:               match.BorrowerOrder.NID,
		InstrumentNID:          match.BorrowerOrder.InstrumentNID,
		InstrumentCode:         match.BorrowerOrder.InstrumentCode,
		Quantity:               match.Quantity,
		Periode:                periode,
		State:                  "S", // Submitted
		FeeFlatVal:             flatFee,
		FeeValDaily:            borrowDailyFee,
		FeeValAccumulated:      0, // Will be updated daily
		MatchedAt:              time.Now(),
		ReimburseAt:            reimbursementDate,
	}

	// Create lender contract
	lenderContract := ledger.Contract{
		NID:                    tradeNID*10 + 2,
		TradeNID:               tradeNID,
		KpeiReff:               kpeiReff + "-LEND",
		Side:                   "LEND",
		AccountNID:             match.LenderOrder.AccountNID,
		AccountCode:            match.LenderOrder.AccountCode,
		AccountSID:             "", // Will be filled by eClear API
		AccountParticipantNID:  match.LenderOrder.ParticipantNID,
		AccountParticipantCode: match.LenderOrder.ParticipantCode,
		OrderNID:               match.LenderOrder.NID,
		InstrumentNID:          match.LenderOrder.InstrumentNID,
		InstrumentCode:         match.LenderOrder.InstrumentCode,
		Quantity:               match.Quantity,
		Periode:                periode,
		State:                  "S", // Submitted
		FeeFlatVal:             0,   // Lender doesn't pay flat fee
		FeeValDaily:            lendDailyFee,
		FeeValAccumulated:      0, // Will be updated daily
		MatchedAt:              time.Now(),
		ReimburseAt:            reimbursementDate,
	}

	// Create trade
	trade := ledger.Trade{
		NID:            tradeNID,
		KpeiReff:       kpeiReff,
		InstrumentNID:  match.BorrowerOrder.InstrumentNID,
		InstrumentCode: match.BorrowerOrder.InstrumentCode,
		Quantity:       match.Quantity,
		Periode:        periode,
		State:          "S", // Submitted
		FeeFlatRate:    flatFeeRate,
		FeeBorrRate:    borrowFeeRate,
		FeeLendRate:    lendFeeRate,
		MatchedAt:      time.Now(),
		ReimburseAt:    reimbursementDate,
		Lender:         []ledger.Contract{lenderContract},
		Borrower:       []ledger.Contract{borrowerContract},
	}

	return trade
}

// GenerateTrades creates multiple trades from a match result
func (tg *TradeGenerator) GenerateTrades(matches []Match) []ledger.Trade {
	trades := make([]ledger.Trade, 0, len(matches))

	for _, match := range matches {
		trade := tg.GenerateTrade(match)
		trades = append(trades, trade)
	}

	return trades
}

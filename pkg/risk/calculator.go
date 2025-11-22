package risk

import (
	"pmeonline/pkg/ledger"
)

// Calculator handles fee and value calculations
type Calculator struct {
	ledger *ledger.LedgerPoint
}

// NewCalculator creates a new calculator instance
func NewCalculator(l *ledger.LedgerPoint) *Calculator {
	return &Calculator{ledger: l}
}

// Static fee rates from design document F.3
const (
	DefaultFlatFeeRate      = 0.0005 // 0.05%
	DefaultBorrowingFeeRate = 0.18   // 18% annual
	DefaultLendingFeeRate   = 0.15   // 15% annual
)

// GetFeeRates returns the current fee rates from parameters or defaults
func (c *Calculator) GetFeeRates() (flatFee, borrowFee, lendFee float64) {
	// Use parameter values if available, otherwise use defaults
	if c.ledger.Parameter.FlatFee > 0 {
		flatFee = c.ledger.Parameter.FlatFee
	} else {
		flatFee = DefaultFlatFeeRate
	}

	if c.ledger.Parameter.BorrowingFee > 0 {
		borrowFee = c.ledger.Parameter.BorrowingFee
	} else {
		borrowFee = DefaultBorrowingFeeRate
	}

	if c.ledger.Parameter.LendingFee > 0 {
		lendFee = c.ledger.Parameter.LendingFee
	} else {
		lendFee = DefaultLendingFeeRate
	}

	return
}

// CalculateBorrowingValue calculates the borrowing value
// Formula: BorrVal = MarketPrice × Quantity
func (c *Calculator) CalculateBorrowingValue(marketPrice, quantity float64) float64 {
	return marketPrice * quantity
}

// CalculateFlatFee calculates the one-time flat fee for borrowing
// Formula: FeeFlat = MarketPrice × Quantity × FlatFeeRate
func (c *Calculator) CalculateFlatFee(marketPrice, quantity float64) float64 {
	flatFeeRate, _, _ := c.GetFeeRates()
	return marketPrice * quantity * flatFeeRate
}

// CalculateBorrowingDailyFee calculates the daily borrowing fee
// Formula: FeeBorrDaily = MarketPrice × Quantity × BorrowingFeeRate / 365
func (c *Calculator) CalculateBorrowingDailyFee(marketPrice, quantity float64) float64 {
	_, borrowFeeRate, _ := c.GetFeeRates()
	return marketPrice * quantity * borrowFeeRate / 365.0
}

// CalculateBorrowingTotalFee calculates total fees for borrowing over a period
// Formula: TotalFee = BorrVal × FeeBorr × Period / 365 + FeeFlat
func (c *Calculator) CalculateBorrowingTotalFee(marketPrice, quantity float64, periode int) float64 {
	flatFee := c.CalculateFlatFee(marketPrice, quantity)
	dailyFee := c.CalculateBorrowingDailyFee(marketPrice, quantity)
	return flatFee + (dailyFee * float64(periode))
}

// CalculateBorrowingAccumulatedFee calculates accumulated borrowing fee
// Formula: FeeBorrAccum = FeeBorrDaily × DaysPassed
func (c *Calculator) CalculateBorrowingAccumulatedFee(marketPrice, quantity float64, daysPassed int) float64 {
	dailyFee := c.CalculateBorrowingDailyFee(marketPrice, quantity)
	return dailyFee * float64(daysPassed)
}

// CalculateLendingDailyFee calculates the daily lending revenue
// Formula: FeeLendDaily = MarketPrice × Quantity × LendingFeeRate / 365
func (c *Calculator) CalculateLendingDailyFee(marketPrice, quantity float64) float64 {
	_, _, lendFeeRate := c.GetFeeRates()
	return marketPrice * quantity * lendFeeRate / 365.0
}

// CalculateLendingTotalFee calculates total lending revenue over a period
// Formula: TotalFee = FeeLendDaily × Period
func (c *Calculator) CalculateLendingTotalFee(marketPrice, quantity float64, periode int) float64 {
	dailyFee := c.CalculateLendingDailyFee(marketPrice, quantity)
	return dailyFee * float64(periode)
}

// CalculateLendingAccumulatedFee calculates accumulated lending revenue
// Formula: FeeLendAccum = FeeLendDaily × DaysPassed
func (c *Calculator) CalculateLendingAccumulatedFee(marketPrice, quantity float64, daysPassed int) float64 {
	dailyFee := c.CalculateLendingDailyFee(marketPrice, quantity)
	return dailyFee * float64(daysPassed)
}

// FeeBreakdown contains detailed fee information
type FeeBreakdown struct {
	MarketPrice           float64
	Quantity              float64
	Periode               int
	BorrowingValue        float64
	FlatFee               float64
	BorrowingDailyFee     float64
	BorrowingTotalFee     float64
	LendingDailyFee       float64
	LendingTotalFee       float64
	FlatFeeRate           float64
	BorrowingFeeRate      float64
	LendingFeeRate        float64
	RequiredTradingLimit  float64
}

// CalculateFeeBreakdown provides a complete breakdown of all fees
func (c *Calculator) CalculateFeeBreakdown(marketPrice, quantity float64, periode int) *FeeBreakdown {
	flatFeeRate, borrowFeeRate, lendFeeRate := c.GetFeeRates()

	borrowVal := c.CalculateBorrowingValue(marketPrice, quantity)
	flatFee := c.CalculateFlatFee(marketPrice, quantity)
	borrowDailyFee := c.CalculateBorrowingDailyFee(marketPrice, quantity)
	borrowTotalFee := c.CalculateBorrowingTotalFee(marketPrice, quantity, periode)
	lendDailyFee := c.CalculateLendingDailyFee(marketPrice, quantity)
	lendTotalFee := c.CalculateLendingTotalFee(marketPrice, quantity, periode)

	return &FeeBreakdown{
		MarketPrice:           marketPrice,
		Quantity:              quantity,
		Periode:               periode,
		BorrowingValue:        borrowVal,
		FlatFee:               flatFee,
		BorrowingDailyFee:     borrowDailyFee,
		BorrowingTotalFee:     borrowTotalFee,
		LendingDailyFee:       lendDailyFee,
		LendingTotalFee:       lendTotalFee,
		FlatFeeRate:           flatFeeRate,
		BorrowingFeeRate:      borrowFeeRate,
		LendingFeeRate:        lendFeeRate,
		RequiredTradingLimit:  borrowVal + borrowTotalFee,
	}
}

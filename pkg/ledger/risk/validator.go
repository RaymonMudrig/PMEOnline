package risk

import (
	"fmt"
	"time"

	"pmeonline/pkg/ledger"
)

// Validator handles pre-trade validation
type Validator struct {
	ledger *ledger.LedgerPoint
	calc   *Calculator
}

// NewValidator creates a new validator instance
func NewValidator(l *ledger.LedgerPoint) *Validator {
	return &Validator{
		ledger: l,
		calc:   NewCalculator(l),
	}
}

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateOrder performs comprehensive pre-trade validation using OrderEntity from ledger
func (v *Validator) ValidateOrder(order ledger.OrderEntity) error {
	// 1. Basic field validation
	if err := v.validateBasicFields(order); err != nil {
		return err
	}

	// 2. Account validation
	if err := v.validateAccount(order); err != nil {
		return err
	}

	// 3. Instrument validation
	if err := v.validateInstrument(order); err != nil {
		return err
	}

	// 4. Participant validation
	if err := v.validateParticipant(order); err != nil {
		return err
	}

	// 5. Date validation (BORR orders only)
	if order.Side == "BORR" {
		if err := v.validateDates(order); err != nil {
			return err
		}
	}

	// 6. Quantity validation
	if err := v.validateQuantity(order); err != nil {
		return err
	}

	// 7. Side-specific validation
	if order.Side == "BORR" {
		if err := v.validateBorrowOrder(order); err != nil {
			return err
		}
	} else if order.Side == "LEND" {
		if err := v.validateLendOrder(order); err != nil {
			return err
		}
	} else {
		return &ValidationError{Field: "Side", Message: "must be BORR or LEND"}
	}

	return nil
}

// validateBasicFields checks required fields are present
func (v *Validator) validateBasicFields(order ledger.OrderEntity) error {
	if order.AccountCode == "" {
		return &ValidationError{Field: "AccountCode", Message: "is required"}
	}
	if order.InstrumentCode == "" {
		return &ValidationError{Field: "InstrumentCode", Message: "is required"}
	}
	if order.ParticipantCode == "" {
		return &ValidationError{Field: "ParticipantCode", Message: "is required"}
	}
	if order.Side == "" {
		return &ValidationError{Field: "Side", Message: "is required"}
	}
	if order.Quantity <= 0 {
		return &ValidationError{Field: "Quantity", Message: "must be greater than 0"}
	}

	// BORR-specific validations (LEND orders don't need periode)
	if order.Side == "BORR" {
		if order.Periode <= 0 {
			return &ValidationError{Field: "Periode", Message: "must be greater than 0"}
		}
	}

	return nil
}

// validateAccount checks if account exists and is active
func (v *Validator) validateAccount(order ledger.OrderEntity) error {
	account, exists := v.ledger.GetAccount(order.AccountCode)
	if !exists {
		return &ValidationError{
			Field:   "AccountCode",
			Message: fmt.Sprintf("account %s not found", order.AccountCode),
		}
	}

	// Verify participant matches
	if account.ParticipantCode != order.ParticipantCode {
		return &ValidationError{
			Field: "ParticipantCode",
			Message: fmt.Sprintf("account %s belongs to participant %s, not %s",
				order.AccountCode, account.ParticipantCode, order.ParticipantCode),
		}
	}

	return nil
}

// validateInstrument checks if instrument exists and is eligible
func (v *Validator) validateInstrument(order ledger.OrderEntity) error {
	instrument, exists := v.ledger.GetInstrument(order.InstrumentCode)
	if !exists {
		return &ValidationError{
			Field:   "InstrumentCode",
			Message: fmt.Sprintf("instrument %s not found", order.InstrumentCode),
		}
	}

	// Check eligibility
	if !instrument.Status {
		return &ValidationError{
			Field:   "InstrumentCode",
			Message: fmt.Sprintf("instrument %s is not eligible for SBL", order.InstrumentCode),
		}
	}

	return nil
}

// validateParticipant checks if participant exists and has eligibility
func (v *Validator) validateParticipant(order ledger.OrderEntity) error {
	participant, exists := v.ledger.GetParticipant(order.ParticipantCode)
	if !exists {
		return &ValidationError{
			Field:   "ParticipantCode",
			Message: fmt.Sprintf("participant %s not found", order.ParticipantCode),
		}
	}

	// Check side-specific eligibility
	if order.Side == "BORR" && !participant.BorrEligibility {
		return &ValidationError{
			Field:   "ParticipantCode",
			Message: fmt.Sprintf("participant %s is not eligible for borrowing", order.ParticipantCode),
		}
	}

	if order.Side == "LEND" && !participant.LendEligibility {
		return &ValidationError{
			Field:   "ParticipantCode",
			Message: fmt.Sprintf("participant %s is not eligible for lending", order.ParticipantCode),
		}
	}

	return nil
}

// validateDates checks settlement and reimbursement dates
func (v *Validator) validateDates(order ledger.OrderEntity) error {
	now := time.Now()
	serverLoc := now.Location()

	// Normalize to date-only comparison (ignore time part)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, serverLoc)

	// Convert settlement date to server timezone first, then normalize to midnight
	settlementInServerTz := order.SettlementDate.In(serverLoc)
	settlementDate := time.Date(settlementInServerTz.Year(), settlementInServerTz.Month(), settlementInServerTz.Day(), 0, 0, 0, 0, serverLoc)

	// Settlement date must be today or in the future (date part only)
	if settlementDate.Before(today) {
		return &ValidationError{
			Field:   "SettlementDate",
			Message: "must be today or in the future",
		}
	}

	// Reimbursement date must be after settlement date
	if order.ReimbursementDate.Before(order.SettlementDate) || order.ReimbursementDate.Equal(order.SettlementDate) {
		return &ValidationError{
			Field:   "ReimbursementDate",
			Message: "must be after settlement date",
		}
	}

	// Calculate expected periode
	days := int(order.ReimbursementDate.Sub(order.SettlementDate).Hours() / 24)
	if days != order.Periode {
		return &ValidationError{
			Field: "Periode",
			Message: fmt.Sprintf("does not match date range (expected %d days, got %d)",
				days, order.Periode),
		}
	}

	// Check maximum periode
	param := v.ledger.GetParameter()
	if order.Periode > param.BorrowMaxOpenDay {
		return &ValidationError{
			Field: "Periode",
			Message: fmt.Sprintf("exceeds maximum allowed periode of %d days",
				param.BorrowMaxOpenDay),
		}
	}

	return nil
}

// validateQuantity checks quantity against limits
func (v *Validator) validateQuantity(order ledger.OrderEntity) error {
	param := v.ledger.GetParameter()

	// Check minimum denomination
	if int(order.Quantity)%param.DenominationLimit != 0 {
		return &ValidationError{
			Field: "Quantity",
			Message: fmt.Sprintf("must be in multiples of %d shares",
				param.DenominationLimit),
		}
	}

	// Check maximum quantity
	if order.Quantity > param.MaxQuantity {
		return &ValidationError{
			Field: "Quantity",
			Message: fmt.Sprintf("exceeds maximum allowed quantity of %.0f shares",
				param.MaxQuantity),
		}
	}

	return nil
}

// validateBorrowOrder performs borrowing-specific validation
func (v *Validator) validateBorrowOrder(order ledger.OrderEntity) error {
	account, exists := v.ledger.GetAccount(order.AccountCode)
	if !exists {
		return &ValidationError{Field: "AccountCode", Message: "account not found"}
	}

	// Calculate borrowing value and fees
	// Formula from F.1.1:
	// BorrVal = MarketPrice × Quantity
	// TotalFee = BorrVal × FeeBorr × Period + FeeFlat
	// TradingLimit >= TotalFee + BorrVal

	borrVal := order.MarketPrice * order.Quantity
	totalFee := v.calc.CalculateBorrowingTotalFee(order.MarketPrice, order.Quantity, order.Periode)

	requiredLimit := totalFee + borrVal

	if account.TradeLimit < requiredLimit {
		return &ValidationError{
			Field: "AccountLimit",
			Message: fmt.Sprintf("insufficient trading limit: required %.2f, available %.2f",
				requiredLimit, account.TradeLimit),
		}
	}

	return nil
}

// validateLendOrder performs lending-specific validation
func (v *Validator) validateLendOrder(order ledger.OrderEntity) error {
	// According to F.1.2, no pool limit check is required for lending orders
	// Just basic validation is sufficient
	return nil
}

// IsPendingNew checks if order should be in Pending-New state
func (v *Validator) IsPendingNew(order ledger.OrderEntity) bool {
	now := time.Now()
	serverLoc := now.Location()

	// Normalize today to midnight in server timezone
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, serverLoc)

	// Convert settlement date to server timezone first, then normalize to midnight
	// This ensures we compare calendar dates in the same timezone, avoiding issues
	// where dates sent as UTC (e.g., "2025-11-26T00:00:00Z") are incorrectly
	// compared with server timezone dates
	settlementInServerTz := order.SettlementDate.In(serverLoc)
	settlementDate := time.Date(settlementInServerTz.Year(), settlementInServerTz.Month(), settlementInServerTz.Day(), 0, 0, 0, 0, serverLoc)

	// Order is pending if settlement date is in the future (not today)
	return settlementDate.After(today)
}

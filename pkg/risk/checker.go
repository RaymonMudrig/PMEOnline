package risk

import (
	"log"

	"pmeonline/pkg/ledger"
)

// Checker monitors eligibility changes for instruments and participants
type Checker struct {
	ledger                  *ledger.LedgerPoint
	onInstrumentIneligible  func(instrumentCode string)
	onInstrumentEligible    func(instrumentCode string)
	onParticipantIneligible func(participantCode string, side string)
}

// NewChecker creates a new eligibility checker
func NewChecker(l *ledger.LedgerPoint) *Checker {
	return &Checker{
		ledger: l,
	}
}

// SetInstrumentIneligibleHandler sets the handler for instrument becoming ineligible
func (c *Checker) SetInstrumentIneligibleHandler(handler func(instrumentCode string)) {
	c.onInstrumentIneligible = handler
}

// SetInstrumentEligibleHandler sets the handler for instrument becoming eligible
func (c *Checker) SetInstrumentEligibleHandler(handler func(instrumentCode string)) {
	c.onInstrumentEligible = handler
}

// SetParticipantIneligibleHandler sets the handler for participant becoming ineligible
func (c *Checker) SetParticipantIneligibleHandler(handler func(participantCode string, side string)) {
	c.onParticipantIneligible = handler
}

// CheckInstrumentEligibility checks if an instrument is eligible
func (c *Checker) CheckInstrumentEligibility(instrumentCode string) (bool, error) {
	instrument, exists := c.ledger.Instrument[instrumentCode]
	if !exists {
		return false, &ValidationError{
			Field:   "InstrumentCode",
			Message: "instrument not found",
		}
	}

	return instrument.Status, nil
}

// CheckParticipantEligibility checks if a participant is eligible for a specific side
func (c *Checker) CheckParticipantEligibility(participantCode string, side string) (bool, error) {
	participant, exists := c.ledger.Participant[participantCode]
	if !exists {
		return false, &ValidationError{
			Field:   "ParticipantCode",
			Message: "participant not found",
		}
	}

	if side == "BORR" {
		return participant.BorrEligibility, nil
	} else if side == "LEND" {
		return participant.LendEligibility, nil
	}

	return false, &ValidationError{
		Field:   "Side",
		Message: "invalid side (must be BORR or LEND)",
	}
}

// MonitorInstrument monitors a specific instrument for eligibility changes
// This should be called by the OMS when subscribing to Instrument events
func (c *Checker) MonitorInstrument(prevStatus bool, instrument ledger.Instrument) {
	// Check if status changed
	if prevStatus != instrument.Status {
		if instrument.Status {
			// Instrument became eligible
			log.Printf("✅ Instrument %s is now ELIGIBLE", instrument.Code)
			if c.onInstrumentEligible != nil {
				c.onInstrumentEligible(instrument.Code)
			}
		} else {
			// Instrument became ineligible
			log.Printf("⚠️  Instrument %s is now INELIGIBLE", instrument.Code)
			if c.onInstrumentIneligible != nil {
				c.onInstrumentIneligible(instrument.Code)
			}
		}
	}
}

// MonitorParticipant monitors a specific participant for eligibility changes
func (c *Checker) MonitorParticipant(prevBorrEligibility, prevLendEligibility bool, participant ledger.Participant) {
	// Check if borrowing eligibility changed
	if prevBorrEligibility != participant.BorrEligibility {
		if !participant.BorrEligibility {
			log.Printf("⚠️  Participant %s is no longer eligible for BORROWING", participant.Code)
			if c.onParticipantIneligible != nil {
				c.onParticipantIneligible(participant.Code, "BORR")
			}
		} else {
			log.Printf("✅ Participant %s is now eligible for BORROWING", participant.Code)
		}
	}

	// Check if lending eligibility changed
	if prevLendEligibility != participant.LendEligibility {
		if !participant.LendEligibility {
			log.Printf("⚠️  Participant %s is no longer eligible for LENDING", participant.Code)
			if c.onParticipantIneligible != nil {
				c.onParticipantIneligible(participant.Code, "LEND")
			}
		} else {
			log.Printf("✅ Participant %s is now eligible for LENDING", participant.Code)
		}
	}
}

// ShouldBlockOrder determines if an order should be blocked based on eligibility
func (c *Checker) ShouldBlockOrder(order ledger.Order) (bool, string) {
	// Check instrument eligibility
	eligible, err := c.CheckInstrumentEligibility(order.InstrumentCode)
	if err != nil || !eligible {
		return true, "Instrument is not eligible"
	}

	// Check participant eligibility
	eligible, err = c.CheckParticipantEligibility(order.ParticipantCode, order.Side)
	if err != nil || !eligible {
		if order.Side == "BORR" {
			return true, "Participant is not eligible for borrowing"
		} else {
			return true, "Participant is not eligible for lending"
		}
	}

	return false, ""
}

// GetIneligibleOrders returns all orders that are now ineligible due to instrument/participant changes
func (c *Checker) GetIneligibleOrders(orders map[int]ledger.OrderEntity) []int {
	var ineligibleNIDs []int

	for nid, order := range orders {
		// Only check Open and Partial orders
		if order.State != "O" && order.State != "P" {
			continue
		}

		blocked, _ := c.ShouldBlockOrder(ledger.Order{
			InstrumentCode:  order.InstrumentCode,
			ParticipantCode: order.ParticipantCode,
			Side:            order.Side,
		})

		if blocked {
			ineligibleNIDs = append(ineligibleNIDs, nid)
		}
	}

	return ineligibleNIDs
}

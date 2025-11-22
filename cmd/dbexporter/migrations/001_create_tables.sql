-- PME Online Database Schema
-- Creates all tables for event sourcing and entities

-- Service Start tracking
CREATE TABLE IF NOT EXISTS service_starts (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL,
    started_at BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Parameters
CREATE TABLE IF NOT EXISTS parameters (
    id SERIAL PRIMARY KEY,
    flat_fee DOUBLE PRECISION NOT NULL,
    lending_fee DOUBLE PRECISION NOT NULL,
    borrowing_fee DOUBLE PRECISION NOT NULL,
    max_quantity DOUBLE PRECISION NOT NULL,
    borrow_max_open_day INTEGER NOT NULL,
    denomination_limit INTEGER NOT NULL,
    updated_at BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Session Times
CREATE TABLE IF NOT EXISTS session_times (
    id SERIAL PRIMARY KEY,
    session1_start TIME NOT NULL,
    session1_end TIME NOT NULL,
    session2_start TIME NOT NULL,
    session2_end TIME NOT NULL,
    updated_at BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Holidays
CREATE TABLE IF NOT EXISTS holidays (
    id SERIAL PRIMARY KEY,
    nid BIGINT UNIQUE NOT NULL,
    year INTEGER NOT NULL,
    date DATE NOT NULL,
    description TEXT,
    last_update BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_holidays_date ON holidays(date);
CREATE INDEX IF NOT EXISTS idx_holidays_year ON holidays(year);

-- Participants
CREATE TABLE IF NOT EXISTS participants (
    id SERIAL PRIMARY KEY,
    nid BIGINT UNIQUE NOT NULL,
    code VARCHAR(10) UNIQUE NOT NULL,
    name VARCHAR(200) NOT NULL,
    borr_eligibility BOOLEAN NOT NULL DEFAULT true,
    lend_eligibility BOOLEAN NOT NULL DEFAULT true,
    last_update BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_participants_code ON participants(code);

-- Instruments
CREATE TABLE IF NOT EXISTS instruments (
    id SERIAL PRIMARY KEY,
    nid BIGINT UNIQUE NOT NULL,
    code VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(200) NOT NULL,
    type VARCHAR(50),
    status BOOLEAN NOT NULL DEFAULT true,
    last_update BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_instruments_code ON instruments(code);
CREATE INDEX IF NOT EXISTS idx_instruments_status ON instruments(status);

-- Accounts
CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,
    nid BIGINT UNIQUE NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    sid VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(200) NOT NULL,
    address TEXT,
    participant_code VARCHAR(10) NOT NULL,
    trade_limit DOUBLE PRECISION NOT NULL DEFAULT 0,
    pool_limit DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_update BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (participant_code) REFERENCES participants(code)
);

CREATE INDEX IF NOT EXISTS idx_accounts_code ON accounts(code);
CREATE INDEX IF NOT EXISTS idx_accounts_sid ON accounts(sid);
CREATE INDEX IF NOT EXISTS idx_accounts_participant ON accounts(participant_code);

-- Orders
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    nid BIGINT UNIQUE NOT NULL,
    prev_nid BIGINT,
    reff_request_id VARCHAR(100),
    account_code VARCHAR(50) NOT NULL,
    participant_code VARCHAR(10) NOT NULL,
    instrument_code VARCHAR(20) NOT NULL,
    side VARCHAR(10) NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    done_quantity DOUBLE PRECISION NOT NULL DEFAULT 0,
    settlement_date TIMESTAMP NOT NULL,
    reimbursement_date TIMESTAMP NOT NULL,
    periode INTEGER NOT NULL,
    state VARCHAR(10) NOT NULL,
    market_price DOUBLE PRECISION NOT NULL,
    rate DOUBLE PRECISION NOT NULL,
    instruction TEXT,
    aro BOOLEAN NOT NULL DEFAULT false,
    entry_at TIMESTAMP NOT NULL,
    amend_at TIMESTAMP,
    withdraw_at TIMESTAMP,
    last_update BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_code) REFERENCES accounts(code),
    FOREIGN KEY (participant_code) REFERENCES participants(code),
    FOREIGN KEY (instrument_code) REFERENCES instruments(code)
);

CREATE INDEX IF NOT EXISTS idx_orders_nid ON orders(nid);
CREATE INDEX IF NOT EXISTS idx_orders_account ON orders(account_code);
CREATE INDEX IF NOT EXISTS idx_orders_participant ON orders(participant_code);
CREATE INDEX IF NOT EXISTS idx_orders_instrument ON orders(instrument_code);
CREATE INDEX IF NOT EXISTS idx_orders_side ON orders(side);
CREATE INDEX IF NOT EXISTS idx_orders_state ON orders(state);
CREATE INDEX IF NOT EXISTS idx_orders_entry_at ON orders(entry_at);

-- Trades
CREATE TABLE IF NOT EXISTS trades (
    id SERIAL PRIMARY KEY,
    nid BIGINT UNIQUE NOT NULL,
    kpei_reff VARCHAR(100) UNIQUE NOT NULL,
    instrument_code VARCHAR(20) NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    periode INTEGER NOT NULL,
    state VARCHAR(10) NOT NULL,
    fee_flat_rate DOUBLE PRECISION NOT NULL,
    fee_borr_rate DOUBLE PRECISION NOT NULL,
    fee_lend_rate DOUBLE PRECISION NOT NULL,
    matched_at TIMESTAMP NOT NULL,
    reimburse_at TIMESTAMP NOT NULL,
    last_update BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (instrument_code) REFERENCES instruments(code)
);

CREATE INDEX IF NOT EXISTS idx_trades_nid ON trades(nid);
CREATE INDEX IF NOT EXISTS idx_trades_kpei_reff ON trades(kpei_reff);
CREATE INDEX IF NOT EXISTS idx_trades_instrument ON trades(instrument_code);
CREATE INDEX IF NOT EXISTS idx_trades_state ON trades(state);
CREATE INDEX IF NOT EXISTS idx_trades_matched_at ON trades(matched_at);

-- Contracts
CREATE TABLE IF NOT EXISTS contracts (
    id SERIAL PRIMARY KEY,
    nid BIGINT UNIQUE NOT NULL,
    trade_nid BIGINT NOT NULL,
    kpei_reff VARCHAR(100) NOT NULL,
    side VARCHAR(10) NOT NULL,
    account_code VARCHAR(50) NOT NULL,
    account_sid VARCHAR(50) NOT NULL,
    account_participant_code VARCHAR(10) NOT NULL,
    order_nid BIGINT NOT NULL,
    instrument_code VARCHAR(20) NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    periode INTEGER NOT NULL,
    state VARCHAR(10) NOT NULL,
    fee_flat_val DOUBLE PRECISION NOT NULL DEFAULT 0,
    fee_val_daily DOUBLE PRECISION NOT NULL DEFAULT 0,
    fee_val_accumulated DOUBLE PRECISION NOT NULL DEFAULT 0,
    matched_at TIMESTAMP NOT NULL,
    reimburse_at TIMESTAMP NOT NULL,
    last_update BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (trade_nid) REFERENCES trades(nid),
    FOREIGN KEY (account_code) REFERENCES accounts(code),
    FOREIGN KEY (account_participant_code) REFERENCES participants(code),
    FOREIGN KEY (order_nid) REFERENCES orders(nid),
    FOREIGN KEY (instrument_code) REFERENCES instruments(code)
);

CREATE INDEX IF NOT EXISTS idx_contracts_nid ON contracts(nid);
CREATE INDEX IF NOT EXISTS idx_contracts_trade_nid ON contracts(trade_nid);
CREATE INDEX IF NOT EXISTS idx_contracts_account ON contracts(account_code);
CREATE INDEX IF NOT EXISTS idx_contracts_participant ON contracts(account_participant_code);
CREATE INDEX IF NOT EXISTS idx_contracts_order ON contracts(order_nid);
CREATE INDEX IF NOT EXISTS idx_contracts_instrument ON contracts(instrument_code);
CREATE INDEX IF NOT EXISTS idx_contracts_state ON contracts(state);

-- Event Log (for audit trail)
CREATE TABLE IF NOT EXISTS event_log (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    timestamp BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_event_log_type ON event_log(event_type);
CREATE INDEX IF NOT EXISTS idx_event_log_timestamp ON event_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_event_log_created_at ON event_log(created_at);

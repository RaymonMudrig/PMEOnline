const API_BASE = '';
const ECLEAR_API = 'http://localhost:8081';

// Global data storage
let allParticipants = [];
let allInstruments = [];
let allAccounts = [];

// Sorting state
let ordersSortColumn = 'nid';
let ordersSortDirection = 'desc';
let contractsSortColumn = 'nid';
let contractsSortDirection = 'desc';

function showTab(tabName) {
    const tabs = document.querySelectorAll('.tab-content');
    tabs.forEach(tab => tab.classList.remove('active'));

    const tabButtons = document.querySelectorAll('.tab');
    tabButtons.forEach(btn => btn.classList.remove('active'));

    document.getElementById(tabName).classList.add('active');
    event.target.classList.add('active');
}

function showStatus(elementId, message, type) {
    const statusEl = document.getElementById(elementId);
    statusEl.textContent = message;
    statusEl.className = 'status ' + type;
    statusEl.style.display = 'block';
}

function hideStatus(elementId) {
    document.getElementById(elementId).style.display = 'none';
}

// Generate auto reference ID
function generateReffId(side) {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    const hour = String(now.getHours()).padStart(2, '0');
    const minute = String(now.getMinutes()).padStart(2, '0');
    const second = String(now.getSeconds()).padStart(2, '0');

    return `${side}${year}${month}${day}${hour}${minute}${second}`;
}

// Load master data from eClear API
async function loadParticipants() {
    try {
        const response = await fetch(ECLEAR_API + '/participant/list');
        const result = await response.json();

        if (result.status === 'success') {
            allParticipants = result.data.participants || [];

            // Populate borrow participant dropdown
            const borrSelect = document.getElementById('borr-participant');
            borrSelect.innerHTML = '<option value="">-- Select Participant --</option>';
            allParticipants.forEach(p => {
                borrSelect.innerHTML += `<option value="${p.code}">${p.code} - ${p.name}</option>`;
            });

            // Populate lend participant dropdown
            const lendSelect = document.getElementById('lend-participant');
            lendSelect.innerHTML = '<option value="">-- Select Participant --</option>';
            allParticipants.forEach(p => {
                lendSelect.innerHTML += `<option value="${p.code}">${p.code} - ${p.name}</option>`;
            });
        }
    } catch (error) {
        console.error('Failed to load participants:', error);
    }
}

async function loadInstruments() {
    try {
        const response = await fetch(ECLEAR_API + '/instrument/list');
        const result = await response.json();

        if (result.status === 'success') {
            allInstruments = result.data.instruments || [];

            // Populate borrow instrument dropdown
            const borrSelect = document.getElementById('borr-instrument');
            borrSelect.innerHTML = '<option value="">-- Select Instrument --</option>';
            allInstruments.filter(i => i.status).forEach(i => {
                borrSelect.innerHTML += `<option value="${i.code}">${i.code} - ${i.name}</option>`;
            });

            // Populate lend instrument dropdown
            const lendSelect = document.getElementById('lend-instrument');
            lendSelect.innerHTML = '<option value="">-- Select Instrument --</option>';
            allInstruments.filter(i => i.status).forEach(i => {
                lendSelect.innerHTML += `<option value="${i.code}">${i.code} - ${i.name}</option>`;
            });
        }
    } catch (error) {
        console.error('Failed to load instruments:', error);
    }
}

async function loadAccounts() {
    try {
        const response = await fetch(ECLEAR_API + '/account/list');
        const result = await response.json();

        if (result.status === 'success') {
            allAccounts = result.data.accounts || [];
        }
    } catch (error) {
        console.error('Failed to load accounts:', error);
    }
}

// Filter accounts by participant
function loadAccountsByParticipant(side) {
    const participantCode = document.getElementById(side + '-participant').value;
    const accountSelect = document.getElementById(side + '-account');

    accountSelect.innerHTML = '<option value="">-- Select Account --</option>';

    if (!participantCode) return;

    const filteredAccounts = allAccounts.filter(a => a.participant_code === participantCode);
    filteredAccounts.forEach(a => {
        accountSelect.innerHTML += `<option value="${a.code}">${a.code} - ${a.name}</option>`;
    });
}

// Calculate periode from dates
function calculatePeriodeFromDates(side) {
    const settlement = document.getElementById(side + '-settlement').value;
    const reimbursement = document.getElementById(side + '-reimbursement').value;

    if (!settlement || !reimbursement) return;

    const settleDate = new Date(settlement);
    const reimburseDate = new Date(reimbursement);
    const diffTime = reimburseDate - settleDate;
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

    if (diffDays > 0) {
        document.getElementById(side + '-periode').value = diffDays;
    }
}

// Calculate reimbursement date from periode
function calculateReimbursementFromPeriode(side) {
    const settlement = document.getElementById(side + '-settlement').value;
    const periode = document.getElementById(side + '-periode').value;

    if (!settlement || !periode) return;

    const settleDate = new Date(settlement);
    settleDate.setDate(settleDate.getDate() + parseInt(periode));

    const year = settleDate.getFullYear();
    const month = String(settleDate.getMonth() + 1).padStart(2, '0');
    const day = String(settleDate.getDate()).padStart(2, '0');

    document.getElementById(side + '-reimbursement').value = `${year}-${month}-${day}`;
}

async function submitBorrowOrder(event) {
    event.preventDefault();
    showStatus('borrow-status', '⏳ Submitting borrow order...', 'loading');

    const data = {
        reff_request_id: generateReffId('BORR'),
        account_code: document.getElementById('borr-account').value,
        participant_code: document.getElementById('borr-participant').value,
        instrument_code: document.getElementById('borr-instrument').value,
        side: 'BORR',
        quantity: parseFloat(document.getElementById('borr-quantity').value),
        settlement_date: document.getElementById('borr-settlement').value + 'T00:00:00Z',
        reimbursement_date: document.getElementById('borr-reimbursement').value + 'T00:00:00Z',
        periode: parseInt(document.getElementById('borr-periode').value),
        market_price: 0,
        rate: 0.18,
        instruction: document.getElementById('borr-instruction').value || '',
        aro: document.getElementById('borr-aro').value === 'true'
    };

    try {
        const response = await fetch(API_BASE + '/api/order/new', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        const result = await response.json();

        if (result.status === 'success') {
            showStatus('borrow-status', '✅ Borrow order submitted! Order NID: ' + result.data.order_nid, 'success');
            setTimeout(() => {
                resetBorrowForm();
                closeModal('borrow');
            }, 2000);
        } else {
            throw new Error(result.message || 'Unknown error');
        }
    } catch (error) {
        showStatus('borrow-status', '❌ Error: ' + error.message, 'error');
    }
}

async function submitLendOrder(event) {
    event.preventDefault();
    showStatus('lend-status', '⏳ Submitting lend order...', 'loading');

    const data = {
        reff_request_id: generateReffId('LEND'),
        account_code: document.getElementById('lend-account').value,
        participant_code: document.getElementById('lend-participant').value,
        instrument_code: document.getElementById('lend-instrument').value,
        side: 'LEND',
        quantity: parseFloat(document.getElementById('lend-quantity').value),
        settlement_date: '1970-01-01T00:00:00Z',
        reimbursement_date: '1970-01-01T00:00:00Z',
        periode: 0,
        market_price: 0,
        rate: 0.15,
        instruction: '',
        aro: false
    };

    try {
        const response = await fetch(API_BASE + '/api/order/new', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        const result = await response.json();

        if (result.status === 'success') {
            showStatus('lend-status', '✅ Lend order submitted! Order NID: ' + result.data.order_nid, 'success');
            setTimeout(() => {
                resetLendForm();
                closeModal('lend');
            }, 2000);
        } else {
            throw new Error(result.message || 'Unknown error');
        }
    } catch (error) {
        showStatus('lend-status', '❌ Error: ' + error.message, 'error');
    }
}

function resetBorrowForm() {
    document.getElementById('borrow-form').reset();
    document.getElementById('borr-rate').value = '18%';
    document.getElementById('borr-account').innerHTML = '<option value="">-- Select Account --</option>';
    setDefaultBorrowDates();
}

function resetLendForm() {
    document.getElementById('lend-form').reset();
    document.getElementById('lend-rate').value = '15%';
    document.getElementById('lend-account').innerHTML = '<option value="">-- Select Account --</option>';
}

function setDefaultBorrowDates() {
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    const settlement = tomorrow.toISOString().split('T')[0];

    const reimburse = new Date(tomorrow);
    reimburse.setDate(reimburse.getDate() + 30);
    const reimbursement = reimburse.toISOString().split('T')[0];

    document.getElementById('borr-settlement').value = settlement;
    document.getElementById('borr-reimbursement').value = reimbursement;
    document.getElementById('borr-periode').value = '30';
}

async function loadOrders() {
    showStatus('orders-status', '⏳ Loading orders...', 'loading');

    const params = new URLSearchParams();
    const participant = document.getElementById('order-participant').value;
    const sid = document.getElementById('order-sid').value;
    const state = document.getElementById('order-state').value;

    if (participant) params.append('participant', participant);
    if (sid) params.append('sid', sid);
    if (state) params.append('state', state);

    try {
        const response = await fetch(API_BASE + '/api/order/list?' + params.toString());
        const text = await response.text();

        // HACK: Replace large integer NIDs with strings before parsing
        // This preserves precision by preventing JSON.parse from converting to Number
        const fixedText = text.replace(/"nid":(\d{15,})/g, '"nid":"$1"');

        const result = JSON.parse(fixedText);

        if (result.status === 'success') {
            const orders = result.data.orders || [];
            displayOrders(orders);
            showStatus('orders-status', '✅ Loaded ' + orders.length + ' orders', 'success');
        } else {
            throw new Error(result.message || 'Unknown error');
        }
    } catch (error) {
        showStatus('orders-status', '❌ Error: ' + error.message, 'error');
        document.getElementById('orders-table').innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><h3>Failed to load orders</h3></div>';
    }
}

function sortOrders(orders, column) {
    // Toggle direction if clicking the same column
    if (ordersSortColumn === column) {
        ordersSortDirection = ordersSortDirection === 'asc' ? 'desc' : 'asc';
    } else {
        ordersSortColumn = column;
        ordersSortDirection = 'asc';
    }

    const sorted = [...orders].sort((a, b) => {
        let aVal = a[column];
        let bVal = b[column];

        // Handle numeric values
        if (typeof aVal === 'number' && typeof bVal === 'number') {
            return ordersSortDirection === 'asc' ? aVal - bVal : bVal - aVal;
        }

        // Handle string values
        aVal = String(aVal || '').toLowerCase();
        bVal = String(bVal || '').toLowerCase();

        if (ordersSortDirection === 'asc') {
            return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
        } else {
            return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
        }
    });

    displayOrders(sorted);
}

function displayOrders(orders) {
    const tableDiv = document.getElementById('orders-table');

    if (orders.length === 0) {
        tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📭</div><h3>No orders found</h3><p>Submit orders using the Entry tabs</p></div>';
        return;
    }

    const getSortIcon = (column) => {
        if (ordersSortColumn !== column) return ' <span class="sort-icon">⇅</span>';
        return ordersSortDirection === 'asc' ? ' <span class="sort-icon active">▲</span>' : ' <span class="sort-icon active">▼</span>';
    };

    let html = '<table><thead><tr>';
    html += '<th class="sortable" onclick="sortOrders(window.currentOrders, \'nid\')">NID' + getSortIcon('nid') + '</th>';
    html += '<th class="sortable" onclick="sortOrders(window.currentOrders, \'side\')">Side' + getSortIcon('side') + '</th>';
    html += '<th class="sortable" onclick="sortOrders(window.currentOrders, \'participant_code\')">Participant' + getSortIcon('participant_code') + '</th>';
    html += '<th class="sortable" onclick="sortOrders(window.currentOrders, \'account_code\')">Account' + getSortIcon('account_code') + '</th>';
    html += '<th class="sortable" onclick="sortOrders(window.currentOrders, \'instrument_code\')">Instrument' + getSortIcon('instrument_code') + '</th>';
    html += '<th class="sortable" onclick="sortOrders(window.currentOrders, \'quantity\')">Quantity' + getSortIcon('quantity') + '</th>';
    html += '<th class="sortable" onclick="sortOrders(window.currentOrders, \'done_quantity\')">Done' + getSortIcon('done_quantity') + '</th>';
    html += '<th class="sortable" onclick="sortOrders(window.currentOrders, \'state\')">State' + getSortIcon('state') + '</th>';
    html += '<th class="sortable" onclick="sortOrders(window.currentOrders, \'entry_at\')">Entry At' + getSortIcon('entry_at') + '</th>';
    html += '<th>Actions</th>';
    html += '</tr></thead><tbody>';

    // Store current orders globally for sorting
    window.currentOrders = orders;

    orders.forEach(o => {
        html += '<tr>';
        html += '<td class="has-tooltip"><strong>' + o.nid + '</strong>';
        // Add tooltip with hidden columns
        html += '<div class="row-tooltip">';
        html += '<div class="tooltip-row"><span class="tooltip-label">Reff Request ID:</span><span class="tooltip-value">' + o.reff_request_id + '</span></div>';
        html += '<div class="tooltip-row"><span class="tooltip-label">Settlement Date:</span><span class="tooltip-value">' + o.settlement_date + '</span></div>';
        html += '<div class="tooltip-row"><span class="tooltip-label">Reimbursement Date:</span><span class="tooltip-value">' + o.reimbursement_date + '</span></div>';
        html += '<div class="tooltip-row"><span class="tooltip-label">Periode:</span><span class="tooltip-value">' + o.periode + ' days</span></div>';
        html += '<div class="tooltip-row"><span class="tooltip-label">Rate:</span><span class="tooltip-value">' + o.rate.toFixed(2) + '%</span></div>';
        html += '<div class="tooltip-row"><span class="tooltip-label">ARO:</span><span class="tooltip-value">' + (o.aro ? 'Yes' : 'No') + '</span></div>';
        html += '<div class="tooltip-row"><span class="tooltip-label">Market Price:</span><span class="tooltip-value">' + o.market_price.toLocaleString() + '</span></div>';
        html += '<div class="tooltip-row"><span class="tooltip-label">Message:</span><span class="tooltip-value">' + (o.message || '-') + '</span></div>';
        html += '</div>';
        html += '</td>';
        html += '<td><span class="badge ' + (o.side === 'BORR' ? 'badge-danger' : 'badge-success') + '">' + o.side + '</span></td>';
        html += '<td>' + o.participant_code + '</td>';
        html += '<td>' + o.account_code + '</td>';
        html += '<td>' + o.instrument_code + '</td>';
        html += '<td>' + o.quantity.toLocaleString() + '</td>';
        html += '<td>' + o.done_quantity.toLocaleString() + '</td>';
        html += '<td><span class="badge badge-' + getStateBadge(o.state) + '">' + getStateLabel(o.state) + '</span></td>';
        html += '<td>' + o.entry_at + '</td>';

        // Action buttons (only show for Open or Partial orders)
        html += '<td class="action-cell">';
        if (o.state === 'O' || o.state === 'P') {
            // Store order index for amendment
            const orderIndex = orders.indexOf(o);
            html += '<button class="btn-small btn-amend-small" onclick="openAmendModalByIndex(' + orderIndex + ')">✏️</button> ';
            html += '<button class="btn-small btn-withdraw-small" onclick="openWithdrawModal(\'' + o.nid + '\')">🗑️</button>';
        } else {
            html += '<span style="color: #adb5bd;">-</span>';
        }
        html += '</td>';

        html += '</tr>';
    });

    html += '</tbody></table>';
    tableDiv.innerHTML = html;

    // Add event listeners for tooltip positioning
    addTooltipPositioning();
}

function addTooltipPositioning() {
    const rows = document.querySelectorAll('#orders-table tbody tr');

    rows.forEach(row => {
        row.addEventListener('mouseenter', function(e) {
            const tooltip = this.querySelector('.row-tooltip');
            if (tooltip) {
                const cell = this.querySelector('td.has-tooltip');
                const rect = cell.getBoundingClientRect();

                // Position tooltip below the row, aligned to the left of the cell
                tooltip.style.left = rect.left + 'px';
                tooltip.style.top = (rect.bottom + 8) + 'px';
            }
        });
    });
}

async function loadContracts() {
    showStatus('contracts-status', '⏳ Loading contracts...', 'loading');

    const params = new URLSearchParams();
    const participant = document.getElementById('contract-participant').value;
    const sid = document.getElementById('contract-sid').value;
    const state = document.getElementById('contract-state').value;

    if (participant) params.append('participant', participant);
    if (sid) params.append('sid', sid);
    if (state) params.append('state', state);

    try {
        const response = await fetch(API_BASE + '/api/contract/list?' + params.toString());
        const result = await response.json();

        if (result.status === 'success') {
            const contracts = result.data.contracts || [];
            displayContracts(contracts);
            showStatus('contracts-status', '✅ Loaded ' + contracts.length + ' contracts', 'success');
        } else {
            throw new Error(result.message || 'Unknown error');
        }
    } catch (error) {
        showStatus('contracts-status', '❌ Error: ' + error.message, 'error');
        document.getElementById('contracts-table').innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><h3>Failed to load contracts</h3></div>';
    }
}

function sortContracts(contracts, column) {
    // Toggle direction if clicking the same column
    if (contractsSortColumn === column) {
        contractsSortDirection = contractsSortDirection === 'asc' ? 'desc' : 'asc';
    } else {
        contractsSortColumn = column;
        contractsSortDirection = 'asc';
    }

    const sorted = [...contracts].sort((a, b) => {
        let aVal = a[column];
        let bVal = b[column];

        // Handle numeric values
        if (typeof aVal === 'number' && typeof bVal === 'number') {
            return contractsSortDirection === 'asc' ? aVal - bVal : bVal - aVal;
        }

        // Handle string values
        aVal = String(aVal || '').toLowerCase();
        bVal = String(bVal || '').toLowerCase();

        if (contractsSortDirection === 'asc') {
            return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
        } else {
            return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
        }
    });

    displayContracts(sorted);
}

function displayContracts(contracts) {
    const tableDiv = document.getElementById('contracts-table');

    if (contracts.length === 0) {
        tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📭</div><h3>No contracts found</h3><p>Contracts appear when orders are matched</p></div>';
        return;
    }

    const getSortIcon = (column) => {
        if (contractsSortColumn !== column) return ' <span class="sort-icon">⇅</span>';
        return contractsSortDirection === 'asc' ? ' <span class="sort-icon active">▲</span>' : ' <span class="sort-icon active">▼</span>';
    };

    let html = '<table><thead><tr>';
    html += '<th class="sortable" onclick="sortContracts(window.currentContracts, \'nid\')">NID' + getSortIcon('nid') + '</th>';
    html += '<th class="sortable" onclick="sortContracts(window.currentContracts, \'kpei_reff\')">KPEI Ref' + getSortIcon('kpei_reff') + '</th>';
    html += '<th class="sortable" onclick="sortContracts(window.currentContracts, \'side\')">Side' + getSortIcon('side') + '</th>';
    html += '<th class="sortable" onclick="sortContracts(window.currentContracts, \'instrument_code\')">Instrument' + getSortIcon('instrument_code') + '</th>';
    html += '<th class="sortable" onclick="sortContracts(window.currentContracts, \'quantity\')">Quantity' + getSortIcon('quantity') + '</th>';
    html += '<th class="sortable" onclick="sortContracts(window.currentContracts, \'periode\')">Periode' + getSortIcon('periode') + '</th>';
    html += '<th class="sortable" onclick="sortContracts(window.currentContracts, \'fee_flat_val\')">Fee Flat' + getSortIcon('fee_flat_val') + '</th>';
    html += '<th class="sortable" onclick="sortContracts(window.currentContracts, \'fee_val_daily\')">Fee Daily' + getSortIcon('fee_val_daily') + '</th>';
    html += '<th class="sortable" onclick="sortContracts(window.currentContracts, \'fee_val_accumulated\')">Fee Accum' + getSortIcon('fee_val_accumulated') + '</th>';
    html += '<th class="sortable" onclick="sortContracts(window.currentContracts, \'state\')">State' + getSortIcon('state') + '</th>';
    html += '<th class="sortable" onclick="sortContracts(window.currentContracts, \'matched_at\')">Matched At' + getSortIcon('matched_at') + '</th>';
    html += '</tr></thead><tbody>';

    // Store current contracts globally for sorting
    window.currentContracts = contracts;

    contracts.forEach(c => {
        html += '<tr>';
        html += '<td><strong>' + c.nid + '</strong></td>';
        html += '<td>' + (c.kpei_reff || '-') + '</td>';
        html += '<td><span class="badge ' + (c.side === 'BORR' ? 'badge-danger' : 'badge-success') + '">' + c.side + '</span></td>';
        html += '<td>' + c.instrument_code + '</td>';
        html += '<td>' + c.quantity.toLocaleString() + '</td>';
        html += '<td>' + c.periode + ' days</td>';
        html += '<td>' + c.fee_flat_val.toLocaleString(undefined, {minimumFractionDigits: 2}) + '</td>';
        html += '<td>' + c.fee_val_daily.toLocaleString(undefined, {minimumFractionDigits: 2}) + '</td>';
        html += '<td>' + c.fee_val_accumulated.toLocaleString(undefined, {minimumFractionDigits: 2}) + '</td>';
        html += '<td><span class="badge badge-' + getStateBadge(c.state) + '">' + getStateLabel(c.state) + '</span></td>';
        html += '<td>' + c.matched_at + '</td>';
        html += '</tr>';
    });

    html += '</tbody></table>';
    tableDiv.innerHTML = html;
}

async function loadSBLDetail() {
    showStatus('sbl-detail-status', '⏳ Loading SBL detail...', 'loading');

    const params = new URLSearchParams();
    const participant = document.getElementById('sbl-participant').value;
    const instrument = document.getElementById('sbl-instrument').value;
    const side = document.getElementById('sbl-side').value;
    const aro = document.getElementById('sbl-aro').value;

    if (participant) params.append('participant', participant);
    if (instrument) params.append('instrument', instrument);
    if (side) params.append('side', side);
    if (aro) params.append('aro', aro);

    try {
        const response = await fetch(API_BASE + '/api/sbl/detail?' + params.toString());
        const result = await response.json();

        if (result.status === 'success') {
            const orders = result.data.orders || [];
            displaySBLDetail(orders);
            showStatus('sbl-detail-status', '✅ Loaded ' + orders.length + ' SBL orders', 'success');
        } else {
            throw new Error(result.message || 'Unknown error');
        }
    } catch (error) {
        showStatus('sbl-detail-status', '❌ Error: ' + error.message, 'error');
        document.getElementById('sbl-detail-table').innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><h3>Failed to load SBL detail</h3></div>';
    }
}

function displaySBLDetail(orders) {
    const tableDiv = document.getElementById('sbl-detail-table');

    if (orders.length === 0) {
        tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📭</div><h3>No SBL orders found</h3><p>Only Open and Partial orders appear in SBL</p></div>';
        return;
    }

    let html = '<table><thead><tr>';
    html += '<th>Participant</th><th>Instrument</th><th>Side</th><th>Quantity</th><th>Done</th><th>Remaining</th>';
    html += '<th>Rate</th><th>Periode</th><th>Settlement</th><th>ARO</th><th>State</th>';
    html += '</tr></thead><tbody>';

    orders.forEach(o => {
        html += '<tr>';
        html += '<td>' + o.participant_code + '</td>';
        html += '<td><strong>' + o.instrument_code + '</strong></td>';
        html += '<td><span class="badge ' + (o.side === 'BORR' ? 'badge-danger' : 'badge-success') + '">' + o.side + '</span></td>';
        html += '<td>' + o.quantity.toLocaleString() + '</td>';
        html += '<td>' + o.done_quantity.toLocaleString() + '</td>';
        html += '<td><strong>' + o.remaining_quantity.toLocaleString() + '</strong></td>';
        html += '<td>' + o.rate.toFixed(2) + '%</td>';
        html += '<td>' + o.periode + ' days</td>';
        html += '<td>' + o.settlement_date + '</td>';
        html += '<td>' + (o.aro ? '✓ ARO' : '') + '</td>';
        html += '<td><span class="badge badge-' + getStateBadge(o.state) + '">' + getStateLabel(o.state) + '</span></td>';
        html += '</tr>';
    });

    html += '</tbody></table>';
    tableDiv.innerHTML = html;
}

async function loadSBLAggregate() {
    showStatus('sbl-aggregate-status', '⏳ Loading SBL aggregate...', 'loading');

    const params = new URLSearchParams();
    const instrument = document.getElementById('agg-instrument').value;
    const side = document.getElementById('agg-side').value;

    if (instrument) params.append('instrument', instrument);
    if (side) params.append('side', side);

    try {
        const response = await fetch(API_BASE + '/api/sbl/aggregate?' + params.toString());
        const result = await response.json();

        if (result.status === 'success') {
            const aggregates = result.data.aggregates || [];
            displaySBLAggregate(aggregates);
            showStatus('sbl-aggregate-status', '✅ Loaded ' + aggregates.length + ' instruments', 'success');
        } else {
            throw new Error(result.message || 'Unknown error');
        }
    } catch (error) {
        showStatus('sbl-aggregate-status', '❌ Error: ' + error.message, 'error');
        document.getElementById('sbl-aggregate-table').innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><h3>Failed to load SBL aggregate</h3></div>';
    }
}

function displaySBLAggregate(aggregates) {
    const tableDiv = document.getElementById('sbl-aggregate-table');

    if (aggregates.length === 0) {
        tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📭</div><h3>No aggregate data found</h3></div>';
        return;
    }

    let html = '<table><thead><tr>';
    html += '<th>Instrument Code</th><th>Instrument Name</th><th>Borrow Quantity</th>';
    html += '<th>Lend Quantity</th><th>Net Quantity</th><th>Net Side</th>';
    html += '</tr></thead><tbody>';

    aggregates.forEach(a => {
        html += '<tr>';
        html += '<td><strong>' + a.instrument_code + '</strong></td>';
        html += '<td>' + a.instrument_name + '</td>';
        html += '<td>' + a.borrow_quantity.toLocaleString() + '</td>';
        html += '<td>' + a.lend_quantity.toLocaleString() + '</td>';
        html += '<td><strong>' + a.net_quantity.toLocaleString() + '</strong></td>';
        html += '<td><span class="badge ' + (a.net_side === 'BORR' ? 'badge-danger' : 'badge-success') + '">' + a.net_side + '</span></td>';
        html += '</tr>';
    });

    html += '</tbody></table>';
    tableDiv.innerHTML = html;
}

function getStateBadge(state) {
    const badges = {
        'S': 'warning', 'O': 'info', 'P': 'primary', 'M': 'success',
        'W': 'danger', 'R': 'danger', 'E': 'warning', 'C': 'success', 'T': 'danger'
    };
    return badges[state] || 'info';
}

function getStateLabel(state) {
    const labels = {
        'S': 'Submitted', 'O': 'Open', 'P': 'Partial', 'M': 'Matched',
        'W': 'Withdrawn', 'R': 'Rejected', 'E': 'Approval', 'C': 'Closed', 'T': 'Terminated'
    };
    return labels[state] || state;
}

function clearOrderFilters() {
    document.getElementById('order-participant').value = '';
    document.getElementById('order-sid').value = '';
    document.getElementById('order-state').value = '';
}

function clearContractFilters() {
    document.getElementById('contract-participant').value = '';
    document.getElementById('contract-sid').value = '';
    document.getElementById('contract-state').value = '';
}

function clearSBLFilters() {
    document.getElementById('sbl-participant').value = '';
    document.getElementById('sbl-instrument').value = '';
    document.getElementById('sbl-side').value = '';
    document.getElementById('sbl-aro').value = '';
}

function clearAggFilters() {
    document.getElementById('agg-instrument').value = '';
    document.getElementById('agg-side').value = '';
}

// Modal functions
function openModal(modalName) {
    const modal = document.getElementById(modalName + '-modal');
    if (modal) {
        modal.classList.add('active');
        document.body.style.overflow = 'hidden';
    }
}

function closeModal(modalName) {
    const modal = document.getElementById(modalName + '-modal');
    if (modal) {
        modal.classList.remove('active');
        document.body.style.overflow = '';

        // Hide status messages when closing
        const statusId = modalName + '-status';
        const statusEl = document.getElementById(statusId);
        if (statusEl) {
            statusEl.style.display = 'none';
        }
    }
}

// Close modal on Escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        const activeModal = document.querySelector('.modal.active');
        if (activeModal) {
            const modalName = activeModal.id.replace('-modal', '');
            closeModal(modalName);
        }
    }
});

// Amend Order
async function submitAmendOrder(event) {
    event.preventDefault();
    showStatus('amend-status', '⏳ Submitting amendment...', 'loading');

    const orderNIDStr = document.getElementById('amend-order-nid').value.trim();

    // Validate order NID
    if (!orderNIDStr || orderNIDStr === '' || orderNIDStr === '0') {
        showStatus('amend-status', '❌ Error: Invalid Order NID', 'error');
        return;
    }

    console.log('[AMEND] Order NID string from input:', orderNIDStr);

    // Try to preserve precision by using the string directly in JSON
    // We'll manually build the JSON to avoid Number precision loss
    const orderNID = orderNIDStr;  // Keep as string for now

    const quantity = document.getElementById('amend-quantity').value;
    const settlement = document.getElementById('amend-settlement').value;
    const reimbursement = document.getElementById('amend-reimbursement').value;
    const periode = document.getElementById('amend-periode').value;
    const aro = document.getElementById('amend-aro').value;
    const instruction = document.getElementById('amend-instruction').value;

    // Build JSON manually to preserve large integer precision
    let jsonBody = `{"order_nid":${orderNID},"reff_request_id":"${generateReffId('AMND')}"`;

    if (quantity) {
        jsonBody += `,"quantity":${parseFloat(quantity)}`;
    }
    if (settlement) {
        jsonBody += `,"settlement_date":"${settlement}T00:00:00Z"`;
    }
    if (reimbursement) {
        jsonBody += `,"reimbursement_date":"${reimbursement}T00:00:00Z"`;
    }
    if (periode) {
        jsonBody += `,"periode":${parseInt(periode)}`;
    }
    if (aro) {
        jsonBody += `,"aro":${aro === 'true'}`;
    }
    if (instruction) {
        jsonBody += `,"instruction":"${instruction.replace(/"/g, '\\"')}"`;
    }

    jsonBody += '}';

    console.log('[AMEND] Submitting amendment request (raw JSON):', jsonBody);

    try {
        const response = await fetch(API_BASE + '/api/order/amend', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: jsonBody
        });

        const result = await response.json();
        console.log('[AMEND] Response:', result);

        if (result.status === 'success') {
            showStatus('amend-status', '✅ Order amended! Original: ' + result.data.original_order_nid + ', New: ' + result.data.new_order_nid, 'success');
            setTimeout(() => {
                resetAmendForm();
                closeModal('amend');
            }, 2000);
        } else {
            throw new Error(result.message || 'Unknown error');
        }
    } catch (error) {
        showStatus('amend-status', '❌ Error: ' + error.message, 'error');
    }
}

function resetAmendForm() {
    document.getElementById('amend-form').reset();
    // Reset all fields to empty
    document.getElementById('amend-quantity').value = '';
    document.getElementById('amend-settlement').value = '';
    document.getElementById('amend-reimbursement').value = '';
    document.getElementById('amend-periode').value = '';
    document.getElementById('amend-aro').value = '';
    document.getElementById('amend-instruction').value = '';
}

// Withdraw Order
async function submitWithdrawOrder(event) {
    event.preventDefault();
    showStatus('withdraw-status', '⏳ Submitting withdrawal...', 'loading');

    const orderNIDStr = document.getElementById('withdraw-order-nid').value.trim();

    // Validate order NID
    if (!orderNIDStr || orderNIDStr === '' || orderNIDStr === '0') {
        showStatus('withdraw-status', '❌ Error: Invalid Order NID', 'error');
        return;
    }

    console.log('[WITHDRAW] Order NID string from input:', orderNIDStr);

    // Keep as string to preserve precision
    const orderNID = orderNIDStr;

    // Build JSON manually to preserve large integer precision
    const jsonBody = `{"order_nid":${orderNID},"reff_request_id":"${generateReffId('WDRW')}"}`;

    console.log('[WITHDRAW] Submitting withdrawal request (raw JSON):', jsonBody);

    try {
        const response = await fetch(API_BASE + '/api/order/withdraw', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: jsonBody
        });

        const result = await response.json();
        console.log('[WITHDRAW] Response:', result);

        if (result.status === 'success') {
            showStatus('withdraw-status', '✅ Order withdrawal submitted! Order NID: ' + result.data.order_nid, 'success');
            setTimeout(() => {
                document.getElementById('withdraw-form').reset();
                closeModal('withdraw');
            }, 2000);
        } else {
            throw new Error(result.message || 'Unknown error');
        }
    } catch (error) {
        showStatus('withdraw-status', '❌ Error: ' + error.message, 'error');
    }
}

// Helper function to open amend modal by order index
function openAmendModalByIndex(index) {
    if (!window.currentOrders || !window.currentOrders[index]) {
        console.error('[AMEND] Order not found at index:', index);
        return;
    }

    const order = window.currentOrders[index];
    openAmendModal(order);
}

// Helper function to open amend modal with pre-filled order data
function openAmendModal(order) {
    openModal('amend');
    console.log('[AMEND] Opening modal for order:', order);

    // Fill in order NID (readonly)
    document.getElementById('amend-order-nid').value = String(order.nid);

    // Pre-fill all fields with current values
    document.getElementById('amend-quantity').value = order.quantity || '';
    document.getElementById('amend-settlement').value = order.settlement_date || '';
    document.getElementById('amend-reimbursement').value = order.reimbursement_date || '';
    document.getElementById('amend-periode').value = order.periode || '';
    document.getElementById('amend-aro').value = order.aro === true ? 'true' : order.aro === false ? 'false' : '';
    document.getElementById('amend-instruction').value = order.instruction || '';

    // Update modal title and show/hide fields based on side
    const modalTitle = document.querySelector('#amend-modal .modal-header h2');
    const quantityGroup = document.getElementById('amend-quantity-group');
    const settlementGroup = document.getElementById('amend-settlement-group');
    const reimbursementGroup = document.getElementById('amend-reimbursement-group');
    const periodeGroup = document.getElementById('amend-periode-group');
    const aroGroup = document.getElementById('amend-aro-group');
    const instructionGroup = document.getElementById('amend-instruction-group');

    if (order.side === 'LEND') {
        modalTitle.textContent = '✏️ Amend Lend Order (Only Quantity)';
        quantityGroup.style.display = 'flex';
        settlementGroup.style.display = 'none';
        reimbursementGroup.style.display = 'none';
        periodeGroup.style.display = 'none';
        aroGroup.style.display = 'none';
        instructionGroup.style.display = 'none';
    } else if (order.side === 'BORR') {
        modalTitle.textContent = '✏️ Amend Borrow Order';
        quantityGroup.style.display = 'flex';
        settlementGroup.style.display = 'flex';
        reimbursementGroup.style.display = 'flex';
        periodeGroup.style.display = 'flex';
        aroGroup.style.display = 'flex';
        instructionGroup.style.display = 'flex';
    }
}

// Helper function to open withdraw modal with pre-filled order NID
function openWithdrawModal(orderNID) {
    openModal('withdraw');
    // Convert string NID to ensure no precision loss
    document.getElementById('withdraw-order-nid').value = String(orderNID);
    console.log('[WITHDRAW] Opening modal for order NID:', orderNID);
}

// Initialize on page load
window.addEventListener('load', async () => {
    await loadParticipants();
    await loadInstruments();
    await loadAccounts();
    setDefaultBorrowDates();
});

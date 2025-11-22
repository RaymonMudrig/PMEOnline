function showTab(tabName) {
    // Hide all tabs
    const tabs = document.querySelectorAll('.tab-content');
    tabs.forEach(tab => tab.classList.remove('active'));

    // Remove active from all tab buttons
    const tabButtons = document.querySelectorAll('.tab');
    tabButtons.forEach(btn => btn.classList.remove('active'));

    // Show selected tab
    document.getElementById(tabName).classList.add('active');
    event.target.classList.add('active');
}

async function loadParticipants() {
    const status = document.getElementById('participants-status');
    const tableDiv = document.getElementById('participants-table');

    status.textContent = '⏳ Loading...';
    status.className = 'status loading';

    try {
        const response = await fetch('/participant/list');
        const data = await response.json();

        if (data.status === 'success') {
            const participants = data.data.participants;

            // Update stats
            document.getElementById('participants-count').textContent = participants.length;
            document.getElementById('participants-borr').textContent =
                participants.filter(p => p.borr_eligibility).length;
            document.getElementById('participants-lend').textContent =
                participants.filter(p => p.lend_eligibility).length;

            if (participants.length === 0) {
                tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📭</div><h3>No participants found</h3><p>Use the test script to insert sample data</p></div>';
            } else {
                let html = '<table><thead><tr><th>Code</th><th>Name</th><th>Borrow Eligible</th><th>Lend Eligible</th></tr></thead><tbody>';
                participants.forEach(p => {
                    html += '<tr>';
                    html += '<td><strong>' + p.code + '</strong></td>';
                    html += '<td>' + p.name + '</td>';
                    html += '<td>' + (p.borr_eligibility ? '<span class="badge badge-success">✓ Yes</span>' : '<span class="badge badge-danger">✗ No</span>') + '</td>';
                    html += '<td>' + (p.lend_eligibility ? '<span class="badge badge-success">✓ Yes</span>' : '<span class="badge badge-danger">✗ No</span>') + '</td>';
                    html += '</tr>';
                });
                html += '</tbody></table>';
                tableDiv.innerHTML = html;
            }

            status.textContent = '✅ Loaded ' + participants.length + ' participants';
            status.className = 'status success';
        } else {
            throw new Error(data.message || 'Unknown error');
        }
    } catch (error) {
        status.textContent = '❌ Error: ' + error.message;
        status.className = 'status error';
        tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><h3>Failed to load data</h3><p>' + error.message + '</p></div>';
    }
}

async function loadInstruments() {
    const status = document.getElementById('instruments-status');
    const tableDiv = document.getElementById('instruments-table');

    status.textContent = '⏳ Loading...';
    status.className = 'status loading';

    try {
        const response = await fetch('/instrument/list');
        const data = await response.json();

        if (data.status === 'success') {
            const instruments = data.data.instruments;

            // Update stats
            document.getElementById('instruments-count').textContent = instruments.length;
            document.getElementById('instruments-eligible').textContent =
                instruments.filter(i => i.status).length;
            document.getElementById('instruments-ineligible').textContent =
                instruments.filter(i => !i.status).length;

            if (instruments.length === 0) {
                tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📭</div><h3>No instruments found</h3><p>Use the test script to insert sample data</p></div>';
            } else {
                let html = '<table><thead><tr><th>Code</th><th>Name</th><th>Type</th><th>Status</th></tr></thead><tbody>';
                instruments.forEach(i => {
                    html += '<tr>';
                    html += '<td><strong>' + i.code + '</strong></td>';
                    html += '<td>' + i.name + '</td>';
                    html += '<td>' + (i.type || '-') + '</td>';
                    html += '<td>' + (i.status ? '<span class="badge badge-success">✓ Eligible</span>' : '<span class="badge badge-danger">✗ Ineligible</span>') + '</td>';
                    html += '</tr>';
                });
                html += '</tbody></table>';
                tableDiv.innerHTML = html;
            }

            status.textContent = '✅ Loaded ' + instruments.length + ' instruments';
            status.className = 'status success';
        } else {
            throw new Error(data.message || 'Unknown error');
        }
    } catch (error) {
        status.textContent = '❌ Error: ' + error.message;
        status.className = 'status error';
        tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><h3>Failed to load data</h3><p>' + error.message + '</p></div>';
    }
}

async function loadAccounts() {
    const status = document.getElementById('accounts-status');
    const tableDiv = document.getElementById('accounts-table');

    status.textContent = '⏳ Loading...';
    status.className = 'status loading';

    try {
        const response = await fetch('/account/list');
        const data = await response.json();

        if (data.status === 'success') {
            const accounts = data.data.accounts;

            // Update stats
            document.getElementById('accounts-count').textContent = accounts.length;

            if (accounts.length === 0) {
                tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📭</div><h3>No accounts found</h3><p>Use the test script to insert sample data</p></div>';
            } else {
                let html = '<table><thead><tr><th>Code</th><th>SID</th><th>Name</th><th>Address</th><th>Participant</th><th>Trade Limit</th><th>Pool Limit</th></tr></thead><tbody>';
                accounts.forEach(a => {
                    html += '<tr>';
                    html += '<td><strong>' + a.code + '</strong></td>';
                    html += '<td>' + (a.sid || '-') + '</td>';
                    html += '<td>' + a.name + '</td>';
                    html += '<td>' + (a.address || '-') + '</td>';
                    html += '<td>' + a.participant_code + '</td>';
                    html += '<td>' + (a.trade_limit ? a.trade_limit.toLocaleString() : '0') + '</td>';
                    html += '<td>' + (a.pool_limit ? a.pool_limit.toLocaleString() : '0') + '</td>';
                    html += '</tr>';
                });
                html += '</tbody></table>';
                tableDiv.innerHTML = html;
            }

            status.textContent = '✅ Loaded ' + accounts.length + ' accounts';
            status.className = 'status success';
        } else {
            throw new Error(data.message || 'Unknown error');
        }
    } catch (error) {
        status.textContent = '❌ Error: ' + error.message;
        status.className = 'status error';
        tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><h3>Failed to load data</h3><p>' + error.message + '</p></div>';
    }
}

// Parameters functions
async function loadParameters() {
    const status = document.getElementById('parameters-status');

    status.textContent = '⏳ Loading...';
    status.className = 'status loading';

    try {
        const response = await fetch('/parameter');
        const data = await response.json();

        if (data.status === 'success') {
            const param = data.data.parameter;

            document.getElementById('param-description').value = param.description || '';
            document.getElementById('param-flatfee').value = param.flat_fee || 0;
            document.getElementById('param-lendingfee').value = param.lending_fee || 0;
            document.getElementById('param-borrowingfee').value = param.borrowing_fee || 0;
            document.getElementById('param-maxqty').value = param.max_quantity || 0;
            document.getElementById('param-maxdays').value = param.borrow_max_open_day || 0;
            document.getElementById('param-denomlimit').value = param.denomination_limit || 0;

            status.textContent = '✅ Parameters loaded';
            status.className = 'status success';
        } else {
            throw new Error(data.message || 'Unknown error');
        }
    } catch (error) {
        status.textContent = '❌ Error: ' + error.message;
        status.className = 'status error';
    }
}

async function saveParameters(event) {
    event.preventDefault();
    const status = document.getElementById('parameters-status');

    status.textContent = '⏳ Saving...';
    status.className = 'status loading';

    const data = {
        description: document.getElementById('param-description').value,
        flat_fee: parseFloat(document.getElementById('param-flatfee').value),
        lending_fee: parseFloat(document.getElementById('param-lendingfee').value),
        borrowing_fee: parseFloat(document.getElementById('param-borrowingfee').value),
        max_quantity: parseFloat(document.getElementById('param-maxqty').value),
        borrow_max_open_day: parseInt(document.getElementById('param-maxdays').value),
        denomination_limit: parseInt(document.getElementById('param-denomlimit').value)
    };

    try {
        const response = await fetch('/parameter/update', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        const result = await response.json();

        if (result.status === 'success') {
            status.textContent = '✅ Parameters saved successfully';
            status.className = 'status success';
        } else {
            throw new Error(result.message || 'Unknown error');
        }
    } catch (error) {
        status.textContent = '❌ Error: ' + error.message;
        status.className = 'status error';
    }
}

// Holidays functions
async function loadHolidays() {
    const status = document.getElementById('holidays-status');
    const tableDiv = document.getElementById('holidays-table');

    status.textContent = '⏳ Loading...';
    status.className = 'status loading';

    try {
        const response = await fetch('/holiday/list');
        const data = await response.json();

        if (data.status === 'success') {
            const holidays = data.data.holidays;

            if (holidays.length === 0) {
                tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📭</div><h3>No holidays found</h3><p>Add holidays using the form above</p></div>';
            } else {
                let html = '<table><thead><tr><th>Date</th><th>Year</th><th>Description</th></tr></thead><tbody>';
                holidays.forEach(h => {
                    html += '<tr>';
                    html += '<td><strong>' + h.date + '</strong></td>';
                    html += '<td>' + h.tahun + '</td>';
                    html += '<td>' + h.description + '</td>';
                    html += '</tr>';
                });
                html += '</tbody></table>';
                tableDiv.innerHTML = html;
            }

            status.textContent = '✅ Loaded ' + holidays.length + ' holidays';
            status.className = 'status success';
        } else {
            throw new Error(data.message || 'Unknown error');
        }
    } catch (error) {
        status.textContent = '❌ Error: ' + error.message;
        status.className = 'status error';
        tableDiv.innerHTML = '<div class="empty-state"><div class="empty-state-icon">⚠️</div><h3>Failed to load data</h3><p>' + error.message + '</p></div>';
    }
}

async function addHoliday(event) {
    event.preventDefault();
    const status = document.getElementById('holidays-status');

    status.textContent = '⏳ Adding...';
    status.className = 'status loading';

    const data = {
        date: document.getElementById('holiday-date').value,
        description: document.getElementById('holiday-description').value
    };

    try {
        const response = await fetch('/holiday/add', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        const result = await response.json();

        if (result.status === 'success') {
            status.textContent = '✅ Holiday added successfully';
            status.className = 'status success';

            // Clear form
            document.getElementById('holiday-form').reset();

            // Reload holidays list
            loadHolidays();
        } else {
            throw new Error(result.message || 'Unknown error');
        }
    } catch (error) {
        status.textContent = '❌ Error: ' + error.message;
        status.className = 'status error';
    }
}

// Session Time functions
async function loadSessionTime() {
    const status = document.getElementById('sessiontime-status');

    status.textContent = '⏳ Loading...';
    status.className = 'status loading';

    try {
        const response = await fetch('/sessiontime');
        const data = await response.json();

        if (data.status === 'success') {
            const st = data.data.sessiontime;

            document.getElementById('session-description').value = st.description || '';
            document.getElementById('session1-start').value = st.session1_start || '';
            document.getElementById('session1-end').value = st.session1_end || '';
            document.getElementById('session2-start').value = st.session2_start || '';
            document.getElementById('session2-end').value = st.session2_end || '';

            status.textContent = '✅ Session time loaded';
            status.className = 'status success';
        } else {
            throw new Error(data.message || 'Unknown error');
        }
    } catch (error) {
        status.textContent = '❌ Error: ' + error.message;
        status.className = 'status error';
    }
}

async function saveSessionTime(event) {
    event.preventDefault();
    const status = document.getElementById('sessiontime-status');

    status.textContent = '⏳ Saving...';
    status.className = 'status loading';

    const data = {
        description: document.getElementById('session-description').value,
        session1_start: document.getElementById('session1-start').value,
        session1_end: document.getElementById('session1-end').value,
        session2_start: document.getElementById('session2-start').value,
        session2_end: document.getElementById('session2-end').value
    };

    try {
        const response = await fetch('/sessiontime/update', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        const result = await response.json();

        if (result.status === 'success') {
            status.textContent = '✅ Session time saved successfully';
            status.className = 'status success';
        } else {
            throw new Error(result.message || 'Unknown error');
        }
    } catch (error) {
        status.textContent = '❌ Error: ' + error.message;
        status.className = 'status error';
    }
}

// Auto-load participants on page load
window.addEventListener('load', () => {
    loadParticipants();
});

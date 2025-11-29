// WebSocket notification handler for APME API Dashboard

let ws = null;
let reconnectTimer = null;
let notificationCount = 0;
const MAX_RECONNECT_DELAY = 30000; // 30 seconds
const INITIAL_RECONNECT_DELAY = 1000; // 1 second
let currentReconnectDelay = INITIAL_RECONNECT_DELAY;

// Sequence tracking for DropCopy recovery
let lastReceivedSeq = 0;
let isRecovering = false;
let recoveryCount = 0;

// Load last sequence from localStorage
function loadLastSequence() {
    const saved = localStorage.getItem('ws_last_seq');
    if (saved) {
        lastReceivedSeq = parseInt(saved, 10) || 0;
        console.log('Loaded last sequence from storage:', lastReceivedSeq);
    }
}

// Save last sequence to localStorage
function saveLastSequence(seq) {
    lastReceivedSeq = seq;
    localStorage.setItem('ws_last_seq', seq.toString());
}

// Auto-connect when notifications tab is shown
window.addEventListener('DOMContentLoaded', () => {
    // Load last sequence number from storage
    loadLastSequence();

    // Listen for tab changes
    const notificationsTab = document.querySelector('.tab[onclick*="notifications"]');
    if (notificationsTab) {
        const originalShowTab = window.showTab;
        window.showTab = function(tabName) {
            originalShowTab(tabName);
            if (tabName === 'notifications' && !ws) {
                // Auto-connect when switching to notifications tab
                setTimeout(() => connectWebSocket(), 500);
            }
        };
    }
});

// Connect to WebSocket
function connectWebSocket() {
    if (ws && (ws.readyState === WebSocket.CONNECTING || ws.readyState === WebSocket.OPEN)) {
        console.log('WebSocket already connected or connecting');
        return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/notifications`;

    updateStatus('🟡 Connecting...', 'connecting');

    try {
        ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            console.log('WebSocket connected');
            updateStatus('🟢 Connected', 'connected');
            currentReconnectDelay = INITIAL_RECONNECT_DELAY;
            clearReconnectTimer();

            // Send subscribe message requesting ALL buffered notifications (from_seq: 0)
            // This ensures we get all available notifications from the server buffer
            const subscribeMsg = {
                type: 'subscribe',
                from_seq: 0  // 0 = request all available notifications
            };
            ws.send(JSON.stringify(subscribeMsg));
            console.log('Sent subscribe request for all buffered notifications');

            addSystemNotification('Connected - Loading all buffered notifications', 'success');
        };

        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);

                // Handle different message types
                if (message.type === 'recovery_start') {
                    handleRecoveryStart(message);
                } else if (message.type === 'recovery_complete') {
                    handleRecoveryComplete(message);
                } else if (message.type === 'buffer_info') {
                    handleBufferInfo(message);
                } else if (message.seq) {
                    // Regular sequenced notification
                    handleSequencedNotification(message);
                } else {
                    console.warn('Unknown message type:', message);
                }
            } catch (error) {
                console.error('Error parsing notification:', error);
            }
        };

        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            updateStatus('🔴 Error', 'error');
        };

        ws.onclose = (event) => {
            console.log('WebSocket closed:', event.code, event.reason);
            updateStatus('🔴 Disconnected', 'disconnected');
            ws = null;

            if (!event.wasClean) {
                addSystemNotification(`Connection lost (code: ${event.code}). Reconnecting...`, 'error');
                scheduleReconnect();
            }
        };

    } catch (error) {
        console.error('Error creating WebSocket:', error);
        updateStatus('🔴 Connection Failed', 'error');
        addSystemNotification(`Failed to connect: ${error.message}`, 'error');
    }
}

// Disconnect WebSocket
function disconnectWebSocket() {
    if (ws) {
        ws.close(1000, 'User requested disconnect');
        ws = null;
        clearReconnectTimer();
        updateStatus('🔴 Disconnected', 'disconnected');
        addSystemNotification('WebSocket disconnected', 'info');
    }
}

// Schedule reconnection with exponential backoff
function scheduleReconnect() {
    clearReconnectTimer();

    reconnectTimer = setTimeout(() => {
        console.log(`Attempting to reconnect... (delay: ${currentReconnectDelay}ms)`);
        connectWebSocket();
        currentReconnectDelay = Math.min(currentReconnectDelay * 2, MAX_RECONNECT_DELAY);
    }, currentReconnectDelay);
}

// Clear reconnection timer
function clearReconnectTimer() {
    if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
    }
}

// Update connection status
function updateStatus(text, status) {
    const statusElement = document.getElementById('ws-status');
    if (statusElement) {
        statusElement.textContent = text;
        statusElement.className = `status-indicator status-${status}`;
    }
}

// Handle recovery start message
function handleRecoveryStart(message) {
    isRecovering = true;
    recoveryCount = 0;
    console.log('Recovery started:', message);
    updateStatus('🔄 Recovering...', 'connecting');

    const msg = `Recovery: ${message.count} messages from seq ${message.requested_seq}`;
    if (!message.all_available) {
        addSystemNotification(`⚠️  ${msg} (some messages lost, oldest: ${message.oldest_seq})`, 'warning');
    } else {
        addSystemNotification(`✅ ${msg}`, 'info');
    }
}

// Handle recovery complete message
function handleRecoveryComplete(message) {
    isRecovering = false;
    console.log('Recovery complete:', message);
    updateStatus('🟢 Connected (Live)', 'connected');
    addSystemNotification(`Recovery complete: ${recoveryCount} messages recovered`, 'success');
}

// Handle buffer info message
function handleBufferInfo(message) {
    console.log('Buffer info:', message);
    const info = `Buffer: ${message.size}/${message.capacity} messages, seq ${message.oldest_seq}-${message.latest_seq}`;
    addSystemNotification(info, 'info');
}

// Handle sequenced notification
function handleSequencedNotification(message) {
    // Update sequence tracking
    if (message.seq) {
        saveLastSequence(message.seq);
        if (isRecovering) {
            recoveryCount++;
        }
    }

    // Display the notification
    displayNotification(message);
}

// Display a notification
function displayNotification(notification) {
    const container = document.getElementById('notifications-container');
    if (!container) return;

    // Remove "no notifications" message
    const noNotifications = container.querySelector('.no-notifications');
    if (noNotifications) {
        noNotifications.remove();
    }

    notificationCount++;

    const entry = document.createElement('div');
    entry.className = 'notification-entry';
    entry.id = `notification-${notificationCount}`;

    // Use timestamp from data if available
    const timestamp = notification.data?.timestamp
        ? new Date(notification.data.timestamp).toLocaleTimeString('en-US', {
            hour12: false,
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            fractionalSecondDigits: 3
        })
        : new Date().toLocaleTimeString('en-US', {
            hour12: false,
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            fractionalSecondDigits: 3
        });

    const eventType = notification.event_type || notification.type || 'unknown';
    const seq = notification.seq || 0;
    const data = notification.data || {};

    // Build single-line compact view with most important fields
    let dataFields = [];

    // Priority fields to show
    const priorityFields = ['account_code', 'order_nid', 'trade_nid', 'contract_nid', 'instrument', 'instrument_code',
                            'side', 'quantity', 'state', 'kpei_reff', 'message'];

    // Add priority fields first
    for (const key of priorityFields) {
        if (data[key] !== undefined && key !== 'timestamp') {
            let value = data[key];
            let valueClass = '';

            // Add special styling for state fields
            if (key === 'state') {
                valueClass = `state-${value}`;
                value = getStateName(value);
            }

            dataFields.push(`<span class="field-label">${key}:</span><span class="${valueClass}">${value}</span>`);
        }
    }

    // Add remaining fields
    for (const [key, value] of Object.entries(data)) {
        if (key !== 'timestamp' && !priorityFields.includes(key)) {
            dataFields.push(`<span class="field-label">${key}:</span><span>${value}</span>`);
        }
    }

    const html = `
        <span class="notification-seq">#${seq}</span>
        <span class="notification-timestamp">${timestamp}</span>
        <span class="notification-type ${eventType}">${eventType.replace(/_/g, ' ')}</span>
        <span class="notification-data">${dataFields.join(' | ')}</span>
    `;

    entry.innerHTML = html;

    // Add to container (newest at top)
    container.insertBefore(entry, container.firstChild);

    // Trim notifications if exceeding max
    trimNotifications();

    // Auto-scroll if enabled
    if (document.getElementById('auto-scroll')?.checked) {
        entry.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
}

// Add system notification
function addSystemNotification(message, type) {
    displayNotification({
        event_type: 'system',
        data: {
            message: message,
            type: type
        }
    });
}

// Get human-readable state name
function getStateName(state) {
    const stateNames = {
        'S': 'Submitted',
        'O': 'Open',
        'P': 'Partial',
        'M': 'Matched',
        'W': 'Withdrawn',
        'R': 'Rejected',
        'G': 'Pending',
        'E': 'Approval',
        'C': 'Closed',
        'T': 'Terminated'
    };
    return stateNames[state] || state;
}

// Clear all notifications
function clearNotifications() {
    const container = document.getElementById('notifications-container');
    if (!container) return;

    container.innerHTML = '<p class="no-notifications">No notifications yet. Click "Connect" to start receiving events.</p>';
    notificationCount = 0;
}

// Trim notifications to max count
function trimNotifications() {
    const maxEvents = parseInt(document.getElementById('max-events')?.value || 100);
    const container = document.getElementById('notifications-container');
    if (!container) return;

    const entries = container.querySelectorAll('.notification-entry');
    if (entries.length > maxEvents) {
        for (let i = maxEvents; i < entries.length; i++) {
            entries[i].remove();
        }
    }
}

// Clean up on page unload
window.addEventListener('beforeunload', () => {
    if (ws) {
        ws.close(1000, 'Page unloading');
    }
    clearReconnectTimer();
});

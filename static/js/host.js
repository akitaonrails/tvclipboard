/* global t, formatTime, getWebSocketURL, getPublicURL, decryptMessage */
// Host-specific functionality
(function() {
    'use strict';

    let receivedFullContent = '';
    let isRevealed = false;
    let ws;
    let timerInterval;
    const sessionTimeout = 600;

    async function showReceivedContent(encryptedContent) {
    const section = document.getElementById('received-section');
    const contentDiv = document.getElementById('received-content');
    const timestamp = document.getElementById('timestamp');
    const revealBtn = document.getElementById('reveal-btn');

    let decrypted;
    try {
        // Try to decrypt content
        decrypted = await decryptMessage(encryptedContent);
        console.log('Successfully decrypted message');
    } catch (error) {
        console.error('Decryption failed:', error);
        // If decryption fails, content is unencrypted
        decrypted = encryptedContent;
    }

    receivedFullContent = decrypted;
    isRevealed = false;

    // Show obfuscated version (first 3 chars + ellipsis)
    const obfuscated = decrypted.length > 3
        ? decrypted.substring(0, 3) + '...'
        : decrypted;

    contentDiv.textContent = obfuscated;
    contentDiv.classList.add('obfuscated');
    contentDiv.onclick = toggleReveal;
    section.classList.add('show');

    // Show reveal button
    revealBtn.textContent = '👁️ ' + t('host.reveal_button');
    revealBtn.style.display = 'inline-block';

    const now = new Date();
    timestamp.textContent = t('host.received_at') + ' ' + now.toLocaleTimeString();

    // Try to copy to clipboard automatically
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(decrypted)
            .then(() => {
                console.log(t('host.auto_copied'));
            })
            .catch(err => {
                console.error(t('host.auto_copy_failed'), err);
            });
    }
}

function copyReceived() {
    console.log('copyReceived called, content length:', receivedFullContent.length);

    if (!receivedFullContent) {
        alert(t('host.no_content_to_copy'));
        return;
    }

    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(receivedFullContent)
            .then(() => {
                console.log('Successfully copied to clipboard');
                alert(t('errors.copied_to_clipboard'));
            })
            .catch(err => {
                console.error(t('errors.failed_to_copy'), err);
                alert(t('errors.failed_to_copy') + ': ' + err.message);
            });
    } else {
        console.error('Clipboard API not supported');
        alert(t('host.clipboard_not_supported_browser'));
    }
}

// Expose functions needed by HTML onclick handlers
window.toggleReveal = toggleReveal;
window.copyReceived = copyReceived;

// Log that functions are exposed
console.log('Host functions exposed: copyReceived, toggleReveal');

function toggleReveal() {
    const contentDiv = document.getElementById('received-content');
    const revealBtn = document.getElementById('reveal-btn');

    if (isRevealed) {
        // Hide content
        const obfuscated = receivedFullContent.length > 3
            ? receivedFullContent.substring(0, 3) + '...'
            : receivedFullContent;
        contentDiv.textContent = obfuscated;
        contentDiv.classList.add('obfuscated');
        contentDiv.onclick = toggleReveal;
        revealBtn.textContent = '👁️ ' + t('host.reveal_button');
        isRevealed = false;
    } else {
        // Show full content
        contentDiv.textContent = receivedFullContent;
        contentDiv.classList.remove('obfuscated');
        contentDiv.onclick = null;
        revealBtn.textContent = '🙈 ' + t('common.hide_content');
        isRevealed = true;
    }
}

function generateQRCode() {
    const url = getPublicURL().replace(/\?.*$/, '');
    const container = document.getElementById('qrcode');
    const urlText = document.getElementById('url-text');

    // Use server-side generated QR code
    const img = document.createElement('img');
    img.src = '/qrcode.png?' + Date.now();
    img.alt = 'QR Code';
    img.style.width = '200px';
    img.style.height = '200px';

    container.innerHTML = '';
    container.appendChild(img);

    urlText.textContent = url + ' (' + t('host.links_to_client') + ')';
}

    function startTimer() {
    let remaining = sessionTimeout;
    const timerEl = document.getElementById('time-remaining');

    timerInterval = setInterval(function() {
        remaining--;
        if (timerEl) {
            timerEl.textContent = formatTime(remaining);
        }

        if (remaining <= 0) {
            clearInterval(timerInterval);
            refreshPage();
        } else if (remaining <= 60) {
            if (timerEl) {
                timerEl.style.color = '#ff6b6b';
                timerEl.style.animation = 'pulse 1s infinite';
            }
        }
    }, 1000);

    if (timerEl) {
        timerEl.textContent = formatTime(remaining);
    }
}

function refreshPage() {
    const timerEl = document.getElementById('timer');
    if (timerEl) {
        timerEl.innerHTML = '⏱️ <span style="color: #4CAF50;">' + t('host.refreshing_qr') + '</span>';
    }
    setTimeout(function() {
        location.reload();
    }, 2000);
}

function connect() {
    // Close existing WebSocket connection before creating a new one
    if (ws) {
        ws.onclose = null; // Prevent reconnection trigger
        ws.close();
    }

    const url = getWebSocketURL();
    console.log('Attempting to connect to WebSocket URL:', url);

    ws = new WebSocket(url);

    ws.onopen = function() {
        const status = document.getElementById('status');
        status.textContent = '🖥️ ' + t('host.status_host_connected');
        status.className = 'status connected';

        const errorEl = document.getElementById('error-message');
        errorEl.style.display = 'none';

        console.log('WebSocket connected');

        startTimer();
    };

    ws.onclose = function(event) {
        // Clear timer when connection closes
        if (timerInterval) {
            clearInterval(timerInterval);
            timerInterval = null;
        }

        const status = document.getElementById('status');
        status.textContent = '🔌 Disconnected';
        status.className = 'status disconnected';
        console.log('WebSocket disconnected. Code:', event.code, 'Reason:', event.reason);

        setTimeout(connect, 3000);
    };

    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
        console.log('Current location:', window.location.href);
        console.log('WebSocket URL:', url);

        const errorEl = document.getElementById('error-message');
        const statusEl = document.getElementById('status');

        if (ws.readyState === WebSocket.CLOSED) {
            errorEl.textContent = t('host.host_already_connected');
            errorEl.style.display = 'block';
            statusEl.textContent = '❌ ' + t('host.connection_rejected');
        }
    };

    ws.onmessage = function(event) {
        const message = JSON.parse(event.data);
        console.log('Received message:', message);

        if (message.type === 'role') {
            handleRoleAssignment(message.role);
        } else if (message.type === 'text' && message.content) {
            showReceivedContent(message.content);
        }
    };
}

function handleRoleAssignment(role) {
    if (role !== 'host') {
        console.warn('Expected host role but got:', role);
    }
    generateQRCode();
}

// Expose functions needed by HTML onclick handlers
window.toggleReveal = toggleReveal;
window.copyReceived = copyReceived;

connect();
})();

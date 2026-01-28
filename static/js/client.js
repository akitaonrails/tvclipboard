/* global t, formatTime, encryptMessage, getWebSocketURL */
// Client-specific functionality
(function() {
    'use strict';

    let ws;
    let sessionExpired = false;
    let connectionFailed = false;
    let timerInterval;
    const appDiv = document.querySelector('.container');
    const sessionTimeout = appDiv ? parseInt(appDiv.getAttribute('data-session-timeout') || '600', 10) : 600;

    async function sendText() {
    const input = document.getElementById('input');
    const content = input.value.trim();

    if (!content) {
        alert(t('errors.please_enter_text'));
        return;
    }

    if (ws && ws.readyState === WebSocket.OPEN) {
        try {
            // Encrypt the content
            const encrypted = await encryptMessage(content);
            const message = {
                type: 'text',
                content: encrypted
            };
            ws.send(JSON.stringify(message));
            console.log('Sent message:', message);
            input.value = '';
        } catch (error) {
            console.error('Encryption failed:', error);
            // Fallback to sending unencrypted
            const message = {
                type: 'text',
                content: content
            };
            ws.send(JSON.stringify(message));
            console.log('Sent unencrypted message:', message);
            input.value = '';
        }
    } else {
        alert(t('errors.not_connected'));
    }
}

function clearInput() {
    document.getElementById('input').value = '';
}

function closeTab() {
    window.close();
}

function copyFromClipboard() {
    if (navigator.clipboard && navigator.clipboard.readText) {
        navigator.clipboard.readText()
            .then(text => {
                document.getElementById('input').value = text;
            })
            .catch(err => {
                console.error(t('errors.failed_to_copy'), err);
                alert(t('errors.clipboard_access_blocked'));
            });
    } else {
        alert(t('errors.clipboard_not_supported'));
        document.getElementById('input').focus();
    }
}

// Allow Enter key to send (with Shift+Enter for new line)
document.addEventListener('DOMContentLoaded', function() {
    const input = document.getElementById('input');
    if (input) {
        input.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                sendText();
            }
        });
    }
});

    function checkToken() {
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token');

    if (!token || token.trim() === '') {
        showError(t('errors.no_token'));
        disableAll();
        return false;
    }

    return true;
}

function showError(message) {
    const errorEl = document.getElementById('error');
    const statusEl = document.getElementById('status');

    if (errorEl) {
        errorEl.textContent = '❌ ' + message;
        errorEl.style.display = 'block';
        errorEl.className = 'status disconnected';
    }

    if (statusEl) {
        statusEl.style.display = 'none';
    }
}

function disableAll() {
    const textarea = document.getElementById('input');
    const buttons = document.querySelectorAll('button');

    if (textarea) {
        textarea.disabled = true;
        textarea.placeholder = t('client.connection_disabled');
    }

    buttons.forEach(btn => {
        btn.disabled = true;
    });
}

function enableAll() {
    const textarea = document.getElementById('input');
    const buttons = document.querySelectorAll('button');

    if (textarea) {
        textarea.disabled = false;
        textarea.placeholder = t('client.input_placeholder');
    }

    buttons.forEach(btn => {
        btn.disabled = false;
    });
}

function startTimer() {
    let remaining = sessionTimeout;
    const timerEl = document.getElementById('time-remaining');

    if (timerEl) {
        timerEl.parentElement.style.display = 'block';
    }

    timerInterval = setInterval(function() {
        remaining--;
        if (timerEl) {
            timerEl.textContent = formatTime(remaining);
        }

        if (remaining <= 0) {
            clearInterval(timerInterval);
            expireSession();
        } else if (remaining <= 60) {
            if (timerEl) {
                timerEl.style.color = '#ff6b6b';
                timerEl.parentElement.style.animation = 'pulse 1s infinite';
            }
        }
    }, 1000);

    if (timerEl) {
        timerEl.textContent = formatTime(remaining);
    }
}

function expireSession() {
    sessionExpired = true;
    clearInterval(timerInterval);

    const timerEl = document.getElementById('timer');
    if (timerEl) {
        timerEl.innerHTML = '⏱️ <span style="color: #ff6b6b;">' + t('client.session_expired') + '</span>';
        timerEl.style.color = '#ff6b6b';
    }

    disableAll();

    const sendBtn = document.getElementById('sendBtn');
    if (sendBtn) {
        sendBtn.textContent = t('client.expired_button');
    }

    showError(t('errors.session_expired_alert'));

    if (ws) {
        ws.close();
    }
}

function connect() {
    if (sessionExpired) {
        return;
    }

    if (!checkToken()) {
        return;
    }

    // Close existing WebSocket connection before creating a new one
    if (ws) {
        ws.onclose = null; // Prevent reconnection trigger
        ws.close();
    }

    const url = getWebSocketURL();
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token');

    ws = new WebSocket(url + '?token=' + token);

    ws.onopen = function() {
        const status = document.getElementById('status');
        const errorEl = document.getElementById('error');

        if (status) {
            status.textContent = '📱 ' + t('client.status_client_connected');
            status.className = 'status connected';
            status.style.display = 'block';
        }

        if (errorEl) {
            errorEl.style.display = 'none';
        }

        enableAll();
        console.log('WebSocket connected');

        startTimer();
    };

    ws.onclose = function(event) {
        if (sessionExpired || connectionFailed) {
            return;
        }

        // Clear timer when connection closes
        if (timerInterval) {
            clearInterval(timerInterval);
            timerInterval = null;
        }

        const status = document.getElementById('status');
        if (status) {
            if (event.code === 1000) {
                status.textContent = '🔌 ' + t('common.status_disconnected');
            } else if (event.code === 1006) {
                status.textContent = '🔌 ' + t('client.status_connection_lost');
            } else if (event.code >= 4000) {
                status.textContent = '❌ ' + t('client.status_connection_failed') + ' (' + event.code + ')';
            } else {
                status.textContent = '🔌 ' + t('client.status_disconnected_code', {code: event.code});
            }
            status.className = 'status disconnected';
        }
        console.log('WebSocket disconnected:', event.code, event.reason);

        if (!sessionExpired && !connectionFailed) {
            setTimeout(connect, 3000);
        }
    };

    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
        connectionFailed = true;
        disableAll();

        const status = document.getElementById('status');
        if (status) {
            status.style.display = 'none';
        }

        showError(t('errors.connection_failed_detailed'));
    };

    ws.onmessage = function(event) {
        const message = JSON.parse(event.data);
        console.log('Received message:', message);

        if (message.type === 'role') {
            handleRoleAssignment(message.role);
        }
    };
}

    function handleRoleAssignment(role) {
        if (role !== 'client') {
            console.warn('Expected client role but got:', role);
            disableAll();
            showError(t('errors.invalid_role'));
        }
    }

    // Expose functions needed by HTML onclick handlers
    // Note: sendText gets session checking wrapper to preserve original behavior
    window.sendText = function() {
        if (sessionExpired) {
            alert(t('errors.session_expired_alert'));
            return;
        }
        return sendText();
    };
    window.copyFromClipboard = copyFromClipboard;
    window.clearInput = clearInput;
    window.closeTab = closeTab;

    connect();
})();

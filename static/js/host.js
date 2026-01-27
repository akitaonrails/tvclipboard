// Host-specific functionality

let receivedFullContent = '';
let isRevealed = false;

async function showReceivedContent(encryptedContent, from) {
    const section = document.getElementById('received-section');
    const contentDiv = document.getElementById('received-content');
    const timestamp = document.getElementById('timestamp');
    const revealBtn = document.getElementById('reveal-btn');

    let decrypted;
    try {
        // Try to decrypt content
        decrypted = await decryptMessage(encryptedContent);
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
    revealBtn.textContent = '👁️ Show Content';
    revealBtn.style.display = 'inline-block';

    const now = new Date();
    timestamp.textContent = `Received at ${now.toLocaleTimeString()}`;

    // Try to copy to clipboard automatically
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(decrypted)
            .then(() => {
                console.log('Auto-copied to clipboard');
            })
            .catch(err => {
                console.error('Auto-copy failed:', err);
            });
    }
}

function copyReceived() {
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(receivedFullContent)
            .then(() => {
                alert('Copied to clipboard!');
            })
            .catch(err => {
                console.error('Failed to copy:', err);
                alert('Failed to copy to clipboard');
            });
    } else {
        alert('Clipboard not supported');
    }
}

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
        revealBtn.textContent = '👁️ Show Content';
        isRevealed = false;
    } else {
        // Show full content
        contentDiv.textContent = receivedFullContent;
        contentDiv.classList.remove('obfuscated');
        contentDiv.onclick = null;
        revealBtn.textContent = '🙈 Hide Content';
        isRevealed = true;
    }
}

function generateQRCode() {
    const url = getPublicURL().replace(/\?.*$/, '?mode=client');
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

    urlText.textContent = url + ' (Scan to connect as client)';
}

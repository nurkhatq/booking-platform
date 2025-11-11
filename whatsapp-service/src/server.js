const express = require('express');
const bodyParser = require('body-parser');
const qrcode = require('qrcode-terminal');
const { Client, LocalAuth } = require('whatsapp-web.js');
const winston = require('winston');

// Configure logger
const logger = winston.createLogger({
    level: 'info',
    format: winston.format.combine(
        winston.format.timestamp(),
        winston.format.json()
    ),
    transports: [
        new winston.transports.Console({
            format: winston.format.combine(
                winston.format.colorize(),
                winston.format.simple()
            )
        })
    ]
});

// Create Express app
const app = express();
app.use(bodyParser.json());

const PORT = process.env.WHATSAPP_SERVICE_PORT || 3000;

// WhatsApp client
let whatsappClient = null;
let isReady = false;
let qrCodeData = null;

// Initialize WhatsApp client
function initializeWhatsApp() {
    logger.info('Initializing WhatsApp client...');

    whatsappClient = new Client({
        authStrategy: new LocalAuth({
            dataPath: './.wwebjs_auth'
        }),
        puppeteer: {
            headless: true,
            args: [
                '--no-sandbox',
                '--disable-setuid-sandbox',
                '--disable-dev-shm-usage',
                '--disable-accelerated-2d-canvas',
                '--no-first-run',
                '--no-zygote',
                '--disable-gpu'
            ]
        }
    });

    // QR Code event
    whatsappClient.on('qr', (qr) => {
        logger.info('QR Code received. Scan with WhatsApp:');
        qrcode.generate(qr, { small: true });
        qrCodeData = qr;
    });

    // Ready event
    whatsappClient.on('ready', () => {
        logger.info('WhatsApp client is ready!');
        isReady = true;
        qrCodeData = null;
    });

    // Authenticated event
    whatsappClient.on('authenticated', () => {
        logger.info('WhatsApp authenticated successfully');
    });

    // Auth failure event
    whatsappClient.on('auth_failure', (msg) => {
        logger.error('WhatsApp authentication failed:', msg);
        isReady = false;
    });

    // Disconnected event
    whatsappClient.on('disconnected', (reason) => {
        logger.warn('WhatsApp disconnected:', reason);
        isReady = false;
        // Attempt to reconnect
        setTimeout(() => {
            logger.info('Attempting to reconnect...');
            whatsappClient.initialize();
        }, 5000);
    });

    // Initialize
    whatsappClient.initialize();
}

// Format phone number to WhatsApp format
function formatPhoneNumber(phone) {
    // Remove all non-digit characters
    let cleaned = phone.replace(/\D/g, '');

    // If number starts with 8, replace with 7 (for Kazakhstan)
    if (cleaned.startsWith('8')) {
        cleaned = '7' + cleaned.substring(1);
    }

    // If number doesn't start with country code, add 7 (Kazakhstan)
    if (!cleaned.startsWith('7')) {
        cleaned = '7' + cleaned;
    }

    // Add WhatsApp suffix
    return cleaned + '@c.us';
}

// Routes

// Health check
app.get('/health', (req, res) => {
    res.json({
        status: isReady ? 'healthy' : 'initializing',
        service: 'whatsapp-service',
        ready: isReady,
        hasQR: qrCodeData !== null
    });
});

// Get QR code (for authentication)
app.get('/qr', (req, res) => {
    if (isReady) {
        res.json({
            authenticated: true,
            message: 'WhatsApp is already authenticated'
        });
    } else if (qrCodeData) {
        res.json({
            authenticated: false,
            qr: qrCodeData,
            message: 'Scan this QR code with WhatsApp'
        });
    } else {
        res.json({
            authenticated: false,
            message: 'QR code not ready yet, please wait...'
        });
    }
});

// Send message
app.post('/send-message', async (req, res) => {
    const { phone, message } = req.body;

    if (!phone || !message) {
        return res.status(400).json({
            error: 'Phone and message are required'
        });
    }

    if (!isReady) {
        return res.status(503).json({
            error: 'WhatsApp service not ready',
            message: 'Please authenticate WhatsApp first'
        });
    }

    try {
        const formattedPhone = formatPhoneNumber(phone);
        logger.info(`Sending message to ${formattedPhone}`);

        // Check if number is registered on WhatsApp
        const isRegistered = await whatsappClient.isRegisteredUser(formattedPhone);

        if (!isRegistered) {
            logger.warn(`Number ${phone} is not registered on WhatsApp`);
            return res.status(400).json({
                error: 'Number not registered on WhatsApp',
                phone: phone
            });
        }

        // Send message
        await whatsappClient.sendMessage(formattedPhone, message);

        logger.info(`Message sent successfully to ${phone}`);

        res.json({
            success: true,
            message: 'Message sent successfully',
            phone: phone
        });

    } catch (error) {
        logger.error('Error sending message:', error);
        res.status(500).json({
            error: 'Failed to send message',
            details: error.message
        });
    }
});

// Send message to multiple recipients
app.post('/send-bulk', async (req, res) => {
    const { phones, message } = req.body;

    if (!phones || !Array.isArray(phones) || phones.length === 0) {
        return res.status(400).json({
            error: 'Phones array is required'
        });
    }

    if (!message) {
        return res.status(400).json({
            error: 'Message is required'
        });
    }

    if (!isReady) {
        return res.status(503).json({
            error: 'WhatsApp service not ready'
        });
    }

    const results = [];

    for (const phone of phones) {
        try {
            const formattedPhone = formatPhoneNumber(phone);
            const isRegistered = await whatsappClient.isRegisteredUser(formattedPhone);

            if (isRegistered) {
                await whatsappClient.sendMessage(formattedPhone, message);
                results.push({ phone, status: 'sent' });
                logger.info(`Bulk message sent to ${phone}`);
            } else {
                results.push({ phone, status: 'not_registered' });
                logger.warn(`Bulk message failed - ${phone} not registered`);
            }

            // Add delay to avoid rate limiting
            await new Promise(resolve => setTimeout(resolve, 1000));

        } catch (error) {
            results.push({ phone, status: 'error', error: error.message });
            logger.error(`Bulk message error for ${phone}:`, error);
        }
    }

    res.json({
        success: true,
        results: results,
        total: phones.length,
        sent: results.filter(r => r.status === 'sent').length,
        failed: results.filter(r => r.status !== 'sent').length
    });
});

// Check if number is registered on WhatsApp
app.post('/check-number', async (req, res) => {
    const { phone } = req.body;

    if (!phone) {
        return res.status(400).json({
            error: 'Phone is required'
        });
    }

    if (!isReady) {
        return res.status(503).json({
            error: 'WhatsApp service not ready'
        });
    }

    try {
        const formattedPhone = formatPhoneNumber(phone);
        const isRegistered = await whatsappClient.isRegisteredUser(formattedPhone);

        res.json({
            phone: phone,
            registered: isRegistered
        });

    } catch (error) {
        logger.error('Error checking number:', error);
        res.status(500).json({
            error: 'Failed to check number',
            details: error.message
        });
    }
});

// Logout/disconnect
app.post('/logout', async (req, res) => {
    try {
        if (whatsappClient) {
            await whatsappClient.logout();
            isReady = false;
            logger.info('WhatsApp logged out');
        }

        res.json({
            success: true,
            message: 'Logged out successfully'
        });

    } catch (error) {
        logger.error('Error logging out:', error);
        res.status(500).json({
            error: 'Failed to logout',
            details: error.message
        });
    }
});

// Start server
app.listen(PORT, () => {
    logger.info(`WhatsApp service started on port ${PORT}`);
    initializeWhatsApp();
});

// Graceful shutdown
process.on('SIGINT', async () => {
    logger.info('Shutting down gracefully...');

    if (whatsappClient) {
        await whatsappClient.destroy();
    }

    process.exit(0);
});

process.on('SIGTERM', async () => {
    logger.info('Shutting down gracefully...');

    if (whatsappClient) {
        await whatsappClient.destroy();
    }

    process.exit(0);
});

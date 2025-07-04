// Jest global setup for HTTPS configuration
import https from 'https';
import { Agent } from 'https';

// Configure Node.js to accept self-signed certificates globally
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

// Configure global HTTPS agent
const httpsAgent = new Agent({
  rejectUnauthorized: false,
});

// Set global HTTP agent
https.globalAgent = httpsAgent;

export default async function() {
  console.log('🔧 Jest setup: Configured HTTPS to accept self-signed certificates');
}


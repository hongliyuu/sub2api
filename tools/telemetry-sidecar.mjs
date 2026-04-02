import { fileURLToPath, pathToFileURL } from 'url';
import fs from 'fs/promises';
import path from 'path';

async function loadAxios() {
  const scriptDir = path.dirname(fileURLToPath(import.meta.url));
  const candidates = [
    path.resolve(scriptDir, '../../restored-src/node_modules/axios/index.js'),
    path.resolve(scriptDir, '../frontend/node_modules/axios/index.js'),
  ];

  for (const candidate of candidates) {
    try {
      await fs.access(candidate);
      const mod = await import(pathToFileURL(candidate).href);
      return mod.default ?? mod;
    } catch {
      // try next
    }
  }

  throw new Error('axios runtime not found for telemetry sidecar');
}

async function readStdin() {
  const chunks = [];
  for await (const chunk of process.stdin) {
    chunks.push(chunk);
  }
  return Buffer.concat(chunks).toString('utf8');
}

function sanitizeHeaders(headers = {}) {
  const out = {};
  for (const [key, value] of Object.entries(headers)) {
    if (typeof value === 'string' && value !== '') {
      out[key] = value;
    }
  }
  return out;
}

async function main() {
  const raw = await readStdin();
  const input = JSON.parse(raw);
  const axios = await loadAxios();
  const headers = sanitizeHeaders(input.headers);
  const payload = Buffer.from(input.payload_base64, 'base64').toString('utf8');

  const response = await axios.post(input.endpoint, payload, {
    timeout: input.timeout_ms ?? 10000,
    headers,
  });

  process.stdout.write(JSON.stringify({
    status: response.status,
    data: response.data,
  }));
}

main().catch((error) => {
  process.stdout.write(JSON.stringify({
    status: 0,
    error: error instanceof Error ? error.message : String(error),
  }));
  process.exitCode = 1;
});

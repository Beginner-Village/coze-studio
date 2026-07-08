#!/usr/bin/env node

import { createServer } from 'node:http';
import { readFileSync } from 'node:fs';

const listenHost = process.env.YNET_WORKFLOW_MCP_PROXY_HOST || '127.0.0.1';
const listenPort = Number(process.env.YNET_WORKFLOW_MCP_PROXY_PORT || 8898);
const upstream =
  process.env.YNET_WORKFLOW_MCP_UPSTREAM ||
  'http://10.10.10.226:8896/api/workflow_mcp/mcp';
const sessionFile =
  process.env.YNET_WORKFLOW_SESSION_FILE || '/tmp/ynet-workflow-session-key';

function readSessionKey() {
  const fromEnv = (process.env.YNET_WORKFLOW_SESSION_KEY || '').trim();
  if (fromEnv) {
    return fromEnv;
  }
  return readFileSync(sessionFile, 'utf8').trim();
}

function proxyHeaders(req) {
  const headers = new Headers();
  for (const [name, value] of Object.entries(req.headers)) {
    if (value === undefined) {
      continue;
    }
    const lower = name.toLowerCase();
    if (
      lower === 'host' ||
      lower === 'connection' ||
      lower === 'content-length' ||
      lower === 'accept-encoding'
    ) {
      continue;
    }
    headers.set(name, Array.isArray(value) ? value.join(', ') : value);
  }
  headers.set('cookie', `session_key=${readSessionKey()}`);
  return headers;
}

async function readBody(req) {
  const chunks = [];
  for await (const chunk of req) {
    chunks.push(chunk);
  }
  return chunks.length > 0 ? Buffer.concat(chunks) : undefined;
}

function responseHeaders(headers) {
  const out = {};
  headers.forEach((value, name) => {
    const lower = name.toLowerCase();
    if (
      lower === 'content-length' ||
      lower === 'content-encoding' ||
      lower === 'transfer-encoding' ||
      lower === 'connection' ||
      lower === 'keep-alive'
    ) {
      return;
    }
    out[name] = value;
  });
  return out;
}

const server = createServer(async (req, res) => {
  try {
    const body = req.method === 'GET' || req.method === 'HEAD'
      ? undefined
      : await readBody(req);
    const upstreamResp = await fetch(upstream, {
      method: req.method,
      headers: proxyHeaders(req),
      body,
    });
    const payload = Buffer.from(await upstreamResp.arrayBuffer());
    res.writeHead(upstreamResp.status, responseHeaders(upstreamResp.headers));
    res.end(payload);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    res.writeHead(502, { 'content-type': 'application/json' });
    res.end(JSON.stringify({ status: 'error', error: message }));
  }
});

server.listen(listenPort, listenHost, () => {
  console.error(
    `workflow MCP proxy listening on http://${listenHost}:${listenPort} -> ${upstream}`,
  );
});

// Prints results.jsonl as a table, one row per run.
import { readFileSync } from 'node:fs';

const rows = readFileSync('results.jsonl', 'utf8').trim().split('\n').map(l => JSON.parse(l));
const cols = [
    ['label', r => r.label],
    ['no client %', r => r.cpuNoClient.total],
    ['client idle %', r => r.cpuClientIdle.total],
    ['clicking %', r => r.cpuClicking.total],
    ['Xvnc %', r => r.cpuClicking.Xtigervnc],
    ['after disc. %', r => r.cpuAfterDisconnect.total],
    ['click p50 ms', r => r.clickLatencyMs.p50],
    ['click p90 ms', r => r.clickLatencyMs.p90],
    ['missed', r => r.clickLatencyMs.missed],
    ['KB/s clicking', r => r.wsClicking.kbPerSec],
];
const table = [cols.map(c => c[0]), ...rows.map(r => cols.map(c => String(c[1](r) ?? '-')))];
const w = cols.map((_, i) => Math.max(...table.map(t => t[i].length)));
for (const t of table) console.log(t.map((v, i) => (i ? v.padStart(w[i]) : v.padEnd(w[i]))).join('  '));

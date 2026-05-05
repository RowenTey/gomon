//go:build js && wasm

package handlers

import "net/http"

func ServeUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>GoMon Dashboard</title>
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{--bg:#f1f5f9;--surface:#fff;--border:#e2e8f0;--text:#1e293b;--text-muted:#64748b;--primary:#3b82f6;--primary-hover:#2563eb;--danger:#ef4444;--danger-hover:#dc2626;--up:#22c55e;--down:#ef4444;--degraded:#f97316;--unknown:#94a3b8;--radius:8px;--shadow:0 1px 3px rgba(0,0,0,.08)}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:var(--bg);color:var(--text);line-height:1.5}
header{background:var(--surface);border-bottom:1px solid var(--border);padding:16px 24px;display:flex;align-items:center;gap:32px;position:sticky;top:0;z-index:50}
header h1{font-size:20px;font-weight:700;color:var(--text);display:flex;align-items:center;gap:6px}
header h1 span{font-size:14px;color:var(--text-muted);font-weight:400}
nav{display:flex;gap:4px}
.tab-btn{padding:8px 16px;border:none;background:transparent;color:var(--text-muted);cursor:pointer;font-size:14px;border-radius:var(--radius);transition:all .15s}
.tab-btn:hover{background:var(--bg);color:var(--text)}
.tab-btn.active{background:var(--primary);color:#fff}
main{max-width:1200px;margin:0 auto;padding:24px}
.tab-content{display:none}
.tab-content.active{display:block}
.toolbar{display:flex;justify-content:space-between;align-items:center;margin-bottom:16px}
.toolbar h2{font-size:18px;font-weight:600}
.btn{padding:8px 16px;border:none;border-radius:var(--radius);font-size:14px;cursor:pointer;transition:all .15s;font-weight:500}
.btn-primary{background:var(--primary);color:#fff}
.btn-primary:hover{background:var(--primary-hover)}
.btn-danger{background:var(--danger);color:#fff}
.btn-danger:hover{background:var(--danger-hover)}
.btn-outline{background:transparent;border:1px solid var(--border);color:var(--text)}
.btn-outline:hover{background:var(--bg)}
.btn-sm{padding:4px 10px;font-size:13px}
table{width:100%;border-collapse:collapse;background:var(--surface);border-radius:var(--radius);overflow:hidden;box-shadow:var(--shadow)}
th,td{padding:10px 14px;text-align:left;border-bottom:1px solid var(--border);font-size:14px}
th{background:#f8fafc;color:var(--text-muted);font-weight:600;text-transform:uppercase;font-size:12px;letter-spacing:.5px}
tr:last-child td{border-bottom:none}
tr:hover td{background:#f8fafc}
.status-dot{display:inline-block;width:10px;height:10px;border-radius:50%;margin-right:6px;vertical-align:middle}
.status-up{background:var(--up)}.status-down{background:var(--down)}.status-degraded{background:var(--degraded)}.status-unknown{background:var(--unknown)}
.actions{display:flex;gap:4px}
.url-cell{max-width:300px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.empty-state{text-align:center;padding:48px 24px;color:var(--text-muted);background:var(--surface);border-radius:var(--radius);box-shadow:var(--shadow)}
.empty-state p{margin-top:8px;font-size:14px}
.loading{text-align:center;padding:24px;color:var(--text-muted)}
.error-banner{background:#fef2f2;color:#991b1b;padding:10px 14px;border-radius:var(--radius);margin-bottom:16px;font-size:14px;display:none}
.modal{display:none;position:fixed;inset:0;background:rgba(0,0,0,.4);z-index:100;align-items:center;justify-content:center}
.modal.open{display:flex}
.modal-content{background:var(--surface);border-radius:var(--radius);padding:24px;width:90%;max-width:500px;max-height:90vh;overflow-y:auto;box-shadow:0 8px 32px rgba(0,0,0,.15)}
.modal-content h2{margin-bottom:16px;font-size:18px}
.modal-close{float:right;font-size:24px;cursor:pointer;color:var(--text-muted);line-height:1}
.form-group{margin-bottom:14px}
.form-group label{display:block;font-size:13px;font-weight:600;color:var(--text);margin-bottom:4px}
.form-group input,.form-group textarea{width:100%;padding:8px 12px;border:1px solid var(--border);border-radius:var(--radius);font-size:14px;font-family:inherit;color:var(--text);background:var(--surface);transition:border-color .15s}
.form-group input:focus,.form-group textarea:focus{outline:none;border-color:var(--primary);box-shadow:0 0 0 3px rgba(59,130,246,.15)}
.form-group textarea{min-height:60px;resize:vertical;font-family:monospace;font-size:13px}
.form-group input[type=checkbox]{width:auto;margin-right:6px}
.form-group .checkbox-label{display:flex;align-items:center;gap:6px;cursor:pointer}
.form-group .checkbox-label input{margin:0}
.form-buttons{display:flex;gap:8px;justify-content:flex-end;margin-top:16px}
.form-error{color:var(--down);font-size:13px;margin-top:4px;display:none}
.badge{padding:2px 8px;border-radius:999px;font-size:11px;font-weight:600;color:#fff;display:inline-block}
.text-muted{color:var(--text-muted);font-size:13px}
@media(max-width:768px){header{flex-wrap:wrap;gap:12px;padding:12px 16px}
main{padding:12px}
table{font-size:13px}
th,td{padding:8px 10px}
.url-cell{max-width:160px}
.toolbar{flex-direction:column;align-items:stretch;gap:8px}
}
</style>
</head>
<body>
<header>
<h1>GoMon <span>Uptime Monitor</span></h1>
<nav>
<button class="tab-btn active" data-tab="websites" onclick="switchTab('websites')">Websites</button>
<button class="tab-btn" data-tab="deliveries" onclick="switchTab('deliveries')">Webhook Deliveries</button>
</nav>
</header>
<main>
<div id="error-banner" class="error-banner"></div>

<section id="tab-websites" class="tab-content active">
<div class="toolbar">
<h2>Monitored Websites</h2>
<button class="btn btn-primary" onclick="openAddModal()">+ Add Website</button>
</div>
<div id="websites-loading" class="loading">Loading websites...</div>
<div id="websites-table-wrapper" style="display:none">
<table>
<thead>
<tr><th>Status</th><th>URL</th><th>Frequency</th><th>Response</th><th>Code</th><th>Last Checked</th><th>Actions</th></tr>
</thead>
<tbody id="websites-tbody"></tbody>
</table>
</div>
<div id="websites-empty" class="empty-state" style="display:none">
<strong>No websites monitored</strong>
<p>Add a website to start monitoring its uptime.</p>
</div>
</section>

<section id="tab-deliveries" class="tab-content">
<div class="toolbar"><h2>Webhook Deliveries</h2></div>
<div id="deliveries-loading" class="loading">Loading deliveries...</div>
<div id="deliveries-table-wrapper" style="display:none">
<table>
<thead>
<tr><th>Event ID</th><th>Website</th><th>Webhook URL</th><th>Attempts</th><th>Status</th><th>Next / Delivered</th><th>Last Error</th></tr>
</thead>
<tbody id="deliveries-tbody"></tbody>
</table>
</div>
<div id="deliveries-empty" class="empty-state" style="display:none">
<strong>No webhook deliveries</strong>
<p>Webhook deliveries will appear here when website status changes trigger notifications.</p>
</div>
</section>
</main>

<div id="website-modal" class="modal">
<div class="modal-content">
<span class="modal-close" onclick="closeModal()">&times;</span>
<h2 id="modal-title">Add Website</h2>
<form id="website-form" onsubmit="saveWebsite(event)">
<input type="hidden" id="form-original-url">
<div class="form-group">
<label for="form-url">URL</label>
<input type="url" id="form-url" required placeholder="https://example.com">
</div>
<div class="form-group">
<label for="form-frequency">Check Frequency (seconds)</label>
<input type="number" id="form-frequency" min="5" required placeholder="300">
</div>
<div class="form-group">
<label for="form-headers">Custom Headers (JSON, optional)</label>
<textarea id="form-headers" placeholder='{"Authorization": "Bearer token"}'></textarea>
</div>
<div class="form-group">
<label class="checkbox-label">
<input type="checkbox" id="form-webhook-enabled" onchange="toggleWebhookFields()"> Webhook Enabled
</label>
</div>
<div id="webhook-fields" style="display:none">
<div class="form-group">
<label for="form-webhook-url">Webhook URL</label>
<input type="url" id="form-webhook-url" placeholder="https://hooks.example.com/notify">
</div>
<div class="form-group">
<label for="form-payload-template">Payload Template (optional)</label>
<textarea id="form-payload-template" placeholder='{"msg": "{{currentStatus}}"}'></textarea>
</div>
</div>
<div id="form-error" class="form-error"></div>
<div class="form-buttons">
<button type="submit" class="btn btn-primary" id="form-submit-btn">Save</button>
<button type="button" class="btn btn-outline" onclick="closeModal()">Cancel</button>
</div>
</form>
</div>
</div>

<div id="delete-modal" class="modal">
<div class="modal-content">
<h2>Delete Website</h2>
<p style="margin-bottom:16px;color:var(--text-muted)">Are you sure you want to delete <strong id="delete-url" style="color:var(--text)"></strong>?</p>
<div class="form-buttons">
<button class="btn btn-danger" id="delete-confirm-btn" onclick="confirmDelete()">Delete</button>
<button class="btn btn-outline" onclick="closeDeleteModal()">Cancel</button>
</div>
</div>
</div>

<script>
let websites = [];
let deleteTarget = null;

function showError(msg) {
const banner = document.getElementById('error-banner');
banner.textContent = msg;
banner.style.display = 'block';
setTimeout(() => { banner.style.display = 'none'; }, 5000);
}

function switchTab(name) {
document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
document.querySelector('.tab-btn[data-tab="' + name + '"]').classList.add('active');
document.getElementById('tab-' + name).classList.add('active');
}

function formatTime(ts) {
if (!ts || ts <= 0) return '-';
const d = new Date(ts * 1000);
const now = Date.now();
const diff = Math.floor((now - d.getTime()) / 1000);
if (diff < 60) return diff + 's ago';
if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
return d.toLocaleDateString();
}

function formatFreq(sec) {
if (!sec) return '-';
if (sec < 60) return sec + 's';
if (sec < 3600) return Math.floor(sec / 60) + 'm';
return Math.floor(sec / 3600) + 'h';
}

function statusClass(status) {
return 'status-' + (status || 'unknown').toLowerCase();
}

function statusDot(status) {
return '<span class="status-dot ' + statusClass(status) + '"></span>' + (status || 'unknown');
}

function deliveryStatus(d) {
if (d.deliveredAt > 0) return '<span class="badge" style="background:var(--up)">Delivered</span>';
if (d.deliveredAt === -1) return '<span class="badge" style="background:var(--down)">Failed</span>';
return '<span class="badge" style="background:var(--primary)">Pending</span>';
}

function escapeHtml(s) {
if (!s) return '';
return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

async function loadWebsites() {
const loading = document.getElementById('websites-loading');
const wrapper = document.getElementById('websites-table-wrapper');
const empty = document.getElementById('websites-empty');
const tbody = document.getElementById('websites-tbody');

try {
const res = await fetch('/api/websites');
const data = await res.json();
if (!data.success) throw new Error(data.error || 'Failed to load');
websites = data.data || [];

if (websites.length === 0) {
loading.style.display = 'none';
wrapper.style.display = 'none';
empty.style.display = 'block';
return;
}

loading.style.display = 'none';
empty.style.display = 'none';
wrapper.style.display = 'block';

tbody.innerHTML = websites.map(w => {
const resp = w.responseTime >= 0 ? w.responseTime + 'ms' : '-';
const code = w.statusCode > 0 ? w.statusCode : '-';
return '<tr><td>' + statusDot(w.status) + '</td><td class="url-cell" title="' + escapeHtml(w.url) + '">' + escapeHtml(w.url) + '</td><td>' + formatFreq(w.frequency) + '</td><td>' + resp + '</td><td>' + code + '</td><td>' + formatTime(w.lastCheckedAt) + '</td><td class="actions"><button class="btn btn-outline btn-sm" onclick="openEditModal(\'' + escapeHtml(w.url) + '\')">Edit</button><button class="btn btn-danger btn-sm" onclick="openDeleteModal(\'' + escapeHtml(w.url) + '\')">Del</button></td></tr>';
}).join('');
} catch (err) {
showError('Failed to load websites: ' + err.message);
loading.style.display = 'none';
wrapper.style.display = 'none';
empty.style.display = 'block';
empty.innerHTML = '<strong>Error loading websites</strong><p>' + escapeHtml(err.message) + '</p>';
}
}

async function loadDeliveries() {
const loading = document.getElementById('deliveries-loading');
const wrapper = document.getElementById('deliveries-table-wrapper');
const empty = document.getElementById('deliveries-empty');
const tbody = document.getElementById('deliveries-tbody');

try {
const res = await fetch('/api/webhook-deliveries');
const data = await res.json();
if (!data.success) throw new Error(data.error || 'Failed to load');

const deliveries = data.data || [];

if (deliveries.length === 0) {
loading.style.display = 'none';
wrapper.style.display = 'none';
empty.style.display = 'block';
return;
}

loading.style.display = 'none';
empty.style.display = 'none';
wrapper.style.display = 'block';

tbody.innerHTML = deliveries.map(d => {
const nextAt = d.deliveredAt > 0 ? formatTime(d.deliveredAt) : formatTime(d.nextAttemptAt);
return '<tr><td style="font-family:monospace;font-size:12px">' + escapeHtml(d.eventId) + '</td><td class="url-cell" title="' + escapeHtml(d.websiteUrl) + '">' + escapeHtml(d.websiteUrl) + '</td><td class="url-cell" title="' + escapeHtml(d.webhookUrl) + '">' + escapeHtml(d.webhookUrl) + '</td><td>' + d.attemptCount + '/' + d.maxAttempts + '</td><td>' + deliveryStatus(d) + '</td><td>' + nextAt + '</td><td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:var(--text-muted)">' + escapeHtml(d.lastError) + '</td></tr>';
}).join('');
} catch (err) {
showError('Failed to load deliveries: ' + err.message);
loading.style.display = 'none';
wrapper.style.display = 'none';
empty.style.display = 'block';
empty.innerHTML = '<strong>Error loading deliveries</strong><p>' + escapeHtml(err.message) + '</p>';
}
}

function openAddModal() {
document.getElementById('modal-title').textContent = 'Add Website';
document.getElementById('form-original-url').value = '';
document.getElementById('form-url').value = '';
document.getElementById('form-url').disabled = false;
document.getElementById('form-frequency').value = '300';
document.getElementById('form-headers').value = '';
document.getElementById('form-webhook-enabled').checked = false;
document.getElementById('form-webhook-url').value = '';
document.getElementById('form-payload-template').value = '';
document.getElementById('form-error').style.display = 'none';
document.getElementById('form-submit-btn').textContent = 'Save';
document.getElementById('webhook-fields').style.display = 'none';
document.getElementById('website-modal').classList.add('open');
}

function openEditModal(url) {
const w = websites.find(s => s.url === url);
if (!w) return;

document.getElementById('modal-title').textContent = 'Edit Website';
document.getElementById('form-original-url').value = w.url;
document.getElementById('form-url').value = w.url;
document.getElementById('form-url').disabled = true;
document.getElementById('form-frequency').value = w.frequency || 300;
document.getElementById('form-headers').value = w.customHeaders && Object.keys(w.customHeaders).length ? JSON.stringify(w.customHeaders, null, 2) : '';
document.getElementById('form-webhook-enabled').checked = w.webhookEnabled || false;
document.getElementById('form-webhook-url').value = w.webhookUrl || '';
document.getElementById('form-payload-template').value = w.webhookPayloadTemplate || '';
document.getElementById('form-error').style.display = 'none';
document.getElementById('form-submit-btn').textContent = 'Update';
toggleWebhookFields();
document.getElementById('website-modal').classList.add('open');
}

function closeModal() {
document.getElementById('website-modal').classList.remove('open');
}

function toggleWebhookFields() {
const enabled = document.getElementById('form-webhook-enabled').checked;
document.getElementById('webhook-fields').style.display = enabled ? 'block' : 'none';
}

async function saveWebsite(e) {
e.preventDefault();
const errEl = document.getElementById('form-error');
errEl.style.display = 'none';

const originalUrl = document.getElementById('form-original-url').value;
const url = document.getElementById('form-url').value.trim();
const frequency = parseInt(document.getElementById('form-frequency').value);
const headersRaw = document.getElementById('form-headers').value.trim();
const webhookEnabled = document.getElementById('form-webhook-enabled').checked;
const webhookUrl = document.getElementById('form-webhook-url').value.trim();
const payloadTemplate = document.getElementById('form-payload-template').value.trim();

let customHeaders = {};
if (headersRaw) {
try { customHeaders = JSON.parse(headersRaw); } catch(_) {
errEl.textContent = 'Custom Headers must be valid JSON';
errEl.style.display = 'block';
return;
}
}

const isEdit = !!originalUrl;

try {
let res;
if (isEdit) {
res = await fetch('/api/websites?websiteUrl=' + encodeURIComponent(originalUrl), {
method: 'PUT',
headers: {'Content-Type': 'application/json'},
body: JSON.stringify({
frequency: frequency,
customHeaders: customHeaders,
webhookEnabled: webhookEnabled,
webhookUrl: webhookEnabled ? webhookUrl : '',
webhookPayloadTemplate: webhookEnabled ? payloadTemplate : ''
})
});
} else {
res = await fetch('/api/websites', {
method: 'POST',
headers: {'Content-Type': 'application/json'},
body: JSON.stringify({
url: url,
frequency: frequency,
customHeaders: customHeaders,
webhookEnabled: webhookEnabled,
webhookUrl: webhookEnabled ? webhookUrl : '',
webhookPayloadTemplate: webhookEnabled ? payloadTemplate : ''
})
});
}

const data = await res.json();
if (!data.success) throw new Error(data.error || 'Request failed');

closeModal();
await loadWebsites();
} catch (err) {
errEl.textContent = err.message;
errEl.style.display = 'block';
}
}

function openDeleteModal(url) {
deleteTarget = url;
document.getElementById('delete-url').textContent = url;
document.getElementById('delete-modal').classList.add('open');
}

function closeDeleteModal() {
deleteTarget = null;
document.getElementById('delete-modal').classList.remove('open');
}

async function confirmDelete() {
if (!deleteTarget) return;
try {
const res = await fetch('/api/websites?websiteUrl=' + encodeURIComponent(deleteTarget), { method: 'DELETE' });
const data = await res.json();
if (!data.success) throw new Error(data.error || 'Delete failed');
closeDeleteModal();
await loadWebsites();
} catch (err) {
showError('Delete failed: ' + err.message);
closeDeleteModal();
}
}

document.addEventListener('DOMContentLoaded', () => {
loadWebsites();
loadDeliveries();
setInterval(() => {
if (document.getElementById('tab-websites').classList.contains('active')) loadWebsites();
}, 15000);
setInterval(() => {
if (document.getElementById('tab-deliveries').classList.contains('active')) loadDeliveries();
}, 15000);
});
</script>
</body>
</html>`

/* IAG DMS — API client for index.html prototype */
(function (global) {
  function apiBase() {
    if (global.DMS_API_BASE) return String(global.DMS_API_BASE).replace(/\/$/, '');
    const p = window.location.pathname || '';
    if (p.startsWith('/api/v1/dms')) return '/api/v1/dms';
    return '';
  }
  const API_BASE = apiBase();

  async function api(path, opts) {
    const url = API_BASE + '/v1' + path;
    const headers = { Accept: 'application/json', ...(opts && opts.headers) };
    if (opts && opts.body) headers['Content-Type'] = 'application/json';
    const token = global.DMS_AUTH_TOKEN;
    if (token) headers.Authorization = 'Bearer ' + token;
    const res = await fetch(url, {
      method: (opts && opts.method) || 'GET',
      headers,
      body: opts && opts.body ? JSON.stringify(opts.body) : undefined,
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || res.statusText || 'request failed');
    }
    if (res.status === 204) return null;
    return res.json();
  }

  function fmtUgx(n) {
    if (n >= 1e6) return 'UGX ' + (n / 1e6).toFixed(1) + 'M';
    if (n >= 1e3) return 'UGX ' + Math.round(n / 1e3) + 'k';
    return 'UGX ' + n;
  }

  function pillClass(channel) {
    const c = (channel || '').toLowerCase();
    if (c.includes('horeca')) return 'info';
    if (c === 'mt' || c.includes('mt')) return 'dms';
    return 'ok';
  }

  async function loadOverview() {
    const data = await api('/overview');
    document.querySelectorAll('.kpi').forEach((el, i) => {
      const k = data.kpis && data.kpis[i];
      if (!k) return;
      const val = el.querySelector('.kpi-val');
      if (val) val.innerHTML = k.value + (k.unit ? '<small>' + k.unit + '</small>' : '');
      const row = el.querySelector('.kpi-row');
      if (row && k.trend) {
        const trend = row.querySelector('.kpi-trend');
        if (trend) trend.textContent = k.trend;
      }
    });
  }

  async function loadOutletsTable() {
    const page = document.getElementById('page-outlets');
    if (!page) return;
    const tbody = page.querySelector('table.t tbody');
    if (!tbody) return;
    const res = await api('/outlets?limit=50');
    const items = res.items || [];
    tbody.innerHTML = items.map(o => {
      const initials = (o.name || '??').split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase();
      const st = o.status === 'active' ? 'ok' : o.status === 'watch' ? 'warn' : 'err';
      return '<tr data-outlet-id="' + o.id + '">' +
        '<td class="t-id"><b>' + o.id + '</b></td>' +
        '<td><div class="cell-ent"><div class="av" style="background:#D97706">' + initials + '</div>' +
        '<div class="nm">' + o.name + '<small>' + (o.address || '') + '</small></div></div></td>' +
        '<td><span class="pill ' + pillClass(o.channel) + '">' + o.channel + '</span></td>' +
        '<td class="mono">' + o.distributorId + '</td>' +
        '<td class="mono">' + o.beatId + '</td>' +
        '<td class="num strong">' + fmtUgx(o.qtdValueUgx || 0) + '</td>' +
        '<td class="num">' + (o.frequency || '') + '</td>' +
        '<td><span class="pill ok">' + (o.score || '') + '</span></td>' +
        '<td><span class="pill ' + st + '">●  ' + o.status + '</span></td></tr>';
    }).join('');
  }

  async function loadNotifications() {
    try {
      const res = await api('/notifications');
      const items = res.items || [];
      const pop = document.getElementById('popNotify');
      if (!pop || !items.length) return;
      const rows = pop.querySelectorAll('.pop-row');
      items.slice(0, 6).forEach((a, i) => {
        if (!rows[i]) return;
        const b = rows[i].querySelector('.bd b');
        const s = rows[i].querySelector('.bd small');
        if (b) b.textContent = a.title;
        if (s) s.textContent = a.detail;
      });
    } catch (_) { /* optional */ }
  }

  function modalField(modalId, selector) {
    const m = document.getElementById(modalId);
    if (!m) return '';
    const el = m.querySelector(selector);
    return el ? (el.value || el.textContent || '').trim() : '';
  }

  function selectedChannel(modalId) {
    const on = document.querySelector('#' + modalId + ' .rcard.on b');
    return on ? on.textContent.trim() : 'GT';
  }

  function parseDistributorId(selectVal) {
    const m = (selectVal || '').match(/^(D-\d+)/);
    return m ? m[1] : 'D-001';
  }

  async function saveOutlet() {
    const modal = 'mAddOutlet';
    const body = {
      name: modalField(modal, 'input[placeholder*="Capital"]') || modalField(modal, 'input[type="text"]'),
      address: modalField(modal, 'input[value*="Plot"]') || modalField(modal, 'input[placeholder]'),
      channel: selectedChannel(modal),
      distributorId: parseDistributorId(modalField(modal, 'select')),
      beatId: 'B-04',
      lat: 0.34288,
      lng: 32.60241,
    };
    const out = await api('/outlets', { method: 'POST', body });
    closeModal(modal);
    toast('Outlet saved · ' + out.id, out.name + ' · synced to reps', 'ok');
    await loadOutletsTable();
    return out;
  }

  async function checkInSubmit() {
    const outletSel = modalField('mCheckin', 'select');
    const outletId = (outletSel.match(/^(OUT-\d+)/) || [])[1] || 'OUT-00214';
    const ci = await api('/field/check-ins', {
      method: 'POST',
      body: { repId: 'FF-04', outletId, lat: 0.34288, lng: 32.60241 },
    });
    closeModal('mCheckin');
    toast('Checked-in', 'Timer running · GPS confirmed', 'dms');
    return ci;
  }

  async function submitVisitReport() {
    const outcomeEl = document.querySelector('#mVisitReport .rcard.on b');
    const outcome = outcomeEl ? outcomeEl.textContent.trim() : 'Productive';
    const notes = modalField('mVisitReport', 'textarea') || '';
    const v = await api('/field/visit-reports', {
      method: 'POST',
      body: { repId: 'FF-04', outletId: 'OUT-00214', outcome, notes, lat: 0.34288, lng: 32.60241 },
    });
    closeModal('mVisitReport');
    toast('Visit report logged', v.id + ' · synced to manager', 'ok');
    return v;
  }

  async function generateInvoice() {
    const dist = parseDistributorId(modalField('mNewInvoice', 'select'));
    const inv = await api('/invoices', {
      method: 'POST',
      body: {
        distributorId: dist,
        amountUgx: 13920979,
        dueDate: new Date(Date.now() + 30 * 864e5).toISOString().slice(0, 10),
        orderId: 'SO-19848',
      },
    });
    closeModal('mNewInvoice');
    toast('Invoice ' + inv.id + ' generated', 'EFRIS validated · queued for email', 'ok');
    return inv;
  }

  const _goPage = global.goPage;
  global.goPage = function (id, el) {
    if (typeof _goPage === 'function') _goPage(id, el);
    if (id === 'overview') loadOverview().catch(e => toast('API', e.message, 'err'));
    if (id === 'outlets') loadOutletsTable().catch(e => toast('API', e.message, 'err'));
  };

  global.DMS = {
    api, loadOverview, loadOutletsTable, loadNotifications,
    saveOutlet, checkInSubmit, submitVisitReport, generateInvoice,
  };

  global.saveOutlet = function () {
    saveOutlet().catch(e => toast('Save outlet failed', e.message, 'err'));
  };
  global.checkInSubmit = function () {
    checkInSubmit().catch(e => toast('Check-in failed', e.message, 'err'));
  };
  global.submitVisitReport = function () {
    submitVisitReport().catch(e => toast('Visit report failed', e.message, 'err'));
  };
  global.generateInvoice = function () {
    generateInvoice().catch(e => toast('Invoice failed', e.message, 'err'));
  };

  document.addEventListener('DOMContentLoaded', function () {
    api('/bootstrap').then(b => {
      console.log('DMS API connected ·', b.service, b.version);
    }).catch(() => console.warn('DMS API offline — static UI only'));
    loadOverview().catch(() => {});
    loadNotifications().catch(() => {});
  });
})(window);

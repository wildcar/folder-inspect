/* folder-inspect web UI. Plain JS, no dependencies. Reads /api/report,
   lets the user browse findings, pick originals in duplicate groups, and
   assemble an action plan that /api/plan saves for `apply`. */
(() => {
  'use strict';

  const state = {
    d: null,          // report
    lang: 'ru',
    S: {},            // strings for the current language
    meta: {},
    view: 'summary',
    filter: '',
    sort: { key: 'size', dir: -1 },
    plan: new Map(),      // path -> action
    originals: new Map(), // group id -> path kept
    lastApply: null,      // result of the last /api/apply, shown until dismissed
    quarantine: null,     // batches from /api/quarantine
  };

  const $ = (sel, el = document) => el.querySelector(sel);
  const t = (k) => state.S[k] ?? k;
  const tf = (k, ...args) => {
    let i = 0;
    return t(k).replace(/%[sd]/g, () => String(args[i++] ?? ''));
  };
  const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

  const KB = 1024, MB = KB * 1024, GB = MB * 1024, TB = GB * 1024;
  function humanSize(n) {
    n = Number(n) || 0;
    const u = (v, k) => `${v.toFixed(1)} ${t('unit.' + k)}`;
    if (n >= TB) return u(n / TB, 'TB');
    if (n >= GB) return u(n / GB, 'GB');
    if (n >= MB) return u(n / MB, 'MB');
    if (n >= KB) return u(n / KB, 'KB');
    return `${n} ${t('unit.B')}`;
  }
  const fmtTime = (s) => {
    if (!s) return '';
    const d = new Date(s);
    if (isNaN(d)) return '';
    const p = (x) => String(x).padStart(2, '0');
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
  };
  const pct = (r) => `${Math.round((r || 0) * 100)}%`;
  const sep = () => (state.meta.os === 'windows' ? '\\' : '/');

  // Absolute path split into root + relative for display.
  function pathHtml(path, rel) {
    if (!rel) return `<span class="rel">${esc(path)}</span>`;
    const root = path.slice(0, path.length - rel.length);
    return `<span class="root">${esc(root)}</span><span class="rel">${esc(rel.split('/').join(sep()))}</span>`;
  }

  const FLAT = ['oversize', 'archive', 'distributive', 'junk', 'empty-dir', 'empty-file'];
  const GROUPED = ['duplicate', 'dir-duplicate', 'dir-overlap'];
  const ORDER = ['oversize', 'archive', 'distributive', 'duplicate', 'dir-duplicate', 'dir-overlap', 'similar-name', 'junk', 'empty-dir', 'empty-file'];

  // ---------- data access ----------
  const findings = (cat) => state.d.findings.filter((f) => f.category === cat);
  const summaryOf = (cat) => state.d.summary.find((s) => s.category === cat);

  // ---------- plan ----------
  function planAdd(a) { state.plan.set(a.path, a); renderPlanBar(); }
  function planRemove(path) { state.plan.delete(path); renderPlanBar(); }
  function planToggleFinding(f, on) {
    if (on) planAdd({ op: 'quarantine', path: f.path, category: f.category, size: f.size, is_dir: !!f.is_dir });
    else planRemove(f.path);
  }
  function originalOf(kind, g) {
    return state.originals.get(kind + ':' + g.id) || g.suggested;
  }
  function setOriginal(kind, g, path) {
    state.originals.set(kind + ':' + g.id, path);
    planRemove(path); // the original can never be in the plan
    // copies already planned now point to the new original
    const members = kind === 'dir' ? g.dirs : g.files;
    for (const m of members) {
      const a = state.plan.get(m.path);
      if (a) a.original = path;
    }
  }
  function planToggleCopy(kind, g, m, on) {
    if (on) planAdd({ op: kind === 'dir' ? 'quarantine-dir' : 'quarantine-duplicate', path: m.path, original: originalOf(kind, g), group: g.id, size: g.size, is_dir: kind === 'dir', category: kind === 'dir' ? 'dir-duplicate' : 'duplicate' });
    else planRemove(m.path);
  }
  function renderPlanBar() {
    const n = state.plan.size;
    let size = 0;
    for (const a of state.plan.values()) size += a.size || 0;
    $('#planInfo').innerHTML = n ? `<b>${esc(t('ui.plan'))}:</b> ${esc(tf('ui.plan_items', n, humanSize(size)))}` : `<span class="muted">${esc(t('ui.plan_empty'))}</span>`;
    for (const id of ['planSave', 'planDry', 'planApply']) $('#' + id).disabled = n === 0;
    $('#planClear').style.visibility = n ? 'visible' : 'hidden';
  }
  async function savePlan() {
    const actions = [...state.plan.values()];
    $('#planStatus').textContent = '…';
    try {
      const r = await fetch('/api/plan', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ actions }) });
      const txt = await r.text();
      if (!r.ok) throw new Error(txt);
      const j = JSON.parse(txt);
      $('#planStatus').textContent = tf('ui.plan_saved', j.path);
    } catch (e) {
      $('#planStatus').textContent = tf('ui.plan_error', e.message);
    }
  }
  async function rescan() {
    const btn = $('#rescanBtn');
    btn.disabled = true;
    $('#rescanStatus').textContent = t('ui.rescanning');
    try {
      const r = await fetch('/api/rescan', { method: 'POST' });
      const txt = await r.text();
      if (!r.ok) throw new Error(txt);
      const j = JSON.parse(txt);
      await load(state.lang);
      $('#rescanStatus').textContent = tf('ui.rescan_done', j.report_path.split(/[\\/]/).pop());
    } catch (e) {
      $('#rescanStatus').textContent = e.message;
    } finally {
      btn.disabled = false;
    }
  }
  async function applyPlan(dry) {
    const actions = [...state.plan.values()];
    let size = 0;
    for (const a of actions) size += a.size || 0;
    if (!dry && !confirm(tf('ui.apply_confirm', actions.length, humanSize(size)))) return;
    $('#planStatus').textContent = t('ui.apply_running');
    for (const id of ['planDry', 'planApply', 'planSave']) $('#' + id).disabled = true;
    try {
      const r = await fetch('/api/apply', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ actions, dry_run: dry, lang: state.lang }) });
      const txt = await r.text();
      if (!r.ok) throw new Error(txt);
      state.lastApply = JSON.parse(txt);
      state.view = 'apply-result';
      $('#planStatus').textContent = '';
      if (!dry) {
        state.plan.clear();
        state.originals.clear();
        render();
        await rescan(); // the report must reflect the moved files
      } else {
        render();
      }
    } catch (e) {
      $('#planStatus').textContent = tf('ui.plan_error', e.message);
      renderPlanBar();
    }
  }
  function renderApplyResult(main) {
    const a = state.lastApply;
    if (!a) { state.view = 'summary'; return renderMain(); }
    let html = `<h2>${esc(t('ui.apply_result'))} <span class="muted">${esc(tf('ui.apply_summary', a.moved, a.stubs, a.problems.length))}</span></h2>`;
    if (a.dry_run) html += `<p class="hint">${esc(t('ui.apply_dry_note'))}</p>`;
    else if (a.manifests.length) {
      html += `<p class="hint">${esc(t('ui.apply_manifest'))}</p><ul class="roots">${a.manifests.map((m) => `<li>${esc(m)}</li>`).join('')}</ul>`;
      html += `<p class="hint"><code>${esc(tf('ui.apply_restore', a.manifests[0]))}</code></p>`;
    }
    if (a.problems.length) {
      html += `<h3>${esc(t('ui.problems'))} (${a.problems.length})</h3><table><tr><th>${esc(t('col.path'))}</th><th>${esc(t('col.error'))}</th></tr>`;
      for (const p of a.problems) html += `<tr><td class="path">${esc(p.path)}</td><td>${esc(p.error)}</td></tr>`;
      html += '</table>';
    }
    if (a.entries.length) {
      html += `<table><tr><th>${esc(t('ui.col_action'))}</th><th>${esc(t('ui.col_from'))}</th><th>${esc(t('ui.col_to'))}</th><th>${esc(t('ui.col_stub'))}</th></tr>`;
      for (const e of a.entries) html += `<tr><td class="tag">${esc(t('op.' + e.op))}</td><td class="path">${esc(e.from)}</td><td class="path">${esc(e.to)}</td><td class="path">${esc(e.stub ? e.stub.split(/[\\/]/).pop() : '')}</td></tr>`;
      html += '</table>';
    }
    html += `<p><button class="btn" id="backBtn">${esc(t('ui.apply_back'))}</button></p>`;
    main.innerHTML = html;
    $('#backBtn', main).addEventListener('click', () => { state.view = 'summary'; state.lastApply = null; render(); });
  }
  async function reveal(path) {
    try { await fetch('/api/reveal', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ path }) }); } catch (_) { /* local only */ }
  }

  // ---------- rendering ----------
  function render() {
    const d = state.d;
    document.documentElement.lang = state.lang;
    $('#version').textContent = state.meta.version || '';
    $('#scanline').textContent = tf('ui.scanned', fmtTime(d.finished), d.roots.join(', '));
    $('#exportLabel').textContent = t('ui.export') + ':';
    for (const f of ['csv', 'xlsx', 'html']) $('#export' + f[0].toUpperCase() + f.slice(1)).href = `/api/export?format=${f}&lang=${state.lang}`;
    $('#langBtn').textContent = t('ui.lang_switch');
    $('#rescanBtn').textContent = t('ui.rescan');
    $('#planSave').textContent = t('ui.plan_save');
    $('#planDry').textContent = t('ui.apply_dry');
    $('#planApply').textContent = t('ui.apply');
    $('#planClear').textContent = t('ui.plan_clear');
    renderNav();
    renderMain();
    renderPlanBar();
  }

  function renderNav() {
    const d = state.d;
    const items = [];
    const item = (id, label, count, cls = '') => items.push(`<div class="item ${cls} ${state.view === id ? 'active' : ''}" data-view="${id}"><span>${esc(label)}</span><span class="count">${count ?? ''}</span></div>`);
    item('summary', t('ui.summary'), '');
    items.push(`<div class="head">${esc(t('scan.summary'))}</div>`);
    for (const c of ORDER) {
      const s = summaryOf(c);
      if (!s) continue;
      item(c, t('cat.' + c), `${s.count} · ${humanSize(s.size)}`);
    }
    items.push('<div class="sep"></div>');
    item('quarantine', t('ui.quarantine'), state.quarantine ? state.quarantine.filter((b) => b.pending > 0).length : '');
    item('top-files', t('scan.top_files'), d.top_files.length);
    item('top-dirs', t('scan.top_dirs'), d.top_dirs.length);
    if (d.errors.length) item('errors', t('html.errors'), d.errors.length);
    if (d.skipped.length) item('skipped', t('html.skipped'), d.skipped.length);
    $('#nav').innerHTML = items.join('');
    $('#nav').querySelectorAll('.item').forEach((el) => el.addEventListener('click', () => { state.view = el.dataset.view; state.filter = ''; render(); }));
  }

  function renderMain() {
    const v = state.view;
    const main = $('#main');
    if (v === 'summary') return renderSummary(main);
    if (FLAT.includes(v)) return renderFlat(main, v);
    if (v === 'duplicate') return renderGroups(main, 'file');
    if (v === 'dir-duplicate') return renderGroups(main, 'dir');
    if (v === 'dir-overlap') return renderOverlaps(main);
    if (v === 'similar-name') return renderNames(main);
    if (v === 'quarantine') return renderQuarantine(main);
    if (v === 'top-files' || v === 'top-dirs') return renderTop(main, v);
    if (v === 'errors') return renderErrors(main);
    if (v === 'skipped') return renderSkipped(main);
    if (v === 'apply-result') return renderApplyResult(main);
    main.innerHTML = '';
  }

  function renderSummary(main) {
    const d = state.d, st = d.stats;
    const stats = [[st.files, t('sum.files')], [st.dirs, t('sum.dirs')], [humanSize(st.total_size), t('sum.total')]];
    if (st.hashed) stats.push([`${st.hashed} / ${humanSize(st.hashed_bytes)}`, t('sum.hashed')]);
    if (st.errors) stats.push([st.errors, t('sum.errors')]);
    let html = `<h2>${esc(t('ui.summary'))} <span class="muted">${esc(state.meta.report_path)}</span></h2>`;
    html += `<ul class="roots">${d.roots.map((r) => `<li>${esc(r)}</li>`).join('')}</ul>`;
    html += `<div class="stats">${stats.map(([v, k]) => `<div><b>${esc(v)}</b><span class="muted">${esc(k)}</span></div>`).join('')}</div>`;
    if (!d.summary.length) { html += `<p class="empty">${esc(t('scan.nothing'))}</p>`; main.innerHTML = html; return; }
    html += `<table><tr><th>${esc(t('scan.category'))}</th><th class="num">${esc(t('scan.count'))}</th><th class="num">${esc(t('scan.size'))}</th></tr>`;
    for (const s of d.summary) html += `<tr class="link" data-view="${s.category}"><td><a href="#">${esc(t('cat.' + s.category))}</a></td><td class="num">${s.count}</td><td class="num">${humanSize(s.size)}</td></tr>`;
    html += '</table>';
    html += `<p class="hint" style="margin-top:14px">${esc(t('ui.plan_hint'))}</p>`;
    if (d.skipped.length) html += `<p class="hint">${esc(t('html.skipped'))}: ${d.skipped.length}</p>`;
    main.innerHTML = html;
    main.querySelectorAll('tr.link').forEach((tr) => tr.addEventListener('click', (e) => { e.preventDefault(); state.view = tr.dataset.view; render(); }));
  }

  function toolbar(extra = '') {
    return `<div class="toolbar"><input type="search" id="filter" placeholder="${esc(t('ui.filter'))}" value="${esc(state.filter)}">${extra}</div>`;
  }
  function bindFilter(main, rerender) {
    const inp = $('#filter', main);
    if (!inp) return;
    inp.addEventListener('input', () => { state.filter = inp.value; rerender(); });
  }
  const matches = (path) => !state.filter || path.toLowerCase().includes(state.filter.toLowerCase());

  function sortRows(rows, keyFn) {
    const { key, dir } = state.sort;
    rows.sort((a, b) => {
      const x = keyFn(a, key), y = keyFn(b, key);
      if (x === y) return 0;
      return (x > y ? 1 : -1) * dir;
    });
  }
  function th(key, label, cls = '') {
    const sorted = state.sort.key === key ? `sorted ${state.sort.dir > 0 ? 'asc' : ''}` : '';
    return `<th class="${cls} ${sorted}" data-key="${key}">${esc(label)}</th>`;
  }
  function bindSort(main, rerender) {
    main.querySelectorAll('th[data-key]').forEach((el) => el.addEventListener('click', () => {
      const k = el.dataset.key;
      if (state.sort.key === k) state.sort.dir = -state.sort.dir; else state.sort = { key: k, dir: k === 'path' ? 1 : -1 };
      rerender();
    }));
  }
  function qualifier(f) {
    const parts = [];
    if (f.rule) parts.push(f.rule);
    if (f.threshold) parts.push('> ' + humanSize(f.threshold));
    if (f.detail) parts.push(t('detail.' + f.detail));
    return parts.join(', ');
  }

  function renderFlat(main, cat) {
    const all = findings(cat);
    const rows = all.filter((f) => matches(f.path));
    sortRows(rows, (f, k) => k === 'path' ? f.path.toLowerCase() : k === 'mtime' ? f.mtime : f.size);
    const s = summaryOf(cat);
    const visibleSelected = rows.length && rows.every((f) => state.plan.has(f.path));
    let html = `<h2>${esc(t('cat.' + cat))} <span class="muted">${s.count} · ${humanSize(s.size)}</span></h2>`;
    html += toolbar(`<button class="btn small" id="selAll">${esc(visibleSelected ? t('ui.clear_visible') : t('ui.select_visible'))}</button><span class="muted">${rows.length}/${all.length}</span>`);
    html += `<table><tr><th class="chk"></th>${th('size', t('col.size'), 'num')}${th('path', t('col.path'))}<th></th>${th('mtime', t('col.mtime'))}<th class="act"></th></tr>`;
    for (const f of rows) {
      const sel = state.plan.has(f.path);
      html += `<tr class="${sel ? 'selected' : ''}" data-path="${esc(f.path)}"><td class="chk"><input type="checkbox" ${sel ? 'checked' : ''} title="${esc(t('ui.to_quarantine'))}"></td><td class="num">${humanSize(f.size)}</td><td class="path">${pathHtml(f.path, f.rel)}${f.is_dir ? ` <span class="muted">${sep()}</span>` : ''}</td><td class="tag">${esc(qualifier(f))}</td><td class="tag">${fmtTime(f.mtime)}</td><td class="act"><button class="reveal" title="${esc(t('ui.reveal'))}">📂</button></td></tr>`;
    }
    if (!rows.length) html += `<tr><td colspan="6" class="empty">${esc(t('ui.no_items'))}</td></tr>`;
    html += '</table>';
    main.innerHTML = html;
    const byPath = new Map(all.map((f) => [f.path, f]));
    main.querySelectorAll('tr[data-path]').forEach((tr) => {
      const f = byPath.get(tr.dataset.path);
      $('input', tr).addEventListener('change', (e) => { planToggleFinding(f, e.target.checked); tr.classList.toggle('selected', e.target.checked); });
      $('.reveal', tr).addEventListener('click', () => reveal(f.path));
    });
    $('#selAll', main).addEventListener('click', () => { for (const f of rows) planToggleFinding(f, !visibleSelected); renderMain(); });
    bindFilter(main, () => renderFlat(main, cat));
    bindSort(main, () => renderFlat(main, cat));
    $('#filter', main).focus();
  }

  function renderGroups(main, kind) {
    const groups = kind === 'dir' ? state.d.dir_duplicates : state.d.duplicates;
    const cat = kind === 'dir' ? 'dir-duplicate' : 'duplicate';
    const s = summaryOf(cat);
    const visible = groups.filter((g) => (kind === 'dir' ? g.dirs : g.files).some((m) => matches(m.path)));
    let html = `<h2>${esc(t('cat.' + cat))} <span class="muted">${esc(tf('dup.header', '', s.count, humanSize(s.size)).replace(/^:\s*/, ''))}</span></h2>`;
    html += `<p class="hint">${esc(t('ui.group_hint'))}</p>`;
    html += toolbar(`<button class="btn small" id="allCopies">${esc(t('ui.select_all_copies'))}</button><span class="muted">${visible.length}/${groups.length}</span>`);
    for (const g of visible) {
      const members = kind === 'dir' ? g.dirs : g.files;
      const orig = originalOf(kind, g);
      const head = kind === 'dir' ? `${tf('dirdup.folders', g.count)}, ${tf('dirdup.files', g.files)}` : tf('dup.copies', g.count);
      html += `<div class="group" data-id="${esc(g.id)}"><div class="ghead"><b>${humanSize(g.size)}</b><span>${esc(head)}</span><span class="muted">${esc(t('col.wasted'))}: ${humanSize(g.wasted)}</span><span class="id">${esc(g.id)}</span><span class="grow"></span><button class="btn small copies">${esc(t('ui.select_copies'))}</button></div><ul>`;
      for (const m of members) {
        const isOrig = m.path === orig, sel = state.plan.has(m.path);
        html += `<li class="${isOrig ? 'orig' : sel ? 'sel' : ''}" data-path="${esc(m.path)}">
          <label><input type="radio" name="o-${esc(g.id)}" ${isOrig ? 'checked' : ''}>${esc(t('ui.original'))}</label>
          <label><input type="checkbox" class="q" ${sel ? 'checked' : ''} ${isOrig ? 'disabled' : ''}>${esc(t('ui.to_quarantine'))}</label>
          <span class="path">${pathHtml(m.path, m.rel)}${kind === 'dir' ? ` <span class="muted">${sep()}</span>` : ''}</span>
          <span class="mt">${fmtTime(m.mtime)}</span>
          <button class="reveal" title="${esc(t('ui.reveal'))}">📂</button></li>`;
      }
      html += '</ul></div>';
    }
    if (!visible.length) html += `<p class="empty">${esc(t('ui.no_items'))}</p>`;
    main.innerHTML = html;
    const byId = new Map(groups.map((g) => [g.id, g]));
    main.querySelectorAll('.group').forEach((el) => {
      const g = byId.get(el.dataset.id);
      const members = kind === 'dir' ? g.dirs : g.files;
      const byPath = new Map(members.map((m) => [m.path, m]));
      el.querySelectorAll('li').forEach((li) => {
        const m = byPath.get(li.dataset.path);
        $('input[type=radio]', li).addEventListener('change', () => { setOriginal(kind, g, m.path); renderMain(); });
        $('input.q', li).addEventListener('change', (e) => { planToggleCopy(kind, g, m, e.target.checked); li.classList.toggle('sel', e.target.checked); });
        $('.reveal', li).addEventListener('click', () => reveal(m.path));
      });
      $('.copies', el).addEventListener('click', () => { selectCopies(kind, g); renderMain(); });
    });
    $('#allCopies', main).addEventListener('click', () => { for (const g of visible) selectCopies(kind, g); renderMain(); });
    bindFilter(main, () => renderGroups(main, kind));
  }
  function selectCopies(kind, g) {
    const orig = originalOf(kind, g);
    for (const m of (kind === 'dir' ? g.dirs : g.files)) if (m.path !== orig) planToggleCopy(kind, g, m, true);
  }

  function renderOverlaps(main) {
    const all = state.d.dir_overlaps;
    const rows = all.filter((o) => matches(o.a.path) || matches(o.b.path));
    sortRows(rows, (o, k) => k === 'path' ? o.a.path.toLowerCase() : k === 'ratio' ? o.ratio : o.shared_bytes);
    const s = summaryOf('dir-overlap');
    let html = `<h2>${esc(t('cat.dir-overlap'))} <span class="muted">${esc(tf('overlap.header', '', s.count, humanSize(s.size)).replace(/^:\s*/, ''))}</span></h2>`;
    html += `<p class="hint">${esc(t('ui.overlap_hint'))}</p>` + toolbar(`<span class="muted">${rows.length}/${all.length}</span>`);
    html += `<table><tr>${th('path', t('col.path'))}<th>${esc(t('col.related'))}</th>${th('size', t('col.size'), 'num')}<th class="num">${esc(t('col.shared_files'))}</th>${th('ratio', t('col.ratio'), 'num')}<th class="num">${esc(t('col.ratio_a'))}</th><th class="num">${esc(t('col.ratio_b'))}</th></tr>`;
    for (const o of rows) {
      html += `<tr><td class="path">${pathHtml(o.a.path, o.a.rel)} <span class="muted">(${humanSize(o.a.size)}, ${o.a.files})</span></td><td class="path">${pathHtml(o.b.path, o.b.rel)} <span class="muted">(${humanSize(o.b.size)}, ${o.b.files})</span></td><td class="num">${humanSize(o.shared_bytes)}</td><td class="num">${o.shared_files}</td><td class="num">${pct(o.ratio)}</td><td class="num">${pct(o.ratio_a)}</td><td class="num">${pct(o.ratio_b)}</td></tr>`;
    }
    if (!rows.length) html += `<tr><td colspan="7" class="empty">${esc(t('ui.no_items'))}</td></tr>`;
    html += '</table>';
    main.innerHTML = html;
    bindFilter(main, () => renderOverlaps(main));
    bindSort(main, () => renderOverlaps(main));
  }

  function renderNames(main) {
    const groups = state.d.similar_names;
    const s = summaryOf('similar-name');
    const visible = groups.filter((g) => g.files.some((m) => matches(m.path)));
    let html = `<h2>${esc(t('cat.similar-name'))} <span class="muted">${esc(tf('names.header', '', s.count, groups.reduce((n, g) => n + g.variants, 0), humanSize(s.size)).replace(/^:\s*/, ''))}</span></h2>`;
    html += `<p class="hint">${esc(t('ui.names_hint'))}</p>` + toolbar(`<span class="muted">${visible.length}/${groups.length}</span>`);
    for (const g of visible) {
      html += `<div class="group" data-id="${esc(g.id)}"><div class="ghead"><b>${esc(g.name)}</b><span class="muted">${g.count}</span><span class="id">${esc(g.id)}</span></div><ul>`;
      for (const m of g.files) {
        const sel = state.plan.has(m.path);
        let tag = m.is_base ? t('names.base') : m.marker;
        if (m.dup_group) tag += ', ' + tf('names.dup', m.dup_group);
        html += `<li class="${m.is_base ? 'orig' : sel ? 'sel' : ''}" data-path="${esc(m.path)}">
          <label><input type="checkbox" class="q" ${sel ? 'checked' : ''}>${esc(t('ui.to_quarantine'))}</label>
          <span class="path">${pathHtml(m.path, m.rel)}</span>
          <span class="mt">${humanSize(m.size)} · [${esc(tag)}] · ${fmtTime(m.mtime)}</span>
          <button class="reveal" title="${esc(t('ui.reveal'))}">📂</button></li>`;
      }
      html += '</ul></div>';
    }
    if (!visible.length) html += `<p class="empty">${esc(t('ui.no_items'))}</p>`;
    main.innerHTML = html;
    const byId = new Map(groups.map((g) => [g.id, g]));
    main.querySelectorAll('.group').forEach((el) => {
      const g = byId.get(el.dataset.id);
      const byPath = new Map(g.files.map((m) => [m.path, m]));
      el.querySelectorAll('li').forEach((li) => {
        const m = byPath.get(li.dataset.path);
        $('input.q', li).addEventListener('change', (e) => {
          if (e.target.checked) planAdd({ op: 'quarantine', path: m.path, category: 'similar-name', size: m.size, original: g.base && g.base !== m.path ? g.base : undefined });
          else planRemove(m.path);
          li.classList.toggle('sel', e.target.checked && !m.is_base);
        });
        $('.reveal', li).addEventListener('click', () => reveal(m.path));
      });
    });
    bindFilter(main, () => renderNames(main));
  }

  async function loadQuarantine() {
    try {
      const r = await fetch('/api/quarantine');
      state.quarantine = r.ok ? await r.json() : [];
    } catch (_) { state.quarantine = []; }
  }
  function renderQuarantine(main) {
    const list = state.quarantine || [];
    let html = `<h2>${esc(t('ui.quarantine'))} <span class="muted">${list.length}</span></h2><p class="hint">${esc(t('ui.quarantine_hint'))}</p>`;
    if (!list.length) { main.innerHTML = html + `<p class="empty">${esc(t('ui.q_empty'))}</p>`; return; }
    html += `<table><tr><th>${esc(t('ui.q_created'))}</th><th>${esc(t('ui.q_status'))}</th><th class="num">${esc(t('ui.q_pending'))}</th><th class="num">${esc(t('ui.q_items'))}</th><th class="num">${esc(t('col.size'))}</th><th>${esc(t('col.path'))}</th><th></th></tr>`;
    list.forEach((b, i) => {
      const status = { active: 'ui.q_active', restored: 'ui.q_restored', partial: 'ui.q_partial', purged: 'ui.q_purged' }[b.status] || b.status;
      html += `<tr data-i="${i}"><td class="tag">${fmtTime(b.created)}</td><td class="tag">${esc(t(status))}</td><td class="num">${b.pending}</td><td class="num">${b.items}</td><td class="num">${humanSize(b.size)}</td><td class="path">${esc(b.dir)} <button class="reveal" title="${esc(t('ui.reveal'))}">📂</button></td><td><button class="btn small restore" ${b.pending ? '' : 'disabled'}>${esc(t('ui.q_restore'))}</button></td></tr>`;
    });
    html += '</table><p id="qStatus" class="muted"></p>';
    main.innerHTML = html;
    main.querySelectorAll('tr[data-i]').forEach((tr) => {
      const b = list[Number(tr.dataset.i)];
      $('.reveal', tr).addEventListener('click', () => reveal(b.dir));
      $('.restore', tr).addEventListener('click', async () => {
        if (!confirm(tf('ui.q_restore_confirm', b.pending, humanSize(b.size), fmtTime(b.created)))) return;
        $('#qStatus').textContent = '…';
        try {
          const r = await fetch('/api/restore', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ manifest: b.path }) });
          const txt = await r.text();
          if (!r.ok) throw new Error(txt);
          const j = JSON.parse(txt);
          await loadQuarantine();
          await rescan();
          state.view = 'quarantine';
          render();
          $('#qStatus').textContent = tf('ui.q_restore_done', j.restored, j.problems.length);
        } catch (e) {
          $('#qStatus').textContent = e.message;
        }
      });
    });
  }

  function renderTop(main, v) {
    const items = v === 'top-files' ? state.d.top_files : state.d.top_dirs;
    const rows = items.filter((i) => matches(i.path));
    let html = `<h2>${esc(t(v === 'top-files' ? 'scan.top_files' : 'scan.top_dirs'))} <span class="muted">${items.length}</span></h2>` + toolbar();
    html += `<table><tr><th class="num">${esc(t('col.size'))}</th><th>${esc(t('col.path'))}</th>${v === 'top-dirs' ? `<th class="num">${esc(t('col.files'))}</th>` : ''}<th class="act"></th></tr>`;
    for (const i of rows) html += `<tr data-path="${esc(i.path)}"><td class="num">${humanSize(i.size)}</td><td class="path">${pathHtml(i.path, i.rel)}</td>${v === 'top-dirs' ? `<td class="num">${i.files || 0}</td>` : ''}<td class="act"><button class="reveal" title="${esc(t('ui.reveal'))}">📂</button></td></tr>`;
    html += '</table>';
    main.innerHTML = html;
    main.querySelectorAll('tr[data-path] .reveal').forEach((b) => b.addEventListener('click', () => reveal(b.closest('tr').dataset.path)));
    bindFilter(main, () => renderTop(main, v));
  }

  function renderErrors(main) {
    let html = `<h2>${esc(t('html.errors'))} <span class="muted">${state.d.errors.length}</span></h2><table><tr><th>${esc(t('col.path'))}</th><th>${esc(t('col.error'))}</th></tr>`;
    for (const e of state.d.errors) html += `<tr><td class="path">${esc(e.path)}</td><td>${esc(e.error)}</td></tr>`;
    main.innerHTML = html + '</table>';
  }
  function renderSkipped(main) {
    main.innerHTML = `<h2>${esc(t('html.skipped'))} <span class="muted">${state.d.skipped.length}</span></h2><ul class="roots">${state.d.skipped.map((p) => `<li>${esc(p)}</li>`).join('')}</ul>`;
  }

  // ---------- boot ----------
  async function load(lang) {
    const r = await fetch('/api/report' + (lang ? '?lang=' + lang : ''));
    if (!r.ok) { $('#main').textContent = await r.text(); return; }
    const j = await r.json();
    state.d = j.report; state.lang = j.lang; state.S = j.strings;
    state.meta = { report_path: j.report_path, plan_dir: j.plan_dir, version: j.version, os: j.os };
    prunePlan();
    await loadQuarantine();
    render();
  }
  // After a rescan the plan may name paths that no longer exist in the
  // report (already moved, or fixed by hand): drop them.
  function prunePlan() {
    if (!state.plan.size) return;
    const known = new Set(state.d.findings.map((f) => f.path));
    for (const g of state.d.duplicates) for (const f of g.files) known.add(f.path);
    for (const g of state.d.dir_duplicates) for (const d of g.dirs) known.add(d.path);
    for (const g of state.d.similar_names || []) for (const f of g.files) known.add(f.path);
    let dropped = 0;
    for (const p of [...state.plan.keys()]) if (!known.has(p)) { state.plan.delete(p); dropped++; }
    if (dropped) $('#planStatus').textContent = '';
  }
  $('#langBtn').addEventListener('click', () => load(state.lang === 'ru' ? 'en' : 'ru'));
  $('#planSave').addEventListener('click', savePlan);
  $('#planDry').addEventListener('click', () => applyPlan(true));
  $('#planApply').addEventListener('click', () => applyPlan(false));
  $('#rescanBtn').addEventListener('click', rescan);
  $('#planClear').addEventListener('click', () => { state.plan.clear(); $('#planStatus').textContent = ''; render(); });
  load();
})();

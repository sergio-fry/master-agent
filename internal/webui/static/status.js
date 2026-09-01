(function () {
  'use strict';

  const api = window.MAAuth.api;

  const $ = (sel) => document.querySelector(sel);
  const flash = $('#flash');

  function showFlash(message, kind) {
    flash.textContent = message;
    flash.className = 'flash' + (kind ? ' ' + kind : '');
    flash.classList.remove('hidden');
  }

  function hideFlash() {
    flash.classList.add('hidden');
  }

  function badge(text, on) {
    return '<span class="badge ' + (on ? 'on' : 'off') + '">' + text + '</span>';
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function renderLocks(locks) {
    const section = $('#locks-section');
    const body = $('#locks-body');
    if (!locks.length) {
      section.classList.add('hidden');
      body.innerHTML = '';
      return;
    }
    section.classList.remove('hidden');
    body.innerHTML = locks.map(function (lock) {
      const project = lock.project_name || lock.project_id;
      const task = lock.task_name || lock.task_id;
      const pid = lock.pid != null ? String(lock.pid) : '—';
      return (
        '<tr>' +
          '<td>' + escapeHtml(project) + '</td>' +
          '<td>' + escapeHtml(task) + '</td>' +
          '<td><code>' + escapeHtml(lock.run_id) + '</code></td>' +
          '<td>' + escapeHtml(lock.acquired_at || '—') + '</td>' +
          '<td>' + escapeHtml(pid) + '</td>' +
        '</tr>'
      );
    }).join('');
  }

  async function loadStatus() {
    hideFlash();
    $('#http-status').innerHTML = '<span class="muted">Loading…</span>';
    $('#db-status').innerHTML = '<span class="muted">Loading…</span>';
    $('#lock-status').innerHTML = '<span class="muted">Loading…</span>';

    try {
      const status = await api('/status');
      $('#http-status').innerHTML = badge('up', status.ok !== false);
      $('#db-status').innerHTML = badge(status.db_ok ? 'connected' : 'unavailable', status.db_ok);
      $('#db-path').innerHTML = status.db_path
        ? '<code>' + escapeHtml(status.db_path) + '</code>'
        : '<span class="muted">—</span>';
      $('#lock-status').innerHTML = status.lock_active
        ? badge('yes', true)
        : badge('none', false);
      renderLocks(status.locks || []);
    } catch (err) {
      $('#http-status').innerHTML = badge('error', false);
      $('#db-status').innerHTML = '<span class="muted">—</span>';
      $('#lock-status').innerHTML = '<span class="muted">—</span>';
      renderLocks([]);
      showFlash(err.message, 'error');
    }
  }

  $('#btn-refresh').addEventListener('click', loadStatus);
  loadStatus();
})();

(function () {
  'use strict';

  const api = window.MAAuth.api;
  const apiText = window.MAAuth.apiText;

  const $ = (sel) => document.querySelector(sel);
  const projectFilter = $('#project-filter');
  const statusFilter = $('#status-filter');
  const runsBody = $('#runs-body');
  const flash = $('#flash');
  const detailDialog = $('#run-detail-dialog');

  let projectsById = {};
  let tasksById = {};

  function showFlash(message, kind) {
    flash.textContent = message;
    flash.className = 'flash' + (kind ? ' ' + kind : '');
    flash.classList.remove('hidden');
  }

  function hideFlash() {
    flash.classList.add('hidden');
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function escapeAttr(s) {
    return escapeHtml(s).replace(/'/g, '&#39;');
  }

  function formatTimestamp(value) {
    if (!value) return '—';
    const d = new Date(value);
    if (Number.isNaN(d.getTime())) return escapeHtml(value);
    return escapeHtml(d.toLocaleString());
  }

  function statusBadge(status) {
    const s = status || 'unknown';
    let cls = 'badge';
    if (s === 'success') cls += ' on';
    else if (s === 'error') cls += ' off';
    else if (s === 'running') cls += ' running';
    else cls += ' neutral';
    return '<span class="' + cls + '">' + escapeHtml(s) + '</span>';
  }

  function projectName(projectId) {
    const p = projectsById[projectId];
    return p ? p.name : projectId;
  }

  function taskLabel(taskId) {
    const t = tasksById[taskId];
    return t ? t.name : taskId;
  }

  function syncFiltersToUrl() {
    const url = new URL(window.location.href);
    const projectId = projectFilter.value;
    const status = statusFilter.value;
    if (projectId) {
      url.searchParams.set('project_id', projectId);
    } else {
      url.searchParams.delete('project_id');
    }
    if (status) {
      url.searchParams.set('status', status);
    } else {
      url.searchParams.delete('status');
    }
    window.history.replaceState(null, '', url.pathname + url.search);
  }

  function applyFiltersFromUrl() {
    const params = new URLSearchParams(window.location.search);
    const projectId = params.get('project_id') || '';
    const status = params.get('status') || '';
    if (projectId && projectsById[projectId]) {
      projectFilter.value = projectId;
    }
    if (status && ['running', 'success', 'error'].includes(status)) {
      statusFilter.value = status;
    }
  }

  function renderRuns(runs) {
    if (!runs.length) {
      runsBody.innerHTML = '<tr><td colspan="7" class="muted">No runs found.</td></tr>';
      return;
    }
    runsBody.innerHTML = runs.map(function (r) {
      const exit = r.exit_code === null || r.exit_code === undefined
        ? '—'
        : escapeHtml(String(r.exit_code));
      return (
        '<tr data-id="' + escapeAttr(r.id) + '">' +
          '<td>' + statusBadge(r.status) + '</td>' +
          '<td>' + escapeHtml(projectName(r.project_id)) + '</td>' +
          '<td>' + escapeHtml(taskLabel(r.task_id)) + '</td>' +
          '<td class="timestamp">' + formatTimestamp(r.started_at) + '</td>' +
          '<td class="timestamp">' + formatTimestamp(r.finished_at) + '</td>' +
          '<td>' + exit + '</td>' +
          '<td class="actions">' +
            '<button type="button" class="link btn-view">View</button>' +
          '</td>' +
        '</tr>'
      );
    }).join('');
  }

  async function loadTasksForProjects(projects) {
    tasksById = {};
    await Promise.all(projects.map(async function (p) {
      try {
        const tasks = await api('/projects/' + encodeURIComponent(p.id) + '/tasks');
        tasks.forEach(function (t) {
          tasksById[t.id] = t;
        });
      } catch (_) {
        // ignore per-project task load errors
      }
    }));
  }

  async function loadProjects() {
    const projects = await api('/projects');
    projectsById = {};
    projects.forEach(function (p) {
      projectsById[p.id] = p;
    });
    const current = projectFilter.value;
    projectFilter.innerHTML = '<option value="">All projects</option>' +
      projects.map(function (p) {
        return '<option value="' + escapeAttr(p.id) + '">' + escapeHtml(p.name) + '</option>';
      }).join('');
    applyFiltersFromUrl();
    if (current && projectsById[current]) {
      projectFilter.value = current;
    }
    await loadTasksForProjects(projects);
  }

  async function loadRuns() {
    hideFlash();
    syncFiltersToUrl();
    runsBody.innerHTML = '<tr><td colspan="7" class="muted">Loading…</td></tr>';
    const params = new URLSearchParams();
    if (projectFilter.value) {
      params.set('project_id', projectFilter.value);
    }
    if (statusFilter.value) {
      params.set('status', statusFilter.value);
    }
    const qs = params.toString();
    const path = '/runs' + (qs ? '?' + qs : '');
    try {
      const runs = await api(path);
      renderRuns(runs);
    } catch (err) {
      runsBody.innerHTML = '<tr><td colspan="7" class="muted">Failed to load runs.</td></tr>';
      showFlash(err.message, 'error');
    }
  }

  function formatDetailValue(value) {
    if (value === null || value === undefined || value === '') return '—';
    return String(value);
  }

  async function openRunDetail(runId) {
    $('#detail-run-id').textContent = runId;
    $('#detail-status').innerHTML = '—';
    $('#detail-started').textContent = '—';
    $('#detail-finished').textContent = '—';
    $('#detail-exit-code').textContent = '—';
    $('#detail-error').textContent = '—';
    $('#detail-log').textContent = 'Loading log…';
    $('#detail-log').className = 'log-viewer muted';
    detailDialog.showModal();

    try {
      const run = await api('/runs/' + encodeURIComponent(runId));
      $('#detail-status').innerHTML = statusBadge(run.status);
      $('#detail-started').textContent = formatDetailValue(run.started_at);
      $('#detail-finished').textContent = formatDetailValue(run.finished_at);
      $('#detail-exit-code').textContent = formatDetailValue(run.exit_code);
      const errMsg = run.error_message;
      $('#detail-error').textContent = errMsg ? String(errMsg) : '—';
      $('#detail-error').className = errMsg ? 'detail-error has-error' : 'detail-error';

      try {
        const logText = await apiText('/runs/' + encodeURIComponent(runId) + '/log');
        $('#detail-log').textContent = logText || '(empty log)';
        $('#detail-log').className = 'log-viewer';
      } catch (logErr) {
        $('#detail-log').textContent = logErr.message || 'Log not available.';
        $('#detail-log').className = 'log-viewer muted';
      }
    } catch (err) {
      showFlash(err.message, 'error');
      detailDialog.close();
    }
  }

  projectFilter.addEventListener('change', loadRuns);
  statusFilter.addEventListener('change', loadRuns);

  runsBody.addEventListener('click', function (ev) {
    const row = ev.target.closest('tr[data-id]');
    if (!row) return;
    if (ev.target.classList.contains('btn-view')) {
      openRunDetail(row.dataset.id);
    }
  });

  $('#detail-close').addEventListener('click', function () {
    detailDialog.close();
  });

  (async function init() {
    try {
      await loadProjects();
      await loadRuns();
    } catch (err) {
      showFlash(err.message, 'error');
    }
  })();
})();

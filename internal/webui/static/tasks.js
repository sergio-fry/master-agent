(function () {
  'use strict';

  const api = window.MAAuth.api;

  const $ = (sel) => document.querySelector(sel);
  const projectSelect = $('#project-select');
  const tasksBody = $('#tasks-body');
  const flash = $('#flash');
  const dialog = $('#task-dialog');
  const form = $('#task-form');
  const formError = $('#form-error');
  const btnNew = $('#btn-new');

  function showFlash(message, kind) {
    flash.textContent = message;
    flash.className = 'flash' + (kind ? ' ' + kind : '');
    flash.classList.remove('hidden');
  }

  function hideFlash() {
    flash.classList.add('hidden');
  }

  function showFormError(message) {
    if (!message) {
      formError.classList.add('hidden');
      formError.textContent = '';
      return;
    }
    formError.textContent = message;
    formError.classList.remove('hidden');
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

  function formatInterval(seconds) {
    if (!seconds && seconds !== 0) return '—';
    return escapeHtml(String(seconds) + 's');
  }

  function selectedProjectId() {
    return projectSelect.value || '';
  }

  function renderTasks(tasks) {
    if (!selectedProjectId()) {
      tasksBody.innerHTML = '<tr><td colspan="6" class="muted">Select a project to view tasks.</td></tr>';
      btnNew.disabled = true;
      return;
    }
    btnNew.disabled = false;
    if (!tasks.length) {
      tasksBody.innerHTML = '<tr><td colspan="6" class="muted">No tasks yet.</td></tr>';
      return;
    }
    tasksBody.innerHTML = tasks.map(function (t) {
      const badge = t.enabled
        ? '<span class="badge on">on</span>'
        : '<span class="badge off">off</span>';
      const toggleLabel = t.enabled ? 'Disable' : 'Enable';
      return (
        '<tr data-id="' + escapeAttr(t.id) + '">' +
          '<td>' + escapeHtml(t.name) + '</td>' +
          '<td>' + formatInterval(t.interval_seconds) + '</td>' +
          '<td>' + badge + '</td>' +
          '<td class="timestamp">' + formatTimestamp(t.last_run_at) + '</td>' +
          '<td class="timestamp">' + formatTimestamp(t.next_run_at) + '</td>' +
          '<td class="actions">' +
            '<button type="button" class="link btn-edit">Edit</button>' +
            '<button type="button" class="link btn-toggle">' + toggleLabel + '</button>' +
          '</td>' +
        '</tr>'
      );
    }).join('');
  }

  async function loadTasks() {
    hideFlash();
    const projectId = selectedProjectId();
    if (!projectId) {
      renderTasks([]);
      return;
    }
    tasksBody.innerHTML = '<tr><td colspan="6" class="muted">Loading…</td></tr>';
    try {
      const tasks = await api('/projects/' + encodeURIComponent(projectId) + '/tasks');
      renderTasks(tasks);
    } catch (err) {
      tasksBody.innerHTML = '<tr><td colspan="6" class="muted">Failed to load tasks.</td></tr>';
      showFlash(err.message, 'error');
    }
  }

  async function loadProjects() {
    try {
      const projects = await api('/projects');
      const current = selectedProjectId();
      projectSelect.innerHTML = '<option value="">Select a project…</option>' +
        projects.map(function (p) {
          return '<option value="' + escapeAttr(p.id) + '">' + escapeHtml(p.name) + '</option>';
        }).join('');
      const params = new URLSearchParams(window.location.search);
      const fromUrl = params.get('project_id') || '';
      if (fromUrl && projects.some(function (p) { return p.id === fromUrl; })) {
        projectSelect.value = fromUrl;
      } else if (current && projects.some(function (p) { return p.id === current; })) {
        projectSelect.value = current;
      }
      await loadTasks();
    } catch (err) {
      showFlash(err.message, 'error');
    }
  }

  function openCreateDialog() {
    if (!selectedProjectId()) return;
    $('#dialog-title').textContent = 'New task';
    $('#task-id').value = '';
    form.reset();
    $('#task-enabled').checked = true;
    showFormError('');
    dialog.showModal();
  }

  function openEditDialog(task) {
    $('#dialog-title').textContent = 'Edit task';
    $('#task-id').value = task.id;
    $('#task-name').value = task.name;
    $('#interval_seconds').value = String(task.interval_seconds || '');
    $('#command').value = task.command;
    $('#prompt').value = task.prompt;
    $('#task-enabled').checked = !!task.enabled;
    showFormError('');
    dialog.showModal();
  }

  async function saveTask(ev) {
    ev.preventDefault();
    showFormError('');
    const projectId = selectedProjectId();
    if (!projectId) {
      showFormError('Select a project first.');
      return;
    }
    const id = $('#task-id').value;
    const interval = parseInt($('#interval_seconds').value, 10);
    const body = {
      name: $('#task-name').value.trim(),
      interval_seconds: interval,
      command: $('#command').value.trim(),
      prompt: $('#prompt').value.trim(),
      enabled: $('#task-enabled').checked,
    };
    try {
      if (id) {
        await api('/tasks/' + encodeURIComponent(id), { method: 'PATCH', body: body });
        showFlash('Task updated.', 'ok');
      } else {
        await api('/projects/' + encodeURIComponent(projectId) + '/tasks', {
          method: 'POST',
          body: body,
        });
        showFlash('Task created.', 'ok');
      }
      dialog.close();
      await loadTasks();
    } catch (err) {
      showFormError(err.message);
    }
  }

  async function toggleEnabled(id, enabled) {
    hideFlash();
    try {
      await api('/tasks/' + encodeURIComponent(id), {
        method: 'PATCH',
        body: { enabled: !enabled },
      });
      await loadTasks();
    } catch (err) {
      showFlash(err.message, 'error');
    }
  }

  projectSelect.addEventListener('change', function () {
    const id = selectedProjectId();
    const url = new URL(window.location.href);
    if (id) {
      url.searchParams.set('project_id', id);
    } else {
      url.searchParams.delete('project_id');
    }
    window.history.replaceState(null, '', url.pathname + url.search);
    loadTasks();
  });

  tasksBody.addEventListener('click', async function (ev) {
    const row = ev.target.closest('tr[data-id]');
    if (!row) return;
    const id = row.dataset.id;
    if (ev.target.classList.contains('btn-edit')) {
      try {
        const task = await api('/tasks/' + encodeURIComponent(id));
        openEditDialog(task);
      } catch (err) {
        showFlash(err.message, 'error');
      }
    }
    if (ev.target.classList.contains('btn-toggle')) {
      const enabled = row.querySelector('.badge.on') !== null;
      await toggleEnabled(id, enabled);
    }
  });

  btnNew.addEventListener('click', openCreateDialog);
  $('#btn-cancel').addEventListener('click', function () { dialog.close(); });
  form.addEventListener('submit', saveTask);

  loadProjects();
})();

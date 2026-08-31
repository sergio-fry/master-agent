(function () {
  'use strict';

  const TOKEN_KEY = 'master_agent_token';

  const $ = (sel) => document.querySelector(sel);
  const projectsBody = $('#projects-body');
  const flash = $('#flash');
  const dialog = $('#project-dialog');
  const form = $('#project-form');
  const formError = $('#form-error');
  const authPanel = $('#auth-panel');
  const keyDialog = $('#key-dialog');
  const keyForm = $('#key-form');
  const keyFormError = $('#key-form-error');
  const keyStatusBadge = $('#key-status-badge');
  const keyPathDisplay = $('#key-path-display');
  const keyFileInput = $('#key-file');

  function getToken() {
    return sessionStorage.getItem(TOKEN_KEY) || '';
  }

  function setToken(value) {
    if (value) {
      sessionStorage.setItem(TOKEN_KEY, value);
    } else {
      sessionStorage.removeItem(TOKEN_KEY);
    }
  }

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

  async function api(path, options) {
    const opts = Object.assign({ headers: {} }, options || {});
    const token = getToken();
    if (token) {
      opts.headers['Authorization'] = 'Bearer ' + token;
    }
    if (opts.body && typeof opts.body === 'object' && !(opts.body instanceof FormData)) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(opts.body);
    }
    const resp = await fetch('/api/v1' + path, opts);
    let data = null;
    const ct = resp.headers.get('content-type') || '';
    if (ct.includes('application/json')) {
      data = await resp.json();
    }
    if (resp.status === 401) {
      authPanel.classList.remove('hidden');
      throw new Error((data && data.error) || 'Unauthorized — enter API token above');
    }
    if (!resp.ok) {
      throw new Error((data && data.error) || ('Request failed: ' + resp.status));
    }
    return data;
  }

  function sshSummary(p) {
    return p.ssh_user + '@' + p.ssh_host + ':' + p.ssh_port;
  }

  function showKeyFormError(message) {
    if (!message) {
      keyFormError.classList.add('hidden');
      keyFormError.textContent = '';
      return;
    }
    keyFormError.textContent = message;
    keyFormError.classList.remove('hidden');
  }

  function renderKeyStatus(present) {
    if (present) {
      keyStatusBadge.textContent = 'configured';
      keyStatusBadge.className = 'badge on';
    } else {
      keyStatusBadge.textContent = 'not configured';
      keyStatusBadge.className = 'badge off';
    }
  }

  async function apiMultipart(path, formData) {
    const opts = { method: 'POST', body: formData, headers: {} };
    const token = getToken();
    if (token) {
      opts.headers['Authorization'] = 'Bearer ' + token;
    }
    const resp = await fetch('/api/v1' + path, opts);
    let data = null;
    const ct = resp.headers.get('content-type') || '';
    if (ct.includes('application/json')) {
      data = await resp.json();
    }
    if (resp.status === 401) {
      authPanel.classList.remove('hidden');
      throw new Error((data && data.error) || 'Unauthorized — enter API token above');
    }
    if (!resp.ok) {
      throw new Error((data && data.error) || ('Request failed: ' + resp.status));
    }
    return data;
  }

  function renderProjects(projects) {
    if (!projects.length) {
      projectsBody.innerHTML = '<tr><td colspan="6" class="muted">No projects yet.</td></tr>';
      return;
    }
    projectsBody.innerHTML = projects.map(function (p) {
      const badge = p.enabled
        ? '<span class="badge on">on</span>'
        : '<span class="badge off">off</span>';
      const toggleLabel = p.enabled ? 'Disable' : 'Enable';
      const keyBadge = p.key_present
        ? '<span class="badge on">yes</span>'
        : '<span class="badge off">no</span>';
      return (
        '<tr data-id="' + escapeAttr(p.id) + '">' +
          '<td>' + escapeHtml(p.name) + '</td>' +
          '<td><code>' + escapeHtml(p.path) + '</code></td>' +
          '<td>' + escapeHtml(sshSummary(p)) + '</td>' +
          '<td>' + badge + '</td>' +
          '<td class="key-cell">' + keyBadge + '</td>' +
          '<td class="actions">' +
            '<button type="button" class="link btn-edit">Edit</button>' +
            '<button type="button" class="link btn-key">Key</button>' +
            '<button type="button" class="link btn-toggle">' + toggleLabel + '</button>' +
          '</td>' +
        '</tr>'
      );
    }).join('');
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

  async function loadProjects() {
    hideFlash();
    projectsBody.innerHTML = '<tr><td colspan="6" class="muted">Loading…</td></tr>';
    try {
      const projects = await api('/projects');
      const withKeyStatus = await Promise.all(projects.map(async function (p) {
        try {
          const status = await api('/projects/' + encodeURIComponent(p.id) + '/key');
          p.key_present = !!(status && status.present);
        } catch (_) {
          p.key_present = false;
        }
        return p;
      }));
      renderProjects(withKeyStatus);
    } catch (err) {
      projectsBody.innerHTML = '<tr><td colspan="6" class="muted">Failed to load projects.</td></tr>';
      showFlash(err.message, 'error');
    }
  }

  function openCreateDialog() {
    $('#dialog-title').textContent = 'New project';
    $('#project-id').value = '';
    form.reset();
    $('#ssh_port').value = '22';
    $('#enabled').checked = true;
    showFormError('');
    dialog.showModal();
  }

  function openEditDialog(project) {
    $('#dialog-title').textContent = 'Edit project';
    $('#project-id').value = project.id;
    $('#name').value = project.name;
    $('#path').value = project.path;
    $('#ssh_host').value = project.ssh_host;
    $('#ssh_user').value = project.ssh_user;
    $('#ssh_port').value = String(project.ssh_port || 22);
    $('#ssh_key_path').value = project.ssh_key_path;
    $('#enabled').checked = !!project.enabled;
    showFormError('');
    dialog.showModal();
  }

  async function saveProject(ev) {
    ev.preventDefault();
    showFormError('');
    const id = $('#project-id').value;
    const body = {
      name: $('#name').value.trim(),
      path: $('#path').value.trim(),
      ssh_host: $('#ssh_host').value.trim(),
      ssh_user: $('#ssh_user').value.trim(),
      ssh_port: parseInt($('#ssh_port').value, 10) || 22,
      ssh_key_path: $('#ssh_key_path').value.trim(),
      enabled: $('#enabled').checked,
    };
    try {
      if (id) {
        await api('/projects/' + encodeURIComponent(id), { method: 'PATCH', body: body });
        showFlash('Project updated.', 'ok');
      } else {
        await api('/projects', { method: 'POST', body: body });
        showFlash('Project created.', 'ok');
      }
      dialog.close();
      await loadProjects();
    } catch (err) {
      showFormError(err.message);
    }
  }

  async function openKeyDialog(project) {
    $('#key-project-id').value = project.id;
    $('#key-project-name').textContent = project.name;
    keyPathDisplay.textContent = project.ssh_key_path || '—';
    keyFileInput.value = '';
    showKeyFormError('');
    try {
      const status = await api('/projects/' + encodeURIComponent(project.id) + '/key');
      renderKeyStatus(!!(status && status.present));
    } catch (err) {
      renderKeyStatus(false);
      showKeyFormError(err.message);
    }
    keyDialog.showModal();
  }

  async function uploadKey(ev) {
    ev.preventDefault();
    showKeyFormError('');
    const id = $('#key-project-id').value;
    const file = keyFileInput.files && keyFileInput.files[0];
    if (!file) {
      showKeyFormError('Choose a key file to upload.');
      return;
    }
    const formData = new FormData();
    formData.append('key', file);
    try {
      const result = await apiMultipart('/projects/' + encodeURIComponent(id) + '/key', formData);
      renderKeyStatus(!!(result && result.present));
      keyFileInput.value = '';
      showFlash('SSH key uploaded.', 'ok');
      const project = await api('/projects/' + encodeURIComponent(id));
      keyPathDisplay.textContent = project.ssh_key_path || '—';
      await loadProjects();
    } catch (err) {
      showKeyFormError(err.message);
    }
  }

  async function toggleEnabled(id, enabled) {
    hideFlash();
    try {
      await api('/projects/' + encodeURIComponent(id), {
        method: 'PATCH',
        body: { enabled: !enabled },
      });
      await loadProjects();
    } catch (err) {
      showFlash(err.message, 'error');
    }
  }

  projectsBody.addEventListener('click', async function (ev) {
    const row = ev.target.closest('tr[data-id]');
    if (!row) return;
    const id = row.dataset.id;
    if (ev.target.classList.contains('btn-edit')) {
      try {
        const project = await api('/projects/' + encodeURIComponent(id));
        openEditDialog(project);
      } catch (err) {
        showFlash(err.message, 'error');
      }
    }
    if (ev.target.classList.contains('btn-key')) {
      try {
        const project = await api('/projects/' + encodeURIComponent(id));
        await openKeyDialog(project);
      } catch (err) {
        showFlash(err.message, 'error');
      }
    }
    if (ev.target.classList.contains('btn-toggle')) {
      const enabled = row.querySelector('.badge.on') !== null;
      await toggleEnabled(id, enabled);
    }
  });

  $('#btn-new').addEventListener('click', openCreateDialog);
  $('#btn-cancel').addEventListener('click', function () { dialog.close(); });
  form.addEventListener('submit', saveProject);

  $('#key-btn-cancel').addEventListener('click', function () { keyDialog.close(); });
  keyForm.addEventListener('submit', uploadKey);

  $('#token-save').addEventListener('click', function () {
    setToken($('#token-input').value.trim());
    authPanel.classList.add('hidden');
    loadProjects();
  });

  $('#token-input').value = getToken();
  loadProjects();
})();

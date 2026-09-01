(function () {
  'use strict';

  const api = window.MAAuth.api;

  const $ = (sel) => document.querySelector(sel);
  const projectsBody = $('#projects-body');
  const flash = $('#flash');
  const dialog = $('#project-dialog');
  const form = $('#project-form');
  const formError = $('#form-error');

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

  function sshSummary(p) {
    return p.ssh_user + '@' + p.ssh_host + ':' + p.ssh_port;
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
      const keyBadge = p.key_configured
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
            '<a class="link btn-tasks" href="/tasks.html?project_id=' + encodeURIComponent(p.id) + '">Tasks</a>' +
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
      renderProjects(projects);
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
    $('#ssh_private_key').required = true;
    $('#key-hint-create').classList.remove('hidden');
    $('#key-hint-edit').classList.add('hidden');
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
    $('#ssh_private_key').value = '';
    $('#ssh_private_key').required = false;
    $('#key-hint-create').classList.add('hidden');
    $('#key-hint-edit').classList.remove('hidden');
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
      enabled: $('#enabled').checked,
    };
    const keyValue = $('#ssh_private_key').value.trim();
    if (keyValue) {
      body.ssh_private_key = keyValue;
    } else if (!id) {
      showFormError('SSH private key is required for new projects.');
      return;
    }
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
    if (ev.target.classList.contains('btn-toggle')) {
      const enabled = row.querySelector('.badge.on') !== null;
      await toggleEnabled(id, enabled);
    }
  });

  $('#btn-new').addEventListener('click', openCreateDialog);
  $('#btn-cancel').addEventListener('click', function () { dialog.close(); });
  form.addEventListener('submit', saveProject);

  loadProjects();
})();

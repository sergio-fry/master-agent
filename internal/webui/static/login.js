(function () {
  'use strict';

  var $ = function (sel) { return document.querySelector(sel); };

  function nextURL() {
    var params = new URLSearchParams(window.location.search);
    var next = params.get('next');
    if (next && next.charAt(0) === '/') {
      return next;
    }
    return '/';
  }

  function showError(message) {
    var el = $('#login-error');
    if (!message) {
      el.classList.add('hidden');
      el.textContent = '';
      return;
    }
    el.textContent = message;
    el.classList.remove('hidden');
  }

  fetch('/api/v1/status', { credentials: 'same-origin' }).then(function (resp) {
    if (resp.ok) {
      window.location.href = nextURL();
    }
  }).catch(function () { /* stay on login */ });

  $('#login-form').addEventListener('submit', async function (ev) {
    ev.preventDefault();
    showError('');
    var username = $('#username').value.trim();
    var password = $('#password').value;
    try {
      var resp = await fetch('/api/v1/auth/login', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: username, password: password }),
      });
      var data = null;
      var ct = resp.headers.get('content-type') || '';
      if (ct.includes('application/json')) {
        data = await resp.json();
      }
      if (resp.status === 401) {
        showError((data && data.error) || 'Invalid username or password');
        return;
      }
      if (!resp.ok) {
        showError((data && data.error) || 'Login failed');
        return;
      }
      window.location.href = nextURL();
    } catch (err) {
      showError(err.message || 'Login failed');
    }
  });
})();

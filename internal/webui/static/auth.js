(function (global) {
  'use strict';

  function loginRedirect() {
    var next = encodeURIComponent(window.location.pathname + window.location.search);
    global.location.href = '/login.html?next=' + next;
  }

  async function parseResponse(resp) {
    var data = null;
    var ct = resp.headers.get('content-type') || '';
    if (ct.includes('application/json')) {
      data = await resp.json();
    }
    return data;
  }

  async function api(path, options) {
    var opts = Object.assign({ credentials: 'same-origin', headers: {} }, options || {});
    if (opts.body && typeof opts.body === 'object' && !(opts.body instanceof FormData)) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(opts.body);
    }
    var resp = await fetch('/api/v1' + path, opts);
    var data = await parseResponse(resp);
    if (resp.status === 401) {
      loginRedirect();
      throw new Error((data && data.error) || 'Unauthorized');
    }
    if (!resp.ok) {
      throw new Error((data && data.error) || ('Request failed: ' + resp.status));
    }
    return data;
  }

  async function apiText(path, options) {
    var opts = Object.assign({ credentials: 'same-origin', headers: {} }, options || {});
    var resp = await fetch('/api/v1' + path, opts);
    if (resp.status === 401) {
      var data = await parseResponse(resp);
      loginRedirect();
      throw new Error((data && data.error) || 'Unauthorized');
    }
    if (!resp.ok) {
      var errData = await parseResponse(resp);
      throw new Error((errData && errData.error) || ('Request failed: ' + resp.status));
    }
    return resp.text();
  }

  async function logout() {
    try {
      await fetch('/api/v1/auth/logout', { method: 'POST', credentials: 'same-origin' });
    } catch (_) { /* ignore */ }
    loginRedirect();
  }

  function bindLogout() {
    var btn = document.getElementById('btn-logout');
    if (btn) {
      btn.addEventListener('click', function (ev) {
        ev.preventDefault();
        logout();
      });
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bindLogout);
  } else {
    bindLogout();
  }

  global.MAAuth = {
    api: api,
    apiText: apiText,
    logout: logout,
    loginRedirect: loginRedirect,
  };
})(window);

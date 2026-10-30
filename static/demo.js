// Страница демо: запуск процессов через HTTP API go-bpm на этом же сервере (/api/v1) и наблюдение за экземпляром.
(function () {
  'use strict';

  var $ = function (sel, el) { return (el || document).querySelector(sel); };
  var $$ = function (sel, el) { return Array.prototype.slice.call((el || document).querySelectorAll(sel)); };

  var page = $('[data-demo]');
  if (!page) return;

  var POLL_MS = 1500;
  var POLL_LIMIT_MS = 10 * 60 * 1000; // дольше не следим: процесс с задачей пользователя может ждать вечно

  var current = null; // {id, process}
  var pollTimer = null;
  var pollUntil = 0;
  var names = {}; // процесс -> {id элемента -> {name, type}}
  var busy = false;

  $$('[data-origin]').forEach(function (el) { el.textContent = location.origin; });

  // --- API ---
  function api(method, path, body) {
    return fetch('/api/v1' + path, {
      method: method,
      headers: body ? { 'Content-Type': 'application/json' } : {},
      body: body ? JSON.stringify(body) : undefined,
    }).then(function (r) {
      if (r.status === 204) return null;
      return r.text().then(function (text) {
        var data = null;
        try { data = text ? JSON.parse(text) : null; } catch (e) {}
        if (!r.ok) {
          var msg = (data && data.error) || text || r.statusText;
          if (r.status === 429 || r.status === 503) msg = 'Слишком много запросов. Подождите немного и попробуйте снова.';
          var err = new Error(msg);
          err.status = r.status;
          throw err;
        }
        return data;
      });
    });
  }

  // --- состояние демо и счётчики экземпляров ---
  var statusBox = $('[data-status]');
  var statusText = $('[data-status-text]');

  function plural(n, one, few, many) {
    var m10 = n % 10, m100 = n % 100;
    if (m10 === 1 && m100 !== 11) return one;
    if (m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20)) return few;
    return many;
  }

  function refreshProcesses() {
    return api('GET', '/processes').then(function (data) {
      statusBox.classList.add('up');
      statusBox.classList.remove('down');
      var total = 0;
      (data.processes || []).forEach(function (p) {
        var c = p.instances || {};
        var n = (c.running || 0) + (c.completed || 0) + (c.failed || 0) + (c.cancelled || 0);
        total += n;
        var card = $('[data-process="' + p.id + '"] [data-count]');
        if (!card) return;
        var parts = [];
        if (c.running) parts.push(c.running + ' в работе');
        if (c.completed) parts.push(c.completed + ' ' + plural(c.completed, 'завершён', 'завершено', 'завершено'));
        if (c.failed) parts.push(c.failed + ' ' + plural(c.failed, 'упал', 'упало', 'упало'));
        if (c.cancelled) parts.push(c.cancelled + ' ' + plural(c.cancelled, 'отменён', 'отменено', 'отменено'));
        card.textContent = parts.join(' · ');
      });
      statusText.textContent = 'Демо работает · ' + total + ' ' + plural(total, 'экземпляр', 'экземпляра', 'экземпляров') + ' в базе';
    }).catch(function () {
      statusBox.classList.add('down');
      statusBox.classList.remove('up');
      statusText.textContent = 'Демо сейчас не отвечает — попробуйте позже';
    });
  }

  // --- имена элементов из BPMN процесса ---
  function loadNames(process) {
    var root = process.split('.')[0];
    if (names[root]) return Promise.resolve(names[root]);
    return api('GET', '/processes/' + encodeURIComponent(root)).then(function (p) {
      var map = {};
      var doc = new DOMParser().parseFromString(p.bpmn || '', 'application/xml');
      $$('*', doc).forEach(function (el) {
        var id = el.getAttribute('id');
        if (!id) return;
        map[id] = { name: el.getAttribute('name') || '', type: el.localName };
      });
      names[root] = map;
      return map;
    }).catch(function () { names[root] = {}; return names[root]; });
  }

  function element(process, id) {
    var m = names[process.split('.')[0]] || {};
    return m[id] || { name: '', type: '' };
  }

  // --- карточка экземпляра ---
  var run = {
    empty: $('[data-run-empty]'),
    body: $('[data-run-body]'),
    proc: $('[data-run-proc]'),
    id: $('[data-run-id]'),
    status: $('[data-run-status]'),
    tasks: $('[data-run-tasks]'),
    timeline: $('[data-run-timeline]'),
    vars: $('[data-run-vars]'),
    varsJSON: $('[data-run-vars-json]'),
    history: $('[data-run-history]'),
    error: $('[data-run-error]'),
  };

  var STATUS = { running: 'в работе', completed: 'завершён', failed: 'упал', cancelled: 'отменён' };
  var EVENT = { activated: 'ждёт', completed: 'пройден', terminated: 'прерван' };
  var TYPE = {
    startEvent: 'старт', endEvent: 'конец', task: 'метод', scriptTask: 'скрипт', serviceTask: 'service task',
    businessRuleTask: 'правило DMN', userTask: 'задача пользователя', exclusiveGateway: 'шлюз',
    parallelGateway: 'параллельный шлюз', intermediateCatchEvent: 'таймер', boundaryEvent: 'граничное событие',
    subProcess: 'подпроцесс', callActivity: 'вызов процесса', receiveTask: 'ждёт сообщение',
  };

  var WITH_TASKS = { TestUserTask: true, TestMultiInstance: true }; // процессы с задачами пользователей

  function showError(msg) {
    run.error.hidden = !msg;
    run.error.textContent = msg || '';
  }

  function escapeHTML(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }

  function renderInstance(inst, records) {
    run.empty.hidden = true;
    run.body.hidden = false;
    run.proc.textContent = inst.process_name + (inst.business_key ? ' · ' + inst.business_key : '');
    run.id.textContent = inst.instance_uid;
    run.status.textContent = STATUS[inst.status] || inst.status;
    run.status.className = 'badge ' + inst.status;
    run.history.href = '/ui/history/?instanceId=' + encodeURIComponent(inst.instance_uid);

    var t0 = records.length ? Date.parse(records[0].created_at) : 0;
    var lastVars = null;
    var items = records.filter(function (r) {
      if (r.variables && !r.parent_instance_uid) lastVars = r.variables;
      return element(r.process_name, r.element_id).type !== 'sequenceFlow';
    }).map(function (r) {
      var el = element(r.process_name, r.element_id);
      var title = el.name || r.element_id;
      var kind = TYPE[el.type] || '';
      var ms = Date.parse(r.created_at) - t0;
      var ts = ms < 1000 ? '+' + ms + ' мс' : '+' + (ms / 1000).toFixed(1) + ' с';
      var sub = r.parent_instance_uid ? r.process_name : '';
      return '<li class="' + r.event + (r.parent_instance_uid ? ' child' : '') + '">' +
        '<span><span class="el">' + escapeHTML(title) + '</span> <span class="ev">' +
        escapeHTML([kind, EVENT[r.event] || r.event].filter(Boolean).join(', ')) + '</span></span>' +
        '<span class="ts">' + ts + '</span>' +
        (sub ? '<span class="sub">в экземпляре ' + escapeHTML(sub) + '</span>' : '') + '</li>';
    });
    run.timeline.innerHTML = items.join('') || '<li class="activated"><span class="ev">Старт записан в очередь…</span></li>';
    run.timeline.scrollTop = run.timeline.scrollHeight;

    run.vars.hidden = !lastVars;
    if (lastVars) run.varsJSON.textContent = JSON.stringify(lastVars, null, 2);
  }

  // --- задачи пользователей: кнопки за менеджера и инициатора ---
  function renderTasks(tasks) {
    if (!tasks.length || busy) {
      if (!busy) run.tasks.hidden = true;
      return;
    }
    var t = tasks[0];
    var actions = [];
    if (t.element_id === 'Approve') {
      actions.push({ label: 'Согласовать как bob', user: 'bob', groups: ['managers'], vars: { approved: true } });
      actions.push({ label: 'Вернуть на исправление', user: 'bob', groups: ['managers'], ghost: true,
        vars: { approved: false, comment: 'нет суммы' } });
    } else if (t.element_id === 'Fix') {
      actions.push({ label: 'Исправить как ' + (t.assignee || 'alice'), user: t.assignee || 'alice', vars: { amount: 100 } });
    } else if (t.element_id === 'Review') {
      actions.push({ label: 'Согласовать как bob', user: 'bob', groups: ['managers'], vars: { approved: true } });
      actions.push({ label: 'Не согласовать: ещё круг', user: 'bob', groups: ['managers'], ghost: true,
        vars: { approved: false } });
    } else {
      var user = t.assignee || 'demo';
      actions.push({ label: 'Выполнить как ' + user, user: user, groups: t.candidate_groups || [], vars: {} });
    }
    var who = t.assignee ? 'назначена ' + t.assignee :
      (t.candidate_groups && t.candidate_groups.length ? 'для групп ' + t.candidate_groups.join(', ') : 'для любого');
    var round = t.variables && t.variables.loopCounter ? ', круг ' + t.variables.loopCounter + ' из 3' : '';
    showActions('<p>Процесс ждёт человека: <strong>' + escapeHTML(t.name || t.element_id) + '</strong>, ' +
      escapeHTML(who + round) + '.</p>', actions.map(function (a) {
      return { label: a.label, ghost: a.ghost, run: function () { return completeTask(t, a); } };
    }));
  }

  // --- ожидание сообщения: кнопка за внешнюю систему ---
  function waitsForMessage(records) {
    var waiting = {};
    records.forEach(function (r) {
      if (element(r.process_name, r.element_id).type !== 'receiveTask') return;
      if (r.event === 'activated') waiting[r.element_id] = true;
      else delete waiting[r.element_id];
    });
    return Object.keys(waiting).length > 0;
  }

  function renderMessage(inst, records) {
    if (busy) return;
    if (inst.status !== 'running' || !inst.business_key || !waitsForMessage(records)) {
      run.tasks.hidden = true;
      return;
    }
    var msg = { name: 'paymentReceived', correlationKey: inst.business_key, variables: { paid: true } };
    showActions('<p>Процесс ждёт сообщение <code>paymentReceived</code> с ключом <code>' +
      escapeHTML(inst.business_key) + '</code> — так платёжная система сообщила бы об оплате.</p>', [
      { label: 'Прислать оплату', run: function () { return api('POST', '/messages', msg); } },
    ]);
  }

  // showActions показывает текст и кнопки; после действия опрос экземпляра начинается заново.
  function showActions(text, actions) {
    run.tasks.hidden = false;
    run.tasks.innerHTML = text + '<div class="actions"></div>';
    var box = $('.actions', run.tasks);
    actions.forEach(function (a) {
      var b = document.createElement('button');
      b.type = 'button';
      b.className = 'btn' + (a.ghost ? ' btn-ghost' : ' btn-primary');
      b.textContent = a.label;
      b.addEventListener('click', function () { act(a.run); });
      box.appendChild(b);
    });
  }

  function act(fn) {
    busy = true;
    $$('button', run.tasks).forEach(function (b) { b.disabled = true; });
    showError(null);
    fn().then(function () {
      run.tasks.hidden = true;
    }).catch(function (e) {
      showError(e.message);
    }).then(function () {
      busy = false;
      startPolling();
    });
  }

  function completeTask(t, a) {
    var claim = t.assignee === a.user ? Promise.resolve() :
      api('POST', '/tasks/' + t.task_uid + '/claim', { userId: a.user, groups: a.groups || [] });
    return claim.then(function () {
      return api('POST', '/tasks/' + t.task_uid + '/complete', { userId: a.user, variables: a.vars });
    });
  }

  // --- опрос экземпляра ---
  function poll() {
    if (!current) return;
    var id = current.id;
    Promise.all([
      api('GET', '/instances/' + id),
      api('GET', '/instances/' + id + '/history?withChildren=true').catch(function (e) {
        if (e.status === 404) return { records: [] };
        throw e;
      }),
      WITH_TASKS[current.process] ? api('GET', '/tasks?instanceId=' + id).catch(function () { return { tasks: [] }; })
        : Promise.resolve({ tasks: [] }),
    ]).then(function (res) {
      if (!current || current.id !== id) return;
      var inst = res[0], records = res[1].records || [], tasks = res[2].tasks || [];
      var processes = {};
      records.forEach(function (r) { processes[r.process_name.split('.')[0]] = true; });
      return Promise.all(Object.keys(processes).map(loadNames)).then(function () {
        renderInstance(inst, records);
        if (current.process === 'TestMessage') renderMessage(inst, records);
        else renderTasks(tasks);
        var waiting = inst.status === 'running';
        if (waiting && Date.now() < pollUntil) {
          pollTimer = setTimeout(poll, tasks.length ? POLL_MS * 2 : POLL_MS);
        } else {
          refreshProcesses();
        }
      });
    }).catch(function (e) {
      if (e.status === 404) {
        try { sessionStorage.removeItem('demo.instance'); } catch (x) {}
        showError('Экземпляр не найден — возможно, его уже удалила очистка демо.');
        return;
      }
      showError('Не удалось получить экземпляр: ' + e.message);
      pollTimer = setTimeout(poll, POLL_MS * 4);
    });
  }

  function startPolling() {
    clearTimeout(pollTimer);
    pollUntil = Date.now() + POLL_LIMIT_MS;
    poll();
  }

  function track(id, process) {
    current = { id: id, process: process };
    try { sessionStorage.setItem('demo.instance', JSON.stringify(current)); } catch (e) {}
    showError(null);
    run.tasks.hidden = true;
    loadNames(process).then(startPolling);
  }

  // --- запуск ---
  $$('[data-start]').forEach(function (b) {
    b.addEventListener('click', function () {
      var process = b.closest('[data-process]').dataset.process;
      var body = JSON.parse(b.dataset.start);
      if (body.businessKey) body.businessKey += '-' + Math.floor(1000 + Math.random() * 9000);
      b.disabled = true;
      showError(null);
      api('POST', '/processes/' + encodeURIComponent(process) + '/instances', body).then(function (res) {
        track(res.id, process);
        if (window.matchMedia('(max-width: 960px)').matches) {
          $('[data-run]').scrollIntoView({ behavior: 'smooth', block: 'start' });
        }
      }).catch(function (e) {
        run.empty.hidden = false;
        showError('Не удалось запустить: ' + e.message);
      }).then(function () {
        setTimeout(function () { b.disabled = false; }, 800);
      });
    });
  });

  refreshProcesses();
  setInterval(function () { if (!document.hidden) refreshProcesses(); }, 20000);

  // экземпляр, за которым следили до перезагрузки страницы
  try {
    var saved = JSON.parse(sessionStorage.getItem('demo.instance') || 'null');
    if (saved && saved.id) track(saved.id, saved.process);
  } catch (e) {}
})();

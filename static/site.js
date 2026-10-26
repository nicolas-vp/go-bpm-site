// Общие скрипты сайта: тема, поиск по документации, копирование кода, меню и оглавление документации.
(function () {
  'use strict';

  var root = document.documentElement;

  // --- тема: системная, пока посетитель не выбрал сам ---
  function systemDark() {
    return window.matchMedia && matchMedia('(prefers-color-scheme: dark)').matches;
  }
  function currentDark() {
    var t = root.dataset.theme;
    return t ? t === 'dark' : systemDark();
  }
  document.querySelectorAll('[data-theme-toggle]').forEach(function (b) {
    b.addEventListener('click', function () {
      var next = currentDark() ? 'light' : 'dark';
      root.dataset.theme = next;
      try { localStorage.setItem('theme', next); } catch (e) {}
    });
  });

  // --- копирование: кнопка у каждого блока кода и у команды установки ---
  function copyText(text, button) {
    var done = function () {
      var label = button.textContent;
      button.textContent = 'Скопировано';
      setTimeout(function () { button.textContent = label; }, 1400);
    };
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text).then(done, function () {});
    }
  }
  document.querySelectorAll('.code').forEach(function (block) {
    var b = document.createElement('button');
    b.type = 'button';
    b.className = 'copy';
    b.textContent = 'Копировать';
    b.addEventListener('click', function () {
      copyText(block.querySelector('code').innerText.replace(/\n$/, ''), b);
    });
    block.appendChild(b);
  });
  document.querySelectorAll('[data-copy]').forEach(function (b) {
    b.addEventListener('click', function () { copyText(b.dataset.copy, b); });
  });

  // --- меню документации на телефоне ---
  var toggle = document.querySelector('[data-sidebar-toggle]');
  var sidebar = document.querySelector('[data-sidebar]');
  if (toggle && sidebar) {
    toggle.addEventListener('click', function () {
      var open = sidebar.classList.toggle('open');
      toggle.setAttribute('aria-expanded', String(open));
    });
  }

  // --- оглавление: подсветка раздела, который сейчас на экране ---
  var tocLinks = Array.prototype.slice.call(document.querySelectorAll('.toc a'));
  if (tocLinks.length && 'IntersectionObserver' in window) {
    var byId = {};
    tocLinks.forEach(function (a) { byId[decodeURIComponent(a.hash.slice(1))] = a; });
    var visible = {};
    var observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) { visible[e.target.id] = e.isIntersecting; });
      var heads = document.querySelectorAll('.prose h2[id], .prose h3[id]');
      var active = null;
      for (var i = 0; i < heads.length; i++) {
        if (visible[heads[i].id]) { active = heads[i].id; break; }
        if (heads[i].getBoundingClientRect().top < 120) active = heads[i].id;
      }
      tocLinks.forEach(function (a) { a.classList.remove('active'); });
      if (active && byId[active]) byId[active].classList.add('active');
    }, { rootMargin: '-60px 0px -60% 0px' });
    document.querySelectorAll('.prose h2[id], .prose h3[id]').forEach(function (h) { observer.observe(h); });
  }

  // --- галерея на главной ---
  document.querySelectorAll('[data-gallery]').forEach(function (g) {
    var tabs = g.querySelectorAll('[data-tab]');
    tabs.forEach(function (t) {
      t.addEventListener('click', function () {
        tabs.forEach(function (x) { x.setAttribute('aria-selected', String(x === t)); });
        g.querySelectorAll('[data-pane]').forEach(function (p) { p.hidden = p.dataset.pane !== t.dataset.tab; });
      });
    });
  });

  // --- поиск по документации: индекс search.json грузится при первом открытии ---
  var dialog = document.querySelector('[data-search]');
  var input = document.querySelector('[data-search-input]');
  var list = document.querySelector('[data-search-results]');
  var index = null;
  var selected = 0;

  function openSearch() {
    dialog.hidden = false;
    input.value = '';
    list.innerHTML = '';
    input.focus();
    if (!index) {
      fetch('/search.json').then(function (r) { return r.json(); }).then(function (data) {
        index = data.map(function (e) {
          e.lt = e.t.toLowerCase();
          e.lx = e.x.toLowerCase();
          return e;
        });
        if (input.value) run();
      });
    }
  }
  function closeSearch() { dialog.hidden = true; }

  function escapeHTML(s) {
    return s.replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }
  function mark(text, words) {
    var html = escapeHTML(text);
    words.forEach(function (w) {
      var re = new RegExp('(' + escapeHTML(w).replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ')', 'gi');
      html = html.replace(re, '<mark>$1</mark>');
    });
    return html;
  }
  function snippet(e, words) {
    var pos = e.lx.indexOf(words[0]);
    if (pos < 0) return e.x.slice(0, 160);
    var start = Math.max(0, pos - 60);
    return (start > 0 ? '…' : '') + e.x.slice(start, start + 180);
  }

  function run() {
    var q = input.value.trim().toLowerCase();
    if (!q || !index) { list.innerHTML = ''; return; }
    var words = q.split(/\s+/);
    var hits = [];
    index.forEach(function (e) {
      var score = 0;
      for (var i = 0; i < words.length; i++) {
        var w = words[i];
        var inTitle = e.lt.indexOf(w) >= 0;
        var inText = e.lx.indexOf(w) >= 0;
        if (!inTitle && !inText) return;
        score += inTitle ? 10 : 1;
        if (e.lt.indexOf(w) === 0) score += 5;
      }
      hits.push({ e: e, score: score });
    });
    hits.sort(function (a, b) { return b.score - a.score; });
    hits = hits.slice(0, 12);
    selected = 0;
    if (!hits.length) {
      list.innerHTML = '<li class="sr-none">Ничего не нашлось</li>';
      return;
    }
    list.innerHTML = hits.map(function (h, i) {
      var e = h.e;
      return '<li><a href="' + escapeHTML(e.u) + '"' + (i === 0 ? ' aria-selected="true"' : '') + '>' +
        '<div class="sr-page">' + escapeHTML(e.p) + '</div>' +
        '<div class="sr-title">' + mark(e.t, words) + '</div>' +
        '<div class="sr-text">' + mark(snippet(e, words), words) + '</div></a></li>';
    }).join('');
  }

  function move(delta) {
    var links = list.querySelectorAll('a');
    if (!links.length) return;
    selected = (selected + delta + links.length) % links.length;
    links.forEach(function (a, i) { a.setAttribute('aria-selected', String(i === selected)); });
    links[selected].scrollIntoView({ block: 'nearest' });
  }

  if (dialog) {
    document.querySelectorAll('[data-search-open]').forEach(function (b) { b.addEventListener('click', openSearch); });
    input.addEventListener('input', run);
    input.addEventListener('keydown', function (e) {
      if (e.key === 'ArrowDown') { e.preventDefault(); move(1); }
      if (e.key === 'ArrowUp') { e.preventDefault(); move(-1); }
      if (e.key === 'Enter') {
        var a = list.querySelectorAll('a')[selected];
        if (a) { closeSearch(); location.href = a.href; }
      }
    });
    list.addEventListener('click', function (e) { if (e.target.closest('a')) closeSearch(); });
    dialog.addEventListener('click', function (e) { if (e.target === dialog) closeSearch(); });
    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && !dialog.hidden) closeSearch();
      var typing = /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement.tagName);
      if (!typing && dialog.hidden && (e.key === '/' || (e.key === 'k' && (e.metaKey || e.ctrlKey)))) {
        e.preventDefault();
        openSearch();
      }
    });
  }
})();

// Подсказки к столбцам графиков: data-tip у строки, показывается у курсора и по фокусу с клавиатуры.
(function () {
  'use strict';
  var rows = document.querySelectorAll('[data-tip]');
  if (!rows.length) return;
  var tip = document.createElement('div');
  tip.className = 'tip';
  tip.hidden = true;
  tip.setAttribute('role', 'tooltip');
  document.body.appendChild(tip);
  function show(el, x, y) {
    tip.textContent = el.dataset.tip;
    tip.hidden = false;
    var w = tip.offsetWidth, h = tip.offsetHeight;
    tip.style.left = Math.max(8, Math.min(x + 14, window.innerWidth - w - 8)) + 'px';
    tip.style.top = Math.max(8, y - h - 12) + 'px';
  }
  rows.forEach(function (el) {
    el.tabIndex = 0;
    el.addEventListener('mousemove', function (e) { show(el, e.clientX, e.clientY); });
    el.addEventListener('mouseleave', function () { tip.hidden = true; });
    el.addEventListener('focus', function () {
      var r = el.getBoundingClientRect();
      show(el, r.left + r.width / 2, r.top);
    });
    el.addEventListener('blur', function () { tip.hidden = true; });
  });
})();

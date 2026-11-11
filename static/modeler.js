// Страница /modeler/: сборки go-bpm-modeler из /downloads/modeler/index.json. Его пишет publish.sh моделера на
// сервере; каналы - latest (релиз по тегу) и dev (последняя сборка main), null - канала ещё нет.
(function () {
  "use strict";

  var en = document.documentElement.lang === 'en';

  var root = document.querySelector("[data-downloads]");
  if (!root) return;
  var status = root.querySelector("[data-downloads-status]");

  // Платформа посетителя: сборки для неё выделяются. Архитектуру процессора браузер надёжно не сообщает -
  // у Mac и Linux выделяются обе сборки
  function guessPlatforms() {
    var p = ((navigator.userAgentData && navigator.userAgentData.platform) || navigator.platform || "").toLowerCase();
    var ua = navigator.userAgent.toLowerCase();
    if (p.indexOf("win") === 0 || ua.indexOf("windows") >= 0) return ["windows-amd64"];
    if (p.indexOf("mac") === 0 || ua.indexOf("mac os") >= 0) return ["darwin-arm64", "darwin-amd64"];
    if (p.indexOf("linux") >= 0 || ua.indexOf("linux") >= 0) return ["linux-amd64", "linux-arm64"];
    return [];
  }
  var mine = guessPlatforms();

  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text != null) e.textContent = text;
    return e;
  }

  function size(bytes) {
    var value = (bytes / 1048576).toFixed(1);
    return (en ? value : value.replace(".", ",")) + (en ? " MB" : " МБ");
  }

  function date(iso) {
    var d = new Date(iso);
    if (isNaN(d)) return iso;
    return d.toLocaleString(en ? "en-US" : "ru-RU", { day: "numeric", month: "long", year: "numeric", hour: "2-digit", minute: "2-digit" });
  }

  function channel(m, title, note) {
    var card = el("article", "modeler-channel");
    var head = el("header");
    head.appendChild(el("h2", null, title));
    var meta = el("p", "modeler-meta");
    var version = el("code", null, m.version);
    meta.appendChild(version);
    meta.appendChild(document.createTextNode(" · " + date(m.date) + (en ? " \u00b7 commit " : " · коммит ")));
    var commit = el("a", null, m.commit.slice(0, 7));
    commit.href = m.commitUrl;
    commit.rel = "noopener";
    meta.appendChild(commit);
    head.appendChild(meta);
    card.appendChild(head);
    if (note) card.appendChild(el("p", "modeler-note", note));

    var list = el("ul", "modeler-files");
    m.files.forEach(function (f) {
      var li = el("li", mine.indexOf(f.platform) >= 0 ? "mine" : null);
      var info = el("div", "modeler-file");
      info.appendChild(el("strong", null, f.label));
      info.appendChild(el("span", null, f.file + " · " + size(f.size)));
      var sum = el("code", "modeler-sum", "sha256 " + f.sha256);
      sum.title = "SHA-256";
      info.appendChild(sum);
      li.appendChild(info);
      var a = el("a", "btn" + (mine.indexOf(f.platform) >= 0 ? " btn-primary" : ""), (en ? "Download" : "Скачать"));
      a.href = m.url + f.file;
      a.setAttribute("download", "");
      li.appendChild(a);
      list.appendChild(li);
    });
    card.appendChild(list);
    var sums = el("a", "modeler-sums", "SHA256SUMS");
    sums.href = m.url + "SHA256SUMS";
    card.appendChild(sums);
    return card;
  }

  fetch("/downloads/modeler/index.json", { cache: "no-cache" })
    .then(function (r) {
      if (!r.ok) throw new Error("HTTP " + r.status);
      return r.json();
    })
    .then(function (index) {
      if (!index.latest && !index.dev) throw new Error((en ? "no builds available yet" : "сборок ещё нет"));
      status.remove();
      if (index.latest) {
        root.appendChild(channel(index.latest, (en ? "Stable release " : "Стабильная версия ") + index.latest.version));
      }
      if (index.dev) {
        root.appendChild(channel(index.dev, (en ? "Build from main" : "Сборка из main"),
          index.latest ? (en ? "The latest changes before release. May be unstable." : "Самые свежие изменения, раньше релиза. Может быть нестабильной.")
            : (en ? "No stable releases yet. This is the latest build from main." : "Стабильных релизов пока нет — это последняя сборка из ветки main.")));
      }
    })
    .catch(function (err) {
      status.textContent = (en ? "Could not load available builds: " : "Не удалось загрузить список сборок: ") + err.message + (en ? ". Builds are also available as GitHub CI artifacts." : ". Сборки есть и в артефактах CI на GitHub.");
    });
})();

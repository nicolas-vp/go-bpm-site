# Контекст go-bpm-site

Журнал фактов и решений для разработки сайта. Новые решения — дописывать сюда.

## Что за проект go-bpm

- Модуль `github.com/nicolas-vp/go-bpm`, Go 1.26, `pgx/v5`. BPM-движок на PostgreSQL: процессы рисуются в
  BPMN/DMN (Camunda Modeler, Camunda 8 / zeebe), генератор `go-bpm-gen` превращает их в типизированный Go-код,
  движок исполняет его на очереди задач в той же базе. Каждый шаг — задача очереди, эффекты шага — в одной транзакции.
- Возможности: service task, задачи и скрипты, шлюзы exclusive/parallel, подпроцессы и callActivity, таймеры,
  граничные события таймера и ошибки, решения DMN (компилируются в Go), задачи пользователей как в Camunda 7,
  экземпляры с бизнес-ключом и историей, `StartProcessTx` — старт процесса в транзакции сервиса,
  HTTP API и веб-консоль `/ui/` с диаграммой bpmn-js и историей экземпляра.
- Генератор проверяет BPMN и отвергает неподдерживаемые элементы — сайт не должен обещать того, чего нет
  (список поддерживаемого — `../go-bpm/docs/bpmn.md`).

## Источник содержимого

Документация go-bpm — `../go-bpm/docs/*.md` (на GitHub — `nicolas-vp/go-bpm`, каталог `docs/`):

| Файл | Что там |
|---|---|
| `README.md` | оглавление |
| `quickstart.md` | демо за пять минут, свой сервис с нуля |
| `concepts.md` | принципы BPMN и DMN, токен, шаги и транзакции, повторы, переменные |
| `bpmn.md` | поддерживаемые элементы, настройки в Camunda Modeler, ограничения |
| `dmn.md` | таблицы решений, hit policy, история правил |
| `generator.md` | запуск генератора, раскладка, модель данных, методы и обработчики |
| `examples.md` | разбор процессов демо |
| `running.md` | настройки, таблицы, масштабирование, Docker, выкладка из CI |
| `api.md` | HTTP API и консоль |
| `context.md` | внутренний журнал решений движка (не для пользователей) |

- Картинки — `../go-bpm/docs/images/`: SVG схем (bpmn-js `saveSVG`, с белым фоном), `demo-*.svg` — процессы демо,
  `console-*.png` и `history-*.png` — скриншоты консоли и истории экземпляров. Исходники мини-схем —
  `docs/images/src/*.bpmn`.
- Документация пишется в go-bpm и меняется вместе с кодом. Сайт её не копирует: забирает на сборке из репозитория
  go-bpm (локально — `../go-bpm/docs`, в CI — checkout `nicolas-vp/go-bpm`). `context.md` на сайт не выводится,
  ссылки на него и на код ведут на GitHub.
- Живое демо работает на сервере (см. ниже) — на него можно ссылаться или проксировать.

## Сайт

- Адрес — `go-bpm.tech` (и `www.`), DNS указывает на сервер. Разделы: `/` главная, `/docs/<страница>/` документация,
  `/demo/` живое демо, `/ui/` и `/api/` — демо go-bpm через прокси nginx.
- Генератор свой, на Go без зависимостей (`cmd/build`): Markdown разбирает `internal/md` — подмножество GFM, которым
  написаны docs go-bpm (заголовки, списки с вложенными блоками, таблицы, цитаты, код, `<details>`); подсветка кода —
  свой простой лексер. Причина: сторонние модули из окружения разработчика не скачать (корпоративный GOPROXY),
  а Hugo и Node требовали бы правок в документации (front matter) и лишнего в CI. Если docs начнут использовать новую
  разметку — дописать парсер и тест в `internal/md/md_test.go`.
- Якоря заголовков — как у GitHub (`#свой-сервис-с-нуля`), чтобы ссылки в docs работали и на GitHub, и на сайте.
  Сборка предупреждает о ссылках на несуществующие страницы и якоря.
- Шаблоны — `templates/*.html` (html/template), статика — `static/`, всё встроено в сборщик (`site.go`).
  Тема светлая и тёмная, поиск по документации на клиенте (`search.json`, режется по заголовкам ## и ###).
- Страница демо (`static/demo.js`) ходит в `/api/v1` того же сайта: запуск процессов демо с готовыми переменными,
  опрос экземпляра и истории, кнопки задач пользователей для `TestUserTask` (bob из managers, alice). Если в go-bpm
  поменяются процессы демо или API — проверить `templates/demo.html` и `static/demo.js`.
- Операции над экземплярами go-bpm (`fe2573c`, `c14f047`): повтор упавшей задачи, исправление переменных, отмена,
  термокарта, динамика. На главной — раздел «Сопровождение» и вкладка «Упавший экземпляр» в галерее, на демо —
  ссылки на вкладки консоли и кнопка `amount = "abc"` у TestBpmProcess (очередь демо — одна попытка, экземпляр
  падает сразу, его можно починить в истории). POST `/api/v1/failures/.../retry` и `/instances/.../cancel`
  проходят через прокси, как и запуск: посетитель может повторить или отменить чужой экземпляр — для демо это
  приемлемо, лимит POST тот же. Статус `cancelled` есть в `demo.js` и в очистке (`deploy/cleanup.sh`).
- Тексты главной и демо написаны руками по README и docs go-bpm: не обещать неподдерживаемого.
- Страница «О нас» (`/docs/about/`, `content/about.md`) — разработчики и их почта; ссылки на неё в шапке и подвале.
- Страница «Тестирование процессов» (`/docs/testing/`) пишется здесь, в `content/testing.md`: страницы с `site: true`
  в списке `pages` (`cmd/build`) берутся из `content/`, остальные — из docs go-bpm. Код в ней не копируется руками:
  блок ```` ```go file=<путь> decl=<имена> ```` сборщик заменяет объявлениями из файла (`cmd/build/include.go`,
  через go/parser); путь `go-bpm/...` — от корня репозитория go-bpm.
- Примеры страницы: `examples/orders` — отдельный модуль, процесс OrderApproval из «Быстрого старта», сгенерирован
  настоящим go-bpm-gen (go-bpm — из `../../../go-bpm`, т. е. соседнего каталога), юнит-тесты метода, условия
  и обработчика с заглушкой почты, e2e на PostgreSQL (`internal/e2e`, база из `GO_BPM_TEST_DSN`, таблицы
  очищаются). `examples/demo/decision_test.go` — тест решения DMN демо go-bpm, подкладывается в его пакет через
  `go test -overlay`, без правок в go-bpm (`go.mod` рядом — только чтобы `go vet ./...` сайта его не трогал).
  `make examples` гоняет всё; job `examples` в CI — то же с PostgreSQL 18 и проверкой, что `go generate` не даёт
  diff. Deploy ждёт и build, и examples: сломанный пример не выкатится.
- Локально: `make build`, `make serve` (`cmd/serve` раздаёт `dist/` и проксирует `/api/`, `/ui/` в демо на
  `localhost:8080`, его можно пробросить `ssh -L`).

## Сервер

- `root@5.180.47.109`, SSH по ключу без пароля (ключ разработчика уже прописан). Ubuntu 24.04, amd64,
  1 ГБ памяти (свободно ~500 МБ), диск 25 ГБ (свободно ~16 ГБ).
- Docker 29 из пакетов Ubuntu (`docker.io`) и плагин `docker-compose-v2`; buildx не установлен.
- Установлен ispmanager: он управляет nginx (1.30, порт 80; 443 пока не слушается). Виртуальные хосты
  ispmanager кладёт в `/etc/nginx/vhosts/<пользователь>/*.conf` (сейчас каталога нет — сайтов не заведено),
  общие вставки — `/etc/nginx/vhosts-includes/` (там `letsencrypt.conf`). Править nginx в обход ispmanager —
  риск, что панель перезапишет конфиг; сайт лучше заводить через панель или класть конфиг в `conf.d`.
  Панель — `https://5.180.47.109:1500`.
- Занято: 22 sshd, 80 и `5.180.47.109:1501` nginx, 1500 ihttpd (ispmanager), 21 proftpd, 53 named,
  3306/33060 MySQL (localhost), 3128 squid в контейнере `my-private-proxy` (localhost), 8080 — демо go-bpm (localhost).
- Демо go-bpm: `/opt/go-bpm` (`compose.yaml`, `.env` с паролем базы — только на сервере, `deploy.sh`),
  контейнеры `go-bpm-go-bpm-1` (`ghcr.io/nicolas-vp/go-bpm`, `127.0.0.1:8080`) и `go-bpm-postgres-1`
  (`postgres:18`, том `go-bpm_pgdata`). Консоль и API без авторизации — наружу только через прокси с ней.
  Наружу — только через nginx сайта (ниже); напрямую порт 8080 слушает localhost.

## Выкладка (как сделано в go-bpm — образец)

- CI go-bpm (`../go-bpm/.github/workflows/ci.yml`): `test` → `image` (сборка и публикация в GHCR
  `ghcr.io/nicolas-vp/go-bpm`, теги `latest`, `sha-<7 знаков>`, semver из `v*`) → `deploy` (только push в main,
  `concurrency: deploy`).
- `deploy` заходит `ssh root@$DEPLOY_HOST "<тег> <actor> <GITHUB_TOKEN>" < compose.yaml`. Ключ CI прописан в
  `/root/.ssh/authorized_keys` с `command="/opt/go-bpm/deploy.sh",restrict` — ничего, кроме скрипта, он не выполнит.
  Скрипт (`../go-bpm/deploy/deploy.sh`): проверяет тег, принимает `compose.yaml` из коммита после
  `docker compose config`, логинится в GHCR во временный `DOCKER_CONFIG`, pull, тег в `.env`, `up -d --wait`,
  удаляет старые образы, ждёт HTTP-ответа.
- Секреты репозитория: `DEPLOY_HOST`, `DEPLOY_SSH_KEY`, `DEPLOY_KNOWN_HOSTS`
  (`5.180.47.109 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILC7PFoour3HJhP1FY05U6X7+57TIW/k0KEZVnxj1pvC`).
- У сайта свой ключ CI и свой скрипт (forced command одного ключа вызывает один скрипт) — см. «Выкладка сайта».

## Выкладка сайта

- nginx: `/etc/nginx/conf.d/go-bpm.tech.conf`, `/etc/nginx/go-bpm-proxy.inc`, `/etc/nginx/go-bpm-tls.inc` — копии
  `deploy/` из репозитория,
  ставятся руками (`nginx -t && systemctl reload nginx`). Корень — `/var/www/go-bpm.tech/current`, симлинк на релиз
  в `releases/`.
- Демо открыто посетителям с ограничениями: `/api/` — только GET/HEAD/POST, POST — 12 в минуту с IP (всплеск 10),
  чтение — 10 в секунду (всплеск 40), тело не больше 16 КБ; `/ui/` — только GET/HEAD (консоль читает, а запускать
  можно из неё же через API). Лишнее — 403, превышение лимита — 429.
- Очистка демо: `/opt/go-bpm-site/cleanup.sh` (копия `deploy/cleanup.sh`), cron `/etc/cron.d/go-bpm-site-cleanup`
  в 4:30, лог `/var/log/go-bpm-site-cleanup.log`. Удаляет завершённые и упавшие экземпляры старше 3 дней и ждущие
  задачу пользователя дольше 14 дней — с потомками, историей, правилами и задачами.
- CI (`.github/workflows/site.yml`): сборка и тесты на push, PR, вручную, по расписанию (раз в сутки — чтобы подтянуть
  docs go-bpm) и по `repository_dispatch` `go-bpm-docs`; deploy — только main: `tar.gz` из `dist/` по SSH
  в `/opt/go-bpm-site/deploy.sh` (копия `deploy/deploy.sh`), аргумент — 7 знаков коммита. Скрипт проверяет архив,
  переключает `current`, хранит 5 релизов и сверяет главную, которую отдаёт nginx.
- Секреты репозитория go-bpm-site: `DEPLOY_HOST`, `DEPLOY_SSH_KEY` (ключ `go-bpm-site-ci` с
  `command="/opt/go-bpm-site/deploy.sh",restrict` в `/root/.ssh/authorized_keys`), `DEPLOY_KNOWN_HOSTS`.
- HTTPS — Let's Encrypt (certbot из пакетов Ubuntu, webroot `/var/www/acme`, сертификат
  `/etc/letsencrypt/live/go-bpm.tech`, выпущен 2026-10-26, без email). Продлевает `certbot.timer`, после продления
  nginx перечитывается хуком `/etc/letsencrypt/renewal-hooks/deploy/reload-nginx` (копия `deploy/reload-nginx`).
  TLS-настройки — `/etc/nginx/go-bpm-tls.inc` (копия `deploy/go-bpm-tls.inc`). HTTP и `www.` — 301 на
  `https://go-bpm.tech`, кроме `/.well-known/acme-challenge/`; включён HSTS на год. Сертификат от регистратора
  домена ещё выпускается; когда придёт, можно заменить пути в `go-bpm-tls.inc` (и выключить продление certbot)
  или остаться на Let's Encrypt.

## Нагрузочный тест и страница «Насколько быстрый go-bpm»

- Повтор методики статьи «Zeebe и Camunda: сравниваем известные BPM-системы под высокими нагрузками» (Хабр, 2021):
  100 тыс. экземпляров процесса «старт → два REST-вызова → конец», скорость на отметках 1 тыс., 10 тыс. … —
  число завершённых, делённое на время от первого старта. Цифры статьи — `content/performance-article.json`
  (строка bpm-light/ClickHouse пересчитана из таблицы времени в мс).
- `bench/` — отдельный модуль (`replace` на `../../go-bpm`): процессы `BenchMethods` (два `task`) и
  `BenchServiceTasks` (две service task), команда `cmd/bench` (свой REST-заглушка, `-truncate` обязателен — таблицы
  `bpm_*` очищаются, `-v` — предупреждения очереди, замер коммитов/с до и после прогона). `run.sh` гоняет шесть
  вариантов по кругу `REPEAT` раз (3) с паузой; результаты — `bench/results/<вариант>/<N>.json`, они в репозитории,
  сборка сайта без них падает. Страница (`content/performance.md`, подстановки `@@chart:…@@`, `@@table@@`,
  `@@stand@@`, `@@v:<вариант>@@` — `cmd/build/perf.go`) показывает медиану прогонов, разброс — в подсказке.
- **Стенд** — рабочая станция `nicolas-vp@192.168.1.10` (та же i7-4770K, что в статье; Go нет — `bin/bench`
  собирается здесь `CGO_ENABLED=0` и копируется, `NO_BUILD=1 ./run.sh`). PostgreSQL 18 — контейнер
  `go-bpm-bench-postgres`, `--network host`, `listen_addresses=localhost`, `shared_buffers=2GB`, `max_wal_size=4GB`,
  том `go-bpm-bench-pgdata`, база `go_bpm_bench` (postgres/postgres).
- Что выяснилось:
  - ноутбук разработчика как стенд не годится: NVMe греется до 73 °C и через минуту записи держит в 4 раза меньше
    коммитов;
  - проброс порта Docker (`docker-proxy`) съедал ядро из четырёх: 571 → 899 экз./с после `--network host`;
  - гонка в `fetchQuery` go-bpm (один fetch брал в аренду несколько задач, лишние висели 40 с) — исправлена в
    go-bpm `d7d82c6`;
  - разброс между прогонами большой (диск стенда и стартеры без ограничения скорости делят 4 ядра с воркерами и
    PostgreSQL), поэтому медиана трёх прогонов.

## Окружение разработчика

- Локально нет доступа к Docker-демону и плагина compose: образы собираются в CI или на сервере.
- Go-модули из сети не скачать (рабочий GOPROXY недоступен), поэтому у сайта нет зависимостей.
- SSH к GitHub из шелла Claude может падать на проверке ключа хоста — пуш тогда делает пользователь.
- Глобальный `~/.gitconfig` — рабочий (ozon), не трогать; автор этого репозитория задан локально.

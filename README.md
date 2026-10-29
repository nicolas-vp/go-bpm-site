# go-bpm-site

Сайт [go-bpm](https://github.com/nicolas-vp/go-bpm) — <https://go-bpm.tech>: главная, документация из `docs/`
репозитория go-bpm и живое демо.

```sh
make build    # dist/ из ../go-bpm/docs
make serve    # http://localhost:8000, /api/ и /ui/ - в демо на localhost:8080
make test
```

Демо с сервера для `make serve`: `ssh -L 8080:127.0.0.1:8080 root@213.239.158.215`.

| Каталог | Что там |
|---|---|
| `cmd/build` | сборка сайта |
| `cmd/serve` | локальный просмотр |
| `internal/md` | Markdown в HTML и подсветка кода |
| `templates/`, `static/` | шаблоны страниц, стили, скрипты |
| `content/` | страницы сайта вне документации go-bpm: тестирование, скорость |
| `examples/` | примеры кода для страницы тестирования, их тесты гоняет CI |
| `bench/` | нагрузочный тест для страницы «Насколько быстрый go-bpm», результаты в `bench/results` |
| `deploy/` | nginx, скрипт выкладки и очистка демо на сервере |
| `docs/context.md` | решения и устройство выкладки |

Выкладка — CI на push в main, раз в сутки и вручную (`.github/workflows/site.yml`).

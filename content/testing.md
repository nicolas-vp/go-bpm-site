# Тестирование процессов

В go-bpm процесс — это Go-код: методы задач, условия шлюзов, решения DMN и обработчики service task — обычные
функции в пакетах вашего сервиса. Поэтому их тестируют обычным `go test`, без моделера, отдельного сервера процессов
и специального фреймворка. Процесс целиком проверяется e2e-тестом на настоящем PostgreSQL за секунды.

| Уровень | Что проверяет | Нужна база | Сколько длится |
|---|---|---|---|
| Генерация | схему, условия, типы переменных, сигнатуры методов | нет | `go generate` |
| Юнит-тесты | методы, условия шлюзов, решения DMN, обработчики с заглушками сервисов | нет | миллисекунды |
| E2E | процесс целиком: очередь, переходы, задачи пользователей, повторы, статусы экземпляра | PostgreSQL | секунды |

Весь код ниже — из примера [`examples/orders`](https://github.com/nicolas-vp/go-bpm-site/tree/main/examples/orders)
в репозитории сайта. CI сайта собирает его и прогоняет все тесты на PostgreSQL при каждой сборке, так что код на этой
странице рабочий.

## Пример: согласование заказа

Процесс `OrderApproval` из [Быстрого старта](quickstart.md#свой-сервис-с-нуля): проверить заказ, крупный — отдать на
согласование менеджеру, затем уведомить клиента.

![Процесс согласования заказа: «Проверить заказ», шлюз «Крупный заказ?», user task «Согласовать заказ» для группы managers, service task «Уведомить клиента»](images/order.svg)

Модель данных и внешние сервисы — рукописные части. Почта — интерфейс: в тестах вместо неё подставляется заглушка.

```go file=examples/orders/internal/bpmgen/model/data_model.go decl=DataModel
```

```go file=examples/orders/internal/bpmgen/services/services.go decl=Services,Mailer
```

## Метод задачи

Метод «Проверить заказ» выполняется синхронно в шаге процесса. Он получает модель и сервисы и меняет поля модели —
движок запишет их в переменные процесса.

```go file=examples/orders/internal/bpmgen/order_approval/method_check_order.go decl=CheckOrder
```

Тест — вызвать функцию и проверить поля. Движок, очередь и база не нужны:

```go file=examples/orders/internal/bpmgen/order_approval/method_check_order_test.go decl=TestCheckOrder
```

## Условие шлюза

Условие `=amount > 1000` генератор превратил в Go-функцию, а не в выражение, которое вычисляется в рантайме:

```go file=examples/orders/internal/bpmgen/order_approval/cond_is_big.go decl=isBig
```

Функция неэкспортируемая, поэтому тест лежит в пакете процесса (`package order_approval`). Границы условия
проверяются таблицей:

```go file=examples/orders/internal/bpmgen/order_approval/method_check_order_test.go decl=TestIsBig
```

## Обработчик service task и заглушки

Обработчик «Уведомить клиента» выполняется асинхронно, отдельной задачей очереди с повторами. Внешний мир он видит
только через `services.Services`:

```go file=examples/orders/internal/bpmgen/handlers/notify_client.go decl=NotifyClient
```

В тесте почта — заглушка, которая запоминает письма или возвращает ошибку. `bpm.Job` — обычная структура, её
заполняют нужными полями:

```go file=examples/orders/internal/bpmgen/handlers/notify_client_test.go
```

## Решение DMN

Таблица решений тоже генерируется в Go — каждое правило становится веткой `if`. Пример из демо go-bpm, процесс
[`TestDecision`](examples.md#testdecision-бизнес-правила-dmn):

```go file=go-bpm/examples/demo/bpmgen/test_decision/decision_decision_1.go decl=decideDecision1
```

Тест проходит по строкам таблицы и проверяет и результат, и сработавшее правило — то, что потом видно в истории
правил консоли. Задачу бизнес-правила тоже можно вызвать напрямую: вне движка запись в историю правил
(`bpm.RecordDecision`) просто пропускается.

```go file=examples/demo/decision_test.go
```

## Процесс целиком: e2e

Юнит-тесты не проверяют, что части собраны в процесс правильно: что шлюз ведёт куда нужно, задача пользователя
появляется у нужной группы, а упавший обработчик повторяется и роняет экземпляр. Это проверяет e2e-тест: настоящие
очередь и движок на тестовой базе PostgreSQL, как в `main.go`, только с заглушками сервисов.

Процесс идёт асинхронно: `StartProcess` только ставит старт в очередь. Поэтому тест ждёт результата — статуса
экземпляра через `HistoryReader.GetInstance` или задачи пользователя через `UserTaskService.ListTasks`.

```go file=examples/orders/internal/e2e/order_test.go decl=TestSmallOrder,TestBigOrderNeedsApproval,TestMailerDown,TestInvalidOrder
```

Окружение теста — движок на тестовой базе. Настройки ускоряют падения и повторы: две попытки на шаг вместо десяти,
повторы через 50 мс вместо секунды.

```go file=examples/orders/internal/e2e/order_test.go decl=testEnv,start,testPool
```

Ожидание статуса и задачи — опрос с таймаутом. Заглушка почты здесь под мьютексом: обработчик вызывают воркеры
очереди из своих горутин.

```go file=examples/orders/internal/e2e/order_test.go decl=testEnv.waitStatus,testEnv.waitTask,fakeMailer,fakeMailer.Send,fakeMailer.Sent
```

### Запуск

Без `GO_BPM_TEST_DSN` e2e-тесты пропускаются, поэтому `go test ./...` работает и без базы. С базой:

```sh
docker run -d --name orders-test-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=orders_test -p 5433:5432 postgres:18
GO_BPM_TEST_DSN='postgres://postgres:postgres@localhost:5433/orders_test?sslmode=disable' go test ./...
```

Четыре e2e-сценария примера проходят примерно за две секунды: таймаутов и долгих задержек в процессе нет,
а повторы ускорены настройками теста.

> **База очищается.** `testPool` применяет миграции и очищает таблицы `bpm_*` перед каждым тестом. Не направляйте
> `GO_BPM_TEST_DSN` на базу с данными. Тесты с базой делят таблицы, поэтому не запускайте их параллельно:
> без `t.Parallel()`, а если e2e-тесты лежат в нескольких пакетах — `go test -p 1`.

### В CI

PostgreSQL поднимается сервисом job. Например, в GitHub Actions:

```yaml
test:
  runs-on: ubuntu-latest
  services:
    postgres:
      image: postgres:18
      env:
        POSTGRES_PASSWORD: postgres
        POSTGRES_DB: orders_test
      ports: ["5432:5432"]
      options: --health-cmd pg_isready --health-interval 2s --health-retries 15
  env:
    GO_BPM_TEST_DSN: postgres://postgres:postgres@localhost:5432/orders_test?sslmode=disable
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version-file: go.mod
    - run: go generate ./... && git diff --exit-code   # сгенерированный код совпадает со схемами
    - run: go test ./...
```

`git diff --exit-code` после генерации ловит схему, которую поменяли в моделере, но забыли перегенерировать.

## Тесты самого go-bpm

В репозитории движка — образцы на все случаи. Они используют процессы демо и внутренний хелпер базы, но устроены
так же, как тесты выше.

| Файл | Что проверяет |
|---|---|
| [`engine/e2e_test.go`](https://github.com/nicolas-vp/go-bpm/blob/main/engine/e2e_test.go) | все процессы демо доходят до End event, история и экземпляры читаются обратно |
| [`engine/user_tasks_e2e_test.go`](https://github.com/nicolas-vp/go-bpm/blob/main/engine/user_tasks_e2e_test.go) | задачи пользователей: очередь группы, `Claim`, `Unclaim`, `Complete`, ошибки, возврат на исправление |
| [`engine/start_tx_e2e_test.go`](https://github.com/nicolas-vp/go-bpm/blob/main/engine/start_tx_e2e_test.go) | `StartProcessTx`: откат транзакции — нет экземпляра, коммит — экземпляр виден сразу |
| [`engine/timers_test.go`](https://github.com/nicolas-vp/go-bpm/blob/main/engine/timers_test.go) | граничные таймеры и ошибки: кто успел первым, отмена таймера, ошибки service task |
| [`engine/decision_test.go`](https://github.com/nicolas-vp/go-bpm/blob/main/engine/decision_test.go) | задача бизнес-правила и история сработавших правил |
| [`engine/call_activity_test.go`](https://github.com/nicolas-vp/go-bpm/blob/main/engine/call_activity_test.go) | вызов процесса и возврат токена в родителя |
| [`internal/generator/codegen/validate_test.go`](https://github.com/nicolas-vp/go-bpm/blob/main/internal/generator/codegen/validate_test.go) | что генератор отвергает и что восстанавливает в BPMN |

Как запускать тесты репозитория go-bpm — в [Запуске и эксплуатации](running.md#тесты-репозитория-go-bpm).

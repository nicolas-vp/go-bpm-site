// Пример сервиса для страницы «Тестирование процессов»: процесс OrderApproval из «Быстрого старта» go-bpm,
// юнит-тесты метода, условия и обработчика, e2e на PostgreSQL. Код страницы вставляется из этих файлов.
// go-bpm берётся из соседнего каталога (../go-bpm рядом с go-bpm-site, в CI - так же).
module example.com/orders

go 1.26.0

require (
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.9.2
	github.com/nicolas-vp/go-bpm v0.0.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
)

replace github.com/nicolas-vp/go-bpm => ../../../go-bpm

tool github.com/nicolas-vp/go-bpm/cmd/go-bpm-gen

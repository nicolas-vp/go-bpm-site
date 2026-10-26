// Нагрузочный тест go-bpm для страницы «Насколько быстрый go-bpm»: процесс из двух REST-вызовов, как в статье
// «Zeebe и Camunda: сравниваем известные BPM-системы под высокими нагрузками» (Хабр, 2021).
// go-bpm берётся из соседнего каталога (../go-bpm рядом с go-bpm-site).
module example.com/bench

go 1.26.0

require (
	github.com/jackc/pgx/v5 v5.9.2
	github.com/nicolas-vp/go-bpm v0.0.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
)

replace github.com/nicolas-vp/go-bpm => ../../go-bpm

tool github.com/nicolas-vp/go-bpm/cmd/go-bpm-gen

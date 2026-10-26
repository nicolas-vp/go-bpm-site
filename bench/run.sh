#!/bin/bash
# Полный набор прогонов для страницы «Насколько быстрый go-bpm»: два варианта процесса, с историей и без,
# по 100 тыс. экземпляров, и оба варианта с историей при synchronous_commit=off (задаётся для соединений теста).
# Каждый вариант прогоняется REPEAT раз с паузой PAUSE секунд: на странице медиана и разброс.
# Результаты - results/<вариант>/<номер>.json, их читает сборка сайта.
#
#	GO_BPM_BENCH_DSN='postgres://user:pass@localhost:5432/go_bpm_bench?sslmode=disable' ./run.sh
#
# Таблицы bpm_* базы очищаются перед каждым прогоном: нужна отдельная база.
set -euo pipefail
cd "$(dirname "$0")"
: "${GO_BPM_BENCH_DSN:?задайте GO_BPM_BENCH_DSN - DSN отдельной базы}"
N=${N:-100000}
REPEAT=${REPEAT:-3}
PAUSE=${PAUSE:-30}

# NO_BUILD=1 - взять готовый bin/bench: стенд без Go, бинарник собран на другой машине
[ -n "${NO_BUILD:-}" ] || go build -o bin/bench ./cmd/bench

# synchronous_commit=off: коммит не ждёт записи WAL на диск, при падении сервера базы теряются последние доли
# секунды транзакций, но база остаётся целой
async="options=-c%20synchronous_commit%3Doff"
variants=(
	"service-history-true service true"
	"service-history-false service false"
	"methods-history-true methods true"
	"methods-history-false methods false"
	"service-history-true-async-commit service true $async"
	"methods-history-true-async-commit methods true $async"
)
for i in $(seq "$REPEAT"); do
	for v in "${variants[@]}"; do
		read -r name process history options <<<"$v"
		mkdir -p "results/$name"
		dsn=$GO_BPM_BENCH_DSN
		[ -z "${options:-}" ] || dsn="$dsn&$options"
		bin/bench -truncate -process "$process" -history="$history" -n "$N" -out "results/$name/$i.json" -dsn "$dsn" "$@"
		echo
		sleep "$PAUSE"
	done
done

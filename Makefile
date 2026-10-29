GO_BPM ?= ../go-bpm

.PHONY: build serve test examples presentation

# собрать сайт в dist/ из документации соседнего репозитория go-bpm
build:
	go run ./cmd/build -docs $(GO_BPM)/docs -out dist -commit $$(git -C $(GO_BPM) rev-parse HEAD)

# посмотреть локально: /api/ и /ui/ уходят в демо на localhost:8080
# (ssh -L 8080:127.0.0.1:8080 root@213.239.158.215 - демо с сервера)
serve: build
	go run ./cmd/serve -dir dist

test:
	go vet ./...
	go test ./...

# тесты примеров страницы «Тестирование процессов»: e2e - только с GO_BPM_TEST_DSN (база очищается!)
examples:
	cd examples/orders && go vet ./... && go test ./...
	# тест решения DMN подкладывается в пакет демо go-bpm без правок в его репозитории
	mkdir -p dist
	printf '{"Replace":{"%s/examples/demo/bpmgen/test_decision/decision_test.go":"%s/examples/demo/decision_test.go"}}' \
		$(abspath $(GO_BPM)) $(CURDIR) > dist/overlay.json
	cd $(GO_BPM) && go test -overlay $(CURDIR)/dist/overlay.json ./examples/demo/bpmgen/test_decision/

# презентация: presentation/go-bpm.md -> presentation/go-bpm.pdf (нужен google-chrome)
presentation:
	go run ./presentation/build -out /tmp/go-bpm-presentation.html
	google-chrome --headless --no-sandbox --no-pdf-header-footer --print-to-pdf=presentation/go-bpm.pdf /tmp/go-bpm-presentation.html

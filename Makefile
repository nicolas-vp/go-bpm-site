GO_BPM ?= ../go-bpm

.PHONY: build serve test

# собрать сайт в dist/ из документации соседнего репозитория go-bpm
build:
	go run ./cmd/build -docs $(GO_BPM)/docs -out dist -commit $$(git -C $(GO_BPM) rev-parse HEAD)

# посмотреть локально: /api/ и /ui/ уходят в демо на localhost:8080
# (ssh -L 8080:127.0.0.1:8080 root@5.180.47.109 - демо с сервера)
serve: build
	go run ./cmd/serve -dir dist

test:
	go vet ./...
	go test ./...

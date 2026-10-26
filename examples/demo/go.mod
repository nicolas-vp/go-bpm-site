// Тест решения DMN из демо go-bpm для страницы «Тестирование процессов». Это не отдельный модуль: файл
// подкладывается в пакет examples/demo/bpmgen/test_decision репозитория go-bpm через go test -overlay
// (make examples). go.mod здесь только затем, чтобы go vet ./... сайта не пытался собрать этот тест.
module example.com/demo-dmn

go 1.26

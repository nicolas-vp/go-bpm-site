// Package services - внешние сервисы для методов task и обработчиков jobType
package services

import "context"

// Services - внешние сервисы, которые получают методы task и обработчики jobType.
// Заполняются при старте и передаются в сгенерированные Processes и Queues.
// Поля удобнее делать интерфейсами, чтобы в тестах подставлять заглушки
type Services struct {
	Rest RestClient
	// Finished вызывается последней задачей процесса: по нему нагрузочный тест считает завершённые экземпляры
	Finished func()
}

// RestClient - REST-вызов, который делает каждая задача процесса
type RestClient interface {
	Call(ctx context.Context) error
}

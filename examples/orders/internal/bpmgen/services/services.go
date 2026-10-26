// Package services - внешние сервисы для методов task и обработчиков jobType
package services

import "context"

// Services - внешние сервисы, которые получают методы task и обработчики jobType.
// Заполняются при старте и передаются в сгенерированные Processes и Queues.
// Поля удобнее делать интерфейсами, чтобы в тестах подставлять заглушки
type Services struct {
	Mailer Mailer
}

// Mailer - отправка писем клиентам
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

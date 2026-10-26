package handlers

import (
	"context"
	"errors"
	"testing"

	"example.com/orders/internal/bpmgen/model"
	"example.com/orders/internal/bpmgen/services"
	"github.com/nicolas-vp/go-bpm/bpm"
)

// fakeMailer - заглушка services.Mailer: запоминает письма или возвращает заданную ошибку.
type fakeMailer struct {
	err  error
	sent []string
}

func (m *fakeMailer) Send(ctx context.Context, to, subject, body string) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, to+": "+subject)
	return nil
}

func TestNotifyClient(t *testing.T) {
	mailer := &fakeMailer{}
	svc := &services.Services{Mailer: mailer}
	job := bpm.Job{InstanceUid: "01a17411-e69b-7ea7", ElementId: "Notify", BusinessKey: "order-2"}
	data := &model.DataModel{Amount: 500, Email: "client@example.com"}

	if err := NotifyClient(context.Background(), svc, job, data); err != nil {
		t.Fatal(err)
	}
	if !data.Notified || len(mailer.sent) != 1 || mailer.sent[0] != "client@example.com: Заказ order-2" {
		t.Errorf("notified = %v, sent = %v", data.Notified, mailer.sent)
	}
}

// Ошибка почты - ошибка обработчика: очередь повторит задачу, а клиент не отмечен уведомлённым.
func TestNotifyClientMailerDown(t *testing.T) {
	svc := &services.Services{Mailer: &fakeMailer{err: errors.New("smtp: connection refused")}}
	data := &model.DataModel{Amount: 500, Email: "client@example.com"}

	if err := NotifyClient(context.Background(), svc, bpm.Job{}, data); err == nil {
		t.Fatal("ошибка почты потерялась")
	}
	if data.Notified {
		t.Error("клиент отмечен уведомлённым без письма")
	}
}

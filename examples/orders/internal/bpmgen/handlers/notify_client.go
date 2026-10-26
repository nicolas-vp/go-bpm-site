package handlers

import (
	"context"
	"fmt"

	"example.com/orders/internal/bpmgen/model"
	"example.com/orders/internal/bpmgen/services"
	"github.com/nicolas-vp/go-bpm/bpm"
)

// NotifyClient - обработчик service task с jobType "notifyClient" во всех процессах, где он встречается
// (job.ProcessName, job.ElementId). Вызывается асинхронно из очереди задач и при ошибке повторяется,
// поэтому должен быть идемпотентным (ключ - job.InstanceUid + job.ElementId)
func NotifyClient(ctx context.Context, svc *services.Services, job bpm.Job, data *model.DataModel) error {
	body := fmt.Sprintf("Заказ на сумму %.2f принят", data.Amount)
	if err := svc.Mailer.Send(ctx, data.Email, "Заказ "+job.BusinessKey, body); err != nil {
		return fmt.Errorf("письмо клиенту: %w", err)
	}
	data.Notified = true
	return nil
}

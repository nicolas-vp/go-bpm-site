package handlers

import (
	"context"

	"example.com/bench/internal/bpmgen/model"
	"example.com/bench/internal/bpmgen/services"
	"github.com/nicolas-vp/go-bpm/bpm"
)

// RestCall - обработчик service task с jobType "restCall" во всех процессах, где он встречается
// (job.ProcessName, job.ElementId). Вызывается асинхронно из очереди задач и при ошибке повторяется,
// поэтому должен быть идемпотентным (ключ - job.InstanceUid + job.ElementId)
func RestCall(ctx context.Context, svc *services.Services, job bpm.Job, data *model.DataModel) error {
	if err := svc.Rest.Call(ctx); err != nil {
		return err
	}
	if job.ElementId == "Rest2" {
		svc.Finished()
	}
	return nil
}

package bench_methods

import (
	"context"

	"example.com/bench/internal/bpmgen/model"
	"example.com/bench/internal/bpmgen/services"
)

// Rest1 - метод задачи Rest1 «Вызов REST №1»
func Rest1(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	return svc.Rest.Call(ctx)
}

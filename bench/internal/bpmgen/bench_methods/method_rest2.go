package bench_methods

import (
	"context"

	"example.com/bench/internal/bpmgen/model"
	"example.com/bench/internal/bpmgen/services"
)

// Rest2 - метод задачи Rest2 «Вызов REST №2»
func Rest2(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	if err := svc.Rest.Call(ctx); err != nil {
		return err
	}
	svc.Finished()
	return nil
}

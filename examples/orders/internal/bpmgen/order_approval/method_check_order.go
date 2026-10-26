package order_approval

import (
	"context"
	"fmt"

	"example.com/orders/internal/bpmgen/model"
	"example.com/orders/internal/bpmgen/services"
)

// CheckOrder - метод задачи CheckOrder «Проверить заказ»
func CheckOrder(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	if data.Amount <= 0 {
		return fmt.Errorf("сумма заказа должна быть положительной: %v", data.Amount)
	}
	data.Checked = true
	return nil
}

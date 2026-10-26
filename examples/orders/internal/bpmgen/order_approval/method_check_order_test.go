package order_approval

import (
	"context"
	"testing"

	"example.com/orders/internal/bpmgen/model"
	"example.com/orders/internal/bpmgen/services"
)

func TestCheckOrder(t *testing.T) {
	ctx, svc := context.Background(), &services.Services{}

	data := &model.DataModel{Amount: 500}
	if err := CheckOrder(ctx, svc, data); err != nil {
		t.Fatal(err)
	}
	if !data.Checked {
		t.Error("заказ не отмечен проверенным")
	}
	if err := CheckOrder(ctx, svc, &model.DataModel{}); err == nil {
		t.Error("пустой заказ прошёл проверку")
	}
}

// Условие шлюза сгенерировано в Go-функцию isBig: её можно проверить из пакета процесса.
func TestIsBig(t *testing.T) {
	tests := []struct {
		amount float64
		want   string
	}{
		{amount: 1000, want: "Flow_small"},
		{amount: 1000.01, want: "Flow_big"},
		{amount: 5000, want: "Flow_big"},
	}
	for _, tt := range tests {
		if got := isBig(&model.DataModel{Amount: tt.amount}); got != tt.want {
			t.Errorf("isBig(%v) = %s, want %s", tt.amount, got, tt.want)
		}
	}
}

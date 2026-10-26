package test_decision

import (
	"context"
	"slices"
	"testing"

	"github.com/nicolas-vp/go-bpm/examples/demo/bpmgen/model"
	"github.com/nicolas-vp/go-bpm/examples/demo/bpmgen/services"
)

// Таблица Decision_1 (hit policy UNIQUE) сгенерирована в decideDecision1: каждое правило - ветка if.
func TestDecideDecision1(t *testing.T) {
	tests := []struct {
		a, b, want, rule string
	}{
		{a: "foo", b: "bar", want: "correct foobar", rule: "DecisionRule_04e4bk8"},
		{a: "some", b: "x", want: "some a", rule: "DecisionRule_1a2n073"},
		{a: "z", b: "some", want: "some b", rule: "DecisionRule_0t56a5i"},
		{a: "z", b: "z", want: "default", rule: "DecisionRule_1kb19kv"},
	}
	for _, tt := range tests {
		got, rules := decideDecision1(&model.DataModel{A: tt.a, B: tt.b})
		if got != tt.want || !slices.Equal(rules, []string{tt.rule}) {
			t.Errorf("a=%s, b=%s: %q %v, want %q [%s]", tt.a, tt.b, got, rules, tt.want, tt.rule)
		}
	}
}

// Задача бизнес-правила кладёт результат решения в модель. Вне движка запись в историю правил пропускается.
func TestBusinessRuleTask(t *testing.T) {
	data := &model.DataModel{A: "foo", B: "bar"}
	if err := Activity0xzljrt(context.Background(), &services.Services{}, data); err != nil {
		t.Fatal(err)
	}
	if data.C != "correct foobar" {
		t.Errorf("c = %q", data.C)
	}
}

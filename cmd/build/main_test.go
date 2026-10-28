package main

import "testing"

func TestShorten(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Коротко.", "Коротко."},
		{"Первое предложение достаточно длинное. Второе не влезет целиком", "Первое предложение достаточно длинное."},
		{"Одно длинное предложение без точки, которое режется по слову", "Одно длинное предложение без точки, которое…"},
	} {
		if got := shorten(tc.in, 45); got != tc.want {
			t.Errorf("shorten(%q) = %q, хотим %q", tc.in, got, tc.want)
		}
	}
}

package main

import "testing"

func TestPort(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{name: "PORT empty falls back to 8080", env: "", want: "8080"},
		{name: "PORT set returns its value", env: "10000", want: "10000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PORT", tt.env)

			if got := port(); got != tt.want {
				t.Errorf("port() = %q, want %q", got, tt.want)
			}
		})
	}
}

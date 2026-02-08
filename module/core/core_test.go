package core

import "testing"

func TestName(t *testing.T) {
	want := "mannaiah core"
	got := Name()
	if got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
}

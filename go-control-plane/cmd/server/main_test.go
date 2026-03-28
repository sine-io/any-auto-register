package main

import "testing"

func TestNewServerCommandDefinesConfigFlag(t *testing.T) {
	cmd := newServerCommand()

	if cmd.Use != "server" {
		t.Fatalf("expected Use=server, got %s", cmd.Use)
	}

	if cmd.Flags().Lookup("config") == nil {
		t.Fatal("expected server command to define --config flag")
	}
}

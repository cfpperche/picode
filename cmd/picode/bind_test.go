package main

import "testing"

func TestKeepsLoopback(t *testing.T) {
	for host, want := range map[string]bool{
		"0.0.0.0": false, "::": false, "127.0.0.1": false, "::1": false, "": false,
		"100.87.149.83": true, "192.168.1.4": true,
	} {
		if got := keepsLoopback(host); got != want {
			t.Errorf("keepsLoopback(%q) = %v, want %v", host, got, want)
		}
	}
}

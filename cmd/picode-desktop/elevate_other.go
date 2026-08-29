//go:build !windows

package main

// Off Windows there is no UAC to raise. `install` already refuses to run here.
func isAdmin() bool { return false }

func elevate() (bool, error) { return false, nil }

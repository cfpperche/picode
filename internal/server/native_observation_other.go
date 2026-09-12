//go:build !linux

package server

import "os"

func openNativeObservationFence(string) (*os.File, error) { return nil, os.ErrNotExist }

//go:build !linux

package llamaservice

import "errors"

func publishCreation(from, to string) error {
	return errors.New("Local services require Linux or WSL.")
}

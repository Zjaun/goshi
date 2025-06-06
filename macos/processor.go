package macos

import (
	"errors"
)

func Processor() (internal.CentralProcessor, error) {
	return internal.CentralProcessor{}, errors.New("not implemented")
}

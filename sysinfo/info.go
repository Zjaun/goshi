/*
 * Copyright 2016-2024 The OSHI Project Contributors
 * SPDX-License-Identifier: MIT
 */

package sysinfo

import (
	"errors"
	"goshi/linux"
	"goshi/macos"
	"goshi/sysinfo/hardware"
	hardware2 "goshi/windows/hardware"
	"runtime"
)

func Processor() (hardware.CentralProcessor, error) {
	var proc hardware.CentralProcessor
	var err error
	switch runtime.GOOS {
	case "windows":
		proc, err = hardware2.Processor()
	case "linux":
		proc, err = linux.Processor()
	case "darwin":
		proc, err = macos.Processor()
	default:
		err = errors.New("unsupported os")
	}
	return proc, err
}

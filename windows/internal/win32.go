/*
 * Copyright 2016-2024 The OSHI Project Contributors
 * SPDX-License-Identifier: MIT
 */

package internal

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"unsafe"
)

const (
	// https://learn.microsoft.com/en-us/windows/win32/api/sysinfoapi/ns-sysinfoapi-system_info#members
	ia64  uint16 = 6
	amd64 uint16 = 9
	arm64 uint16 = 12
)

var (
	kernel32           = windows.NewLazySystemDLL("kernel32.dll")
	psapi              = windows.NewLazySystemDLL("Psapi.dll")
	nativeSystemInfo   = kernel32.NewProc("GetNativeSystemInfo")
	perfInfo           = psapi.NewProc("GetPerformanceInfo")
	Windows7OrGreater  bool
	VistaOrGreater     bool
	Windows10OrGreater bool
)

type PerformanceInformation struct {
	Cb                uint32
	CommitTotal       uintptr
	CommitLimit       uintptr
	CommitPeak        uintptr
	PhysicalTotal     uintptr
	PhysicalAvailable uintptr
	SystemCache       uintptr
	KernelTotal       uintptr
	KernelPaged       uintptr
	KernelNonPaged    uintptr
	PageSize          uintptr
	HandleCount       uint32
	ProcessCount      uint32
	ThreadCount       uint32
}

func isWindowsVersionOrGreater(currentMajor, currentMinor, major, minor uint16) bool {
	return currentMajor > major || (currentMajor == major && currentMinor >= minor)
}

func Is64bit() bool {
	var sysInfo struct {
		arch uint16
	}
	_, _, err := nativeSystemInfo.Call(uintptr(unsafe.Pointer(&sysInfo)))
	if !errors.Is(err, windows.ERROR_SUCCESS) {
		return false
	}
	return sysInfo.arch == amd64 || sysInfo.arch == arm64 || sysInfo.arch == ia64
}

func GetPerformanceInfo() (PerformanceInformation, error) {
	pi := PerformanceInformation{}
	cb := unsafe.Sizeof(pi)
	pi.Cb = uint32(cb)
	_, _, err := perfInfo.Call(
		uintptr(unsafe.Pointer(&pi)),
		cb,
	)
	if !errors.Is(err, windows.ERROR_SUCCESS) {
		err = fmt.Errorf("perfInfo: failed to get performance info: %w ", err)
		return PerformanceInformation{}, err
	}
	return pi, nil
}

func init() {
	// https://learn.microsoft.com/en-us/cpp/porting/modifying-winver-and-win32-winnt?view=msvc-170#remarks
	ver, err := windows.GetVersion()
	if err != nil {
		err = fmt.Errorf("windows: error getting operating system version: %w", err)
		panic(err)
	}
	currentMajor := uint16(ver & 0xFF)
	currentMinor := uint16((ver >> 8) & 0xFF)
	Windows10OrGreater = isWindowsVersionOrGreater(currentMajor, currentMinor, 10, 0)
	Windows7OrGreater = isWindowsVersionOrGreater(currentMajor, currentMinor, 6, 1)
	VistaOrGreater = isWindowsVersionOrGreater(currentMajor, currentMinor, 6, 0)
}

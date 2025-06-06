/*
 * Copyright 2016-2024 The OSHI Project Contributors
 * SPDX-License-Identifier: MIT
 */

package internal

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"math/bits"
	"unsafe"
)

var (
	lpi = kernel32.NewProc("GetLogicalProcessorInformation")

	// used for retriving the returned length value of GetLogicalProcessorInformation syscall
	// not sure why the program panics when it is turned into a local variable
	lpiRl uint32
)

type SystemLogicalProcessorInformation interface {
	GetProcessorMask() uintptr
}

type ProcessorCore struct {
	ProcessorMask uintptr
	Relationship  uint32
	Flags         uint8
}

func (p ProcessorCore) GetProcessorMask() uintptr {
	return p.ProcessorMask
}

func getBitMatchingPackageNumber(maskList []int64, logProc int32) int {
	for i, mask := range maskList {
		if mask&(1<<logProc) != 0 {
			return i
		}
	}
	return 0
}

func getSystemLogicalProcessorInformation() ([]SystemLogicalProcessorInformation, error) {
	_, _, err := lpi.Call(uintptr(0), uintptr(unsafe.Pointer(&lpiRl)))
	if !errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) {
		err = fmt.Errorf("lpi: failed to get buffer length: %w", err)
		return nil, err
	}
	buf := make([]byte, lpiRl)
	res, _, err := lpi.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&lpiRl)))
	if res == 0 {
		return nil, err
	}
	type lpi struct {
		pm uintptr
		r  uint32
		pi [16]byte
	}
	arr := make([]SystemLogicalProcessorInformation, 0)
	s := int(unsafe.Sizeof(lpi{}))
	for off := 0; off+s <= int(lpiRl); off += s {
		ptr := unsafe.Pointer(&buf[off])
		str := (*lpi)(ptr)
		switch str.r {
		case RelationProcessorPackage:
			fallthrough
		case RelationProcessorCore:
			arr = append(arr, ProcessorCore{
				ProcessorMask: str.pm,
				Relationship:  str.r,
				Flags:         str.pi[0],
			})
		default:
		}
	}
	return arr, nil
}

func GetLogicalProcessorInformation() (LogicalProcessorInformation, error) {
	packageMaskList := make([]int64, 0)
	coreMaskList := make([]int64, 0)
	procInfo, err := getSystemLogicalProcessorInformation()
	if err != nil {
		return LogicalProcessorInformation{}, err
	}
	for _, info := range procInfo {
		mask := int64(info.GetProcessorMask())
		switch info.GetProcessorMask() {
		case RelationProcessorPackage:
			packageMaskList = append(packageMaskList, mask)
		case RelationProcessorCore:
			coreMaskList = append(coreMaskList, mask)
		default:
		}
	}
	logProcs := make([]LogicalProcessor, 0)
	for core := 0; core < len(coreMaskList); core++ {
		mask := uint(coreMaskList[core])
		lowBit := int32(bits.TrailingZeros(mask))
		hiBit := 63 - int32(bits.LeadingZeros(mask))
		for i := lowBit; i <= hiBit; i++ {
			if mask&(1<<i) == 0 {
				continue
			}
			logProc := LogicalProcessor{
				ProcessorNumber:         int(i),
				PhysicalProcessorNumber: core,
				PhysicalPackageNumber:   getBitMatchingPackageNumber(packageMaskList, i),
			}
			logProcs = append(logProcs, logProc)
		}
	}
	res := LogicalProcessorInformation{
		logProcs,
		nil,
	}
	return res, nil
}

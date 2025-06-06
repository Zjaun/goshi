/*
 * Copyright 2016-2024 The OSHI Project Contributors
 * SPDX-License-Identifier: MIT
 */

package internal

import (
	_ "embed"
	"fmt"
	"github.com/yusufpapurcu/wmi"
)

const (
	Processor                   = "Win32_Processor"
	VideoController             = "Win32_VideoController"
	PhysicalMemory              = "Win32_PhysicalMemory"
	PerfRawDataPerfOSPagingFile = "Win32_PerfRawData_PerfOS_PagingFile"
	PerfRawDataPerfOSMemory     = "Win32_PerfRawData_PerfOS_Memory"
)

func queryClass[T any](class string) ([]T, error) {
	var res []T
	q := wmi.CreateQuery(&res, "", class)
	err := wmi.Query(q, &res)
	if err != nil {
		return nil, wrapErrors(class, err)
	}
	return res, nil
}

func wrapErrors(class string, err error) error {
	if err != nil {
		return fmt.Errorf("wmi: failed to execute query for class %q: %w", class, err)
	}
	return nil
}

type Win32PerfRawDataPerfOSPagingFile struct {
	PercentUsage uint32
}

type Win32PerfRawDataPerfOSMemory struct {
	PagesInputPerSec  uint32
	PagesOutputPerSec uint32
}

type Win32Processor struct {
	ProcessorId string
}

type Win32VideoController struct {
	Name, PNPDeviceID, AdapterCompatibility, DriverVersion string
	AdapterRAM                                             uint32
}

type Win32PhysicalMemory struct {
	BankLabel, Manufacturer, PartNumber, SerialNumber string
	Capacity                                          uint64
	SMBiosMemoryType, Speed                           uint32
	MemoryType                                        uint16
}

func QueryWmiGraphicsCards() ([]Win32VideoController, error) {
	return queryClass[Win32VideoController](VideoController)
}

func WmiQueryPhysicalMemory() ([]Win32PhysicalMemory, error) {
	var mems []Win32PhysicalMemory
	if Windows10OrGreater {
		type c struct {
			BankLabel, Manufacturer, PartNumber, SerialNumber string
			Capacity                                          uint64
			SMBiosMemoryType, Speed                           uint32
		}
		res, err := queryClass[c](PhysicalMemory)
		if err != nil {
			return nil, err
		}
		for _, mem := range res {
			mems = append(mems, Win32PhysicalMemory{
				BankLabel:        mem.BankLabel,
				Manufacturer:     mem.Manufacturer,
				PartNumber:       mem.PartNumber,
				SerialNumber:     mem.SerialNumber,
				Capacity:         mem.Capacity,
				SMBiosMemoryType: mem.SMBiosMemoryType,
				Speed:            mem.Speed,
			})
		}
	} else {
		type c struct {
			BankLabel, Manufacturer, PartNumber, SerialNumber string
			Capacity                                          uint64
			Speed                                             uint32
			MemoryType                                        uint16
		}
		res, err := queryClass[c](PhysicalMemory)
		if err != nil {
			return nil, err
		}
		for _, mem := range res {
			mems = append(mems, Win32PhysicalMemory{
				BankLabel:    mem.BankLabel,
				Manufacturer: mem.Manufacturer,
				PartNumber:   mem.PartNumber,
				SerialNumber: mem.SerialNumber,
				Capacity:     mem.Capacity,
				MemoryType:   mem.MemoryType,
				Speed:        mem.Speed,
			})
		}
	}
	return mems, nil
}

func WmiQueryPerfRawDataPagingFile() ([]Win32PerfRawDataPerfOSPagingFile, error) {
	return queryClass[Win32PerfRawDataPerfOSPagingFile](PerfRawDataPerfOSPagingFile)
}

func WmiQueryPerfRawDataMemory() ([]Win32PerfRawDataPerfOSMemory, error) {
	return queryClass[Win32PerfRawDataPerfOSMemory](PerfRawDataPerfOSMemory)
}

func WmiQueryProcessor() ([]Win32Processor, error) {
	return queryClass[Win32Processor](Processor)
}

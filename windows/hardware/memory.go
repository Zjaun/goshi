/*
 * Copyright 2016-2024 The OSHI Project Contributors
 * SPDX-License-Identifier: MIT
 */

package hardware

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"goshi/sysinfo/hardware"
	"goshi/util"
	"goshi/windows/internal"
	"strconv"
)

var (
	//go:embed memoryTypes.json
	memoryTypes []byte

	memory map[uint16]string
	smBios map[uint32]string
)

type WindowsVirtualMemory struct {
	global WindowsGlobalMemory
}

func (w WindowsVirtualMemory) SwapUsed() int64 {
	return w.global.PageSize() * querySwapUsed()
}

func (w WindowsVirtualMemory) SwapTotal() int64 {
	a, _, _ := querySwapTotalVirtMaxVirtUsed()
	return w.global.PageSize() * a
}

func (w WindowsVirtualMemory) VirtualMax() int64 {
	_, b, _ := querySwapTotalVirtMaxVirtUsed()
	return w.global.PageSize() * b
}

func (w WindowsVirtualMemory) VirtualInUse() int64 {
	_, _, c := querySwapTotalVirtMaxVirtUsed()
	return w.global.PageSize() * c

}

func (w WindowsVirtualMemory) SwapPagesIn() int64 {
	a, _ := queryPageSwaps()
	return a
}

func (w WindowsVirtualMemory) SwapPagesOut() int64 {
	_, b := queryPageSwaps()
	return b
}

type WindowsGlobalMemory struct {
}

func (w WindowsGlobalMemory) Available() int64 {
	a, _, _ := readPerfInfo()
	return a
}

func (w WindowsGlobalMemory) Total() int64 {
	_, b, _ := readPerfInfo()
	return b
}

func (w WindowsGlobalMemory) PageSize() int64 {
	_, _, c := readPerfInfo()
	return c
}

func (w WindowsGlobalMemory) VirtualMemory() hardware.VirtualMemory {
	return WindowsVirtualMemory{
		global: w,
	}
}

func (w WindowsGlobalMemory) PhysicalMemory() []hardware.PhysicalMemory {
	q, err := internal.WmiQueryPhysicalMemory()
	if err != nil {
		return nil
	}
	memories := make([]hardware.PhysicalMemory, 0)
	for _, mems := range q {
		var memoryType string
		var exists bool
		if internal.Windows10OrGreater {
			memoryType, exists = smBios[mems.SMBiosMemoryType]
		} else {
			memoryType, exists = memory[mems.MemoryType]
		}
		if !exists {
			memoryType = util.Unknown
		}
		pmem := hardware.NewPhysicalMemory(
			mems.BankLabel,
			mems.Manufacturer,
			memoryType,
			mems.PartNumber,
			mems.SerialNumber,
			int64(mems.Capacity),
			int64(mems.Speed*1000000),
		)
		memories = append(memories, pmem)
	}
	return memories
}

func parseMemoryTypes[T uint16 | uint32](d map[string]string, bits int) (map[T]string, error) {
	res := make(map[T]string)
	for k, v := range d {
		i, err := strconv.ParseUint(k, 10, bits)
		if err != nil {
			return nil, err
		}
		res[T(i)] = v
	}
	return res, nil
}

func readPerfInfo() (int64, int64, int64) {
	pi, err := internal.GetPerformanceInfo()
	if err != nil {
		return 0, 0, 4098
	}
	pageSize := int64(pi.PageSize)
	memAvailable := pageSize * int64(pi.PhysicalAvailable)
	memTotal := pageSize * int64(pi.PhysicalTotal)
	return memAvailable, memTotal, pageSize
}

func querySwapTotalVirtMaxVirtUsed() (int64, int64, int64) {
	pi, err := internal.GetPerformanceInfo()
	if err != nil {
		return 0, 0, 0
	}
	a := int64(pi.CommitLimit - pi.PhysicalTotal)
	b := int64(pi.CommitLimit)
	c := int64(pi.CommitTotal)
	return a, b, c
}

func querySwapUsed() int64 {
	pi, err := internal.WmiQueryPerfRawDataPagingFile()
	if err != nil {
		return 0
	}
	return int64(pi[0].PercentUsage)
}

func queryPageSwaps() (int64, int64) {
	pi, err := internal.WmiQueryPerfRawDataMemory()
	if err != nil {
		return 0, 0
	}
	return int64(pi[0].PagesInputPerSec), int64(pi[0].PagesOutputPerSec)
}

func GlobalMemory() hardware.GlobalMemory {
	return WindowsGlobalMemory{}
}

func init() {
	var data map[string]map[string]string
	err := json.Unmarshal(memoryTypes, &data)
	if err != nil {
		err = fmt.Errorf("memory: error unmarshalling memory types: %w", err)
		panic(err)
	}
	memory, err = parseMemoryTypes[uint16](data["memory"], 16)
	if err != nil {
		err = fmt.Errorf("memory: error parsing memory hardware: %w", err)
		panic(err)
	}
	smBios, err = parseMemoryTypes[uint32](data["smBios"], 32)
	if err != nil {
		err = fmt.Errorf("memory: error parsing smBios hardware: %w", err)
		panic(err)
	}
}

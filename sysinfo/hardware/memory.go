/*
 * Copyright 2016-2024 The OSHI Project Contributors
 * SPDX-License-Identifier: MIT
 */

package hardware

type PhysicalMemory struct {
	bankLabel, manufacturer, memoryType, partNumber, serialNumber string
	capacity, clockSpeed                                          int64
}

func (p PhysicalMemory) BankLabel() string {
	return p.bankLabel
}

func (p PhysicalMemory) Capacity() int64 {
	return p.capacity
}

func (p PhysicalMemory) ClockSpeed() int64 {
	return p.clockSpeed
}

func (p PhysicalMemory) Manufacturer() string {
	return p.manufacturer
}

func (p PhysicalMemory) MemoryType() string {
	return p.memoryType
}

func (p PhysicalMemory) PartNumber() string {
	return p.partNumber
}

func (p PhysicalMemory) SerialNumber() string {
	return p.serialNumber
}

func NewPhysicalMemory(
	bankLabel, manufacturer, memoryType, partNumber, serialNumber string,
	capacity, clockSpeed int64,
) PhysicalMemory {
	return PhysicalMemory{
		bankLabel:    bankLabel,
		capacity:     capacity,
		clockSpeed:   clockSpeed,
		manufacturer: manufacturer,
		memoryType:   memoryType,
		partNumber:   partNumber,
		serialNumber: serialNumber,
	}
}

type VirtualMemory interface {
	SwapTotal() int64
	SwapUsed() int64
	VirtualMax() int64
	VirtualInUse() int64
	SwapPagesIn() int64
	SwapPagesOut() int64
}

type GlobalMemory interface {
	Total() int64
	Available() int64
	PageSize() int64
	VirtualMemory() VirtualMemory
	PhysicalMemory() []PhysicalMemory
}

/*
 * Copyright 2016-2024 The OSHI Project Contributors
 * SPDX-License-Identifier: MIT
 */

package hardware

import (
	"fmt"
	set "github.com/deckarep/golang-set/v2"
	"golang.org/x/sys/windows/registry"
	"goshi/sysinfo/hardware"
	"goshi/windows/internal"
	"strings"
)

const (
	cpuRegistryPath = `HARDWARE\DESCRIPTION\System\CentralProcessor\`
)

type WindowsCentralProcessor struct {
	processorIdentifier    hardware.ProcessorIdentifier
	physicalPackageCount   int
	physicalProcessorCount int
	logicalProcessorCount  int
}

func (w WindowsCentralProcessor) ProcessorIdentifier() hardware.ProcessorIdentifier {
	return w.processorIdentifier
}

func (w WindowsCentralProcessor) PhysicalPackageCount() int {
	return w.physicalPackageCount
}

func (w WindowsCentralProcessor) PhysicalProcessorCount() int {
	return w.physicalProcessorCount
}

func (w WindowsCentralProcessor) LogicalProcessorCount() int {
	return w.logicalProcessorCount
}

func processorCounts() (internal.LogicalProcessorInformation, error) {
	if internal.Windows7OrGreater {
		return internal.GetLogicalProcessorInformationEx()
	}
	return internal.GetLogicalProcessorInformation()
}

func parseIdentifier(id, key string) string {
	found := false
	for _, match := range strings.Fields(id) {
		if found {
			return match
		}
		found = match == key
	}
	return ""
}

func processorIdentifier() (hardware.ProcessorIdentifier, error) {
	var err error
	acc := uint32(registry.QUERY_VALUE | registry.ENUMERATE_SUB_KEYS)
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, cpuRegistryPath, acc)
	if err != nil {
		err = fmt.Errorf("registry: cannot open cpu registry key: %w", err)
		return hardware.ProcessorIdentifier{}, err
	}
	defer func() {
		if derr := key.Close(); derr != nil && err == nil {
			err = fmt.Errorf("registry: cannot close cpu registry key: %w", derr)
			err = derr
		}
	}()
	subKeys, err := key.ReadSubKeyNames(-1)
	if err != nil {
		err = fmt.Errorf("registry: cannot read cpu subkeys: %w", err)
		return hardware.ProcessorIdentifier{}, err
	}
	var vendor, name, identifier, family, model, stepping, processorID string
	var freq int64
	if len(subKeys) > 0 {
		subKeyPath := fmt.Sprintf(`%s\%s`, cpuRegistryPath, subKeys[0])
		subKey, err := registry.OpenKey(registry.LOCAL_MACHINE, subKeyPath, registry.QUERY_VALUE)
		if err != nil {
			err = fmt.Errorf("registry: cannot open cpu registry key: %w", err)
			return hardware.ProcessorIdentifier{}, err
		}
		defer func() {
			if derr := subKey.Close(); derr != nil && err == nil {
				err = fmt.Errorf("registry: cannot close cpu registry key: %w", derr)
				err = derr
			}
		}()
		vendor, _, err = subKey.GetStringValue("VendorIdentifier")
		if err != nil {
			err = fmt.Errorf("registry: failed to get value of VendorIdentifier: %w", err)
			return hardware.ProcessorIdentifier{}, err
		}
		name, _, err = subKey.GetStringValue("ProcessorNameString")
		if err != nil {
			err = fmt.Errorf("registry: failed to get value of ProcessorNameString: %w", err)
			return hardware.ProcessorIdentifier{}, err
		}
		identifier, _, err = subKey.GetStringValue("Identifier")
		if err != nil {
			err = fmt.Errorf("registry: failed to get value of Identifier: %w", err)
			return hardware.ProcessorIdentifier{}, err
		}
		f, _, err := subKey.GetIntegerValue("~MHz")
		if err == nil {
			freq = int64(f * 1_000_000)
		}
	}
	if len(identifier) != 0 {
		family = parseIdentifier(identifier, "Family")
		model = parseIdentifier(identifier, "Model")
		stepping = parseIdentifier(identifier, "Stepping")
	}
	procId := hardware.NewProcessorIdentifier(
		vendor, name, family, model, stepping, processorID, internal.Is64bit(), freq,
	)
	return procId, nil
}

func Processor() (hardware.CentralProcessor, error) {
	procId, err := processorIdentifier()
	if err != nil {
		return nil, err
	}
	counts, err := processorCounts()
	if err != nil {
		return nil, err
	}
	var phyProcs []internal.PhysicalProcessor
	if counts.PhysicalProcessors == nil {
		keys := set.NewSet[int]()
		for _, core := range counts.LogicalProcessors {
			key := core.PhysicalPackageNumber<<16 + core.PhysicalProcessorNumber
			keys.Add(key)
		}
		pkgCoreKeys := keys.ToSlice()
		for _, key := range pkgCoreKeys {
			phyProcs = append(phyProcs, internal.PhysicalProcessor{
				PhysicalPackageNumber:   key >> 16,
				PhysicalProcessorNumber: key & 0xFFFF,
			})
		}
	} else {
		phyProcs = counts.PhysicalProcessors
	}
	physPkgs := set.NewSet[int]()
	for _, logProc := range counts.LogicalProcessors {
		physPkgs.Add(logProc.PhysicalPackageNumber)
	}
	proc := WindowsCentralProcessor{
		processorIdentifier:    procId,
		physicalProcessorCount: len(counts.PhysicalProcessors),
		physicalPackageCount:   physPkgs.Cardinality(),
		logicalProcessorCount:  len(counts.LogicalProcessors),
	}
	return proc, nil
}

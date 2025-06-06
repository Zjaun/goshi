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

const (
	RelationProcessorCore = iota
	RelationNumaNode
	RelationProcessorPackage
	RelationAll = 0xFFFF
)

var (
	lpiEx = kernel32.NewProc("GetLogicalProcessorInformationEx")

	// used for retriving the returned length value of GetLogicalProcessorInformationEx syscall
	// not sure why the program panics when it is turned into a local variable
	lpiExRl uint32
)

type SystemLogicalProcessorInformationEx interface {
	GetRelationship() uint32
}

type LogicalProcessorInformation struct {
	LogicalProcessors  []LogicalProcessor
	PhysicalProcessors []PhysicalProcessor
}

type LogicalProcessor struct {
	ProcessorNumber         int
	PhysicalProcessorNumber int
	PhysicalPackageNumber   int
	NumaNode                uint32
	ProcessorGroup          uint16
}

type PhysicalProcessor struct {
	PhysicalPackageNumber   int
	PhysicalProcessorNumber int
	Efficiency              uint8
	IdString                string
}

type ProcessorRelationship struct {
	Relationship           uint32
	Flags, EfficiencyClass uint8
	_                      [20]uint8
	GroupCount             uint16
	GroupMasks             []GroupAffinity
}

func (p ProcessorRelationship) GetRelationship() uint32 {
	return p.Relationship
}

type NumaNodeRelationship struct {
	Relationship uint32
	NodeNumber   uint32
	_            [18]uint8
	GroupCount   uint16
	GroupMasks   []GroupAffinity
}

func (n NumaNodeRelationship) GetRelationship() uint32 {
	return n.Relationship
}

type GroupAffinity struct {
	Mask  uintptr
	Group uint16
	_     [3]uint16
}

func getPhysicalProcessors(cores []GroupAffinity, coreEfficiencyMap map[GroupAffinity]uint8, corePkgMap map[int]int, pkgCpuidMap map[int]string) []PhysicalProcessor {
	procs := make([]PhysicalProcessor, 0)
	for i, core := range cores {
		efficiency := coreEfficiencyMap[core]
		cpuid := pkgCpuidMap[i]
		pkgId := corePkgMap[i]
		procs = append(procs, PhysicalProcessor{
			PhysicalPackageNumber:   pkgId,
			PhysicalProcessorNumber: i,
			Efficiency:              efficiency,
			IdString:                cpuid,
		})
	}
	return procs
}

func matchingCore(cores []GroupAffinity, g uint16, lp int32) int {
	for i, core := range cores {
		if core.Mask&(1<<lp) != 0 && core.Group == g {
			return i
		}
	}
	return 0
}

func matchingPackage(packages [][]GroupAffinity, g uint16, lp int32) int {
	for i, row := range packages {
		for _, col := range row {
			if (col.Mask&(1<<lp)) != 0 && col.Group == g {
				return i
			}
		}
	}
	return 0
}

func parseProcessorRelationship(uPtr unsafe.Pointer) ProcessorRelationship {
	type header struct {
		f, ec uint8
		_     [20]uint8
		gc    uint16
	}
	h := (*header)(uPtr)
	arrPtr := unsafe.Pointer(uintptr(uPtr) + unsafe.Sizeof(header{}))
	arr := unsafe.Slice((*GroupAffinity)(arrPtr), h.gc)
	return ProcessorRelationship{
		Flags:           h.f,
		EfficiencyClass: h.ec,
		GroupCount:      h.gc,
		GroupMasks:      arr,
	}
}

func parseNumaRelationship(uPtr unsafe.Pointer) NumaNodeRelationship {
	type header struct {
		nn uint32
		_  [18]uint8
		gc uint16
	}
	h := (*header)(uPtr)
	arrPtr := unsafe.Pointer(uintptr(uPtr) + unsafe.Sizeof(header{}))
	arr := unsafe.Slice((*GroupAffinity)(arrPtr), h.gc)
	return NumaNodeRelationship{
		NodeNumber: h.nn,
		GroupCount: h.gc,
		GroupMasks: arr,
	}
}

func GetSystemLogicalProcessorInformationEx() ([]SystemLogicalProcessorInformationEx, error) {
	_, _, err := lpiEx.Call(uintptr(RelationAll), 0, uintptr(unsafe.Pointer(&lpiExRl)))
	if !errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) {
		err = fmt.Errorf("lpiex: failed to get buffer length: %w", err)
		return nil, err
	}
	buf := make([]byte, lpiExRl)
	res, _, err := lpiEx.Call(uintptr(RelationAll), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&lpiExRl)))
	if res == 0 {
		err = fmt.Errorf("lpiex: %w", err)
		return nil, err
	}
	relationships := make([]SystemLogicalProcessorInformationEx, 0)
	type procRel struct {
		rel  uint32
		size uint32
	}
	for off := 0; off < int(lpiExRl); {
		ptr := unsafe.Pointer(&buf[off])
		info := (*procRel)(ptr)
		headerPtr := unsafe.Pointer(uintptr(ptr) + unsafe.Sizeof(procRel{}))
		switch info.rel {
		case RelationProcessorCore:
			fallthrough
		case RelationProcessorPackage:
			rel := parseProcessorRelationship(headerPtr)
			rel.Relationship = info.rel
			relationships = append(relationships, rel)
		case RelationNumaNode:
			rel := parseNumaRelationship(headerPtr)
			rel.Relationship = info.rel
			relationships = append(relationships, rel)
		default:

		}
		off += int(info.size)
	}
	return relationships, nil
}

func GetLogicalProcessorInformationEx() (LogicalProcessorInformation, error) {
	procInfo, err := GetSystemLogicalProcessorInformationEx()
	if err != nil {
		return LogicalProcessorInformation{}, err
	}
	packages := make([][]GroupAffinity, 0)
	cores := make([]GroupAffinity, 0)
	numaNodes := make([]NumaNodeRelationship, 0)
	coreEfficiencyMap := make(map[GroupAffinity]uint8)
	for _, info := range procInfo {
		switch info.GetRelationship() {
		case RelationProcessorCore:
			core := info.(ProcessorRelationship)
			cores = append(cores, core.GroupMasks...)
			if Windows10OrGreater {
				coreEfficiencyMap[core.GroupMasks[0]] = core.EfficiencyClass
			}
		case RelationNumaNode:
			numa := info.(NumaNodeRelationship)
			numaNodes = append(numaNodes, numa)
		case RelationProcessorPackage:
			packages = append(packages, info.(ProcessorRelationship).GroupMasks)
		default:
		}
	}

	processorIdMap := make(map[int]string)
	processorId, err := WmiQueryProcessor()
	if err != nil {
		return LogicalProcessorInformation{}, err
	}

	for i, value := range processorId {
		processorIdMap[i] = value.ProcessorId
	}

	logProcs := make([]LogicalProcessor, 0)
	corePkgMap := make(map[int]int)
	pkgCpuidMap := make(map[int]string)

	for _, numa := range numaNodes {
		group := numa.GroupMasks[0].Group
		mask := uint(numa.GroupMasks[0].Mask)
		lowBit := int32(bits.TrailingZeros(mask))
		hiBit := 63 - int32(bits.LeadingZeros(mask))
		for lp := lowBit; lp <= hiBit; lp++ {
			if (mask & 1 << lp) == 0 {
				continue
			}
			coreId := matchingCore(cores, group, lp)
			pkgId := matchingPackage(packages, group, lp)
			corePkgMap[coreId] = pkgId
			pkgCpuidMap[coreId] = processorIdMap[pkgId]
			logProcs = append(logProcs, LogicalProcessor{
				ProcessorNumber:         int(lp),
				PhysicalProcessorNumber: coreId,
				PhysicalPackageNumber:   pkgId,
				NumaNode:                numa.NodeNumber,
				ProcessorGroup:          group,
			})
		}
	}

	physicalProcs := getPhysicalProcessors(cores, coreEfficiencyMap, corePkgMap, pkgCpuidMap)
	res := LogicalProcessorInformation{
		logProcs,
		physicalProcs,
	}
	return res, nil
}

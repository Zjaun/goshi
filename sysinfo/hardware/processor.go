/*
 * Copyright 2016-2024 The OSHI Project Contributors
 * SPDX-License-Identifier: MIT
 */

package hardware

import (
	_ "embed"
	"fmt"
	"goshi/util"
	"regexp"
	"strings"
)

var (
	//go:embed oshi.architecture.properties
	propsAsBytes []byte
	archPops     = make(map[string]string)
	freqRegex    = regexp.MustCompile("@ (.*)$")
)

func populateProperties() {
	buf := make([]byte, 0)
	for _, b := range propsAsBytes {
		if b != '\n' {
			buf = append(buf, b)
			continue
		}
		line := string(buf)
		if line == "" || strings.HasPrefix(line, "#") {
			buf = buf[:0]
			continue
		}
		pair := strings.Split(line, "=")
		if len(pair) != 2 {
			continue
		}
		archPops[pair[0]] = pair[1]
		buf = buf[:0]
	}
}

func queryVendorFromImplementer(vendor string) string {
	key := fmt.Sprintf("hw_impl.%s", vendor)
	val, present := archPops[key]
	if !present {
		return vendor
	}
	return val
}

type ProcessorIdentifier struct {
	vendor, name, family, model, stepping, processorID, identifier, microarchitecture string
	is64bit                                                                           bool
	frequency                                                                         int64
}

func (procId *ProcessorIdentifier) queryMicroarchitecture() string {
	var a string
	sb := strings.Builder{}
	uc := strings.ToUpper(procId.vendor)
	if strings.Contains(uc, "AMD") {
		sb.WriteString("amd.")
	} else if strings.Contains(uc, "ARM") {
		sb.WriteString("arm.")
	} else if strings.Contains(uc, "IBM") {
		powerIdx := strings.Index(procId.name, "_POWER")
		if powerIdx > 0 {
			a = procId.name[powerIdx+1:]
		}
	} else if strings.Contains(uc, "APPLE") {
		sb.WriteString("apple.")
	}
	if strings.TrimSpace(a) == "" && sb.String() != "arm." {
		sb.WriteString(procId.family)
		a = archPops[sb.String()]
	}
	if strings.TrimSpace(a) == "" {
		sb.WriteString(".")
		sb.WriteString(procId.model)
		a = archPops[sb.String()]
	}
	if strings.TrimSpace(a) == "" {
		sb.WriteString(".")
		sb.WriteString(procId.stepping)
		a = archPops[sb.String()]
	}
	if strings.TrimSpace(a) == "" {
		return util.Unknown
	}
	return a
}

func (procId *ProcessorIdentifier) Vendor() string {
	return procId.vendor
}

func (procId *ProcessorIdentifier) Name() string {
	return procId.name
}

func (procId *ProcessorIdentifier) Family() string {
	return procId.family
}

func (procId *ProcessorIdentifier) Model() string {
	return procId.model
}

func (procId *ProcessorIdentifier) Stepping() string {
	return procId.stepping
}

func (procId *ProcessorIdentifier) ProcessorID() string {
	return procId.processorID
}

func (procId *ProcessorIdentifier) Identifier() string {
	return procId.identifier
}

func (procId *ProcessorIdentifier) Microarchitecture() string {
	if procId.microarchitecture == "" {
		procId.microarchitecture = procId.queryMicroarchitecture()
	}
	return procId.microarchitecture
}

func (procId *ProcessorIdentifier) Is64Bit() bool {
	return procId.is64bit
}

func (procId *ProcessorIdentifier) Frequency() int64 {
	return procId.frequency
}

func NewProcessorIdentifier(
	vendor, name, family, model, stepping, processorID string,
	is64bit bool,
	frequency int64,
) ProcessorIdentifier {
	var identifier string
	if vendor == "GenuineIntel" {
		if is64bit {
			identifier = "Intel64"
		} else {
			identifier = "x86"
		}
	} else {
		identifier = vendor
	}
	identifier = fmt.Sprintf("%s Family %s Model %s Stepping %s", identifier, family, model, stepping)
	if frequency < 1 {
		if freqRegex.MatchString(name) {
			unit := freqRegex.FindStringSubmatch(name)[1]
			frequency = util.ParseHertz(unit)
		} else {
			frequency = -1
		}
	}
	proc := ProcessorIdentifier{
		vendor:      vendor,
		name:        name,
		family:      family,
		model:       model,
		stepping:    stepping,
		processorID: processorID,
		identifier:  identifier,
		is64bit:     is64bit,
		frequency:   frequency,
	}
	return proc
}

type ProcOption func(processor *CentralProcessor)

type CentralProcessor interface {
	ProcessorIdentifier() ProcessorIdentifier
	PhysicalPackageCount() int
	PhysicalProcessorCount() int
	LogicalProcessorCount() int
}

func init() {
	populateProperties()
}

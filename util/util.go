/*
 * Copyright 2016-2024 The OSHI Project Contributors
 * SPDX-License-Identifier: MIT
 */

package util

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"runtime"
	"strconv"
)

var (
	multipliers = map[string]float64{
		"Hz":  1,
		"KHz": 1_000,
		"MHz": 1_000_000,
		"GHz": 1_000_000_000,
		"THz": 1_000_000_000_000,
		"PHz": 1_000_000_000_000_000,
	}
	hertzPattern = regexp.MustCompile("(\\d+(.\\d+)?) ?([kKMGT]?Hz).*")
)

// https://go.googlesource.com/sys/+/9a28524796a519b225fedd6aaaf4b1bf6c06c002/cpu/byteorder.go

// HostByteOrder might be replaced by a simple unsafe pointer check for byte order checking
func HostByteOrder() binary.ByteOrder {
	switch runtime.GOARCH {
	case "386", "amd64", "amd64p32",
		"alpha",
		"arm", "arm64",
		"loong64",
		"mipsle", "mips64le", "mips64p32le",
		"nios2",
		"ppc64le",
		"riscv", "riscv64",
		"sh":
		return binary.LittleEndian
	case "armbe", "arm64be",
		"m68k",
		"mips", "mips64", "mips64p32",
		"ppc", "ppc64",
		"s390", "s390x",
		"shbe",
		"sparc", "sparc64":
		return binary.BigEndian
	}
	panic("unknown architecture")
}

func ParseInt64OrDefault(value string, defaultValue int64) int64 {
	if val, err := strconv.ParseInt(value, 10, 64); err == nil {
		return val
	}
	return defaultValue
}

func StringValueOrDefault(value string, defaultValue string) string {
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func ParseHertz(s string) int64 {
	if !hertzPattern.MatchString(s) {
		return -1
	}
	matches := hertzPattern.FindStringSubmatch(s)
	if len(matches) != 3 {
		return -1
	}
	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		// should not happen
		err = fmt.Errorf("error parsing hertz: %w", err)
		panic(err)
	}
	multiplier, exists := multipliers[matches[2]]
	if !exists {
		multiplier = -1
	}
	val = val * multiplier
	return int64(val)
}

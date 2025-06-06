/*
 * Copyright 2016-2024 The OSHI Project Contributors
 * SPDX-License-Identifier: MIT
 */

package hardware

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"goshi/sysinfo/hardware"
	"goshi/util"
	"goshi/windows/internal"
	"regexp"
	"strconv"
	"strings"
)

const (
	displaysRegistryPath = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
)

var (
	gpuRegex = regexp.MustCompile(`.*(?:VID|VEN)_([[:xdigit:]]{4})&(?:PID|DEV)_([[:xdigit:]]{4})(.*)\\(.*)`)
)

type WindowsGraphicsCard struct {
	name, deviceId, vendor, versionInfo string
	vRam                                int64
}

func (w WindowsGraphicsCard) Name() string {
	return w.name
}

func (w WindowsGraphicsCard) DeviceId() string {
	return w.deviceId
}

func (w WindowsGraphicsCard) Vendor() string {
	return w.vendor
}

func (w WindowsGraphicsCard) VersionInfo() string {
	return w.versionInfo
}

func (w WindowsGraphicsCard) VRam() int64 {
	return w.vRam
}

func parseGPUDeviceID(devId string) []string {
	if gpuRegex.MatchString(devId) {
		matches := gpuRegex.FindStringSubmatch(devId)
		vend := "0x" + strings.ToLower(matches[1])
		prod := "0x" + strings.ToLower(matches[2])
		serial := matches[4]
		if len(matches[3]) != 0 || strings.Contains(serial, "&") {
			serial = ""
		}
		return []string{vend, prod, serial}
	}
	return nil
}

func wmiGraphicsCards() ([]hardware.GraphicsCard, error) {
	if internal.VistaOrGreater {
		return nil, errors.New("wmi: gpu query requires vista or greater")
	}
	gpus := make([]hardware.GraphicsCard, 0)
	q, err := internal.QueryWmiGraphicsCards()
	if err != nil {
		return nil, err
	}
	for _, v := range q {
		matches := parseGPUDeviceID(v.PNPDeviceID)
		var deviceId string
		if matches == nil {
			deviceId = util.Unknown
		} else {
			deviceId = matches[1]
		}
		// can be refactored
		vendor := v.AdapterCompatibility
		if matches != nil {
			if len(vendor) == 0 {
				deviceId = matches[0]
			} else {
				vendor = fmt.Sprintf("%s (%s)", vendor, matches[0])
			}
		}
		versionInfo := v.DriverVersion
		if len(versionInfo) != 0 {
			versionInfo = fmt.Sprintf("DriverVersion=%s", versionInfo)
		} else {
			versionInfo = util.Unknown
		}
		vram := v.AdapterRAM
		gpu := WindowsGraphicsCard{
			name:        util.StringValueOrDefault(v.Name, util.Unknown),
			deviceId:    deviceId,
			vendor:      util.StringValueOrDefault(vendor, util.Unknown),
			versionInfo: versionInfo,
			vRam:        int64(vram),
		}
		gpus = append(gpus, gpu)
	}
	return gpus, nil
}

func registryGraphicsCards() ([]hardware.GraphicsCard, error) {
	var err error
	gpus := make([]hardware.GraphicsCard, 0)
	acc := registry.QUERY_VALUE | registry.ENUMERATE_SUB_KEYS
	keys, err := registry.OpenKey(registry.LOCAL_MACHINE, displaysRegistryPath, uint32(acc))
	if err != nil {
		err = fmt.Errorf("error opening registry key: %w", err)
		return nil, err
	}
	defer func() {
		if derr := keys.Close(); derr != nil && err == nil {
			err = derr
		}
	}()
	subKeys, err := keys.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}
	numOfGpus := 1
	// ignore all access denied errors
	for _, v := range subKeys {
		if !strings.HasPrefix(v, "0") {
			continue
		}
		dispKey := fmt.Sprintf("%s\\%s", displaysRegistryPath, v)
		disp, err := registry.OpenKey(registry.LOCAL_MACHINE, dispKey, registry.QUERY_VALUE)
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			continue
		} else if err != nil {
			err = fmt.Errorf("registry: failed to open registry key: %w", err)
			return nil, err
		}
		_, _, err = disp.GetValue("HardwareInformation.AdapterString", nil)
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) || errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			continue
		} else if err != nil {
			err = fmt.Errorf("registry: failed to get value of HardwareInformation.AdapterString: %w", err)
			return nil, err
		}
		name, _, err := disp.GetStringValue("DriverDesc")
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			continue
		} else if err != nil {
			err = fmt.Errorf("registry: failed to get value of DriverDesc: %w", err)
			return nil, err
		}
		deviceId := "VideoController" + strconv.Itoa(numOfGpus)
		numOfGpus++
		vendor, _, err := disp.GetStringValue("ProviderName")
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			continue
		} else if err != nil {
			err = fmt.Errorf("registry: failed to get value of ProviderName: %w", err)
			return nil, err
		}
		versionInfo, _, err := disp.GetStringValue("DriverVersion")
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			continue
		} else if err != nil {
			err = fmt.Errorf("registry: failed to get value of DriverVersion: %w", err)
			return nil, err
		}
		vram := uint64(0)
		if val, _, err := disp.GetIntegerValue("HardwareInformation.qwMemorySize"); err == nil {
			vram = val
		} else if val, _, err := disp.GetIntegerValue("HardwareInformation.MemorySize"); err == nil {
			vram = val
		}
		gpu := WindowsGraphicsCard{
			name:        util.StringValueOrDefault(name, util.Unknown),
			deviceId:    util.StringValueOrDefault(deviceId, util.Unknown),
			vendor:      util.StringValueOrDefault(vendor, util.Unknown),
			versionInfo: util.StringValueOrDefault(versionInfo, util.Unknown),
			vRam:        int64(vram),
		}
		gpus = append(gpus, gpu)
	}
	return gpus, nil
}

func GPUs() ([]hardware.GraphicsCard, error) {
	gpus, err := registryGraphicsCards()
	if len(gpus) == 0 {
		return wmiGraphicsCards()
	}
	return gpus, err
}

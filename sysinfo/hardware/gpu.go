package hardware

type GraphicsCard interface {
	Name() string
	DeviceId() string
	Vendor() string
	VersionInfo() string
	VRam() int64
}

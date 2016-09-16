package darwin

// An Option is a configuration function, which configures the device.
type Option func(*Device) error

// OptPeripheralRole configures the device to perform Peripheral tasks.
func OptPeripheralRole() Option {
	return func(d *Device) error {
		d.role = 1
		return nil
	}
}

// OptCentralRole configures the device to perform Central tasks.
func OptCentralRole() Option {
	return func(d *Device) error {
		d.role = 0
		return nil
	}
}

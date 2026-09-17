package usb

import "context"

// Inventory combines demo USB targets with the real embedded firmware catalog.
func (m DummyManager) Inventory(ctx context.Context) (Inventory, error) {
	if err := wait(ctx, m.delay()/3); err != nil {
		return Inventory{}, err
	}
	images, err := Images(m.Assets)
	if err != nil {
		return Inventory{}, err
	}
	return Inventory{
		Devices: []Device{
			{Port: "/dev/cu.usbserial-0001"},
			{Port: "/dev/cu.usbmodem-ESP32S3"},
		},
		Images: images,
	}, nil
}

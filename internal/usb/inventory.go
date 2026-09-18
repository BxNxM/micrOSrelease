package usb

import "context"

// Inventory combines supported USB serial endpoints with the embedded firmware catalog.
func (m ReleaseManager) Inventory(ctx context.Context) (Inventory, error) {
	images, err := Images(m.Assets)
	if err != nil {
		return Inventory{}, err
	}
	for index := range images {
		config, _, err := loadInstallConfig(m.Assets, images[index])
		if err != nil {
			return Inventory{}, err
		}
		images[index].InstallHint = config.Hint
	}
	discover := m.Discover
	if discover == nil {
		discover = discoverDevices
	}
	devices, err := discover(ctx)
	if err != nil {
		return Inventory{}, err
	}
	return Inventory{
		Devices: devices,
		Images:  images,
	}, nil
}

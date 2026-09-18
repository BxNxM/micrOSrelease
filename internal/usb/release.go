package usb

import (
	"fmt"
	"io/fs"
)

// preparedRelease owns validated inputs for one operation. Load every payload
// before interrupting the application, and use the same config during flashing.
type preparedRelease struct {
	config    InstallConfig
	offset    uint32
	firmware  []byte
	resources []preparedResource
}

func prepareRelease(files fs.FS, image Image) (preparedRelease, error) {
	config, offset, err := loadInstallConfig(files, image)
	if err != nil {
		return preparedRelease{}, err
	}
	resources, err := prepareResources(files, config)
	if err != nil {
		return preparedRelease{}, err
	}
	firmware, err := fs.ReadFile(files, image.Path)
	if err != nil {
		return preparedRelease{}, fmt.Errorf("read firmware %s: %w", image.Path, err)
	}
	if len(firmware) == 0 {
		return preparedRelease{}, fmt.Errorf("firmware %s is empty", image.Path)
	}
	return preparedRelease{config: config, offset: offset, firmware: firmware, resources: resources}, nil
}

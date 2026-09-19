package usb

import "golang.org/x/sys/windows"

// Windows does not support directory fsync through os.File. Request a
// synchronous move after archiveBackup has flushed the complete file.
func publishBackup(temporary, name string) error {
	from, err := windows.UTF16PtrFromString(temporary)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(from, to, windows.MOVEFILE_WRITE_THROUGH)
}

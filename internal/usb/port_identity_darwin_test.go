package usb

import "testing"

func TestUSBRegistryAssociatesPortWithParentDevice(t *testing.T) {
	data := []byte(`<plist><array><dict>
 <key>IOObjectClass</key><string>IOUSBHostDevice</string>
 <key>idVendor</key><integer>12346</integer><key>idProduct</key><integer>4097</integer>
 <key>kUSBSerialNumberString</key><string>board-a</string>
 <key>locationID</key><integer>1048576</integer>
 <key>irrelevantData</key><data>AAAA</data>
 <key>IORegistryEntryChildren</key><array><dict>
 <key>IOObjectClass</key><string>IOUSBHostInterface</string>
 <key>IORegistryEntryChildren</key><array><dict>
 <key>IOCalloutDevice</key><string>/dev/cu.usbmodem1</string>
 </dict></array></dict></array></dict></array></plist>`)
	devices, err := parseUSBRegistry(data)
	if err != nil || len(devices) != 1 {
		t.Fatalf("%+v %v", devices, err)
	}
	want := Device{Port: "/dev/cu.usbmodem1", USBSerial: "board-a", USBVID: "303A", USBPID: "1001", USBLocation: "00100000"}
	if devices[0] != want {
		t.Fatalf("wrong USB association: %+v", devices[0])
	}
}

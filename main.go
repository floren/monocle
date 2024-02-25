package main

import (
	"fmt"
	"strings"
	"time"
	"tinygo.org/x/bluetooth"
)

var (
	serviceUUID = bluetooth.ServiceUUIDNordicUART
	rxUUID      = bluetooth.CharacteristicUUIDUARTRX
	txUUID      = bluetooth.CharacteristicUUIDUARTTX
)

var adapter = bluetooth.DefaultAdapter

func main() {
	// Enable BLE interface.
	err := adapter.Enable()
	if err != nil {
		println("could not enable the BLE stack:", err.Error())
		return
	}

	// The address to connect to. Set during scanning and read afterwards.
	var foundDevice bluetooth.ScanResult

	// Scan for NUS peripheral.
	println("Scanning...")
	err = adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if !result.AdvertisementPayload.HasServiceUUID(serviceUUID) {
			return
		}
		foundDevice = result

		// Stop the scan.
		err := adapter.StopScan()
		if err != nil {
			// Unlikely, but we can't recover from this.
			println("failed to stop the scan:", err.Error())
		}
	})
	if err != nil {
		println("could not start a scan:", err.Error())
		return
	}

	// Found a device: print this event.
	if name := foundDevice.LocalName(); name == "" {
		print("Connecting to ", foundDevice.Address.String(), "...")
		println()
	} else {
		print("Connecting to ", name, " (", foundDevice.Address.String(), ")...")
		println()
	}

	// Found a NUS peripheral. Connect to it.
	device, err := adapter.Connect(foundDevice.Address, bluetooth.ConnectionParams{})
	if err != nil {
		println("Failed to connect:", err.Error())
		return
	}

	// Connected. Look up the Nordic UART Service.
	println("Discovering service...")
	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		println("Failed to discover the Nordic UART Service:", err.Error())
		return
	}
	service := services[0]

	// Get the two characteristics present in this service.
	chars, err := service.DiscoverCharacteristics([]bluetooth.UUID{rxUUID, txUUID})
	if err != nil {
		println("Failed to discover RX and TX characteristics:", err.Error())
		return
	}
	rx := chars[0]
	tx := chars[1]
	// Enable notifications to receive incoming data.
	err = tx.EnableNotifications(func(value []byte) {
		println(string(value))
	})
	if err != nil {
		println("Failed to enable TX notifications:", err.Error())
		return
	}
	// send a ctrl-c + carriage return
	sendUartCommand(rx, string([]byte{0x03, 0x0d}))

	format := `import display, time, uasyncio
display.brightness(4)
time.time(%d)
time.zone("-8:00")
def gettime():
    t=time.now()
    return f'{t["month"]}/{t["day"]} {t["hour"]:02}:{t["minute"]:02}:{t["second"]:02}'

async def showtime():
    while True:
        t=display.Text(gettime(), 50, 50, display.YELLOW)
        display.show(t)
        await uasyncio.sleep(1)

t.cancel()
t=uasyncio.run(showtime())
`

	cmd := fmt.Sprintf(format, time.Now().Unix())
	if err := sendUartCommand(rx, cmd); err != nil {
		panic(err)
	}
	time.Sleep(1 * time.Second)
}

func sendUartCommand(rx bluetooth.DeviceCharacteristic, commandString string) error {
	// It expects carriage returns instead of newlines. Fine.
	cmd := []byte(strings.ReplaceAll(commandString, "\n", "\r"))
	for len(cmd) != 0 {
		// Chop off up to 20 bytes from the sendbuf.
		partlen := 20
		if len(cmd) < 20 {
			partlen = len(cmd)
		}
		part := cmd[:partlen]
		cmd = cmd[partlen:]
		_, err := rx.WriteWithoutResponse(part)
		if err != nil {
			return err
		}
	}
	return nil
}

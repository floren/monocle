package monocle

import (
	"sync"
	"time"

	"errors"
	"fmt"
	"strings"
	"tinygo.org/x/bluetooth"
)

var (
	serviceUUID = bluetooth.ServiceUUIDNordicUART
	rxUUID      = bluetooth.CharacteristicUUIDUARTRX
	txUUID      = bluetooth.CharacteristicUUIDUARTTX
)

type Monocle struct {
	sync.Mutex
	// Whether or not we're connected to the device
	connected bool
	// The device we've connected to
	device  *bluetooth.Device
	adapter *bluetooth.Adapter
	rxUART  *bluetooth.DeviceCharacteristic
	txUART  *bluetooth.DeviceCharacteristic
}

func NewMonocle() (*Monocle, error) {
	m := &Monocle{
		adapter: bluetooth.DefaultAdapter,
	}
	m.adapter.SetConnectHandler(func(device bluetooth.Device, connected bool) {
		m.connectHandler(device, connected)
	})
	m.adapter.Enable()
	return m, nil
}

func (m *Monocle) connectHandler(device bluetooth.Device, connected bool) {
	fmt.Printf("connectHandler called with %v, %v\n", device, connected)
	if m.device == nil {
		return
	}
	// Just a little check to make sure it's the right device
	if device.Address == m.device.Address {
		m.Lock()
		if connected {
			m.device = &device
			m.discoverServices()
		}
		m.connected = connected
		m.Unlock()
	}
}

// ConnectToAny attempts to find any device with the appropriate service UUID and name ("monocle").
// It will block until it finds a matching device.
// If timeout is non-zero, it will give up after approximately that much time.
func (m *Monocle) ConnectToAny(timeout time.Duration) error {
	m.Lock()
	defer m.Unlock()
	resultChan := make(chan bluetooth.ScanResult)
	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()
	fmt.Println("starting scan")
	go func() {
		err := m.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			fmt.Printf("scan found %v %+v\n", result.LocalName(), result)
			if !result.AdvertisementPayload.HasServiceUUID(serviceUUID) {
				return
			}
			if result.LocalName() != "monocle" {
				return
			}
			// We found it! Stop the scan and send the result out
			m.adapter.StopScan()
			resultChan <- result
		})
		if err != nil {
			errChan <- err
		}
		wg.Done()
	}()

	// set up the timer if a timeout was specified
	timer := time.NewTimer(timeout)
	if timeout == 0 && !timer.Stop() {
		// drain the channel if timeout was zero
		<-timer.C
	}
	select {
	case result := <-resultChan:
		fmt.Printf("got result: %+v\n", result)
		// We got a result, connect to it
		device, err := m.adapter.Connect(result.Address, bluetooth.ConnectionParams{})
		if err != nil {
			return err
		}
		m.device = &device
		m.connected = true
		m.discoverServices()
	case <-timer.C:
		m.adapter.StopScan()
		return errors.New("timeout")
	}

	return nil
}

// discoverServices should be called after the device has been found and connected to.
// Caller should hold the lock.
func (m *Monocle) discoverServices() error {
	if m.device == nil {
		return errors.New("device not initialized")
	}
	services, err := m.device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		return err
	}
	service := services[0]

	// Get the two characteristics present in this service.
	chars, err := service.DiscoverCharacteristics([]bluetooth.UUID{rxUUID, txUUID})
	if err != nil {
		return err
	}
	m.rxUART = &chars[0]
	m.txUART = &chars[1]

	return nil
}

func (m *Monocle) EnableTxUartNotifications(cb func([]byte)) error {
	m.Lock()
	defer m.Unlock()
	if !m.connected || m.txUART == nil {
		return errors.New("Not connected")
	}
	return m.txUART.EnableNotifications(cb)
}

func (m *Monocle) SendUartCommand(commandString string) error {
	m.Lock()
	defer m.Unlock()

	if !m.connected || m.rxUART == nil {
		return errors.New("Not connected")
	}

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
		_, err := m.rxUART.WriteWithoutResponse(part)
		if err != nil {
			return err
		}
	}
	return nil
}

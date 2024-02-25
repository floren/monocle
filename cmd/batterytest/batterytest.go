package main

import (
	"fmt"
	"github.com/floren/monocle/pkg/monocle"
	"log"
	"time"
)

func main() {
	log.Printf("Starting")
	m, err := monocle.NewMonocle()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Connecting...")
	if err := m.ConnectToAny(0); err != nil {
		log.Fatal(err)
	}
	log.Printf("Connected")
	// Enable notifications to receive incoming data.
	err = m.EnableTxUartNotifications(func(value []byte) {
		println(string(value))
	})
	if err != nil {
		log.Fatalf("Failed to enable TX notifications: %v", err)
	}
	// send a ctrl-c + carriage return
	m.SendUartCommand(string([]byte{0x03, 0x0d}))

	format := `import display, time, device
# blank the screen
display.show()
# set the time
time.time(%d)
time.zone("-8:00")
def gettime():
    t=time.now()
    return f'{t["month"]}/{t["day"]} {t["hour"]:02}:{t["minute"]:02}:{t["second"]:02}'

while True:
    print(gettime())
    print(device.battery_level())
    time.sleep(60)

`

	cmd := fmt.Sprintf(format, time.Now().Unix())
	if err := m.SendUartCommand(cmd); err != nil {
		panic(err)
	}
	for {
		time.Sleep(10 * time.Second)
		if err := m.SendUartCommand("\r"); err != nil {
			panic(err)
		}
	}
}

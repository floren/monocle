# Go tools for Brilliant Labs Monocle

## Go library

The `pkg/monocle` subdirectory implements a basic library for using the Monocle.

Instantiate a `monocle.Monocle`, call `ConnectToAny` to connect to any nearby Monocle device, then use `SendUartCommand` to send strings to the REPL.

Here's a Go program that will turn your Monocle into a clock:

```
package main

import (
	"fmt"
	"github.com/floren/monocle/pkg/monocle"
	"log"
	"time"
)

func main() {
	m, err := monocle.NewMonocle()
	if err != nil {
		log.Fatal(err)
	}
	if err := m.ConnectToAny(0); err != nil {
		log.Fatal(err)
	}

	// send a ctrl-c + carriage return to interrupt anything that's running
	m.SendUartCommand(string([]byte{0x03, 0x0d}))

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
	if err := m.SendUartCommand(cmd); err != nil {
		panic(err)
	}
}
```

## Battery Life Tester

To see how about how long your battery lasts with the display off and periodic communication from the host:

```
go run github.com/floren/monocle/cmd/batterytest@latest
```

Every minute, the Monocle should send the current time and the battery percentage. When the device finally dies (or you put it back into the case), the program will exit, reporting how long it ran.

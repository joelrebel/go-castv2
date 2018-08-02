package castv2

import (
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

//Chromecasts are chatty so we wouldn't need to worry too much about lots of devices in one network. It's not really feasible.
const deviceBufferSearchSize = 100

//FindDevices searches the LAN for chromecast devices via mDNS and sends them to a channel.
func FindDevices(timeout time.Duration, devices chan<- Device) {

	// Make a channel for results and start listening
	entries := make(chan *mdns.ServiceEntry, deviceBufferSearchSize)

	go createDeviceObjects(entries, devices)
	go lookupChromecastMDNSENtries(entries, timeout)
}

func createDeviceObjects(entries <-chan *mdns.ServiceEntry, devices chan<- Device) {
	defer close(devices)
	for entry := range entries {
		if !strings.Contains(entry.Name, chromecastServiceName) {
			return
		}
		device, err := NewDevice(entry.Addr, entry.Port)
		if err != nil {
			return
		}
		devices <- device
	}
}
func lookupChromecastMDNSENtries(entries chan<- *mdns.ServiceEntry, timeout time.Duration) {
	defer close(entries)
	mdns.Query(&mdns.QueryParam{
		Service: chromecastServiceName,
		Timeout: timeout,
		Entries: entries,
	})
}

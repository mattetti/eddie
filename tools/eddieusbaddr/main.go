package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var unixLsDevQuery = "/dev/cu.usbserial-*"

func main() {
	devices, err := filepath.Glob(unixLsDevQuery)
	if err != nil {
		fmt.Println("error looking for devices", err)
	}
	if len(devices) == 0 {
		fmt.Println("No usb devices found")
		os.Exit(1)
	}
	fmt.Println("Try connecting to one of the following devices (and press enter once connected):")
	for _, dev := range devices {
		fmt.Printf("\tscreen -L %s –L\n", dev)
	}
}

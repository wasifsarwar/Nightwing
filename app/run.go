package app

import (
	"./system"
)


// Run connects to the bigIP devices and exports data
func Run() {
	system.ConnectAndExportSNMP()
}
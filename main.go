package main

import (
	"os"
)

func main() {
	InitLogger()
	GetLogger().Info("minimcd starting")
	if err := LoadConfig("config.yml"); err != nil {
		os.Exit(1)
	}

}

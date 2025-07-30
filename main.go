package main

import (
	"sync"

	"github.com/DHCPCD9/go-swaga-bot/configuration"
	"github.com/DHCPCD9/go-swaga-bot/database"
	"github.com/DHCPCD9/go-swaga-bot/discord"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	logrus.SetLevel(logrus.DebugLevel)

	if _, err := configuration.ReadConfig(); err != nil {
		logrus.Fatalf("Failed to read configuration: %v", err)
	}

	if err := database.InitDatabase(); err != nil {
		logrus.Fatalf("Failed to initialize database: %v", err)
	}

	if err := discord.Init(); err != nil {
		logrus.Fatalf("Failed to initialize Discord: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}

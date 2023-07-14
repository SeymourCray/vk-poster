package main

import (
	"vk-poster/config"
	"vk-poster/internal/app"
)

func main() {
	cfg := config.NewConfigFromEnv()

	app.Run(cfg)
}

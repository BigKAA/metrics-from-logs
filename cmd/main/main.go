package main

import (
	"log"

	"github.com/BigKAA/metrics-from-logs/app/instance"
	"github.com/joho/godotenv"
)

func init() {
	// load values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	inst := instance.NewInstance()
	if inst == nil {
		return
	}
	inst.Start()
}

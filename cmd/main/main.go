package main

import (
	"log"

	"github.com/BigKAA/metrics-from-logs/app/instance"
	"github.com/joho/godotenv"
)

func main() {
	var inst *instance.Instance
	inst = instance.NewInstance()
	if inst == nil {
		return
	}
	inst.Start()
}

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

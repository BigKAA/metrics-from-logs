package main

import (
	"github.com/BigKAA/metrics-from-logs/app/instance"
	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load(".env")
}

func main() {
	inst := instance.NewInstance()
	if inst == nil {
		return
	}
	inst.Start()
}

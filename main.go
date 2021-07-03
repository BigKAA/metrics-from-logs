package main

import (
	"github.com/BigKAA/metrics-from-logs/app/instance"
)

func main() {
	inst := instance.NewInstance()
	if inst == nil {
		return
	}
	inst.Start()
}

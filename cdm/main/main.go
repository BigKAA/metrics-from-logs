package main

import "github.com/BigKAA/metrics-from-logs/app/instance"

func main() {
	instance := instance.NewInstance()
	if instance == nil {
		return
	}
	instance.Start()
}

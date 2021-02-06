package main

import "github.com/BigKAA/metrics-from-logs/app/instance"

func main() {
	var inst *instance.Instance
	inst = instance.NewInstance()
	if inst == nil {
		return
	}
	inst.Start()
}

package main

import (
	"math/rand"
	"testing"
	"time"
)

func TestSleep(t *testing.T) {
	time.Sleep(1 * time.Second)
	
	if rand.Float64() < 0.1 {
		t.Fatal("Random test failure (10% chance)")
	}
}
package main

import (
	"testing"
	"time"
)

func TestNextWait(t *testing.T) {
	hour := time.Hour
	if got := nextWait(exitClean, hour); got != hour {
		t.Errorf("clean run should wait the interval, got %s", got)
	}
	if got := nextWait(exitProblems, hour); got != hour {
		t.Errorf("a run with problems completed and should wait the interval, got %s", got)
	}
	if got := nextWait(exitFailed, hour); got != retry {
		t.Errorf("a failed run should retry soon, got %s", got)
	}
	if got := nextWait(exitFailed, 30*time.Second); got != 30*time.Second {
		t.Errorf("retry never waits longer than the interval, got %s", got)
	}
}

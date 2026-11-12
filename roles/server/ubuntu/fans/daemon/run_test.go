package fans

import (
	"context"
	"testing"
	"time"
)

func TestRunTicksUntilCancelledThenHandsTheFansBack(t *testing.T) {
	root := driver(t)
	writeAll(t, root+"/hwmon0", map[string]string{"name": "coretemp", "temp1_label": "Package id 0", "temp1_input": "50000"})
	cfg, _ := Parse([]byte(dtConfig))
	cfg.Interval = 0.001
	d := New(cfg, Hwmon{OSSysfs{}, root}, nil, func(string) {})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error)
	go func() { done <- d.Run(ctx) }()
	deadline := time.Now().Add(5 * time.Second)
	for file(t, root, "pwm2_enable") != "1\n" {
		if time.Now().After(deadline) {
			t.Fatal("the fan was never taken from the board")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run = %v, want nil", err)
	}
	if got := file(t, root, "pwm2_enable"); got != "2\n" {
		t.Errorf("pwm2_enable after Run = %q, want 2", got)
	}
}

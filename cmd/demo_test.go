/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/bwagner5/inflate/pkg/demo"
)

func TestDemoCommandDefaultsToMediumPreset(t *testing.T) {
	t.Parallel()

	cmd := newDemoCommand(&globalOptions{}, nil)
	want := map[string]string{
		"preset":       "medium",
		"hold":         "2m0s",
		"max-memory":   "5Gi",
		"max-replicas": "1",
		"max-nodes":    "20",
		"max-cpu":      "10",
	}
	for name, value := range want {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("flag %q not found", name)
		}
		if flag.DefValue != value {
			t.Errorf("%s default = %q, want %q", name, flag.DefValue, value)
		}
	}
}

func TestApplyDemoPreset(t *testing.T) {
	t.Parallel()

	config := demo.Config{}
	if err := applyDemoPreset(&config, "low", func(string) bool { return false }); err != nil {
		t.Fatal(err)
	}

	if config.MinNodes != 5 || config.MaxNodes != 10 {
		t.Fatalf("node range = %d..%d, want 5..10", config.MinNodes, config.MaxNodes)
	}
	if config.MinReplicas != 1 || config.MaxReplicas != 3 {
		t.Fatalf("replica range = %d..%d, want 1..3", config.MinReplicas, config.MaxReplicas)
	}
	if config.MaxCPU != "1500m" || config.MaxMemory != "1536Mi" {
		t.Fatalf("resource maximums = %s/%s, want 1500m/1536Mi", config.MaxCPU, config.MaxMemory)
	}
	if config.Interval != 15*time.Second || config.Hold != time.Minute {
		t.Fatalf("timing = %s/%s, want 15s/1m", config.Interval, config.Hold)
	}
}

func TestApplyDemoPresetKeepsExplicitFlags(t *testing.T) {
	t.Parallel()

	config := demo.Config{MaxNodes: 12, MaxCPU: "4", Hold: 3 * time.Minute}
	changed := map[string]bool{"max-nodes": true, "max-cpu": true, "hold": true}
	if err := applyDemoPreset(&config, "high", changedValue(changed)); err != nil {
		t.Fatal(err)
	}

	if config.MaxNodes != 12 || config.MaxCPU != "4" || config.Hold != 3*time.Minute {
		t.Fatalf("explicit values were overwritten: max-nodes=%d max-cpu=%s hold=%s", config.MaxNodes, config.MaxCPU, config.Hold)
	}
	if config.MinNodes != 20 || config.MaxReplicas != 1 || config.Interval != 5*time.Second {
		t.Fatalf("preset values were not applied: min-nodes=%d max-replicas=%d interval=%s", config.MinNodes, config.MaxReplicas, config.Interval)
	}
}

func TestApplyDemoPresetRejectsUnknownPreset(t *testing.T) {
	t.Parallel()

	err := applyDemoPreset(&demo.Config{}, "huge", func(string) bool { return false })
	if err == nil || !strings.Contains(err.Error(), "must be low, medium, or high") {
		t.Fatalf("error = %v, want preset choices", err)
	}
}

func changedValue(changed map[string]bool) func(string) bool {
	return func(name string) bool {
		return changed[name]
	}
}

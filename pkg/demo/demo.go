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

// Package demo runs repeatable inflate workload waves for autoscaler demonstrations.
package demo

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/bwagner5/inflate/pkg/inflater"
)

const (
	cpuStepMilli   = int64(50)
	memoryStepByte = int64(64 * 1024 * 1024)
)

// Config controls the size and timing of demo waves.
type Config struct {
	Namespace   string
	Image       string
	MinNodes    int
	MaxNodes    int
	MinReplicas int
	MaxReplicas int
	MinCPU      string
	MaxCPU      string
	MinMemory   string
	MaxMemory   string
	Interval    time.Duration
	Hold        time.Duration
	Cycles      int
	Seed        int64
	Cleanup     bool
	CPUArch     string
	OS          string
}

// Workload describes one randomly sized Deployment in a wave.
type Workload struct {
	Name     string
	Replicas int32
	CPU      string
	Memory   string
}

// Runner grows and shrinks a set of demo Deployments.
type Runner struct {
	inflater *inflater.Inflater
	config   Config
	demoID   string
	random   *rand.Rand
	logf     func(string, ...any)
	minCPU   int64
	maxCPU   int64
	minMem   int64
	maxMem   int64
}

// New validates a demo configuration and constructs a Runner.
func New(inflate *inflater.Inflater, config Config, logf func(string, ...any)) (*Runner, error) {
	if inflate == nil {
		return nil, fmt.Errorf("inflater must not be nil")
	}
	if config.MinNodes < 1 || config.MaxNodes < config.MinNodes {
		return nil, fmt.Errorf("node range must satisfy 1 <= min-nodes <= max-nodes")
	}
	if config.MinReplicas < 1 || config.MaxReplicas < config.MinReplicas {
		return nil, fmt.Errorf("replica range must satisfy 1 <= min-replicas <= max-replicas")
	}
	if config.Interval < 0 || config.Hold < 0 {
		return nil, fmt.Errorf("interval and hold must not be negative")
	}
	if config.Cycles < 0 {
		return nil, fmt.Errorf("cycles must not be negative")
	}

	minCPU, maxCPU, err := parseCPURange(config.MinCPU, config.MaxCPU)
	if err != nil {
		return nil, err
	}
	minMem, maxMem, err := parseMemoryRange(config.MinMemory, config.MaxMemory)
	if err != nil {
		return nil, err
	}
	if config.Seed == 0 {
		config.Seed = time.Now().UnixNano()
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}

	return &Runner{
		inflater: inflate,
		config:   config,
		demoID:   fmt.Sprintf("run-%x", uint64(time.Now().UnixNano())),
		random:   rand.New(rand.NewSource(config.Seed)), //nolint:gosec // Demo randomness is intentional.
		logf:     logf,
		minCPU:   minCPU,
		maxCPU:   maxCPU,
		minMem:   minMem,
		maxMem:   maxMem,
	}, nil
}

// Run continuously grows and shrinks workload waves until the context is canceled.
func (r *Runner) Run(ctx context.Context) error {
	if err := r.inflater.EnsureNamespace(ctx, r.config.Namespace); err != nil {
		return fmt.Errorf("create namespace: %w", err)
	}
	if r.config.Cleanup {
		defer r.cleanup()
	}

	r.logf("demo %s started; press Ctrl-C to stop", r.demoID)
	for cycle := 1; r.config.Cycles == 0 || cycle <= r.config.Cycles; cycle++ {
		target := r.randomBetween(r.config.MinNodes, r.config.MaxNodes)
		wave := r.planWave(cycle, target)
		r.logf("cycle %d: growing toward %d demo nodes", cycle, target)
		for index, workload := range wave {
			_, err := r.inflater.Apply(ctx, inflater.Options{
				Name:       workload.Name,
				Namespace:  r.config.Namespace,
				Image:      r.config.Image,
				CPU:        workload.CPU,
				Memory:     workload.Memory,
				Replicas:   workload.Replicas,
				DemoID:     r.demoID,
				OnePerNode: true,
				CPUArch:    r.config.CPUArch,
				OS:         r.config.OS,
			})
			if err != nil {
				return fmt.Errorf("create %s: %w", workload.Name, err)
			}
			r.logf(
				"  created %s: replicas=%d cpu=%s memory=%s",
				workload.Name, workload.Replicas, workload.CPU, workload.Memory,
			)
			if index < len(wave)-1 && !wait(ctx, r.config.Interval) {
				return nil
			}
		}

		r.logf("cycle %d: holding at %d demo nodes for %s", cycle, target, r.config.Hold)
		if !wait(ctx, r.config.Hold) {
			return nil
		}

		r.logf("cycle %d: shrinking the workload", cycle)
		for index, workload := range slices.Backward(wave) {
			if err := r.inflater.Delete(ctx, r.config.Namespace, workload.Name); err != nil {
				return fmt.Errorf("delete %s: %w", workload.Name, err)
			}
			r.logf("  deleted %s", workload.Name)
			if index > 0 && !wait(ctx, r.config.Interval) {
				return nil
			}
		}

		if r.config.Cycles != 0 && cycle == r.config.Cycles {
			break
		}
		r.logf("cycle %d: holding empty for %s", cycle, r.config.Hold)
		if !wait(ctx, r.config.Hold) {
			return nil
		}
	}
	return nil
}

func (r *Runner) planWave(cycle, target int) []Workload {
	remaining := target
	var workloads []Workload
	for index := 1; remaining > 0; index++ {
		replicas := r.randomBetween(r.config.MinReplicas, r.config.MaxReplicas)
		replicas = min(replicas, remaining)
		workloads = append(workloads, Workload{
			Name:     fmt.Sprintf("inflate-demo-%02d-%02d", cycle, index),
			Replicas: int32(replicas),
			CPU:      resource.NewMilliQuantity(r.randomStepped(r.minCPU, r.maxCPU, cpuStepMilli), resource.DecimalSI).String(),
			Memory:   resource.NewQuantity(r.randomStepped(r.minMem, r.maxMem, memoryStepByte), resource.BinarySI).String(),
		})
		remaining -= replicas
	}
	return workloads
}

func (r *Runner) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := r.inflater.DeleteDemo(ctx, r.config.Namespace, r.demoID); err != nil {
		r.logf("cleanup failed: %v", err)
		return
	}
	r.logf("demo workloads cleaned up")
}

func (r *Runner) randomBetween(minimum, maximum int) int {
	return minimum + r.random.Intn(maximum-minimum+1)
}

func (r *Runner) randomStepped(minimum, maximum, step int64) int64 {
	steps := (maximum - minimum) / step
	return minimum + r.random.Int63n(steps+1)*step
}

func parseCPURange(minimum, maximum string) (int64, int64, error) {
	minQuantity, err := resource.ParseQuantity(minimum)
	if err != nil || minQuantity.Sign() <= 0 {
		return 0, 0, fmt.Errorf("invalid min-cpu %q", minimum)
	}
	maxQuantity, err := resource.ParseQuantity(maximum)
	if err != nil || maxQuantity.Sign() <= 0 {
		return 0, 0, fmt.Errorf("invalid max-cpu %q", maximum)
	}
	minValue, maxValue := minQuantity.MilliValue(), maxQuantity.MilliValue()
	if maxValue < minValue {
		return 0, 0, fmt.Errorf("min-cpu must not exceed max-cpu")
	}
	return minValue, maxValue, nil
}

func parseMemoryRange(minimum, maximum string) (int64, int64, error) {
	minQuantity, err := resource.ParseQuantity(minimum)
	if err != nil || minQuantity.Sign() <= 0 {
		return 0, 0, fmt.Errorf("invalid min-memory %q", minimum)
	}
	maxQuantity, err := resource.ParseQuantity(maximum)
	if err != nil || maxQuantity.Sign() <= 0 {
		return 0, 0, fmt.Errorf("invalid max-memory %q", maximum)
	}
	minValue, maxValue := minQuantity.Value(), maxQuantity.Value()
	if maxValue < minValue {
		return 0, 0, fmt.Errorf("min-memory must not exceed max-memory")
	}
	return minValue, maxValue, nil
}

func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

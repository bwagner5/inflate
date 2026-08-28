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

package demo

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/bwagner5/inflate/pkg/inflater"
)

func TestPlanWaveHonorsBounds(t *testing.T) {
	t.Parallel()

	runner, err := New(inflater.New(fake.NewClientset()), Config{
		Namespace: "inflate", Image: "pause", MinNodes: 5, MaxNodes: 10,
		MinReplicas: 1, MaxReplicas: 3, MinCPU: "250m", MaxCPU: "1500m",
		MinMemory: "256Mi", MaxMemory: "1536Mi", Seed: 42,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	for target := 5; target <= 10; target++ {
		wave := runner.planWave(1, target)
		total := 0
		for _, workload := range wave {
			total += int(workload.Replicas)
			if workload.Replicas < 1 || workload.Replicas > 3 {
				t.Fatalf("replicas = %d, want 1..3", workload.Replicas)
			}
			cpu := resource.MustParse(workload.CPU)
			if cpu.Cmp(resource.MustParse("250m")) < 0 || cpu.Cmp(resource.MustParse("1500m")) > 0 {
				t.Fatalf("cpu = %s, outside configured range", workload.CPU)
			}
			memory := resource.MustParse(workload.Memory)
			if memory.Cmp(resource.MustParse("256Mi")) < 0 || memory.Cmp(resource.MustParse("1536Mi")) > 0 {
				t.Fatalf("memory = %s, outside configured range", workload.Memory)
			}
		}
		if total != target {
			t.Fatalf("total replicas = %d, want %d", total, target)
		}
	}
}

func TestNewRejectsInvertedRanges(t *testing.T) {
	t.Parallel()

	_, err := New(inflater.New(fake.NewClientset()), Config{
		Namespace: "inflate", Image: "pause", MinNodes: 10, MaxNodes: 5,
		MinReplicas: 1, MaxReplicas: 3, MinCPU: "250m", MaxCPU: "1",
		MinMemory: "256Mi", MaxMemory: "1Gi",
	}, nil)
	if err == nil {
		t.Fatal("expected invalid node range error")
	}
}

func TestRunOneCycleCleansUp(t *testing.T) {
	t.Parallel()

	client := fake.NewClientset()
	inflate := inflater.New(client)
	runner, err := New(inflate, Config{
		Namespace: "inflate", Image: "pause", MinNodes: 5, MaxNodes: 5,
		MinReplicas: 1, MaxReplicas: 3, MinCPU: "250m", MaxCPU: "1",
		MinMemory: "256Mi", MaxMemory: "1Gi", Interval: 0, Hold: 0,
		Cycles: 1, Seed: 42, Cleanup: true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runner.Run(ctx); err != nil {
		t.Fatal(err)
	}
	deployments, err := inflate.List(context.Background(), "inflate")
	if err != nil {
		t.Fatal(err)
	}
	if len(deployments) != 0 {
		t.Fatalf("remaining Deployments = %d, want 0", len(deployments))
	}
}

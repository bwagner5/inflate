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

// Command inflate creates Kubernetes workloads for scheduling and autoscaling demos.
package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/bwagner5/inflate/pkg/demo"
	"github.com/bwagner5/inflate/pkg/inflater"
)

const defaultDemoPreset = "medium"

type demoPreset struct {
	minNodes    int
	maxNodes    int
	minReplicas int
	maxReplicas int
	minCPU      string
	maxCPU      string
	minMemory   string
	maxMemory   string
	interval    time.Duration
	hold        time.Duration
}

var demoPresets = map[string]demoPreset{
	"low": {
		minNodes: 5, maxNodes: 10,
		minReplicas: 1, maxReplicas: 3,
		minCPU: "250m", maxCPU: "1500m",
		minMemory: "256Mi", maxMemory: "1536Mi",
		interval: 15 * time.Second, hold: time.Minute,
	},
	"medium": {
		minNodes: 5, maxNodes: 20,
		minReplicas: 1, maxReplicas: 1,
		minCPU: "250m", maxCPU: "10",
		minMemory: "256Mi", maxMemory: "5Gi",
		interval: 15 * time.Second, hold: 2 * time.Minute,
	},
	"high": {
		minNodes: 20, maxNodes: 50,
		minReplicas: 1, maxReplicas: 1,
		minCPU: "250m", maxCPU: "10",
		minMemory: "256Mi", maxMemory: "5Gi",
		interval: 5 * time.Second, hold: time.Minute,
	},
}

func newDemoCommand(
	global *globalOptions,
	clientFactory func() (kubernetes.Interface, error),
) *cobra.Command {
	opts := demo.Config{}
	presetName := defaultDemoPreset
	defaults := demoPresets[defaultDemoPreset]
	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Run a foreground autoscaling demo",
		Long: "Run a foreground loop that grows and shrinks randomized inflate Deployments. " +
			"Required hostname anti-affinity keeps each demo pod on a separate node. " +
			"Presets provide workload defaults; explicitly set sizing and timing flags take precedence.",
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return applyDemoPreset(&opts, presetName, cmd.Flags().Changed)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.Namespace = global.namespace
			client, err := clientFactory()
			if err != nil {
				return err
			}
			runner, err := demo.New(
				inflater.New(client),
				opts,
				func(format string, args ...any) {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), format+"\n", args...)
				},
			)
			if err != nil {
				return err
			}
			return runner.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&opts.Image, "image", "i", defaultImage, "container image")
	cmd.Flags().StringVar(&presetName, "preset", defaultDemoPreset, "workload preset: low, medium, or high")
	cmd.Flags().IntVar(&opts.MinNodes, "min-nodes", defaults.minNodes, "minimum demo pod/node target per cycle")
	cmd.Flags().IntVar(&opts.MaxNodes, "max-nodes", defaults.maxNodes, "maximum demo pod/node target per cycle")
	cmd.Flags().IntVar(&opts.MinReplicas, "min-replicas", defaults.minReplicas, "minimum replicas per Deployment")
	cmd.Flags().IntVar(&opts.MaxReplicas, "max-replicas", defaults.maxReplicas, "maximum replicas per Deployment")
	cmd.Flags().StringVar(&opts.MinCPU, "min-cpu", defaults.minCPU, "minimum CPU request per pod")
	cmd.Flags().StringVar(&opts.MaxCPU, "max-cpu", defaults.maxCPU, "maximum CPU request per pod")
	cmd.Flags().StringVar(&opts.MinMemory, "min-memory", defaults.minMemory, "minimum memory request per pod")
	cmd.Flags().StringVar(&opts.MaxMemory, "max-memory", defaults.maxMemory, "maximum memory request per pod")
	cmd.Flags().DurationVar(&opts.Interval, "interval", defaults.interval, "delay between Deployment changes")
	cmd.Flags().DurationVar(&opts.Hold, "hold", defaults.hold, "delay at the top and bottom of each cycle")
	cmd.Flags().IntVar(&opts.Cycles, "cycles", 0, "number of cycles to run (0 runs until interrupted)")
	cmd.Flags().Int64Var(&opts.Seed, "seed", 0, "random seed (0 uses the current time)")
	cmd.Flags().BoolVar(&opts.Cleanup, "cleanup", true, "delete this run's Deployments on exit")
	cmd.Flags().StringVarP(&opts.CPUArch, "cpu-arch", "c", "", "select a CPU architecture")
	cmd.Flags().StringVar(&opts.OS, "os", "", "select an operating system")
	return cmd
}

func applyDemoPreset(config *demo.Config, name string, changed func(string) bool) error {
	preset, ok := demoPresets[name]
	if !ok {
		return fmt.Errorf("invalid preset %q (must be low, medium, or high)", name)
	}

	if !changed("min-nodes") {
		config.MinNodes = preset.minNodes
	}
	if !changed("max-nodes") {
		config.MaxNodes = preset.maxNodes
	}
	if !changed("min-replicas") {
		config.MinReplicas = preset.minReplicas
	}
	if !changed("max-replicas") {
		config.MaxReplicas = preset.maxReplicas
	}
	if !changed("min-cpu") {
		config.MinCPU = preset.minCPU
	}
	if !changed("max-cpu") {
		config.MaxCPU = preset.maxCPU
	}
	if !changed("min-memory") {
		config.MinMemory = preset.minMemory
	}
	if !changed("max-memory") {
		config.MaxMemory = preset.maxMemory
	}
	if !changed("interval") {
		config.Interval = preset.interval
	}
	if !changed("hold") {
		config.Hold = preset.hold
	}
	return nil
}

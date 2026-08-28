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
	"math/rand"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/bwagner5/inflate/pkg/inflater"
)

const defaultImage = "registry.k8s.io/pause:3.10.2"

type createOptions struct {
	image              string
	cpu                string
	memory             string
	replicas           int32
	dryRun             bool
	randomSuffix       bool
	onePerNode         bool
	hostnameSpread     bool
	zonalSpread        bool
	capacityTypeSpread bool
	hostNetwork        bool
	cpuArch            string
	os                 string
}

func newCreateCommand(
	global *globalOptions,
	clientFactory func() (kubernetes.Interface, error),
) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create or update one inflate Deployment",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "inflate"
			if len(args) == 1 {
				name = args[0]
			}
			if opts.randomSuffix {
				name = fmt.Sprintf("%s-%08x", name, rand.Uint32()) //nolint:gosec
			}
			deploymentOptions := inflater.Options{
				Name:               name,
				Namespace:          global.namespace,
				Image:              opts.image,
				CPU:                opts.cpu,
				Memory:             opts.memory,
				Replicas:           opts.replicas,
				OnePerNode:         opts.onePerNode,
				HostnameSpread:     opts.hostnameSpread,
				ZonalSpread:        opts.zonalSpread,
				CapacityTypeSpread: opts.capacityTypeSpread,
				HostNetwork:        opts.hostNetwork,
				CPUArch:            opts.cpuArch,
				OS:                 opts.os,
			}
			if opts.dryRun {
				deployment, err := inflater.Deployment(deploymentOptions)
				if err != nil {
					return err
				}
				output, err := yaml.Marshal(deployment)
				if err != nil {
					return fmt.Errorf("encode Deployment: %w", err)
				}
				_, err = fmt.Fprint(cmd.OutOrStdout(), string(output))
				return err
			}

			client, err := clientFactory()
			if err != nil {
				return err
			}
			inflate := inflater.New(client)
			if err := inflate.EnsureNamespace(cmd.Context(), global.namespace); err != nil {
				return fmt.Errorf("create namespace: %w", err)
			}
			deployment, err := inflate.Apply(cmd.Context(), deploymentOptions)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(), "Deployment %s/%s applied\n", deployment.Namespace, deployment.Name,
			)
			return err
		},
	}

	cmd.Flags().StringVarP(&opts.image, "image", "i", defaultImage, "container image")
	cmd.Flags().StringVar(&opts.cpu, "cpu", "500m", "CPU request per replica")
	cmd.Flags().StringVar(&opts.memory, "memory", "256Mi", "memory request per replica")
	cmd.Flags().Int32VarP(&opts.replicas, "replicas", "r", 1, "number of replicas")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "print the Deployment YAML without applying it")
	cmd.Flags().BoolVar(&opts.randomSuffix, "random-suffix", false, "append a random suffix to the name")
	cmd.Flags().BoolVar(&opts.onePerNode, "one-per-node", false, "require replicas to run on separate nodes")
	cmd.Flags().BoolVar(&opts.hostnameSpread, "hostname-spread", false, "spread replicas across hostnames")
	cmd.Flags().BoolVarP(&opts.zonalSpread, "zonal-spread", "z", false, "spread replicas across zones")
	cmd.Flags().BoolVar(&opts.capacityTypeSpread, "capacity-type-spread", false, "spread replicas across Karpenter capacity types")
	cmd.Flags().BoolVar(&opts.hostNetwork, "host-network", false, "use host networking")
	cmd.Flags().StringVarP(&opts.cpuArch, "cpu-arch", "c", "", "select a CPU architecture")
	cmd.Flags().StringVar(&opts.os, "os", "", "select an operating system")
	return cmd
}

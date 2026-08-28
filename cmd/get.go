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
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/bwagner5/inflate/pkg/inflater"
)

func newGetCommand(
	global *globalOptions,
	clientFactory func() (kubernetes.Interface, error),
) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "get [name]",
		Short: "List inflate Deployments",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFactory()
			if err != nil {
				return err
			}
			inflate := inflater.New(client)
			var deployments []appsv1.Deployment
			if len(args) == 1 {
				deployment, err := inflate.Get(cmd.Context(), global.namespace, args[0])
				if err != nil {
					return err
				}
				deployments = []appsv1.Deployment{*deployment}
			} else {
				deployments, err = inflate.List(cmd.Context(), global.namespace)
				if err != nil {
					return err
				}
			}
			sort.Slice(deployments, func(left, right int) bool {
				return deployments[left].Name < deployments[right].Name
			})

			switch output {
			case "table":
				return printDeploymentTable(cmd, deployments)
			case "yaml":
				encoded, err := yaml.Marshal(deployments)
				if err != nil {
					return fmt.Errorf("encode Deployments: %w", err)
				}
				_, err = fmt.Fprint(cmd.OutOrStdout(), string(encoded))
				return err
			default:
				return fmt.Errorf("unknown output %q (expected table or yaml)", output)
			}
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "table", "output format: table or yaml")
	return cmd
}

func printDeploymentTable(cmd *cobra.Command, deployments []appsv1.Deployment) error {
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "NAME\tREPLICAS\tCPU\tMEMORY"); err != nil {
		return err
	}
	for _, deployment := range deployments {
		container := deployment.Spec.Template.Spec.Containers[0]
		cpu := container.Resources.Requests.Cpu().String()
		memory := container.Resources.Requests.Memory().String()
		if _, err := fmt.Fprintf(
			writer, "%s\t%d\t%s\t%s\n", deployment.Name, *deployment.Spec.Replicas, cpu, memory,
		); err != nil {
			return err
		}
	}
	return writer.Flush()
}

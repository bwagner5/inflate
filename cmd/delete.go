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

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/bwagner5/inflate/pkg/inflater"
)

func newDeleteCommand(
	global *globalOptions,
	clientFactory func() (kubernetes.Interface, error),
) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete inflate Deployments",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !all {
				return fmt.Errorf("specify a Deployment name or --all")
			}
			if len(args) == 1 && all {
				return fmt.Errorf("a Deployment name and --all cannot be used together")
			}
			client, err := clientFactory()
			if err != nil {
				return err
			}
			inflate := inflater.New(client)
			if all {
				err = inflate.DeleteAll(cmd.Context(), global.namespace)
			} else {
				err = inflate.Delete(cmd.Context(), global.namespace, args[0])
			}
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Deployments deleted")
			return err
		},
	}
	cmd.Flags().BoolVarP(&all, "all", "a", false, "delete all inflate Deployments in the namespace")
	return cmd
}

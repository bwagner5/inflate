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
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/bwagner5/inflate/pkg/inflater"
)

type GetOptions struct{}

type GetTableOutput struct {
	Namespace string `table:"namespace"`
	Name      string `table:"name"`
}

var (
	cmdGet = &cobra.Command{
		Use:   "get [name]",
		Short: "get an inflatable or maybe a few",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			clientset := kubeClientset()
			inflate := inflater.New(clientset)
			listFilters := inflater.ListFilters{}
			if rootCmd.Flag("namespace").Changed {
				listFilters.Namespace = globalOpts.Namespace
			}
			if len(args) > 0 {
				listFilters.Name = args[0]
			}

			deployments, err := inflate.List(cmd.Context(), listFilters)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			switch globalOpts.Output {
			case OutputYAML:
				fmt.Println(PrettyEncode(deployments))
			case OutputTableShort, OutputTableWide:
				rows := lo.Map(deployments, func(deployment appsv1.Deployment, _ int) GetTableOutput {
					return GetTableOutput{
						Name:      deployment.Name,
						Namespace: deployment.Namespace,
					}
				})
				sort.SliceStable(rows, func(i, j int) bool {
					if strings.EqualFold(rows[i].Namespace, rows[j].Namespace) {
						return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
					}
					return strings.ToLower(rows[i].Namespace) < strings.ToLower(rows[j].Namespace)
				})
				fmt.Println(PrettyTable(rows, globalOpts.Output == OutputTableWide))
			default:
				fmt.Printf("unknown output options %s\n", globalOpts.Output)
				os.Exit(1)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(cmdGet)
}

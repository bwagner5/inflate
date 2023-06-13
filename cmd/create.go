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

	"github.com/bwagner5/inflate/pkg/inflater"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	ZonalSpread        bool
	HostnameSpread     bool
	CapacityTypeSpread bool
}

var (
	createOptions = &CreateOptions{}
	cmdCreate     = &cobra.Command{
		Use:   "create",
		Short: "create an inflatable or maybe a few",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			clientset := kubeClientset()
			deployment, err := inflater.New(clientset).Inflate(cmd.Context(), inflater.Options{
				Namespace:   globalOpts.Namespace,
				ZonalSpread: createOptions.ZonalSpread,
			})
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Printf("Created %s/%s", deployment.GetNamespace(), deployment.GetName())
		},
	}
)

func init() {
	cmdCreate.Flags().BoolVarP(&createOptions.ZonalSpread, "zonal-spread", "z", false, "add a zonal topology spread constraint")
	cmdCreate.Flags().BoolVar(&createOptions.HostnameSpread, "hostname-spread", false, "add a hostname topology spread constraint")
	cmdCreate.Flags().BoolVar(&createOptions.CapacityTypeSpread, "capacity-type-spread", false, "add a capacity-type topology spread constraint")
	rootCmd.AddCommand(cmdCreate)
}

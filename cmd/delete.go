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

type DeleteOptions struct {
	All bool
}

var (
	deleteOptions = &DeleteOptions{}
	cmdDelete     = &cobra.Command{
		Use:   "delete [name]",
		Short: "delete an inflatable or maybe a few",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if !rootCmd.Flag("namespace").Changed && len(args) == 0 && !deleteOptions.All {
				fmt.Println("must specify --namespace OR name OR --all")
				os.Exit(1)
			}
			clientset := kubeClientset()
			inflate := inflater.New(clientset)
			deleteFilters := inflater.DeleteFilters{}
			if rootCmd.Flag("namespace").Changed {
				deleteFilters.Namespace = globalOpts.Namespace
			}
			if len(args) > 0 {
				deleteFilters.Name = args[0]
			}

			err := inflate.Delete(cmd.Context(), deleteFilters)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("Successfully Deleted Inflates")
		},
	}
)

func init() {
	cmdDelete.Flags().BoolVarP(&deleteOptions.All, "all", "a", false, "delete all inflates")
	rootCmd.AddCommand(cmdDelete)
}

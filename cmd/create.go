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

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/bwagner5/inflate/pkg/inflater"
)

type CreateOptions struct {
	DryRun             bool
	RandomSuffix       bool
	Image              string
	ZonalSpread        bool
	HostnameSpread     bool
	CapacityTypeSpread bool
	HostNetwork        bool
	CPUArch            string
	OS                 string
	Service            bool
}

var (
	createOptions = &CreateOptions{}
	cmdCreate     = &cobra.Command{
		Use:   "create",
		Short: "create an inflatable or maybe a few",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			var clientset *kubernetes.Clientset
			if !createOptions.DryRun {
				clientset = kubeClientset()
			}
			inflate := inflater.New(clientset)
			options := inflater.Options{
				RandomSuffix:       createOptions.RandomSuffix,
				Namespace:          globalOpts.Namespace,
				Image:              createOptions.Image,
				ZonalSpread:        createOptions.ZonalSpread,
				HostnameSpread:     createOptions.HostnameSpread,
				CapacityTypeSpread: createOptions.CapacityTypeSpread,
				HostNetwork:        createOptions.HostNetwork,
				CPUArch:            createOptions.CPUArch,
				OS:                 createOptions.OS,
				Service:            createOptions.Service,
				DryRun:             createOptions.DryRun,
			}
			inflateCollection, err := inflate.Inflate(cmd.Context(), options)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			// Output
			if createOptions.DryRun || globalOpts.Output == OutputYAML {
				fmt.Println(PrettyEncode(inflateCollection.Deployment))
				if createOptions.Service {
					fmt.Println("---")
					fmt.Println(PrettyEncode(inflateCollection.Service))
				}
			} else {
				fmt.Printf("Created Deployment %s/%s\n", inflateCollection.Deployment.GetNamespace(), inflateCollection.Deployment.GetName())
				if createOptions.Service {
					fmt.Printf("Created Service %s/%s\n", inflateCollection.Service.GetNamespace(), inflateCollection.Service.GetName())
				}
			}
		},
	}
)

func init() {
	cmdCreate.Flags().StringVarP(&createOptions.Image, "image", "i", "public.ecr.aws/eks-distro/kubernetes/pause:3.7", "Container image to use")
	cmdCreate.Flags().BoolVarP(&createOptions.ZonalSpread, "zonal-spread", "z", false, "add a zonal topology spread constraint")
	cmdCreate.Flags().BoolVar(&createOptions.HostnameSpread, "hostname-spread", false, "add a hostname topology spread constraint")
	cmdCreate.Flags().BoolVar(&createOptions.CapacityTypeSpread, "capacity-type-spread", false, "add a capacity-type topology spread constraint")
	cmdCreate.Flags().BoolVar(&createOptions.HostNetwork, "host-network", false, "use host networking")
	cmdCreate.Flags().StringVarP(&createOptions.CPUArch, "cpu-arch", "c", "", "CPU Architecture to use for nodeSelector")
	cmdCreate.Flags().StringVar(&createOptions.OS, "os", "", "Operating System to use for nodeSelector")
	cmdCreate.Flags().BoolVar(&createOptions.RandomSuffix, "random-suffix", false, "add a random suffix to the deployment name")
	cmdCreate.Flags().BoolVar(&createOptions.Service, "service", true, "Create a K8s service")
	cmdCreate.Flags().BoolVar(&createOptions.DryRun, "dry-run", false, "Dry-run prints the K8s manifests without applying")
	rootCmd.AddCommand(cmdCreate)
}

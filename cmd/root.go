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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var version = "dev"

type globalOptions struct {
	kubeconfig string
	context    string
	namespace  string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := newRootCommand().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	global := &globalOptions{}
	cmd := &cobra.Command{
		Use:           "inflate",
		Short:         "Create Kubernetes workloads for scheduling and autoscaling demos",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.PersistentFlags().StringVar(&global.kubeconfig, "kubeconfig", "", "path to a kubeconfig (uses standard loading rules by default)")
	cmd.PersistentFlags().StringVar(&global.context, "context", "", "kubeconfig context to use")
	cmd.PersistentFlags().StringVarP(&global.namespace, "namespace", "n", "inflate", "Kubernetes namespace")

	client := func() (kubernetes.Interface, error) {
		return newKubeClient(global)
	}
	cmd.AddCommand(
		newDemoCommand(global, client),
		newCreateCommand(global, client),
		newGetCommand(global, client),
		newDeleteCommand(global, client),
	)
	return cmd
}

func newKubeClient(opts *globalOptions) (kubernetes.Interface, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.kubeconfig != "" {
		rules.ExplicitPath = opts.kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: opts.context}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}
	return client, nil
}

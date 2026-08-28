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

package inflater

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDeploymentDemoAntiAffinity(t *testing.T) {
	t.Parallel()

	deployment, err := Deployment(Options{
		Name:       "inflate-demo-1",
		Namespace:  "inflate",
		Image:      "registry.k8s.io/pause:3.10.2",
		CPU:        "750m",
		Memory:     "512Mi",
		Replicas:   3,
		DemoID:     "demo-1",
		OnePerNode: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := *deployment.Spec.Replicas; got != 3 {
		t.Fatalf("replicas = %d, want 3", got)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if got := container.Resources.Requests.Cpu(); got.Cmp(resource.MustParse("750m")) != 0 {
		t.Fatalf("cpu = %s, want 750m", got.String())
	}
	if got := container.Resources.Requests.Memory(); got.Cmp(resource.MustParse("512Mi")) != 0 {
		t.Fatalf("memory = %s, want 512Mi", got.String())
	}
	terms := deployment.Spec.Template.Spec.Affinity.PodAntiAffinity.
		RequiredDuringSchedulingIgnoredDuringExecution
	if len(terms) != 1 || terms[0].TopologyKey != corev1.LabelHostname {
		t.Fatalf("unexpected anti-affinity terms: %#v", terms)
	}
	if got := terms[0].LabelSelector.MatchLabels[LabelDemo]; got != "demo-1" {
		t.Fatalf("anti-affinity demo label = %q, want demo-1", got)
	}
}

func TestApplyUpdatesExistingDeployment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := fake.NewClientset()
	inflate := New(client)
	opts := Options{
		Name: "inflate", Namespace: "inflate", Image: "registry.k8s.io/pause:3.10.2",
		CPU: "250m", Memory: "256Mi", Replicas: 1,
	}
	if _, err := inflate.Apply(ctx, opts); err != nil {
		t.Fatal(err)
	}
	opts.Replicas = 4
	opts.CPU = "1"
	updated, err := inflate.Apply(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := *updated.Spec.Replicas; got != 4 {
		t.Fatalf("replicas = %d, want 4", got)
	}
	if got := updated.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu(); got.Cmp(resource.MustParse("1")) != 0 {
		t.Fatalf("cpu = %s, want 1", got.String())
	}
}

func TestDeploymentRejectsInvalidResources(t *testing.T) {
	t.Parallel()

	_, err := Deployment(Options{
		Name: "inflate", Namespace: "inflate", Image: "pause", CPU: "nope",
		Memory: "256Mi", Replicas: 1,
	})
	if err == nil {
		t.Fatal("expected an invalid CPU error")
	}
}

func TestEnsureNamespacePreservesExistingNamespace(t *testing.T) {
	t.Parallel()

	client := fake.NewClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "inflate", Labels: map[string]string{"owner": "user"}},
	})
	if err := New(client).EnsureNamespace(context.Background(), "inflate"); err != nil {
		t.Fatal(err)
	}
	namespace, err := client.CoreV1().Namespaces().Get(
		context.Background(), "inflate", metav1.GetOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := namespace.Labels["owner"]; got != "user" {
		t.Fatalf("owner label = %q, want user", got)
	}
}

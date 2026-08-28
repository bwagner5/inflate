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

// Package inflater creates and manages deliberately schedulable Kubernetes workloads.
package inflater

import (
	"context"
	"fmt"
	"maps"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// Labels placed on all inflate resources.
const (
	LabelName      = "app.kubernetes.io/name"
	LabelInstance  = "app.kubernetes.io/instance"
	LabelManagedBy = "app.kubernetes.io/managed-by"
	LabelDemo      = "inflate.bwagner.dev/demo"

	legacyLabelApp       = "app"
	legacyLabelManagedBy = "managed-by"
	managedByValue       = "inflate"
)

// Options describes one inflate Deployment.
type Options struct {
	Name               string
	Namespace          string
	Image              string
	CPU                string
	Memory             string
	Replicas           int32
	DemoID             string
	OnePerNode         bool
	HostnameSpread     bool
	ZonalSpread        bool
	CapacityTypeSpread bool
	HostNetwork        bool
	CPUArch            string
	OS                 string
}

// Inflater manages inflate Deployments through a Kubernetes client.
type Inflater struct {
	client kubernetes.Interface
}

// New creates an Inflater.
func New(client kubernetes.Interface) *Inflater {
	return &Inflater{client: client}
}

// EnsureNamespace creates the target namespace when it does not already exist.
func (i *Inflater) EnsureNamespace(ctx context.Context, namespace string) error {
	if _, err := i.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{}); err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return err
	}

	_, err := i.client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				LabelManagedBy: managedByValue,
			},
		},
	}, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// Deployment builds a Deployment without contacting Kubernetes.
func Deployment(opts Options) (*appsv1.Deployment, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("name must not be empty")
	}
	if opts.Namespace == "" {
		return nil, fmt.Errorf("namespace must not be empty")
	}
	if opts.Image == "" {
		return nil, fmt.Errorf("image must not be empty")
	}
	if opts.Replicas < 1 {
		return nil, fmt.Errorf("replicas must be at least 1")
	}

	cpu, err := resource.ParseQuantity(opts.CPU)
	if err != nil || cpu.Sign() <= 0 {
		return nil, fmt.Errorf("invalid CPU request %q", opts.CPU)
	}
	memory, err := resource.ParseQuantity(opts.Memory)
	if err != nil || memory.Sign() <= 0 {
		return nil, fmt.Errorf("invalid memory request %q", opts.Memory)
	}

	selectorLabels := map[string]string{
		legacyLabelApp:       opts.Name,
		legacyLabelManagedBy: managedByValue,
	}
	podLabels := maps.Clone(selectorLabels)
	podLabels[LabelName] = "inflate"
	podLabels[LabelInstance] = opts.Name
	podLabels[LabelManagedBy] = managedByValue
	if opts.DemoID != "" {
		podLabels[LabelDemo] = opts.DemoID
	}

	podSpec := corev1.PodSpec{
		HostNetwork:                   opts.HostNetwork,
		TerminationGracePeriodSeconds: new(int64),
		Containers: []corev1.Container{{
			Name:  "inflate",
			Image: opts.Image,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    cpu,
					corev1.ResourceMemory: memory,
				},
			},
		}},
		NodeSelector:              nodeSelector(opts),
		TopologySpreadConstraints: topologySpreadConstraints(opts, selectorLabels),
	}
	if opts.OnePerNode {
		antiAffinityLabels := selectorLabels
		if opts.DemoID != "" {
			antiAffinityLabels = map[string]string{
				legacyLabelManagedBy: managedByValue,
				LabelDemo:            opts.DemoID,
			}
		}
		podSpec.Affinity = &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{
					LabelSelector: &metav1.LabelSelector{MatchLabels: antiAffinityLabels},
					TopologyKey:   corev1.LabelHostname,
				}},
			},
		}
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
			Labels:    podLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: new(opts.Replicas),
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
				Spec:       podSpec,
			},
		},
	}, nil
}

// Apply creates a Deployment or updates its mutable fields when it already exists.
func (i *Inflater) Apply(ctx context.Context, opts Options) (*appsv1.Deployment, error) {
	desired, err := Deployment(opts)
	if err != nil {
		return nil, err
	}

	current, err := i.client.AppsV1().Deployments(opts.Namespace).Get(
		ctx, opts.Name, metav1.GetOptions{},
	)
	if apierrors.IsNotFound(err) {
		return i.client.AppsV1().Deployments(opts.Namespace).Create(
			ctx, desired, metav1.CreateOptions{},
		)
	}
	if err != nil {
		return nil, err
	}

	// A Deployment selector is immutable. Return a useful error rather than sending
	// an update that the API server will reject if this name belongs to something else.
	if !labels.Equals(labels.Set(current.Spec.Selector.MatchLabels), labels.Set(desired.Spec.Selector.MatchLabels)) {
		return nil, fmt.Errorf("deployment %s/%s exists with a different selector", opts.Namespace, opts.Name)
	}
	current.Labels = desired.Labels
	current.Spec.Replicas = desired.Spec.Replicas
	current.Spec.Template = desired.Spec.Template
	return i.client.AppsV1().Deployments(opts.Namespace).Update(
		ctx, current, metav1.UpdateOptions{},
	)
}

// List returns inflate-managed Deployments in one namespace.
func (i *Inflater) List(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	list, err := i.client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set{legacyLabelManagedBy: managedByValue}.String(),
	})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// Get returns one Deployment by name.
func (i *Inflater) Get(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	return i.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

// Delete removes one Deployment by name.
func (i *Inflater) Delete(ctx context.Context, namespace, name string) error {
	err := i.client.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// DeleteAll removes all inflate-managed Deployments in one namespace.
func (i *Inflater) DeleteAll(ctx context.Context, namespace string) error {
	return i.deleteCollection(ctx, namespace, labels.Set{legacyLabelManagedBy: managedByValue}.String())
}

// DeleteDemo removes all Deployments belonging to one demo run.
func (i *Inflater) DeleteDemo(ctx context.Context, namespace, demoID string) error {
	return i.deleteCollection(ctx, namespace, labels.Set{
		legacyLabelManagedBy: managedByValue,
		LabelDemo:            demoID,
	}.String())
}

func (i *Inflater) deleteCollection(ctx context.Context, namespace, selector string) error {
	return i.client.AppsV1().Deployments(namespace).DeleteCollection(
		ctx,
		metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: selector},
	)
}

func nodeSelector(opts Options) map[string]string {
	selector := map[string]string{}
	if opts.CPUArch != "" {
		selector[corev1.LabelArchStable] = opts.CPUArch
	}
	if opts.OS != "" {
		selector[corev1.LabelOSStable] = opts.OS
	}
	if len(selector) == 0 {
		return nil
	}
	return selector
}

func topologySpreadConstraints(
	opts Options,
	matchLabels map[string]string,
) []corev1.TopologySpreadConstraint {
	var constraints []corev1.TopologySpreadConstraint
	add := func(enabled bool, topologyKey string) {
		if !enabled {
			return
		}
		constraints = append(constraints, corev1.TopologySpreadConstraint{
			MaxSkew:           1,
			TopologyKey:       topologyKey,
			WhenUnsatisfiable: corev1.DoNotSchedule,
			LabelSelector:     &metav1.LabelSelector{MatchLabels: matchLabels},
		})
	}
	add(opts.HostnameSpread, corev1.LabelHostname)
	add(opts.ZonalSpread, corev1.LabelTopologyZone)
	add(opts.CapacityTypeSpread, "karpenter.sh/capacity-type")
	return constraints
}

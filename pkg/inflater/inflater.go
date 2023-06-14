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
	"fmt"
	"math/rand"

	"github.com/imdario/mergo"
	"github.com/samber/lo"
	"go.uber.org/multierr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	DefaultOptions = Options{
		Namespace:   "inflate",
		ZonalSpread: false,
	}
)

type Options struct {
	RandomSuffix       bool
	Namespace          string
	Image              string
	ZonalSpread        bool
	HostnameSpread     bool
	CapacityTypeSpread bool
	HostNetwork        bool
	CPUArch            string
	OS                 string
}

type Inflater struct {
	clientset *kubernetes.Clientset
}

func New(clientset *kubernetes.Clientset) *Inflater {
	return &Inflater{
		clientset: clientset,
	}
}

func GetDefaultOptions() Options {
	return Options{
		Namespace:   "inflate",
		ZonalSpread: false,
	}
}

func (i Inflater) CreateNamespace(ctx context.Context, namespace string) error {
	_, err := i.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"managed-by": "inflate",
			},
		},
	}, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (i Inflater) GetInflateDeployment(_ context.Context, opts Options) (*appsv1.Deployment, error) {
	opts, err := mergeOptions(opts)
	if err != nil {
		return nil, err
	}
	appName := "inflate"
	if opts.RandomSuffix {
		//nolint:gosec
		appName += fmt.Sprintf("-%d", rand.Intn(9_999_999_999))
	}
	return &appsv1.Deployment{
		ObjectMeta: i.objectMeta(opts.Namespace, appName),
		Spec: appsv1.DeploymentSpec{
			Replicas: lo.ToPtr(int32(0)),
			Selector: &metav1.LabelSelector{
				MatchLabels: i.defaultLabels(appName),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: i.defaultLabels(appName),
				},
				Spec: corev1.PodSpec{
					HostNetwork:                   opts.HostNetwork,
					TerminationGracePeriodSeconds: lo.ToPtr(int64(0)),
					Containers: []corev1.Container{
						{
							Name:  appName,
							Image: opts.Image,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewQuantity(256, resource.BinarySI),
								},
							},
						},
					},
					TopologySpreadConstraints: i.topologySpread(opts, i.defaultLabels(appName)),
					NodeSelector:              i.nodeSelector(opts),
				},
			},
		},
	}, nil
}

func (i Inflater) Inflate(ctx context.Context, opts Options) (*appsv1.Deployment, error) {
	opts, err := mergeOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := i.CreateNamespace(ctx, opts.Namespace); err != nil {
		return nil, err
	}
	deployment, err := i.GetInflateDeployment(ctx, opts)
	if err != nil {
		return nil, err
	}
	deploymentFromAPI, err := i.clientset.AppsV1().Deployments(opts.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			deploymentFromAPI, err = i.clientset.AppsV1().Deployments(opts.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
			return deploymentFromAPI, err
		}
	}
	return deploymentFromAPI, err
}

type ListFilters struct {
	Namespace string
	Name      string
}

func (i Inflater) List(ctx context.Context, filters ListFilters) ([]appsv1.Deployment, error) {
	var namespaces []string
	var deployments []appsv1.Deployment
	if filters.Namespace != "" {
		namespaces = append(namespaces, filters.Namespace)
	} else {
		namespaceList, err := i.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
			LabelSelector: "managed-by=inflate",
		})
		if err != nil {
			return nil, err
		}
		namespaces = lo.Map(namespaceList.Items, func(ns corev1.Namespace, _ int) string { return ns.Name })
	}
	if filters.Name != "" {
		for _, ns := range namespaces {
			deployment, err := i.clientset.AppsV1().Deployments(ns).Get(ctx, filters.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			deployments = append(deployments, *deployment)
		}
		return deployments, nil
	}

	var errs error
	for _, ns := range namespaces {
		deploymentList, err := i.clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{
			LabelSelector: "managed-by=inflate",
		})
		errs = multierr.Append(errs, err)
		deployments = append(deployments, deploymentList.Items...)
	}
	return deployments, errs
}

type DeleteFilters struct {
	Namespace string
	Name      string
}

func (i Inflater) Delete(ctx context.Context, filters DeleteFilters) error {
	var namespaces []string
	if filters.Namespace != "" {
		namespaces = append(namespaces, filters.Namespace)
	} else {
		namespaceList, err := i.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
			LabelSelector: "managed-by=inflate",
		})
		if err != nil {
			return err
		}
		namespaces = lo.Map(namespaceList.Items, func(ns corev1.Namespace, _ int) string { return ns.Name })
	}
	if filters.Name != "" {
		for _, ns := range namespaces {
			if err := i.clientset.AppsV1().Deployments(ns).Delete(ctx, filters.Name, metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
		return nil
	}

	var errs error
	for _, ns := range namespaces {
		if err := i.clientset.AppsV1().Deployments(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: "managed-by=inflate",
		}); err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return errs
}

func (i Inflater) nodeSelector(opts Options) map[string]string {
	nodeSelector := map[string]string{}
	if opts.CPUArch != "" {
		nodeSelector["kubernetes.io/arch"] = opts.CPUArch
	}
	if opts.OS != "" {
		nodeSelector["kubernetes.io/os"] = opts.OS
	}
	return lo.Ternary(len(nodeSelector) == 0, nil, nodeSelector)
}

func (i Inflater) topologySpread(opts Options, matchLabels map[string]string) []corev1.TopologySpreadConstraint {
	var topologySpreadConstraints []corev1.TopologySpreadConstraint
	if opts.ZonalSpread {
		topologySpreadConstraints = append(topologySpreadConstraints, corev1.TopologySpreadConstraint{
			MaxSkew:           int32(1),
			TopologyKey:       corev1.LabelTopologyZone,
			WhenUnsatisfiable: "DoNotSchedule",
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		})
	}
	if opts.HostnameSpread {
		topologySpreadConstraints = append(topologySpreadConstraints, corev1.TopologySpreadConstraint{
			MaxSkew:           int32(1),
			TopologyKey:       corev1.LabelHostname,
			WhenUnsatisfiable: "DoNotSchedule",
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		})
	}
	if opts.CapacityTypeSpread {
		topologySpreadConstraints = append(topologySpreadConstraints, corev1.TopologySpreadConstraint{
			MaxSkew:           int32(1),
			TopologyKey:       "karpenter.sh/capacity-type",
			WhenUnsatisfiable: "DoNotSchedule",
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		})
	}
	return topologySpreadConstraints
}

func (i Inflater) objectMeta(namespace string, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    i.defaultLabels(name),
	}
}

func (i Inflater) defaultLabels(app string) map[string]string {
	return map[string]string{
		"app":        app,
		"managed-by": "inflate",
	}
}

func mergeOptions(opts Options) (Options, error) {
	options := GetDefaultOptions()
	if err := mergo.MergeWithOverwrite(&options, opts); err != nil {
		return options, err
	}
	return options, nil
}

package inflater

import (
	"context"

	"github.com/imdario/mergo"
	"github.com/samber/lo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	DefaultOptions = Options{
		Namespace:   "inflate",
		ZonalSpread: false,
	}
)

type Options struct {
	Namespace   string
	ZonalSpread bool
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
		ObjectMeta: v1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	return err
}

func mergeOptions(opts Options) (Options, error) {
	options := GetDefaultOptions()
	if err := mergo.MergeWithOverwrite(&options, opts); err != nil {
		return options, err
	}
	return options, nil
}

func (i Inflater) Inflate(ctx context.Context, opts Options) (*appsv1.Deployment, error) {
	opts, err := mergeOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := i.CreateNamespace(ctx, opts.Namespace); err != nil {
		return nil, err
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      "inflate",
			Namespace: "inflate",
			Labels: map[string]string{
				"app": "inflate",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: lo.ToPtr(int32(0)),
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "inflate",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"app": "inflate",
					},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: lo.ToPtr(int64(0)),
					Containers: []corev1.Container{
						{
							Name:  "inflate",
							Image: "public.ecr.aws/eks-distro/kubernetes/pause:3.7",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewQuantity(256, resource.BinarySI),
								},
							},
						},
					},
				},
			},
		},
	}
	return i.clientset.AppsV1().Deployments("inflate").Create(ctx, deployment, metav1.CreateOptions{})
}

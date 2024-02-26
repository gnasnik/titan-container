package builder

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

var whiteRegistry = "registry.cn-hongkong.aliyuncs.com"

type Deployment interface {
	workloadBase
	Create() (*appsv1.Deployment, error)
	Update(obj *appsv1.Deployment) (*appsv1.Deployment, error)
}

type deployment struct {
	Workload
}

var _ Deployment = (*deployment)(nil)

func NewDeployment(workload Workload) Deployment {
	d := &deployment{
		Workload: workload,
	}
	return d
}

func (b *deployment) Create() (*appsv1.Deployment, error) { // nolint:golint,unparam
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: b.Name(),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: b.labels(),
			},
			Replicas: b.replicas(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: b.labels(),
				},
				Spec: corev1.PodSpec{
					Containers:       []corev1.Container{b.container()},
					ImagePullSecrets: b.imagePullSecrets(),
					NodeSelector:     map[string]string{titanNodeSelector: b.osType()},
					Tolerations:      b.tolerations(),
				},
			},
		},
	}

	if strings.Contains(b.container().Image, whiteRegistry) {
		deployment.Spec.Template.Spec.ImagePullSecrets = append(deployment.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: whiteRegistry})
	}

	return deployment, nil
}

func (b *deployment) Update(obj *appsv1.Deployment) (*appsv1.Deployment, error) { // nolint:golint,unparam
	obj.Labels = b.labels()
	obj.Spec.Selector.MatchLabels = b.labels()
	obj.Spec.Replicas = b.replicas()
	obj.Spec.Template.Labels = b.labels()
	obj.Spec.Template.Spec.Containers = []corev1.Container{b.container()}
	obj.Spec.Template.Spec.ImagePullSecrets = b.imagePullSecrets()
	obj.Spec.Template.Spec.NodeSelector = map[string]string{titanNodeSelector: b.osType()}

	var exist bool

	for _, item := range obj.Spec.Template.Spec.ImagePullSecrets {
		if item.Name == whiteRegistry {
			exist = true
		}
	}

	if !exist {
		if strings.Contains(b.container().Image, whiteRegistry) {
			obj.Spec.Template.Spec.ImagePullSecrets = append(obj.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: whiteRegistry})
		}

	}

	return obj, nil
}

package kube

// nolint:deadcode,golint

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"os"

	"github.com/Filecoin-Titan/titan-container/node/impl/provider/kube/builder"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func applyNS(ctx context.Context, kc kubernetes.Interface, b builder.NS) error {
	obj, err := kc.CoreV1().Namespaces().Get(ctx, b.Name(), metav1.GetOptions{})

	switch {
	case err == nil:
		obj, err = b.Update(obj)
		if err == nil {
			_, err = kc.CoreV1().Namespaces().Update(ctx, obj, metav1.UpdateOptions{})
		}
	case errors.IsNotFound(err):
		obj, err = b.Create()
		if err == nil {
			_, err = kc.CoreV1().Namespaces().Create(ctx, obj, metav1.CreateOptions{})
		}
	}
	return err
}

// Apply list of Network Policies
func applyNetPolicies(ctx context.Context, kc kubernetes.Interface, b builder.NetPol) error {
	var err error

	policies, err := b.Create()
	if err != nil {
		return err
	}

	for _, pol := range policies {
		obj, err := kc.NetworkingV1().NetworkPolicies(b.NS()).Get(ctx, pol.Name, metav1.GetOptions{})

		switch {
		case err == nil:
			_, err = b.Update(obj)
			if err == nil {
				_, err = kc.NetworkingV1().NetworkPolicies(b.NS()).Update(ctx, pol, metav1.UpdateOptions{})
			}
		case errors.IsNotFound(err):
			_, err = kc.NetworkingV1().NetworkPolicies(b.NS()).Create(ctx, pol, metav1.CreateOptions{})
		}
		if err != nil {
			break
		}
	}

	return err
}

func applyDeployment(ctx context.Context, kc kubernetes.Interface, b builder.Deployment) error {
	obj, err := kc.AppsV1().Deployments(b.NS()).Get(ctx, b.Name(), metav1.GetOptions{})

	switch {
	case err == nil:
		obj, err = b.Update(obj)

		if err == nil {
			_, err = kc.AppsV1().Deployments(b.NS()).Update(ctx, obj, metav1.UpdateOptions{})

		}
	case errors.IsNotFound(err):
		obj, err = b.Create()
		if err == nil {
			_, err = kc.AppsV1().Deployments(b.NS()).Create(ctx, obj, metav1.CreateOptions{})
		}
	}
	return err
}

func applyStatefulSet(ctx context.Context, kc kubernetes.Interface, b builder.StatefulSet) error {
	obj, err := kc.AppsV1().StatefulSets(b.NS()).Get(ctx, b.Name(), metav1.GetOptions{})

	switch {
	case err == nil:
		obj, err = b.Update(obj)

		if err == nil {
			_, err = kc.AppsV1().StatefulSets(b.NS()).Update(ctx, obj, metav1.UpdateOptions{})

		}
	case errors.IsNotFound(err):
		obj, err = b.Create()
		if err == nil {
			_, err = kc.AppsV1().StatefulSets(b.NS()).Create(ctx, obj, metav1.CreateOptions{})
		}
	}
	return err
}

func applyService(ctx context.Context, kc kubernetes.Interface, b builder.Service) error {
	obj, err := kc.CoreV1().Services(b.NS()).Get(ctx, b.Name(), metav1.GetOptions{})

	switch {
	case err == nil:
		obj, err = b.Update(obj)
		if err == nil {
			_, err = kc.CoreV1().Services(b.NS()).Update(ctx, obj, metav1.UpdateOptions{})
		}
	case errors.IsNotFound(err):
		obj, err = b.Create()
		if err == nil {
			_, err = kc.CoreV1().Services(b.NS()).Create(ctx, obj, metav1.CreateOptions{})
		}
	}
	return err
}

func applyIngress(ctx context.Context, kc kubernetes.Interface, b builder.Ingress) error {
	obj, err := kc.NetworkingV1().Ingresses(b.NS()).Get(ctx, b.Name(), metav1.GetOptions{})

	switch {
	case err == nil:
		obj, err = b.Update(obj)
		if err == nil {
			_, err = kc.NetworkingV1().Ingresses(b.NS()).Update(ctx, obj, metav1.UpdateOptions{})
		}
	case errors.IsNotFound(err):
		obj, err = b.Create()
		if err == nil {
			_, err = kc.NetworkingV1().Ingresses(b.NS()).Create(ctx, obj, metav1.CreateOptions{})
		}
	}
	return err
}

func getOrCreateSecretFromHostname(ctx context.Context, kc kubernetes.Interface, ns, hostname, certificate, certificateKey string) (*corev1.Secret, error) {
	secret, err := kc.CoreV1().Secrets(ns).Get(ctx, hostname, metav1.GetOptions{})
	if err == nil {
		return secret, nil
	}

	certFile, err := os.ReadFile(certificate)
	if err != nil {
		return nil, err
	}

	certKeyFile, err := os.ReadFile(certificateKey)
	if err != nil {
		return nil, err
	}

	return kc.CoreV1().Secrets(ns).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: hostname,
		},
		Data: map[string][]byte{
			"cert": certFile,
			"key":  certKeyFile,
		},
		Type: corev1.SecretTypeOpaque,
	}, metav1.CreateOptions{})
}

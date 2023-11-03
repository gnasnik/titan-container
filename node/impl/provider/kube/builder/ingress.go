package builder

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Ingress interface {
	workloadBase
	Create() (*netv1.Ingress, error)
	Update(obj *netv1.Ingress) (*netv1.Ingress, error)
}

type ingress struct {
	Workload
	directive *HostnameDirective
}

var _ Ingress = (*ingress)(nil)

func BuildIngress(workload Workload, directive *HostnameDirective) Ingress {
	return &ingress{
		Workload:  workload,
		directive: directive,
	}
}

func (b *ingress) Create() (*netv1.Ingress, error) {
	directive := b.directive

	ingressClassName := directive.IngressName
	obj := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        directive.Hostname,
			Labels:      b.labels(),
			Annotations: kubeNginxIngressAnnotations(directive),
		},
		Spec: netv1.IngressSpec{
			Rules: ingressRules(directive.Hostname, directive.ServiceName, directive.ServicePort),
		},
	}

	if len(ingressClassName) > 0 {
		obj.Spec.IngressClassName = &ingressClassName
	}

	if directive.UseCaddyIngress {
		obj.ObjectMeta.Annotations = cadyyIngressAnnotations(directive)
	}

	return obj, nil
}

func (b *ingress) Update(obj *netv1.Ingress) (*netv1.Ingress, error) {
	directive := b.directive
	rules := ingressRules(directive.Hostname, directive.ServiceName, directive.ServicePort)

	obj.ObjectMeta.Labels = b.labels()
	obj.Spec.Rules = rules
	return obj, nil
}

func (b *ingress) Name() string {
	return b.directive.Hostname
}

func cadyyIngressAnnotations(directive *HostnameDirective) map[string]string {
	const root = "nginx.ingress.kubernetes.io"

	result := kubeNginxIngressAnnotations(directive)
	result[fmt.Sprintf("%s/enable-cors", root)] = "true"
	result["kubernetes.io/ingress.class"] = "caddy"

	return result
}

func kubeNginxIngressAnnotations(directive *HostnameDirective) map[string]string {
	// For kubernetes/ingress-nginx
	// https://github.com/kubernetes/ingress-nginx
	const root = "nginx.ingress.kubernetes.io"

	readTimeout := math.Ceil(float64(directive.ReadTimeout) / 1000.0)
	sendTimeout := math.Ceil(float64(directive.SendTimeout) / 1000.0)
	result := map[string]string{
		fmt.Sprintf("%s/proxy-read-timeout", root): fmt.Sprintf("%d", int(readTimeout)),
		fmt.Sprintf("%s/proxy-send-timeout", root): fmt.Sprintf("%d", int(sendTimeout)),

		fmt.Sprintf("%s/proxy-next-upstream-tries", root): strconv.Itoa(int(directive.NextTries)),
		fmt.Sprintf("%s/proxy-body-size", root):           strconv.Itoa(int(directive.MaxBodySize)),
	}

	nextTimeoutKey := fmt.Sprintf("%s/proxy-next-upstream-timeout", root)
	nextTimeout := 0 // default magic value for disable
	if directive.NextTimeout > 0 {
		nextTimeout = int(math.Ceil(float64(directive.NextTimeout) / 1000.0))
	}

	result[nextTimeoutKey] = fmt.Sprintf("%d", nextTimeout)

	strBuilder := strings.Builder{}

	for i, v := range directive.NextCases {
		first := string(v[0])
		isHTTPCode := strings.ContainsAny(first, "12345")

		if isHTTPCode {
			strBuilder.WriteString("http_")
		}
		strBuilder.WriteString(v)

		if i != len(directive.NextCases)-1 {
			// The actual separator is the space character for kubernetes/ingress-nginx
			strBuilder.WriteRune(' ')
		}
	}

	result[fmt.Sprintf("%s/proxy-next-upstream", root)] = strBuilder.String()
	return result
}

func ingressRules(hostname string, kubeServiceName string, kubeServicePort int32) []netv1.IngressRule {
	// for some reason we need to pass a pointer to this
	pathTypeForAll := netv1.PathTypePrefix
	ruleValue := netv1.HTTPIngressRuleValue{
		Paths: []netv1.HTTPIngressPath{{
			Path:     "/",
			PathType: &pathTypeForAll,
			Backend: netv1.IngressBackend{
				Service: &netv1.IngressServiceBackend{
					Name: kubeServiceName,
					Port: netv1.ServiceBackendPort{
						Number: kubeServicePort,
					},
				},
			},
		}},
	}

	return []netv1.IngressRule{{
		Host:             hostname,
		IngressRuleValue: netv1.IngressRuleValue{HTTP: &ruleValue},
	}}
}

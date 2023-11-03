package builder

import (
	"fmt"
	"strings"

	"github.com/Filecoin-Titan/titan-container/node/impl/provider/kube/manifest"
)

type HostnameDirective struct {
	IngressName     string
	Hostname        string
	ServiceName     string
	ServicePort     int32
	ReadTimeout     uint32
	SendTimeout     uint32
	NextTimeout     uint32
	MaxBodySize     uint32
	NextTries       uint32
	NextCases       []string
	UseCaddyIngress bool
}

func BuildHostNameDirective(ns, hostName, serviceName, ingressName string, serviceExpose *manifest.ServiceExpose, useCaddyIngress bool) *HostnameDirective {
	// Build the directive based off the event
	directive := &HostnameDirective{
		Hostname:        NewHostName(ns, hostName),
		ServiceName:     serviceName,
		ServicePort:     int32(serviceExpose.ExternalPort),
		IngressName:     ingressName,
		UseCaddyIngress: useCaddyIngress,
	}
	/*
		Populate the configuration options
		selectedExpose.HttpOptions has zero values if this is from an earlier CRD. Just insert
		defaults and move on
	*/
	if serviceExpose.HTTPOptions.MaxBodySize == 0 {
		directive.ReadTimeout = 60000
		directive.SendTimeout = 60000
		directive.NextTimeout = 60000
		directive.MaxBodySize = 1048576
		directive.NextTries = 3
		directive.NextCases = []string{"error", "timeout"}
	} else {
		directive.ReadTimeout = serviceExpose.HTTPOptions.ReadTimeout
		directive.SendTimeout = serviceExpose.HTTPOptions.SendTimeout
		directive.NextTimeout = serviceExpose.HTTPOptions.NextTimeout
		directive.MaxBodySize = serviceExpose.HTTPOptions.MaxBodySize
		directive.NextTries = serviceExpose.HTTPOptions.NextTries
		directive.NextCases = serviceExpose.HTTPOptions.NextCases
	}

	return directive
}

func NewHostName(ns string, hostName string) string {
	hostNamePrefix := strings.Replace(ns, "-", "", -1)
	return fmt.Sprintf("%s.%s", hostNamePrefix, hostName)
}

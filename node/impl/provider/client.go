package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"

	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/Filecoin-Titan/titan-container/node/config"
	"github.com/Filecoin-Titan/titan-container/node/impl/provider/kube"
	"github.com/Filecoin-Titan/titan-container/node/impl/provider/kube/builder"
	"github.com/Filecoin-Titan/titan-container/node/impl/provider/kube/manifest"
	logging "github.com/ipfs/go-log/v2"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/remotecommand"
)

var log = logging.Logger("provider")

type Client interface {
	GetStatistics(ctx context.Context) (*types.ResourcesStatistics, error)
	CreateDeployment(ctx context.Context, deployment *types.Deployment) error
	UpdateDeployment(ctx context.Context, deployment *types.Deployment) error
	CloseDeployment(ctx context.Context, deployment *types.Deployment) error
	GetDeployment(ctx context.Context, id types.DeploymentID) (*types.Deployment, error)
	GetLogs(ctx context.Context, id types.DeploymentID) ([]*types.ServiceLog, error)
	GetEvents(ctx context.Context, id types.DeploymentID) ([]*types.ServiceEvent, error)
	GetDomains(ctx context.Context, id types.DeploymentID) ([]*types.DeploymentDomain, error)
	AddDomain(ctx context.Context, id types.DeploymentID, hostname string) error
	DeleteDomain(ctx context.Context, id types.DeploymentID, index int64) error
	ImportCertificate(ctx context.Context, id types.DeploymentID, cert *types.Certificate) error
	Exec(ctx context.Context, id types.DeploymentID, podIndex int, stdin io.Reader, stdout, stderr io.Writer, cmd []string, tty bool,
		terminalSizeQueue remotecommand.TerminalSizeQueue) (types.ExecResult, error)
}

type client struct {
	kc          kube.Client
	providerCfg *config.ProviderCfg
}

var _ Client = (*client)(nil)

func NewClient(config *config.ProviderCfg) (Client, error) {
	kubeClient, err := kube.NewClient(config.KubeConfigPath, config)
	if err != nil {
		return nil, err
	}
	return &client{kc: kubeClient, providerCfg: config}, nil
}

func (c *client) GetStatistics(ctx context.Context) (*types.ResourcesStatistics, error) {
	nodeResources, err := c.kc.FetchNodeResources(ctx)
	if err != nil {
		return nil, err
	}

	if nodeResources == nil {
		return nil, fmt.Errorf("nodes resources do not exist")
	}

	statistics := &types.ResourcesStatistics{}
	for _, node := range nodeResources {
		statistics.CPUCores.MaxCPUCores += node.CPU.Capacity.AsApproximateFloat64()
		statistics.CPUCores.Available += node.CPU.Allocatable.AsApproximateFloat64()
		statistics.CPUCores.Active += node.CPU.Allocated.AsApproximateFloat64()

		statistics.Memory.MaxMemory += uint64(node.Memory.Capacity.AsApproximateFloat64())
		statistics.Memory.Available += uint64(node.Memory.Allocatable.AsApproximateFloat64())
		statistics.Memory.Active += uint64(node.Memory.Allocated.AsApproximateFloat64())

		statistics.Storage.MaxStorage += uint64(node.EphemeralStorage.Capacity.AsApproximateFloat64())
		statistics.Storage.Available += uint64(node.EphemeralStorage.Allocatable.AsApproximateFloat64())
		statistics.Storage.Active += uint64(node.EphemeralStorage.Allocated.AsApproximateFloat64())
	}

	statistics.CPUCores.Available = statistics.CPUCores.Available - statistics.CPUCores.Active
	statistics.Memory.Available = statistics.Memory.Available - statistics.Memory.Active
	statistics.Storage.Available = statistics.Storage.Available - statistics.Storage.Active

	return statistics, nil
}

func (c *client) CreateDeployment(ctx context.Context, deployment *types.Deployment) error {
	if deployment.Authority {
		deployment.ProviderExposeIP = c.providerCfg.ExposeIP
	}

	cDeployment, err := ClusterDeploymentFromDeployment(deployment)
	if err != nil {
		log.Errorf("CreateDeployment %s", err.Error())
		return err
	}

	did := cDeployment.DeploymentID()
	ns := builder.DidNS(did)

	if isExist, err := c.isExistDeploymentOrStatefulSet(ctx, ns); err != nil {
		return err
	} else if isExist {
		return fmt.Errorf("deployment %s already exist", deployment.ID)
	}

	ctx = context.WithValue(ctx, builder.SettingsKey, builder.NewDefaultSettings())
	return c.kc.Deploy(ctx, cDeployment)
}

func (c *client) UpdateDeployment(ctx context.Context, deployment *types.Deployment) error {
	if deployment.Authority {
		deployment.ProviderExposeIP = c.providerCfg.ExposeIP
	}

	k8sDeployment, err := ClusterDeploymentFromDeployment(deployment)
	if err != nil {
		log.Errorf("UpdateDeployment %s", err.Error())
		return err
	}

	did := k8sDeployment.DeploymentID()
	ns := builder.DidNS(did)

	if isExist, err := c.isExistDeploymentOrStatefulSet(ctx, ns); err != nil {
		return err
	} else if !isExist {
		return fmt.Errorf("deployment %s do not exist", deployment.ID)
	}

	group := k8sDeployment.ManifestGroup()
	for _, service := range group.Services {
		// check service if exist
		if c.isPersistent(&service) {
			statefulSet, err := c.kc.GetStatefulSet(ctx, ns, service.Name)
			if err != nil {
				return err
			}

			if !c.checkStatefulSetStorage(service.Resources.Storage, statefulSet.Spec.VolumeClaimTemplates) {
				return fmt.Errorf("can not change storage size in persistent status")
			}

		} else {
			_, err := c.kc.GetDeployment(ctx, ns, service.Name)
			if err != nil {
				return err
			}
		}
	}

	ctx = context.WithValue(ctx, builder.SettingsKey, builder.NewDefaultSettings())
	return c.kc.Deploy(ctx, k8sDeployment)
}

func (c *client) CloseDeployment(ctx context.Context, deployment *types.Deployment) error {
	k8sDeployment, err := ClusterDeploymentFromDeployment(deployment)
	if err != nil {
		log.Errorf("CloseDeployment %s", err.Error())
		return err
	}

	did := k8sDeployment.DeploymentID()
	ns := builder.DidNS(did)
	if len(ns) == 0 {
		return fmt.Errorf("can not get ns from deployment id %s and owner %s", deployment.ID, deployment.Owner)
	}

	return c.kc.DeleteNS(ctx, ns)
}

func (c *client) GetDeployment(ctx context.Context, id types.DeploymentID) (*types.Deployment, error) {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	services, err := c.getTitanServices(ctx, ns)
	if err != nil {
		return nil, err
	}

	serviceList, err := c.kc.ListServices(ctx, ns)
	if err != nil {
		return nil, err
	}

	portMap, err := k8sServiceToPortMap(serviceList)
	if err != nil {
		return nil, err
	}

	for i := range services {
		name := services[i].Name
		if ports, ok := portMap[name]; ok {
			services[i].Ports = ports
		}
	}

	return &types.Deployment{ID: id, Services: services, ProviderExposeIP: c.providerCfg.ExposeIP}, nil
}

func (c *client) GetLogs(ctx context.Context, id types.DeploymentID) ([]*types.ServiceLog, error) {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	pods, err := c.getPods(ctx, ns)
	if err != nil {
		return nil, err
	}

	logMap := make(map[string][]types.Log)

	for podName, serviceName := range pods {
		buf, err := c.getPodLogs(ctx, ns, podName)
		if err != nil {
			return nil, err
		}
		log := string(buf)

		logs, ok := logMap[serviceName]
		if !ok {
			logs = make([]types.Log, 0)
		}
		logs = append(logs, types.Log(log))
		logMap[serviceName] = logs
	}

	serviceLogs := make([]*types.ServiceLog, 0, len(logMap))
	for serviceName, logs := range logMap {
		serviceLog := &types.ServiceLog{ServiceName: serviceName, Logs: logs}
		serviceLogs = append(serviceLogs, serviceLog)
	}

	return serviceLogs, nil
}

func (c *client) GetEvents(ctx context.Context, id types.DeploymentID) ([]*types.ServiceEvent, error) {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	pods, err := c.getPods(ctx, ns)
	if err != nil {
		return nil, err
	}

	podEventMap, err := c.getEvents(ctx, ns)
	if err != nil {
		return nil, err
	}

	serviceEventMap := make(map[string][]types.Event)
	for podName, serviceName := range pods {
		es, ok := serviceEventMap[serviceName]
		if !ok {
			es = make([]types.Event, 0)
		}

		if podEvents, ok := podEventMap[podName]; ok {
			es = append(es, podEvents...)
		}
		serviceEventMap[serviceName] = es
	}

	serviceEvents := make([]*types.ServiceEvent, 0, len(serviceEventMap))
	for serviceName, events := range serviceEventMap {
		serviceEvent := &types.ServiceEvent{ServiceName: serviceName, Events: events}
		serviceEvents = append(serviceEvents, serviceEvent)
	}

	return serviceEvents, nil
}

func (c *client) getPods(ctx context.Context, ns string) (map[string]string, error) {
	pods := make(map[string]string)
	podList, err := c.kc.ListPods(context.Background(), ns, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if podList == nil {
		return nil, nil
	}

	for _, pod := range podList.Items {
		pods[pod.Name] = pod.ObjectMeta.Labels[builder.TitanManifestServiceLabelName]
	}

	return pods, nil
}

func (c *client) getPodLogs(ctx context.Context, ns string, podName string) ([]byte, error) {
	reader, err := c.kc.PodLogs(ctx, ns, podName)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *client) getEvents(ctx context.Context, ns string) (map[string][]types.Event, error) {
	eventList, err := c.kc.Events(ctx, ns, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if eventList == nil {
		return nil, nil
	}

	eventMap := make(map[string][]types.Event)
	for _, event := range eventList.Items {
		podName := event.InvolvedObject.Name
		events, ok := eventMap[podName]
		if !ok {
			events = make([]types.Event, 0)
		}

		events = append(events, types.Event(event.Message))
		eventMap[podName] = events
	}

	return eventMap, nil
}

// this is titan services not k8s services
func (c *client) getTitanServices(ctx context.Context, ns string) ([]*types.Service, error) {
	deploymentList, err := c.kc.ListDeployments(ctx, ns)
	if err != nil {
		return nil, err
	}

	if deploymentList != nil && len(deploymentList.Items) > 0 {
		return k8sDeploymentsToServices(deploymentList)
	}

	statefulSets, err := c.kc.ListStatefulSets(ctx, ns)
	if err != nil {
		return nil, err
	}
	return k8sStatefulSetsToServices(statefulSets)
}

func (c *client) isPersistent(service *manifest.Service) bool {
	for i := range service.Resources.Storage {
		attrVal := service.Resources.Storage[i].Attributes.Find(builder.StorageAttributePersistent)
		if persistent, _ := attrVal.AsBool(); persistent {
			return true
		}
	}

	return false
}

func (c *client) checkStatefulSetStorage(storages []*manifest.Storage, pvcs []corev1.PersistentVolumeClaim) bool {
	if len(storages) != len(pvcs) {
		return false
	}

	for i := 0; i < len(storages); i++ {
		storage := storages[i]
		pvc := pvcs[i]
		quantity := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		if storage.Quantity.Val.Int64() != quantity.Value() {
			return false
		}
	}

	return true
}

func (c *client) isExistDeploymentOrStatefulSet(ctx context.Context, ns string) (bool, error) {
	deploymentList, err := c.kc.ListDeployments(ctx, ns)
	if err != nil {
		return false, err
	}

	if deploymentList != nil && len(deploymentList.Items) > 0 {
		return true, nil
	}

	statefulSets, err := c.kc.ListStatefulSets(ctx, ns)
	if err != nil {
		return false, err
	}
	if statefulSets != nil && len(statefulSets.Items) > 0 {
		return true, nil
	}

	return false, nil
}

func (c *client) GetDomains(ctx context.Context, id types.DeploymentID) ([]*types.DeploymentDomain, error) {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	var out []*types.DeploymentDomain
	ingress, err := c.kc.GetIngress(ctx, ns, c.providerCfg.HostName)
	if err != nil {
		return nil, err
	}

	for _, rule := range ingress.Spec.Rules {
		out = append(out, &types.DeploymentDomain{
			Host: rule.Host,
		})
	}

	return out, nil
}

func (c *client) AddDomain(ctx context.Context, id types.DeploymentID, hostname string) error {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	ingress, err := c.kc.GetIngress(ctx, ns, c.providerCfg.HostName)
	if err != nil {
		return err
	}

	newRule := netv1.IngressRule{
		Host: hostname,
	}

	if len(ingress.Spec.Rules) == 0 {
		return errors.New("please config hostname and expose port first")
	}

	newRule.IngressRuleValue = ingress.Spec.Rules[0].IngressRuleValue
	ingress.Spec.Rules = append(ingress.Spec.Rules, newRule)

	_, err = c.kc.UpdateIngress(ctx, ns, ingress)
	return nil
}

func (c *client) DeleteDomain(ctx context.Context, id types.DeploymentID, index int64) error {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	ingress, err := c.kc.GetIngress(ctx, ns, c.providerCfg.HostName)
	if err != nil {
		return err
	}

	if len(ingress.Spec.Rules) == 0 || len(ingress.Spec.Rules) < int(index) {
		return nil
	}

	newRules := append(ingress.Spec.Rules[:index-1], ingress.Spec.Rules[index:]...)
	ingress.Spec.Rules = newRules

	_, err = c.kc.UpdateIngress(ctx, ns, ingress)
	return nil
}

func (c *client) ImportCertificate(ctx context.Context, id types.DeploymentID, cert *types.Certificate) error {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	data := map[string][]byte{
		corev1.TLSPrivateKeyKey: cert.Key,
		corev1.TLSCertKey:       cert.Cert,
	}

	secret, err := c.kc.CreateSecret(ctx, corev1.SecretTypeTLS, ns, cert.Host, data)
	if err != nil {
		return err
	}

	return c.updateDomain(ctx, id, cert.Host, secret.Name)
}

func (c *client) updateDomain(ctx context.Context, id types.DeploymentID, hostname, secretName string) error {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	ingress, err := c.kc.GetIngress(ctx, ns, c.providerCfg.HostName)
	if err != nil {
		return err
	}

	newRule := netv1.IngressRule{
		Host: hostname,
	}

	if len(ingress.Spec.Rules) == 0 {
		return errors.New("please config hostname and expose port first")
	}

	newRule.IngressRuleValue = ingress.Spec.Rules[0].IngressRuleValue
	ingress.Spec.Rules = append(ingress.Spec.Rules, newRule)
	ingress.Spec.TLS = append(ingress.Spec.TLS, netv1.IngressTLS{
		Hosts:      []string{hostname},
		SecretName: secretName,
	})

	_, err = c.kc.UpdateIngress(ctx, ns, ingress)
	return err
}

func (c *client) Exec(ctx context.Context, id types.DeploymentID, podIndex int, stdin io.Reader, stdout, stderr io.Writer, cmd []string, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) (types.ExecResult, error) {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	pods, err := c.kc.ListPods(context.Background(), ns, metav1.ListOptions{})
	if err != nil {
		return types.ExecResult{Code: 0}, err
	}

	if pods == nil || len(pods.Items) == 0 {
		return types.ExecResult{Code: 0}, errors.New("no pod found")
	}

	if len(pods.Items) < podIndex {
		return types.ExecResult{Code: 0}, errors.New("pod index not exist")
	}

	podName := pods.Items[podIndex].Name
	result, err := c.kc.Exec(ctx, c.providerCfg.KubeConfigPath, ns, podName, stdin, stdout, stderr, cmd, tty, terminalSizeQueue)

	return types.ExecResult{Code: result.ExitCode()}, err

}

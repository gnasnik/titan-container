package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	netv1 "k8s.io/api/networking/v1"

	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/Filecoin-Titan/titan-container/node/config"
	"github.com/Filecoin-Titan/titan-container/node/impl/provider/kube"
	"github.com/Filecoin-Titan/titan-container/node/impl/provider/kube/builder"
	"github.com/Filecoin-Titan/titan-container/node/impl/provider/kube/manifest"
	logging "github.com/ipfs/go-log/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var log = logging.Logger("provider")

type Manager interface {
	GetStatistics(ctx context.Context) (*types.ResourcesStatistics, error)
	CreateDeployment(ctx context.Context, deployment *types.Deployment) error
	UpdateDeployment(ctx context.Context, deployment *types.Deployment) error
	CloseDeployment(ctx context.Context, deployment *types.Deployment) error
	GetDeployment(ctx context.Context, id types.DeploymentID) (*types.Deployment, error)
	GetLogs(ctx context.Context, id types.DeploymentID) ([]*types.ServiceLog, error)
	GetEvents(ctx context.Context, id types.DeploymentID) ([]*types.ServiceEvent, error)
	GetDeploymentDomains(ctx context.Context, id types.DeploymentID) ([]*types.DeploymentDomain, error)
	AddDeploymentDomain(ctx context.Context, id types.DeploymentID, hostname string) error
	DeleteDeploymentDomain(ctx context.Context, id types.DeploymentID, index int64) error
}

type manager struct {
	kc          kube.Client
	providerCfg *config.ProviderCfg
}

var _ Manager = (*manager)(nil)

func NewManager(config *config.ProviderCfg) (Manager, error) {
	client, err := kube.NewClient(config.KubeConfigPath, config)
	if err != nil {
		return nil, err
	}
	return &manager{kc: client, providerCfg: config}, nil
}

func (m *manager) GetStatistics(ctx context.Context) (*types.ResourcesStatistics, error) {
	nodeResources, err := m.kc.FetchNodeResources(ctx)
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

func (m *manager) CreateDeployment(ctx context.Context, deployment *types.Deployment) error {
	if deployment.Authority {
		deployment.ProviderExposeIP = m.providerCfg.ExposeIP
	}

	cDeployment, err := ClusterDeploymentFromDeployment(deployment)
	if err != nil {
		log.Errorf("CreateDeployment %s", err.Error())
		return err
	}

	did := cDeployment.DeploymentID()
	ns := builder.DidNS(did)

	if isExist, err := m.isDeploymentExist(ctx, ns); err != nil {
		return err
	} else if isExist {
		return fmt.Errorf("deployment %s already exist", deployment.ID)
	}

	ctx = context.WithValue(ctx, builder.SettingsKey, builder.NewDefaultSettings())
	return m.kc.Deploy(ctx, cDeployment)
}

func (m *manager) UpdateDeployment(ctx context.Context, deployment *types.Deployment) error {
	if deployment.Authority {
		deployment.ProviderExposeIP = m.providerCfg.ExposeIP
	}

	k8sDeployment, err := ClusterDeploymentFromDeployment(deployment)
	if err != nil {
		log.Errorf("UpdateDeployment %s", err.Error())
		return err
	}

	did := k8sDeployment.DeploymentID()
	ns := builder.DidNS(did)

	if isExist, err := m.isDeploymentExist(ctx, ns); err != nil {
		return err
	} else if !isExist {
		return fmt.Errorf("deployment %s do not exist", deployment.ID)
	}

	ctx = context.WithValue(ctx, builder.SettingsKey, builder.NewDefaultSettings())
	return m.kc.Deploy(ctx, k8sDeployment)
}

func (m *manager) CloseDeployment(ctx context.Context, deployment *types.Deployment) error {
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

	return m.kc.DeleteNS(ctx, ns)
}

func (m *manager) GetDeployment(ctx context.Context, id types.DeploymentID) (*types.Deployment, error) {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	services, err := m.getServices(ctx, ns)
	if err != nil {
		return nil, err
	}

	serviceList, err := m.kc.ListServices(ctx, ns)
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

	return &types.Deployment{ID: id, Services: services, ProviderExposeIP: m.providerCfg.ExposeIP}, nil
}

func (m *manager) GetLogs(ctx context.Context, id types.DeploymentID) ([]*types.ServiceLog, error) {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	pods, err := m.getPods(ctx, ns)
	if err != nil {
		return nil, err
	}

	logMap := make(map[string][]types.Log)

	for podName, serviceName := range pods {
		buf, err := m.getPodLogs(ctx, ns, podName)
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

func (m *manager) GetEvents(ctx context.Context, id types.DeploymentID) ([]*types.ServiceEvent, error) {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	pods, err := m.getPods(ctx, ns)
	if err != nil {
		return nil, err
	}

	podEventMap, err := m.getEvents(ctx, ns)
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

func (m *manager) getPods(ctx context.Context, ns string) (map[string]string, error) {
	pods := make(map[string]string)
	podList, err := m.kc.ListPods(context.Background(), ns, metav1.ListOptions{})
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

func (m *manager) getPodLogs(ctx context.Context, ns string, podName string) ([]byte, error) {
	reader, err := m.kc.PodLogs(ctx, ns, podName)
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

func (m *manager) getEvents(ctx context.Context, ns string) (map[string][]types.Event, error) {
	eventList, err := m.kc.Events(ctx, ns, metav1.ListOptions{})
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

func (m *manager) getServices(ctx context.Context, ns string) ([]*types.Service, error) {
	deploymentList, err := m.kc.ListDeployments(ctx, ns)
	if err != nil {
		return nil, err
	}

	if deploymentList != nil && len(deploymentList.Items) > 0 {
		return k8sDeploymentsToServices(deploymentList)
	}

	statefulSets, err := m.kc.ListStatefulSets(ctx, ns)
	if err != nil {
		return nil, err
	}
	return k8sStatefulSetsToServices(statefulSets)
}

func (m *manager) isDeploymentExist(ctx context.Context, ns string) (bool, error) {
	deploymentList, err := m.kc.ListDeployments(ctx, ns)
	if err != nil {
		return false, err
	}

	if deploymentList != nil && len(deploymentList.Items) > 0 {
		return true, nil
	}

	statefulSets, err := m.kc.ListStatefulSets(ctx, ns)
	if err != nil {
		return false, err
	}
	if statefulSets != nil && len(statefulSets.Items) > 0 {
		return true, nil
	}

	return false, nil
}

func (m *manager) GetDeploymentDomains(ctx context.Context, id types.DeploymentID) ([]*types.DeploymentDomain, error) {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	var out []*types.DeploymentDomain
	ingress, err := m.kc.GetIngress(ctx, ns, m.providerCfg.HostName)
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

func (m *manager) AddDeploymentDomain(ctx context.Context, id types.DeploymentID, hostname string) error {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	ingress, err := m.kc.GetIngress(ctx, ns, m.providerCfg.HostName)
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

	_, err = m.kc.UpdateIngress(ctx, ns, ingress)
	return nil
}

func (m *manager) DeleteDeploymentDomain(ctx context.Context, id types.DeploymentID, index int64) error {
	deploymentID := manifest.DeploymentID{ID: string(id)}
	ns := builder.DidNS(deploymentID)

	ingress, err := m.kc.GetIngress(ctx, ns, m.providerCfg.HostName)
	if err != nil {
		return err
	}

	if len(ingress.Spec.Rules) == 0 || len(ingress.Spec.Rules) < int(index) {
		return nil
	}

	newRules := append(ingress.Spec.Rules[:index-1], ingress.Spec.Rules[index:]...)
	ingress.Spec.Rules = newRules

	_, err = m.kc.UpdateIngress(ctx, ns, ingress)
	return nil
}

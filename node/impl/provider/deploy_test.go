package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/Filecoin-Titan/titan-container/node/config"
	"github.com/Filecoin-Titan/titan-container/node/impl/provider/kube"
	"github.com/Filecoin-Titan/titan-container/node/impl/provider/kube/builder"
	"github.com/Filecoin-Titan/titan-container/node/impl/provider/kube/manifest"
	"github.com/stretchr/testify/require"
)

func TestCreateDeploy(t *testing.T) {
	config := &config.ProviderCfg{KubeConfigPath: "./test/config", ExposeIP: "192.168.0.132"}
	client, err := NewClient(config)
	require.NoError(t, err)

	port := types.Port{Port: 6379}
	service := types.Service{Image: "nginx:1.24.0", Ports: []types.Port{port}, ComputeResources: types.ComputeResources{CPU: 0.1, Memory: 100, Storage: []*types.Storage{&types.Storage{Quantity: 100}}}}
	deploy := types.Deployment{
		ID:       types.DeploymentID("2222"),
		Owner:    "test",
		Services: []*types.Service{&service},
	}

	err = client.CreateDeployment(context.Background(), &deploy)
	require.NoError(t, err)
}

func TestUplodateDeploy(t *testing.T) {
	config := &config.ProviderCfg{KubeConfigPath: "./test/config", ExposeIP: "192.168.0.132"}
	client, err := NewClient(config)
	require.NoError(t, err)

	port := types.Port{Port: 6379}
	ports := types.Ports([]types.Port{port})
	service := types.Service{Image: "test", Ports: ports, ComputeResources: types.ComputeResources{CPU: 0.1, Memory: 100, Storage: []*types.Storage{&types.Storage{Quantity: 100}}}}
	deploy := types.Deployment{
		ID:       types.DeploymentID("ccc"),
		Owner:    "test",
		Services: []*types.Service{&service},
	}

	err = client.UpdateDeployment(context.Background(), &deploy)
	require.NoError(t, err)
}

func TestDeleteDeploy(t *testing.T) {
	kubeconfig := "./test/config"
	client, err := kube.NewClient(kubeconfig, config.DefaultProviderCfg())
	require.NoError(t, err)

	deploy := types.Deployment{
		ID:       types.DeploymentID("4444"),
		Owner:    "test",
		Services: []*types.Service{},
	}

	ns := builder.DidNS(manifest.DeploymentID{ID: string(deploy.ID)})
	err = client.DeleteNS(context.Background(), ns)
	require.NoError(t, err)
}

func TestResourcesStatistics(t *testing.T) {
	config := &config.ProviderCfg{KubeConfigPath: "./test/config", ExposeIP: "192.168.0.132"}
	client, err := NewClient(config)
	require.NoError(t, err)

	statistics, err := client.GetStatistics(context.Background())
	require.NoError(t, err)

	t.Logf("nodeResources %#v", *statistics)

}

func TestGetDeployment(t *testing.T) {
	config := &config.ProviderCfg{KubeConfigPath: "./test/config", ExposeIP: "192.168.0.132"}
	client, err := NewClient(config)
	require.NoError(t, err)

	deployment, err := client.GetDeployment(context.Background(), types.DeploymentID("2222"))
	require.NoError(t, err)

	for _, service := range deployment.Services {
		t.Logf("deployment:%#v", *service)
	}

	t.Logf("deployment:%#v", *deployment)

}

func TestListDeployment(t *testing.T) {
	kubeconfig := "./test/config"
	client, err := kube.NewClient(kubeconfig, config.DefaultProviderCfg())
	require.NoError(t, err)

	deploymentList, err := client.ListDeployments(context.Background(), "bbbbb")
	require.NoError(t, err)

	if deploymentList == nil {
		t.Logf("deploymentList == nil")
	}
	if len(deploymentList.Items) == 0 {
		t.Logf("len(deploymentList.Items) == 0")
	}
	t.Logf("deployment:%#v", *deploymentList)
	for _, deployment := range deploymentList.Items {
		buf, _ := json.Marshal(deployment.Status.Conditions)
		t.Logf("deployment:%s", string(buf))
	}
}

func TestGetLogs(t *testing.T) {
	config := &config.ProviderCfg{KubeConfigPath: "./test/config", ExposeIP: "192.168.0.132"}
	client, err := NewClient(config)
	require.NoError(t, err)

	logs, err := client.GetLogs(context.Background(), types.DeploymentID("1111"))
	require.NoError(t, err)

	for _, serviceLog := range logs {
		t.Logf("log len:%d", len(serviceLog.Logs))
		for _, log := range serviceLog.Logs {
			podLogs := formatLogs(string(log))
			for _, podLog := range podLogs {
				t.Logf("%s", podLog)
			}
		}
	}
}

func formatLogs(log string) []string {
	logLines := strings.Split(log, "\n")
	return logLines
}

func TestGetEvents(t *testing.T) {
	config := &config.ProviderCfg{KubeConfigPath: "./test/config", ExposeIP: "192.168.0.132"}
	client, err := NewClient(config)
	require.NoError(t, err)

	events, err := client.GetEvents(context.Background(), types.DeploymentID("2222"))
	require.NoError(t, err)

	for _, serviceEvent := range events {
		t.Logf("event len:%d", len(serviceEvent.Events))
		for _, event := range serviceEvent.Events {
			t.Logf("event:%s", string(event))
		}
	}
}

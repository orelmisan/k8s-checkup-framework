package nads

import (
	"context"
	"encoding/json"
	"fmt"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

const (
	NetworkConfigTypeFiledName    = "type"
	NetworkConfigPluginsFiledName = "plugins"
)

func GetNetworkType(virtClient kubecli.KubevirtClient, namespace, name string) (string, error) {
	nad, err := virtClient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).
		Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get netwrok type: %v", err)
	}

	var netConf map[string]interface{}
	err = json.Unmarshal([]byte(nad.Spec.Config), &netConf)
	if err != nil {
		return "", fmt.Errorf("failed to get netwrok type: failed to unmarshal NetworkAttachmentDefinitions config: %v", err)
	}

	var networkType string
	// Identify target is single CNI config or plugins
	if pluginsRaw, exists := netConf[NetworkConfigPluginsFiledName]; exists {
		// CNI conflist
		plugins := pluginsRaw.([]interface{})
		for _, pluginRaw := range plugins {
			plugin := pluginRaw.(map[string]interface{})
			pluginType := fmt.Sprintf("%v", plugin[NetworkConfigTypeFiledName])
			if isSupportedCNIPlugin(pluginType) {
				networkType = pluginType
				break
			}
		}
	} else {
		// single CNI config
		networkType = fmt.Sprintf("%v", netConf[NetworkConfigTypeFiledName])
	}

	return networkType, nil
}

var supportedCNIPlugins = []string{"bridge", "bridge-cnv", "sriov"}

func isSupportedCNIPlugin(name string) bool {
	for _, supportedPlugin := range supportedCNIPlugins {
		if supportedPlugin == name {
			return true
		}
	}
	return false
}

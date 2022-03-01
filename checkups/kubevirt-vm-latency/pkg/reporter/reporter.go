package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

func WriteToStdout(data map[string]string) error {
	rawResult, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %v", err)
	}
	_, err = fmt.Fprintln(os.Stdout, string(rawResult))
	if err != nil {
		return fmt.Errorf("failed write results to stdout: %v", err)
	}

	return nil
}

func WriteToConfigMap(virtClient kubecli.KubevirtClient, namespace, name string, data map[string]string) error {
	const errMsgPrefix = "failed to write check results to ConfigMap"

	configMap, err := virtClient.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("%s: failed to get ConfigMap: %v", errMsgPrefix, err)
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string, len(data))
	}
	for k, v := range data {
		configMap.Data[k] = v
	}

	log.Printf("updating to ConfigMap %s/%s..", configMap.Namespace, configMap.Name)
	if _, err := virtClient.CoreV1().ConfigMaps(configMap.Namespace).Update(context.Background(), configMap, k8smetav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("%s: failed to update ConfigMap: %v", errMsgPrefix, err)
	}

	return nil
}

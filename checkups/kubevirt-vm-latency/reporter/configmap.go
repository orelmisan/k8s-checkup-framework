package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

type reporter struct {
	configMapName      string
	configMapNamespace string
}

func NewConfigMapReporter(name, namesapce string) reporter {
	return reporter{configMapName: name, configMapNamespace: namesapce}
}

func (r *reporter) WriteToConfigMap(obj interface{}) error {
	const errMsgPrefix = "failed to write check results to ConfigMap"

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return fmt.Errorf("%s: failed to obtain KubeVirt client: %v", errMsgPrefix, err)
	}

	log.Printf("Writing to ConfigMap %s/%s\n", r.configMapNamespace, r.configMapName)
	raw, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("%s: failed to marshal object: %v", errMsgPrefix, err)
	}
	configMap, err := virtClient.CoreV1().ConfigMaps(r.configMapNamespace).Get(context.Background(), r.configMapName, k8smetav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("%s: failed to get ConfigMap: %v", errMsgPrefix, err)
	}
	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}
	configMap.Data["result"] = string(raw)
	if _, err := virtClient.CoreV1().ConfigMaps(configMap.Namespace).Update(context.Background(), configMap, k8smetav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("%s: failed to update ConfigMap: %v", errMsgPrefix, err)
	}

	return nil
}

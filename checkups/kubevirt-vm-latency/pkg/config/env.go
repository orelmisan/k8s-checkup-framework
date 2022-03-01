package config

import (
	"fmt"
	"os"
)

const (
	ResultsConfigMapNamespaceEnvVarName = "RESULT_CONFIGMAP_NAMESPACE"
	ResultsConfigMapNameEnvVarName      = "RESULT_CONFIGMAP_NAME"
	NetworkNamespaceEnvVarName          = "NETWORK_NAMESPACE"
	NetworkNameEnvVarName               = "NETWORK_NAME"
	SampleDurationEnvVarName            = "SAMPLE_DURATION"
	SourceNodeNameEnvVarName            = "SOURCE_NODE"
	TargetNodeNameEnvVarName            = "TARGET_NODE"
	DesiredMaxLatencyEnvVarName         = "MAX_DESIRED_LATENCY"
)

func LoadEnvVars() (map[string]string, error) {
	const errMsgFormat = "failed to load %s environment variable"

	var exists bool

	envVarNames := []string{
		ResultsConfigMapNamespaceEnvVarName,
		ResultsConfigMapNameEnvVarName,
		NetworkNamespaceEnvVarName,
		NetworkNameEnvVarName,
		SampleDurationEnvVarName,
		SourceNodeNameEnvVarName,
		TargetNodeNameEnvVarName,
		DesiredMaxLatencyEnvVarName,
	}

	envVars := make(map[string]string, len(envVarNames))
	for _, envVarName := range envVarNames {
		envVars[envVarName], exists = os.LookupEnv(envVarName)
		if !exists {
			return envVars, fmt.Errorf(errMsgFormat, envVarName)
		}
	}

	return envVars, nil
}

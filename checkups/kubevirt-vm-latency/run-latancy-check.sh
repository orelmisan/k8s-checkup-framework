#! /bin/bash

set -e 

SCRIPT_PATH=$(dirname "$(realpath "$0")")
MANIFESTS="$SCRIPT_PATH/manifests"

CRI="${CRI:-podman}"

# build container and push to local registry
image="kubevirt-latency-check"
CRI="$CRI" IMAGE="$image" TAG="$tag" $SCRIPT_PATH/build/build.sh

# push to local registry
registry="localhost:5000"
$CRI tag "$image" "$registry/$image"
$CRI push "$registry/$image"

trap "kubectl delete -f $MANIFESTS" EXIT

## Create kubevirt network latency checkup prequisits
kubectl apply -f $MANIFESTS/nads.yaml

# Create the Roles that are necessary for the checkup to work.
# The ServiceAccount that runs the checkup pod will be granted with those Roles.
kubectl apply -f $MANIFESTS/roles.yaml

echo "starting Kubevirt network latency checkup"
cat $MANIFESTS/latency-check-config.yaml
echo ""
sleep 2

kubectl apply -f $MANIFESTS/namespace.yaml
kubectl apply -f $MANIFESTS/serviceaccount.yaml
kubectl apply -f $MANIFESTS/rolebindings.yaml
kubectl apply -f $MANIFESTS/results-configmap.yaml

kubectl apply -f $MANIFESTS/latency-check-config.yaml
kubectl apply -f $MANIFESTS/latency-check-job.yaml

# follow the checkup logs..
working_ns=$(cat $MANIFESTS/namespace.yaml | grep -Po "name: \K.*")
checkup_job=$(cat $MANIFESTS/latency-check-job.yaml | grep metadata: -A 2 | grep -Po "name: \K.*")
job_name_label="job-name=$checkup_job"

kubectl get job $checkup_job -n $working_ns
echo "waiting for job pod to start.."
timeout 30s bash -c "until kubectl get pod -n $working_ns -l $job_name_label --field-selector status.phase=Running --no-headers | grep . ; do sleep 2; done" || true

pod=$(kubectl get po -n $working_ns -l $job_name_label --no-headers | head -1 | awk '{print $1}')
kubectl logs $pod -n $working_ns --follow | tee

kubectl wait job -n $working_ns $checkup_job --for condition=complete

# print latency check results from the result ConfigMap
results_configmap=$(cat $MANIFESTS/results-configmap.yaml | grep -Po "name: \K.*")
results_configmap_ns=$(cat $MANIFESTS/results-configmap.yaml | grep -Po "namespace: \K.*")
kubectl get cm $results_configmap -n $results_configmap_ns -o jsonpath='{.data}' | jq

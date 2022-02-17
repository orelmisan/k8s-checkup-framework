#! /bin/bash

set -e 

SCRIPT_PATH=$(dirname "$(realpath "$0")")
MANIFESTS="$SCRIPT_PATH/manifests"

# build container and push to local registry
image="kubevirt-latency-check"
registry="localhost:5000"
IMAGE="$image" REGISTRY="$registry" $SCRIPT_PATH/build/build.sh

trap "kubectl delete -f $MANIFESTS" EXIT

### User input for the k8s-checkup-framwork ###
## Create kubevirt network latency checkup prequisits
kubectl apply -f $MANIFESTS/nad.yaml
# Create the Roles that are necessary for the checkup to work.
# The ServiceAccount that runs the checkup pod will be granted with those Roles.
kubectl apply -f $MANIFESTS/roles.yaml
# Create ConfigMap with all necesary args
kubectl apply -f $MANIFESTS/checkup-config.yaml

echo "Starting Kubevirt latency check:"
cat $MANIFESTS/checkup-config.yaml
sleep 2

### TODO: Automate (framework part) ###
kubectl apply -f $MANIFESTS/ns.yaml
kubectl apply -f $MANIFESTS/serviceaccount.yaml

checkup_configmap=$(cat $MANIFESTS/checkup-config.yaml | grep -Po "name: \K.*")
checkup_configmap_ns=$(cat $MANIFESTS/checkup-config.yaml | grep -Po "namespace: \K.*")
checkup_ns=$(cat $MANIFESTS/ns.yaml | grep -Po "name: \K.*")

# copy the user provided checkup ConfigMap to the checkup working namespace
# in order to assosicate the ConfigMap with the Job underlying Pod spec
# and set the ConfigMap content as env vars on the Pod.
# The env vars are consumed by the checkup.
kubectl get cm $checkup_configmap -n $checkup_configmap_ns -o yaml \
  | sed "s?namespace: $checkup_configmap_ns?namespace: $checkup_ns?" \
  | kubectl create -f -

# Finally, create RoleBindings for each Role the checkup requiers,
# including Role and RoleBinding that enable the checkup Pod to write
# the results to the provided checkup ConfigMap.
# Run the latancy check with a Job.
kubectl apply -f $MANIFESTS/kubevirt-latency-checkup.yaml

# follow the checkup logs..

checkup_job=$(cat $MANIFESTS/kubevirt-latency-checkup.yaml | grep "Job" -A 2 | grep -Po "name: \K.*")
job_name_label="job-name=$checkup_job"

kubectl get job $checkup_job -n $checkup_ns 
until kubectl get po -n $checkup_ns -l $job_name_label --no-headers | grep .; do echo "waiting for job pod to start.."; sleep 2; done 
kubectl wait pod -n $checkup_ns -l $job_name_label --for condition=ready
pod=$(kubectl get po -n $checkup_ns -l $job_name_label --no-headers | head -1 | awk '{print $1}')
kubectl logs $pod -n $checkup_ns --follow | tee

kubectl wait job -n $checkup_ns $checkup_job --for condition=Complete || true

# print latency check results from the result ConfigMap
kubectl get cm $checkup_configmap -n $checkup_configmap_ns -o jsonpath='{.data}'  | jq .result -r | jq

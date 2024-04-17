### Create Operator
```shell
operator-sdk init --domain charolia.io --repo github.com/ankitcharolia/autoscaling-operator
operator-sdk create api --group=k8s --version=v1beta1 --kind=AutoScaler --resource --controller
```

### Run your operator locally
```shell
make install run
```

### Build and Push the Operator
Update the docker image variables
```bash
IMAGE_TAG_BASE ?= ankitcharolia/autoscaling-operator
IMG ?= $(IMAGE_TAG_BASE):$(VERSION)
```

```shell
# Be sure to be logged to your registry, then build and push your operator:
make docker-build docker-push
```
**NOTE:** autoscaling-operator is available here: `ankitcharolia/autoscaling-operator:1.0.0`

### Deploy to Kubernetes
```bash
make deploy
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

### References
* [Kubernetes Operator Example](https://docs.okd.io/latest/operators/operator_sdk/golang/osdk-golang-tutorial.html#osdk-golang-create-api-controller_osdk-golang-tutorial)
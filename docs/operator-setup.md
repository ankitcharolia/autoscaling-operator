### Create Operator
```shell
operator-sdk init --domain charolia.io --repo github.com/ankitcharolia/autoscaling-operator
operator-sdk create api --group=k8s --version=v1beta1 --kind=AutoScaler --resource --controller
```

### References
* [Kubernetes Operator Example](https://docs.okd.io/latest/operators/operator_sdk/golang/osdk-golang-tutorial.html#osdk-golang-create-api-controller_osdk-golang-tutorial)
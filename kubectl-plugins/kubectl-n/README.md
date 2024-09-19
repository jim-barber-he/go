
# kubectl-n

A tool for listing nodes in Kubernetes clusters.

Similar to `kubectl get nodes`, but with some different information.

In particular, in AWS environments it shows the following information about nodes in a cluster:
- The instance type of the node.
- The availabilty zone (AZ) the node is running in.
- The name of the instance group / node group the node belongs to.

It also sorts the nodes by their instance group name; then by the AZ; and finally by their names.

## Usage

```shell
kubectl n [ --context CONTEXT ]
```

## Comparison to `kubectl get nodes`

### EKS cluster

```shell
$ kubectl get nodes
```
```
NAME                                               STATUS   ROLES    AGE    VERSION
ip-10-160-24-50.ap-southeast-2.compute.internal    Ready    <none>   337d    v1.28.1-eks-43840fb
ip-10-160-41-121.ap-southeast-2.compute.internal   Ready    <none>   337d    v1.28.1-eks-43840fb
```

```shell
$ kubectl n
```
```
NAME              OK  AGE    VERSION              RUNTIME  TYPE         SPOT  AZ  INSTANCE-ID          INSTANCE-GROUP
ip-10-160-24-50   ✓   48w1d  v1.28.1-eks-43840fb  1.6.19   c6in.xlarge  x     a   i-0fd3c1eb68a092efa  ng-1
ip-10-160-41-121  ✓   48w1d  v1.28.1-eks-43840fb  1.6.19   c6in.xlarge  x     b   i-000d3e4b6f78aed19  ng-1
```

### kOps cluster

```shell
$ kubectl get nodes
```
```
NAME                  STATUS   ROLES              AGE    VERSION
i-02c87764c5d7884b3   Ready    node,spot-worker   2d4h   v1.29.9
i-0630694be7a879cc4   Ready    node               2d4h   v1.29.9
i-065f30faa9db7f949   Ready    node,spot-worker   2d5h   v1.29.9
i-081f41e1d8e630e0c   Ready    node,spot-worker   29h    v1.29.9
i-08e004186079e74e2   Ready    node,spot-worker   2d4h   v1.29.9
i-0a41c827b6e581efe   Ready    node,spot-worker   2d4h   v1.29.9
i-0a76386295da6fe83   Ready    node               2d4h   v1.29.9
i-0af469ea75aa4c82b   Ready    node               2d4h   v1.29.9
i-0b568d75ecb3153d0   Ready    control-plane      2d6h   v1.29.9
i-0e63a4a348096dcf5   Ready    control-plane      2d5h   v1.29.9
i-0ed734f56ed35c352   Ready    node               2d5h   v1.29.9
i-0ed7cb8a38a7b4d35   Ready    control-plane      2d5h   v1.29.9
i-0f9bff5d2c23a5a95   Ready    node               2d5h   v1.29.9
```

```shell
$ kubectl n
```
```
NAME                 OK  AGE   VERSION  RUNTIME  TYPE              SPOT  AZ  IP-ADDRESS    INSTANCE-GROUP
i-0b568d75ecb3153d0  ✓   2d6h  v1.29.9  1.7.16   t3.xlarge         x     a   10.8.36.6     control-plane-ap-southeast-2a
i-0ed7cb8a38a7b4d35  ✓   2d5h  v1.29.9  1.7.16   t3.xlarge         x     b   10.8.82.50    control-plane-ap-southeast-2b
i-0e63a4a348096dcf5  ✓   2d5h  v1.29.9  1.7.16   t3.xlarge         x     c   10.8.124.42   control-plane-ap-southeast-2c
i-0ed734f56ed35c352  ✓   2d5h  v1.29.9  1.7.16   r7i.2xlarge       x     b   10.8.87.250   elasticsearch
i-0f9bff5d2c23a5a95  ✓   2d5h  v1.29.9  1.7.16   r7i.2xlarge       x     b   10.8.66.40    elasticsearch
i-0af469ea75aa4c82b  ✓   2d4h  v1.29.9  1.7.16   c7i.large         x     a   10.8.128.98   ingress-controller-a
i-0a76386295da6fe83  ✓   2d4h  v1.29.9  1.7.16   c7i.large         x     b   10.8.129.170  ingress-controller-b
i-0630694be7a879cc4  ✓   2d4h  v1.29.9  1.7.16   c7i.large         x     c   10.8.130.112  ingress-controller-c
i-081f41e1d8e630e0c  ✓   1d5h  v1.29.9  1.7.16   m7i.xlarge        ✓     a   10.8.45.171   node
i-0a41c827b6e581efe  ✓   2d4h  v1.29.9  1.7.16   m7i.xlarge        ✓     a   10.8.49.17    node
i-02c87764c5d7884b3  ✓   2d4h  v1.29.9  1.7.16   c7i-flex.2xlarge  ✓     b   10.8.70.157   node
i-08e004186079e74e2  ✓   2d4h  v1.29.9  1.7.16   c7i-flex.2xlarge  ✓     b   10.8.82.184   node
i-065f30faa9db7f949  ✓   2d5h  v1.29.9  1.7.16   c7i-flex.2xlarge  ✓     c   10.8.97.206   node
```

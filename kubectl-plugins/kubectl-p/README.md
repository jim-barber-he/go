
# kubectl-p

A tool for listing pods in Kubernetes clusters.

Similar to `kubectl get pods`, but with some different information.

In particular, in AWS environments it shows the following information about the pods:
- If the node the pod is running on is a spot instance or not.
- The availability zone (AZ) that the node the pod is running on is in.

## Usage

```shell
$ kubectl p --help
```
```
  -A, --all-namespaces       List the pods across all namespaces. Overrides --namespace / -n
      --context string       The name of the kubeconfig context to use
  -n, --namespace string     If present, the namespace scope for this CLI request
      --profile-cpu string   Produce pprof cpu profiling output in supplied file
      --profile-mem string   Produce pprof memory profiling output in supplied file
  -l, --selector string      Selector (label query) to filter on
```

## Comparison to `kubectl get pods`

```shell
$ kubectl get pods
```
```
NAME                                              READY   STATUS    RESTARTS       AGE
aws-cloud-controller-manager-h4fjj                1/1     Running   0              2d5h
aws-cloud-controller-manager-njltb                1/1     Running   0              2d5h
aws-cloud-controller-manager-t2sss                1/1     Running   0              2d6h
aws-iam-authenticator-6rvh5                       1/1     Running   0              2d5h
aws-iam-authenticator-dw7fp                       1/1     Running   0              2d6h
aws-iam-authenticator-s769n                       1/1     Running   0              2d5h
aws-node-4fzrk                                    2/2     Running   0              2d5h
aws-node-5pd5m                                    2/2     Running   0              2d5h
aws-node-6wpkg                                    2/2     Running   0              2d5h
aws-node-9svms                                    2/2     Running   0              2d4h
aws-node-b9w4r                                    2/2     Running   0              2d4h
aws-node-bjvwn                                    2/2     Running   0              2d4h
aws-node-bvz9d                                    2/2     Running   0              29h
aws-node-c9tc7                                    2/2     Running   0              2d4h
aws-node-jn988                                    2/2     Running   0              2d5h
aws-node-prtkv                                    2/2     Running   0              2d5h
aws-node-qqpgk                                    2/2     Running   0              2d6h
aws-node-termination-handler-74b857fdd7-cmmqf     1/1     Running   0              2d5h
aws-node-termination-handler-74b857fdd7-x6drw     1/1     Running   0              2d5h
aws-node-vwmm6                                    2/2     Running   0              2d4h
aws-node-xrv85                                    2/2     Running   0              2d5h
cert-manager-559975d55c-pw6wg                     1/1     Running   1 (2d6h ago)   2d6h
cert-manager-cainjector-868f54ccf5-shvvf          1/1     Running   1 (2d5h ago)   2d5h
cert-manager-webhook-f8484455c-ks756              1/1     Running   0              2d6h
coredns-7d47876df6-4w5l8                          1/1     Running   0              30h
coredns-7d47876df6-6vvmx                          1/1     Running   0              2d4h
coredns-7d47876df6-tbxvk                          1/1     Running   0              2d4h
coredns-autoscaler-5fdfd9d499-ttrrl               1/1     Running   0              2d5h
ebs-csi-controller-6dc5dcbbb8-67qkt               7/7     Running   0              2d5h
ebs-csi-controller-6dc5dcbbb8-bpm6d               7/7     Running   0              2d5h
ebs-csi-node-62vs7                                3/3     Running   0              2d6h
ebs-csi-node-6g72f                                3/3     Running   0              29h
ebs-csi-node-6mhlz                                3/3     Running   0              2d5h
ebs-csi-node-7rmzx                                3/3     Running   0              2d5h
ebs-csi-node-9qwsx                                3/3     Running   0              2d5h
ebs-csi-node-9rbpp                                3/3     Running   0              2d5h
ebs-csi-node-cckv4                                3/3     Running   0              2d4h
ebs-csi-node-d8v68                                3/3     Running   0              2d5h
ebs-csi-node-l5lfx                                3/3     Running   0              2d4h
ebs-csi-node-vhljt                                3/3     Running   0              2d4h
ebs-csi-node-vn4kn                                3/3     Running   0              2d4h
ebs-csi-node-wkvw9                                3/3     Running   0              2d5h
ebs-csi-node-xg94z                                3/3     Running   0              2d4h
etcd-manager-events-i-0b568d75ecb3153d0           1/1     Running   0              2d6h
etcd-manager-events-i-0e63a4a348096dcf5           1/1     Running   0              2d5h
etcd-manager-events-i-0ed7cb8a38a7b4d35           1/1     Running   0              2d5h
etcd-manager-main-i-0b568d75ecb3153d0             1/1     Running   0              2d6h
etcd-manager-main-i-0e63a4a348096dcf5             1/1     Running   0              2d5h
etcd-manager-main-i-0ed7cb8a38a7b4d35             1/1     Running   0              2d5h
external-dns-78fbf59cd-b7knz                      1/1     Running   0              2d5h
kops-controller-6xhb6                             1/1     Running   0              2d5h
kops-controller-gfdxd                             1/1     Running   0              2d6h
kops-controller-gwrvn                             1/1     Running   0              2d5h
kube-apiserver-i-0b568d75ecb3153d0                2/2     Running   2 (2d6h ago)   2d6h
kube-apiserver-i-0e63a4a348096dcf5                2/2     Running   2 (2d5h ago)   2d5h
kube-apiserver-i-0ed7cb8a38a7b4d35                2/2     Running   2 (2d5h ago)   2d5h
kube-controller-manager-i-0b568d75ecb3153d0       1/1     Running   4 (2d6h ago)   2d6h
kube-controller-manager-i-0e63a4a348096dcf5       1/1     Running   3 (2d5h ago)   2d5h
kube-controller-manager-i-0ed7cb8a38a7b4d35       1/1     Running   4 (2d5h ago)   2d5h
kube-proxy-i-02c87764c5d7884b3                    1/1     Running   0              2d4h
kube-proxy-i-0630694be7a879cc4                    1/1     Running   0              2d4h
kube-proxy-i-065f30faa9db7f949                    1/1     Running   0              2d5h
kube-proxy-i-081f41e1d8e630e0c                    1/1     Running   0              29h
kube-proxy-i-08e004186079e74e2                    1/1     Running   0              2d4h
kube-proxy-i-0a41c827b6e581efe                    1/1     Running   0              2d5h
kube-proxy-i-0a76386295da6fe83                    1/1     Running   0              2d4h
kube-proxy-i-0af469ea75aa4c82b                    1/1     Running   0              2d4h
kube-proxy-i-0b568d75ecb3153d0                    1/1     Running   0              2d6h
kube-proxy-i-0e63a4a348096dcf5                    1/1     Running   0              2d5h
kube-proxy-i-0ed734f56ed35c352                    1/1     Running   0              2d5h
kube-proxy-i-0ed7cb8a38a7b4d35                    1/1     Running   0              2d5h
kube-proxy-i-0f9bff5d2c23a5a95                    1/1     Running   0              2d5h
kube-scheduler-i-0b568d75ecb3153d0                1/1     Running   0              2d6h
kube-scheduler-i-0e63a4a348096dcf5                1/1     Running   0              2d5h
kube-scheduler-i-0ed7cb8a38a7b4d35                1/1     Running   0              2d5h
metrics-server-97767c4f8-k2tnk                    1/1     Running   0              2d5h
metrics-server-97767c4f8-srtx4                    1/1     Running   0              2d5h
node-local-dns-44qtk                              1/1     Running   0              2d5h
node-local-dns-667rq                              1/1     Running   0              2d4h
node-local-dns-7df7b                              1/1     Running   0              2d4h
node-local-dns-82gtf                              1/1     Running   0              2d5h
node-local-dns-8hg7d                              1/1     Running   0              2d5h
node-local-dns-b9m22                              1/1     Running   0              2d5h
node-local-dns-blqdd                              1/1     Running   0              2d6h
node-local-dns-ksvdh                              1/1     Running   0              2d5h
node-local-dns-lhdv9                              1/1     Running   0              2d4h
node-local-dns-rsppq                              1/1     Running   0              2d5h
node-local-dns-sk269                              1/1     Running   0              2d4h
node-local-dns-slz9d                              1/1     Running   0              2d4h
node-local-dns-z92rb                              1/1     Running   0              29h
node-problem-detector-4bnwl                       1/1     Running   0              2d4h
node-problem-detector-cdspw                       1/1     Running   0              2d4h
node-problem-detector-d7nkk                       1/1     Running   0              2d4h
node-problem-detector-f5dpb                       1/1     Running   0              2d5h
node-problem-detector-fltf6                       1/1     Running   0              2d5h
node-problem-detector-fvqtm                       1/1     Running   0              2d5h
node-problem-detector-mwwck                       1/1     Running   0              2d6h
node-problem-detector-nphm4                       1/1     Running   0              29h
node-problem-detector-p48n6                       1/1     Running   0              2d4h
node-problem-detector-pm56j                       1/1     Running   0              2d5h
node-problem-detector-rdz5r                       1/1     Running   0              2d5h
node-problem-detector-wpc7h                       1/1     Running   0              2d4h
node-problem-detector-z8q4s                       1/1     Running   0              2d5h
pod-identity-webhook-79974dcd9c-2nqzf             1/1     Running   0              2d4h
pod-identity-webhook-79974dcd9c-n276c             1/1     Running   0              2d5h
pod-identity-webhook-79974dcd9c-trg6r             1/1     Running   0              2d5h
```

```shell
$ kubectl p
```
```
NAME                                             READY  STATUS   RESTARTS      AGE   IP            NODE                 SPOT  AZ
aws-cloud-controller-manager-h4fjj               1/1    Running  0             2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
aws-cloud-controller-manager-njltb               1/1    Running  0             2d5h  10.8.124.42   i-0e63a4a348096dcf5  x     c
aws-cloud-controller-manager-t2sss               1/1    Running  0             2d6h  10.8.36.6     i-0b568d75ecb3153d0  x     a
aws-iam-authenticator-6rvh5                      1/1    Running  0             2d5h  10.8.124.42   i-0e63a4a348096dcf5  x     c
aws-iam-authenticator-dw7fp                      1/1    Running  0             2d6h  10.8.36.6     i-0b568d75ecb3153d0  x     a
aws-iam-authenticator-s769n                      1/1    Running  0             2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
aws-node-4fzrk                                   2/2    Running  0             2d5h  10.8.87.250   i-0ed734f56ed35c352  x     b
aws-node-5pd5m                                   2/2    Running  0             2d5h  10.8.66.40    i-0f9bff5d2c23a5a95  x     b
aws-node-6wpkg                                   2/2    Running  0             2d5h  10.8.97.206   i-065f30faa9db7f949  ✓     c
aws-node-9svms                                   2/2    Running  0             2d4h  10.8.129.170  i-0a76386295da6fe83  x     b
aws-node-b9w4r                                   2/2    Running  0             2d4h  10.8.130.112  i-0630694be7a879cc4  x     c
aws-node-bjvwn                                   2/2    Running  0             2d4h  10.8.82.184   i-08e004186079e74e2  ✓     b
aws-node-bvz9d                                   2/2    Running  0             1d5h  10.8.45.171   i-081f41e1d8e630e0c  ✓     a
aws-node-c9tc7                                   2/2    Running  0             2d4h  10.8.70.157   i-02c87764c5d7884b3  ✓     b
aws-node-jn988                                   2/2    Running  0             2d5h  10.8.124.42   i-0e63a4a348096dcf5  x     c
aws-node-prtkv                                   2/2    Running  0             2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
aws-node-qqpgk                                   2/2    Running  0             2d6h  10.8.36.6     i-0b568d75ecb3153d0  x     a
aws-node-termination-handler-74b857fdd7-cmmqf    1/1    Running  0             2d5h  10.8.45.111   i-0a41c827b6e581efe  ✓     a
aws-node-termination-handler-74b857fdd7-x6drw    1/1    Running  0             2d5h  10.8.111.158  i-065f30faa9db7f949  ✓     c
aws-node-vwmm6                                   2/2    Running  0             2d4h  10.8.128.98   i-0af469ea75aa4c82b  x     a
aws-node-xrv85                                   2/2    Running  0             2d5h  10.8.49.17    i-0a41c827b6e581efe  ✓     a
cert-manager-559975d55c-pw6wg                    1/1    Running  1 (2d6h ago)  2d6h  10.8.42.212   i-0b568d75ecb3153d0  x     a
cert-manager-cainjector-868f54ccf5-shvvf         1/1    Running  1 (2d5h ago)  2d5h  10.8.74.98    i-0ed7cb8a38a7b4d35  x     b
cert-manager-webhook-f8484455c-ks756             1/1    Running  0             2d6h  10.8.42.211   i-0b568d75ecb3153d0  x     a
coredns-7d47876df6-4w5l8                         1/1    Running  0             1d6h  10.8.128.227  i-0af469ea75aa4c82b  x     a
coredns-7d47876df6-6vvmx                         1/1    Running  0             2d4h  10.8.129.83   i-0a76386295da6fe83  x     b
coredns-7d47876df6-tbxvk                         1/1    Running  0             2d4h  10.8.130.33   i-0630694be7a879cc4  x     c
coredns-autoscaler-5fdfd9d499-ttrrl              1/1    Running  0             2d5h  10.8.111.10   i-065f30faa9db7f949  ✓     c
ebs-csi-controller-6dc5dcbbb8-67qkt              7/7    Running  0             2d5h  10.8.45.101   i-0a41c827b6e581efe  ✓     a
ebs-csi-controller-6dc5dcbbb8-bpm6d              7/7    Running  0             2d5h  10.8.114.85   i-065f30faa9db7f949  ✓     c
ebs-csi-node-62vs7                               3/3    Running  0             2d6h  10.8.42.208   i-0b568d75ecb3153d0  x     a
ebs-csi-node-6g72f                               3/3    Running  0             1d5h  10.8.47.192   i-081f41e1d8e630e0c  ✓     a
ebs-csi-node-6mhlz                               3/3    Running  0             2d5h  10.8.65.160   i-0f9bff5d2c23a5a95  x     b
ebs-csi-node-7rmzx                               3/3    Running  0             2d5h  10.8.45.96    i-0a41c827b6e581efe  ✓     a
ebs-csi-node-9qwsx                               3/3    Running  0             2d5h  10.8.114.208  i-0e63a4a348096dcf5  x     c
ebs-csi-node-9rbpp                               3/3    Running  0             2d5h  10.8.111.144  i-065f30faa9db7f949  ✓     c
ebs-csi-node-cckv4                               3/3    Running  0             2d4h  10.8.80.16    i-02c87764c5d7884b3  ✓     b
ebs-csi-node-d8v68                               3/3    Running  0             2d5h  10.8.74.96    i-0ed7cb8a38a7b4d35  x     b
ebs-csi-node-l5lfx                               3/3    Running  0             2d4h  10.8.128.224  i-0af469ea75aa4c82b  x     a
ebs-csi-node-vhljt                               3/3    Running  0             2d4h  10.8.76.144   i-08e004186079e74e2  ✓     b
ebs-csi-node-vn4kn                               3/3    Running  0             2d4h  10.8.130.16   i-0630694be7a879cc4  x     c
ebs-csi-node-wkvw9                               3/3    Running  0             2d5h  10.8.80.144   i-0ed734f56ed35c352  x     b
ebs-csi-node-xg94z                               3/3    Running  0             2d4h  10.8.129.80   i-0a76386295da6fe83  x     b
etcd-manager-events-i-0b568d75ecb3153d0          1/1    Running  0             2d6h  10.8.36.6     i-0b568d75ecb3153d0  x     a
etcd-manager-events-i-0e63a4a348096dcf5          1/1    Running  0             2d5h  10.8.124.42   i-0e63a4a348096dcf5  x     c
etcd-manager-events-i-0ed7cb8a38a7b4d35          1/1    Running  0             2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
etcd-manager-main-i-0b568d75ecb3153d0            1/1    Running  0             2d6h  10.8.36.6     i-0b568d75ecb3153d0  x     a
etcd-manager-main-i-0e63a4a348096dcf5            1/1    Running  0             2d5h  10.8.124.42   i-0e63a4a348096dcf5  x     c
etcd-manager-main-i-0ed7cb8a38a7b4d35            1/1    Running  0             2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
external-dns-78fbf59cd-b7knz                     1/1    Running  0             2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
kops-controller-6xhb6                            1/1    Running  0             2d5h  10.8.124.42   i-0e63a4a348096dcf5  x     c
kops-controller-gfdxd                            1/1    Running  0             2d6h  10.8.36.6     i-0b568d75ecb3153d0  x     a
kops-controller-gwrvn                            1/1    Running  0             2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
kube-apiserver-i-0b568d75ecb3153d0               2/2    Running  2 (2d6h ago)  2d6h  10.8.36.6     i-0b568d75ecb3153d0  x     a
kube-apiserver-i-0e63a4a348096dcf5               2/2    Running  2 (2d5h ago)  2d5h  10.8.124.42   i-0e63a4a348096dcf5  x     c
kube-apiserver-i-0ed7cb8a38a7b4d35               2/2    Running  2 (2d5h ago)  2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
kube-controller-manager-i-0b568d75ecb3153d0      1/1    Running  4 (2d6h ago)  2d6h  10.8.36.6     i-0b568d75ecb3153d0  x     a
kube-controller-manager-i-0e63a4a348096dcf5      1/1    Running  3 (2d5h ago)  2d5h  10.8.124.42   i-0e63a4a348096dcf5  x     c
kube-controller-manager-i-0ed7cb8a38a7b4d35      1/1    Running  4 (2d5h ago)  2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
kube-proxy-i-02c87764c5d7884b3                   1/1    Running  0             2d4h  10.8.70.157   i-02c87764c5d7884b3  ✓     b
kube-proxy-i-0630694be7a879cc4                   1/1    Running  0             2d4h  10.8.130.112  i-0630694be7a879cc4  x     c
kube-proxy-i-065f30faa9db7f949                   1/1    Running  0             2d5h  10.8.97.206   i-065f30faa9db7f949  ✓     c
kube-proxy-i-081f41e1d8e630e0c                   1/1    Running  0             1d5h  10.8.45.171   i-081f41e1d8e630e0c  ✓     a
kube-proxy-i-08e004186079e74e2                   1/1    Running  0             2d4h  10.8.82.184   i-08e004186079e74e2  ✓     b
kube-proxy-i-0a41c827b6e581efe                   1/1    Running  0             2d5h  10.8.49.17    i-0a41c827b6e581efe  ✓     a
kube-proxy-i-0a76386295da6fe83                   1/1    Running  0             2d4h  10.8.129.170  i-0a76386295da6fe83  x     b
kube-proxy-i-0af469ea75aa4c82b                   1/1    Running  0             2d4h  10.8.128.98   i-0af469ea75aa4c82b  x     a
kube-proxy-i-0b568d75ecb3153d0                   1/1    Running  0             2d6h  10.8.36.6     i-0b568d75ecb3153d0  x     a
kube-proxy-i-0e63a4a348096dcf5                   1/1    Running  0             2d5h  10.8.124.42   i-0e63a4a348096dcf5  x     c
kube-proxy-i-0ed734f56ed35c352                   1/1    Running  0             2d5h  10.8.87.250   i-0ed734f56ed35c352  x     b
kube-proxy-i-0ed7cb8a38a7b4d35                   1/1    Running  0             2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
kube-proxy-i-0f9bff5d2c23a5a95                   1/1    Running  0             2d5h  10.8.66.40    i-0f9bff5d2c23a5a95  x     b
kube-scheduler-i-0b568d75ecb3153d0               1/1    Running  0             2d6h  10.8.36.6     i-0b568d75ecb3153d0  x     a
kube-scheduler-i-0e63a4a348096dcf5               1/1    Running  0             2d5h  10.8.124.42   i-0e63a4a348096dcf5  x     c
kube-scheduler-i-0ed7cb8a38a7b4d35               1/1    Running  0             2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
metrics-server-97767c4f8-k2tnk                   1/1    Running  0             2d5h  10.8.111.155  i-065f30faa9db7f949  ✓     c
metrics-server-97767c4f8-srtx4                   1/1    Running  0             2d5h  10.8.48.96    i-0a41c827b6e581efe  ✓     a
node-local-dns-44qtk                             1/1    Running  0             2d5h  10.8.82.50    i-0ed7cb8a38a7b4d35  x     b
node-local-dns-667rq                             1/1    Running  0             2d4h  10.8.130.112  i-0630694be7a879cc4  x     c
node-local-dns-7df7b                             1/1    Running  0             2d4h  10.8.129.170  i-0a76386295da6fe83  x     b
node-local-dns-82gtf                             1/1    Running  0             2d5h  10.8.66.40    i-0f9bff5d2c23a5a95  x     b
node-local-dns-8hg7d                             1/1    Running  0             2d5h  10.8.124.42   i-0e63a4a348096dcf5  x     c
node-local-dns-b9m22                             1/1    Running  0             2d5h  10.8.49.17    i-0a41c827b6e581efe  ✓     a
node-local-dns-blqdd                             1/1    Running  0             2d6h  10.8.36.6     i-0b568d75ecb3153d0  x     a
node-local-dns-ksvdh                             1/1    Running  0             2d5h  10.8.97.206   i-065f30faa9db7f949  ✓     c
node-local-dns-lhdv9                             1/1    Running  0             2d4h  10.8.82.184   i-08e004186079e74e2  ✓     b
node-local-dns-rsppq                             1/1    Running  0             2d5h  10.8.87.250   i-0ed734f56ed35c352  x     b
node-local-dns-sk269                             1/1    Running  0             2d4h  10.8.70.157   i-02c87764c5d7884b3  ✓     b
node-local-dns-slz9d                             1/1    Running  0             2d4h  10.8.128.98   i-0af469ea75aa4c82b  x     a
node-local-dns-z92rb                             1/1    Running  0             1d5h  10.8.45.171   i-081f41e1d8e630e0c  ✓     a
node-problem-detector-4bnwl                      1/1    Running  0             2d4h  10.8.80.17    i-02c87764c5d7884b3  ✓     b
node-problem-detector-cdspw                      1/1    Running  0             2d4h  10.8.76.145   i-08e004186079e74e2  ✓     b
node-problem-detector-d7nkk                      1/1    Running  0             2d4h  10.8.129.81   i-0a76386295da6fe83  x     b
node-problem-detector-f5dpb                      1/1    Running  0             2d5h  10.8.74.97    i-0ed7cb8a38a7b4d35  x     b
node-problem-detector-fltf6                      1/1    Running  0             2d5h  10.8.111.145  i-065f30faa9db7f949  ✓     c
node-problem-detector-fvqtm                      1/1    Running  0             2d5h  10.8.65.161   i-0f9bff5d2c23a5a95  x     b
node-problem-detector-mwwck                      1/1    Running  0             2d6h  10.8.42.209   i-0b568d75ecb3153d0  x     a
node-problem-detector-nphm4                      1/1    Running  0             1d5h  10.8.46.0     i-081f41e1d8e630e0c  ✓     a
node-problem-detector-p48n6                      1/1    Running  0             2d4h  10.8.130.17   i-0630694be7a879cc4  x     c
node-problem-detector-pm56j                      1/1    Running  0             2d5h  10.8.115.112  i-0e63a4a348096dcf5  x     c
node-problem-detector-rdz5r                      1/1    Running  0             2d5h  10.8.45.97    i-0a41c827b6e581efe  ✓     a
node-problem-detector-wpc7h                      1/1    Running  0             2d4h  10.8.128.226  i-0af469ea75aa4c82b  x     a
node-problem-detector-z8q4s                      1/1    Running  0             2d5h  10.8.79.160   i-0ed734f56ed35c352  x     b
pod-identity-webhook-79974dcd9c-2nqzf            1/1    Running  0             2d4h  10.8.80.30    i-02c87764c5d7884b3  ✓     b
pod-identity-webhook-79974dcd9c-n276c            1/1    Running  0             2d5h  10.8.45.109   i-0a41c827b6e581efe  ✓     a
pod-identity-webhook-79974dcd9c-trg6r            1/1    Running  0             2d5h  10.8.114.94   i-065f30faa9db7f949  ✓     c
```

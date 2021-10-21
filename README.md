### aws-services

K8s Controller for create, delete, update, tag kinesis streams on AWS environment.

# example resource definition

```
apiVersion: awsservices.k8s.io/v1alpha1
kind: Stream
metadata:
  name: test-stream-1
  namespace: default
spec:
  streamName: test-stream-1
  shardCount: 1
  tags:
  - tagKey: owner
    tagValue: operations
  - tagKey: namespace
    tagValue: default
```

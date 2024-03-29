apiVersion: msvc.kloudlite.io/v1
kind: MySqlDB
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
  ownerReferences:
    - apiVersion: {{.APIVersion}}
      kind: {{.Kind}}
      name: {{.Name}}
      uid: {{.UID}}
      controller: true
      blockOwnerDeletion: true
spec:
  global:
    storageClass: do-block-storage
  auth:
    rootPassword: {{index .Spec.Inputs "root_password"}}
  architecture: replication
  primary:
    persistence:
      enabled: true
      size: {{index .Spec.Inputs "size" }}
  secondary:
    persistence:
      enabled: true
      size: {{index .Spec.Inputs "size" }}
    replicaCount: {{index .Spec.Inputs "replica_count" | default 1 }}

{{- $nodeSelector := get . "node-selector" -}}
{{- $name := get . "name"  -}}
{{- $namespace := get . "namespace"  -}}
{{- $ownerRefs := get . "owner-refs"  -}}

{{- $awsKey := get . "aws-key"  -}}
{{- $awsSecret := get . "aws-secret"  -}}

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cluster-svc-account
  namespace: {{$namespace}}
  ownerReferences: {{$ownerRefs | toYAML |nindent 4}}

---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{$namespace}}-cluster-svc-account-rb
  ownerReferences: {{$ownerRefs | toYAML |nindent 4}}
subjects:
  - kind: ServiceAccount
    name: cluster-svc-account
    namespace: {{$namespace}}
    apiGroup: ""
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: ""

---

apiVersion: csi.helm.kloudlite.io/v1
kind: AwsEbsCsiDriver
metadata:
  name: {{$name}}
  namespace: {{$namespace}}
  ownerReferences: {{$ownerRefs | toYAML |nindent 4}}
spec:
  fullnameOverride: {{$name}}
  driverName: {{$name}}
  controller:
    replicaCount: 1
    env:
      - name: AWS_ACCESS_KEY_ID
        value: {{$awsKey}}
      - name: AWS_SECRET_ACCESS_KEY
        value: {{$awsSecret}}
      - name: CSI_DRIVER_NAME
        value: "{{$name}}"
    serviceAccount:
      create: false
      name: "cluster-svc-account"
    {{- if $nodeSelector }}
    nodeSelector: {{$nodeSelector | toYAML | nindent 6}}
    {{- end }}
    tolerations:
      - effect: NoSchedule
        operator: Exists
      - effect: NoExecute
        operator: Exists
  node:
    env:
      - name: AWS_ACCESS_KEY_ID
        value: {{$awsKey}}
      - name: AWS_SECRET_ACCESS_KEY
        value: {{$awsSecret}}
      - name: CSI_DRIVER_NAME
        value: "{{$name}}"
    tolerateAllTaints: false
    {{- if $nodeSelector }}
    nodeSelector: {{$nodeSelector | toYAML| nindent 6}}
    {{- end }}
    tolerations:
      - effect: NoSchedule
        operator: Exists
      - effect: NoExecute
        operator: Exists
    serviceAccount:
      create: false
      name: "cluster-svc-account"
---

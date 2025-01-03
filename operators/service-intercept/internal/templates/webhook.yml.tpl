{{- with . }}
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: service-intercept
webhooks:
  - name: {{.ServiceName}}.kloudlite.io
    clientConfig:
      service:
        name: {{ .ServiceName | squote }}
        namespace: {{.ServiceNamespace | squote}}
        path: /mutate/pod
        port: 443
      caBundle: {{ .CaBundle | b64enc | squote }}

    namespaceSelector: {{.NamespaceSelector | toJson }}
    {{- /* matchExpressions: */}}
    {{- /*   - key: kloudlite.io/gateway.enabled */}}
    {{- /*     operator: In */}}
    {{- /*     values: ["true"] */}}

    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: [""]
        apiVersions: ["v1"]
        resources: ["pods"]
    admissionReviewVersions: ["v1"]
    sideEffects: None

---

apiVersion: v1
kind: Service
metadata:
  name: {{ .ServiceName }}
  namespace: {{.ServiceNamespace}}
spec:
  selector: {{.ServiceSelector | toJson }}
  ports:
    - name: webhook
      port: 443
      protocol: TCP
      targetPort: {{.ServiceHTTPSPort}}
{{- end }}

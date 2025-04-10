---
{{- with . }}

{{ if .EnableCodeServer }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: code-server-{{.Metadata.Name}}
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
spec:
  rules:
  - host: code-server.{{.Metadata.Name}}.{{.WorkMachineName}}.{{.KloudliteDomain}}
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: {{.Metadata.Name}}
            port:
              number: {{.PortConfig.CodeServerPort}}
{{ end }}

---

{{ if .EnableJupyterNotebook }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nb-{{.Metadata.Name}}
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
spec:
  rules:
  - host: notebook.{{.Metadata.Name}}.{{.WorkMachineName}}.{{.KloudliteDomain}}
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: {{.Metadata.Name}}
            port:
              number: {{.PortConfig.NotebookPort}}
{{ end }}

---
{{ if .EnableTTYD }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ttyd-{{.Metadata.Name}}
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
spec:
  rules:
  - host: ttyd.{{.Metadata.Name}}.{{.WorkMachineName}}.{{.KloudliteDomain}}
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: {{.Metadata.Name}}
            port:
              number: {{.PortConfig.TTYDPort}}
{{ end }}

{{- end }}

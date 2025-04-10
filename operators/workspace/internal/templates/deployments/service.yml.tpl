---
{{- with . }}
apiVersion: v1
kind: Service
metadata: {{.Metadata | toJson }}
spec:
  ports:
    - name: "ssh"
      protocol: "TCP"
      port: {{.PortConfig.SSHPort}}
      targetPort: {{.PortConfig.SSHPort}}

{{ if .EnableTTYD }}
    - name: "ttyd-server"
      protocol: "TCP"
      port: {{.PortConfig.TTYDPort}}
      targetPort: {{.PortConfig.TTYDPort}}
{{ end }}
    
{{ if .EnableJupyterNotebook }}
    - name: "jupyter-server"
      protocol: "TCP"
      port: {{.PortConfig.NotebookPort}}
      targetPort: {{.PortConfig.NotebookPort}}
{{ end }}

{{ if .EnableCodeServer }}
    - name: "code-server"
      protocol: "TCP"
      port: {{.PortConfig.CodeServerPort}}
      targetPort: {{.PortConfig.CodeServerPort}}
{{ end }}

{{- end }}

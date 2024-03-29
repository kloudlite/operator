piVersion: batch/v1
kind: Job
metadata:
  generateName: {{.Name}}-
  namespace: {{.Namespace}}
spec:
  template:
    spec:
      serviceAccount: {{.ServiceAccount}}
      containers:
        - name: {{.Name}}
          image: {{.Image}}
          imagePullPolicy: {{.ImagePullPolicy | default "Always"}}
        {{- if .Command }}
          command:
          {{- range .Command}}
          - {{. | squote}}
          {{- end}}
        {{- end }}
        {{- if .Args }}
          args:
            {{- range .Args}}
            - {{. | squote}}
            {{- end}}
        {{- end }}
        {{- if .Env }}
          env:
          {{- range $key, $value := .Env }}
            - name: {{$key}}
              value: {{$value}}
          {{- end}}
        {{- end }}
      restartPolicy: OnFailure
  backoffLimit: 2

{{- with . }}
apiVersion: apps/v1
kind: StatefulSet
metadata: {{.Metadata | toJson }}
spec:
  replicas: {{ if .IsOn }}1{{ else }}0{{ end }}
  selector:
    matchLabels:
      app: {{.Metadata.Name | squote}}
  template:
    metadata:
      labels:
        app: {{.Metadata.Name | squote}}
        kloudlite.io/gateway.enabled: "false"
    spec:
      securityContext:
        fsGroup: 1000
      hostname: {{.Metadata.Name}}
      nodeName: {{.WorkMachineName}}
      # serviceAccount: {{.ServiceAccountName | squote}}
      tolerations:
        - key: "kloudlite.io/workmachine.name"
          operator: "Equal"
          value: {{.WorkMachineName |squote}}
          effect: "NoExecute"
      initContainers:
        - name: init-home-dir
          image: {{.ImageInitContainer}}
          imagePullPolicy: Always
          env:
            - name: KL_WORKSPACE
              value: {{.Metadata.Name}}

            - name: HOME
              value: "/home/kl"

            - name: KL_BOX_MODE
              value: "true"

          securityContext:
            runAsUser: 1000
            runAsGroup: 1000
          command:
          - "bash"
          - "-c"
          - |
            set -e
            set +x
            if [ ! -d "/home/kl/.ssh" ]; then
              mkdir -p /home/kl/.ssh
            fi
            if [ -f "/home/kl/.ssh/authorized_keys" ]; then
              if ! cmp -s /tmp/authorized_keys /home/kl/.ssh/authorized_keys; then
                echo "authorized_keys file differs, copying new one"
                cp /tmp/authorized_keys /home/kl/.ssh/authorized_keys
              fi
              echo "authorized_keys file is up to date"
            else
              echo "authorized_keys file not found, copying new one"
              cp /tmp/authorized_keys /home/kl/.ssh/authorized_keys
            fi
            if [ ! -d "/nix/store" ]; then
              curl -L https://nixos.org/nix/install | sh
              mkdir -p ~/.config/nix
              echo 'experimental-features = nix-command flakes' > ~/.config/nix/nix.conf
            fi
            kl_bin_dir="/home/kl/.local/bin"
            if [ ! -f "$kl_bin_dir/kl" ]; then
              mkdir -p $kl_bin_dir
              pushd $kl_bin_dir
              curl https://i.jpillora.com/kloudlite/kl@v1.1.87-nightly | bash
              popd
            fi

            workspace_dir="/home/kl/workspaces/$(KL_WORKSPACE)"
            if [ ! -d "$workspace_dir" ]; then
              mkdir -p $workspace_dir
              pushd $workspace_dir
              export PATH=$PATH:/home/kl/.nix-profile/bin:/home/kl/.local/bin 
              kl init
              popd
            fi

            if [ -f "$workspace_dir/kl.yaml" ] || [ -f "$workspace_dir/kl.yml" ]; then
              pushd $workspace_dir
              PATH=$PATH:/home/kl/.nix-profile/bin:/home/kl/.local/bin /home/kl/.local/bin/kl shell -r >  /env/.env
              PATH=$PATH:/home/kl/.nix-profile/bin:/home/kl/.local/bin /home/kl/.local/bin/kl get env >  /env/.connected_env
              popd
            fi

            if [ ! -f "/home/kl/.zshrc" ]; then
              mkdir -p "/home/kl/.config/zsh"
              cp /tmp/.zshrc /home/kl/.zshrc
              cp /tmp/.aliasrc /home/kl/.config/aliasrc
            fi

            if [ ! -f "/home/kl/.local/bin/starship" ]; then
              curl -sS https://starship.rs/install.sh | sh -s -- -y -b /home/kl/.local/bin
            fi
            
            if [ ! -d "/home/kl/.config/zsh/zsh-autosuggestions" ]; then
              mkdir -p "/home/kl/.config/zsh"
              git clone https://github.com/zsh-users/zsh-autosuggestions /home/kl/.config/zsh/zsh-autosuggestions
            fi

            if [ ! -d "/home/kl/.config/zsh/zsh-syntax-highlighting" ]; then
              mkdir -p "/home/kl/.config/zsh"
              git clone https://github.com/zsh-users/zsh-syntax-highlighting.git  "/home/kl/.config/zsh/zsh-syntax-highlighting"
            fi

            {{- /* if [ ! -d "/home/kl/.kl" ]; then */}}
            {{- /*   mkdir -p /home/kl/.kl */}}
            {{- /*   sh -c 'cat > /home/kl/.kl/kl-session.yaml <<EOF */}}
            {{- /*   session: {{.KloudliteSessionID}} */}}
            {{- /*   team: {{.KloudliteTeam}} */}}
            {{- /*   EOF' */}}
            {{- /* fi */}}
            if [ ! -f "/home/kl/.local/bin/kubectl" ]; then
              pushd /home/kl/.local/bin
              curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
              chmod +x /home/kl/.local/bin/kubectl
              popd
            fi

          volumeMounts: &volume-mounts
            - mountPath: /home/kl
              name: home-dir
            
            - mountPath: /tmp/authorized_keys
              name: ssh-keys
              subPath: authorized_keys

            - mountPath: /nix
              name: nix-dir

            - mountPath: /env
              name: containerenv

      containers:
        - name: ssh
          image: {{.ImageSSH | squote}}
          imagePullPolicy: {{.ImagePullPolicy | default "IfNotPresent" }}
          env: &env
            - name: KL_WORKSPACE
              value: "{{.Metadata.Name}}"
            - name: KL_WORKSPACE_DIR
              value: "/home/kl/workspaces/{{.Metadata.Name}}"
            - name: KL_DEVICE_NAME
              value: {{.KloudliteDeviceFQDN}}
            - name: NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: DEPLOYMENT_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.labels['app']
          ports:
            - containerPort: 22
          volumeMounts: *volume-mounts

      {{ if .EnableTTYD }}
      - name: ttyd
        image: {{.ImageTTYD}}
        imagePullPolicy: {{.ImagePullPolicy}}
        env: *env
        ports:
        - containerPort: 54535
        volumeMounts: *volume-mounts
      {{ end }}

      {{ if .EnableJupyterNotebook }}
      - name: jupyter
        image: {{.ImageJupyterNotebook}}
        imagePullPolicy: {{.ImagePullPolicy}}
        env: *env
        ports:
        - containerPort: 8888
        volumeMounts: *volume-mounts
        securityContext:
          runAsUser: 1000
          runAsGroup: 1000
      {{ end }}

      {{ if .EnableCodeServer }}
      - name: code-server
        image: {{.ImageCodeServer}}
        imagePullPolicy: {{.ImagePullPolicy}}
        env: *env
        volumeMounts: *volume-mounts
        securityContext:
          runAsUser: 1000
          runAsGroup: 1000
      {{ end }}

      {{ if .EnableVSCodeServer }}
      - name: vscode-server
        {{- /* image: ghcr.io/kloudlite/iac/vscode-server:latest */}}
        image: {{.ImageVscodeServer}}
        imagePullPolicy: {{.ImagePullPolicy}}
        env: *env
        volumeMounts: *volume-mounts
        securityContext:
          runAsUser: 1000
          runAsGroup: 1000
      {{ end }}

      volumes:
      - name: containerenv
        emptyDir: {}
      
      - name: home-dir
        hostPath:
          path: /external-volume/user-home

      - name: nix-dir
        hostPath:
          path: /external-volume/nix
      
      - name: ssh-keys
        secret:
          secretName: ssh-public-keys
          defaultMode: 0400
          items:
          - key: authorized_keys
            path: authorized_keys
---
apiVersion: v1
kind: Service
metadata: {{.Metadata | toJson }}
spec:
  ports:
    - name: "ssh"
      protocol: "TCP"
      port: 22
      targetPort: 22

    - name: "ttyd-server"
      protocol: "TCP"
      port: 54535
      targetPort: 54535

    - name: "jupyter-server"
      protocol: "TCP"
      port: 8888
      targetPort: 8888

    - name: "code-server"
      protocol: "TCP"
      port: 8080
      targetPort: 8080
---

apiVersion: crds.kloudlite.io/v1
kind: Router
metadata: {{.Metadata | toJson }}
spec: {{.RouterSpec | toJson }}
---
{{- end }}

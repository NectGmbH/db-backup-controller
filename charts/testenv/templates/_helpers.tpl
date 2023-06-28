{{- define "com.nect.cockroachdb.fullname" -}}
  {{- $name := "cockroachdb" -}}
  {{- if contains $name .Release.Name -}}
      {{- .Release.Name | trunc 56 | trimSuffix "-" -}}
  {{- else -}}
      {{- printf "%s-%s" .Release.Name $name | trunc 56 | trimSuffix "-" -}}
  {{- end -}}
{{- end -}}

{{- define "com.nect.mount-crdb-certs-volume" -}}
- name: cockroach-certs
  projected:
    defaultMode: 0o400
    sources:
    - secret:
        items:
          - key: ca.crt
            mode: 0o400
            path: ca.crt
          - key: tls.crt
            mode: 0o400
            path: client.root.crt
          - key: tls.key
            mode: 0o400
            path: client.root.key
        name: crdb-ca-root
{{- end -}}


{{- define "com.nect.wait-crdb-conn" -}}
- name: wait-for-crdb-conn
  image: {{ .Values.cockroachdb.image.repository }}:{{ .Values.cockroachdb.image.tag }}
  command:
    - sh
    - -c
    - |
      # VERY dirty hack: Mount has 0o440 perms on the keys, libpq is
      # pissing itself over that so we take the keys out of there and
      # apply "proper" permissions
      mkdir /cockroach-certs-permfix
      cp /cockroach-certs/* /cockroach-certs-permfix/
      chmod 0400 /cockroach-certs-permfix/*

      until /cockroach/cockroach sql \
        --host={{ include "com.nect.cockroachdb.fullname" . }}-public \
        --port={{ .Values.cockroachdb.conf.port }} \
        --certs-dir=/cockroach-certs-permfix \
        --user=root \
        --execute="SHOW DATABASES;"; do
        echo "waiting for db..."
        sleep 2
      done
  resources:
    limits:
      cpu: 200m
      memory: "1Gi"
    requests:
      memory: "1Gi"
  volumeMounts:
  - name: cockroach-certs
    mountPath: /cockroach-certs
{{- end -}}

{{- define "com.nect.wait-db-exists" -}}
- name: wait-for-db
  image: {{ .Values.cockroachdb.image.repository }}:{{ .Values.cockroachdb.image.tag }}
  command:
    - sh
    - -c
    - |
      # VERY dirty hack: Mount has 0o440 perms on the keys, libpq is
      # pissing itself over that so we take the keys out of there and
      # apply "proper" permissions
      mkdir /cockroach-certs-permfix
      cp /cockroach-certs/* /cockroach-certs-permfix/
      chmod 0400 /cockroach-certs-permfix/*

      until /cockroach/cockroach sql \
        --host={{ include "com.nect.cockroachdb.fullname" . }}-public \
        --port={{ .Values.cockroachdb.conf.port }} \
        --certs-dir=/cockroach-certs-permfix \
        --user=root \
        --execute="SHOW DATABASES;" | grep -q "^\s*{{ .Values.database.database }}\s"; do
        echo "waiting for db..."
        sleep 2
      done
  resources:
    limits:
      cpu: 200m
      memory: "1Gi"
    requests:
      memory: "1Gi"
  volumeMounts:
  - name: cockroach-certs
    mountPath: /cockroach-certs
{{- end -}}


{{/* vim: set ft=mustache: */}}

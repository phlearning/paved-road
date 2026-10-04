{{- define "service.name" -}}
{{- required "name is required" .Values.name -}}
{{- end -}}

{{- define "service.host" -}}
{{- default (printf "%s.localhost" (include "service.name" .)) .Values.ingress.host -}}
{{- end -}}

{{- define "service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "service.labels" -}}
{{ include "service.selectorLabels" . }}
app.kubernetes.io/version: {{ .Values.image.tag | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: paved-road
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}

{{/* Environment shared by the service container and its jobs. */}}
{{- define "service.env" -}}
- name: PORT
  value: {{ .Values.port | quote }}
{{- if .Values.postgres.enabled }}
- name: DATABASE_URL
  valueFrom:
    secretKeyRef:
      name: {{ include "service.name" . }}-db-app
      key: uri
{{- end }}
{{- with .Values.env }}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{- define "service.envFrom" -}}
{{- if .Values.secrets }}
envFrom:
  - secretRef:
      name: {{ include "service.name" . }}-secrets
{{- end }}
{{- end -}}

{{/* Non-root, read-only and without capabilities: also what OpenShift's restricted SCC expects. */}}
{{- define "service.podSecurityContext" -}}
runAsNonRoot: true
seccompProfile:
  type: RuntimeDefault
{{- end -}}

{{- define "service.containerSecurityContext" -}}
allowPrivilegeEscalation: false
readOnlyRootFilesystem: true
capabilities:
  drop: ["ALL"]
{{- end -}}

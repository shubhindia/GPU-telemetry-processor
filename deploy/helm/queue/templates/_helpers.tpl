{{- define "queue.name" -}}
{{- .Chart.Name -}}
{{- end }}

{{- define "queue.fullname" -}}
  {{- printf "%s-%s" .Release.Name (include "queue.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{- define "queue.labels" -}}
app.kubernetes.io/name: {{ include "queue.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

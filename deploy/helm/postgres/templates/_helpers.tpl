{{- define "postgres.name" -}}
{{- .Chart.Name -}}
{{- end }}

{{- define "postgres.fullname" -}}
  {{- printf "%s-%s" .Release.Name (include "postgres.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{- define "postgres.labels" -}}
app.kubernetes.io/name: {{ include "postgres.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

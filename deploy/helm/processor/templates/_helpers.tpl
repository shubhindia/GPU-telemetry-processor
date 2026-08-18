{{- define "processor.name" -}}
{{- .Chart.Name -}}
{{- end }}

{{- define "processor.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "processor.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{- define "processor.labels" -}}
app.kubernetes.io/name: {{ include "processor.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

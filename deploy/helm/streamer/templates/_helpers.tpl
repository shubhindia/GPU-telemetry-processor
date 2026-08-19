{{- define "streamer.name" -}}
{{- .Chart.Name -}}
{{- end }}

{{- define "streamer.fullname" -}}
  {{- printf "%s-%s" .Release.Name (include "streamer.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{- define "streamer.labels" -}}
app.kubernetes.io/name: {{ include "streamer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "streamer.csvPath" -}}
  {{- if .Values.csv.path -}}
{{- .Values.csv.path -}}
  {{- else -}}
    {{- printf "%s/%s" (trimSuffix "/" .Values.csv.mountPath) .Values.csv.fileName -}}
  {{- end -}}
{{- end }}

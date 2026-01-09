{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cert.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "app.name.nginx" -}}
cert-nginx
{{- end }}

{{- define "app.fullname.nginx" -}}
cert-nginx
{{- end }}

{{- define "app.labels.nginx" -}}
helm.sh/chart: {{ include "cert.chart" . | quote}}
{{ include "app.selectorLabels.nginx" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service | quote }}
{{- end }}

{{- define "app.selectorLabels.nginx" -}}
app.kubernetes.io/name: {{ include "app.name.nginx" . | quote }}
app.kubernetes.io/instance: {{ .Release.Name | quote }}
{{- end }}



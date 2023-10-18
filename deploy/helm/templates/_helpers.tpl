{{/* app name */}}
{{- define "ubt.name" -}}
{{- printf "ubt" -}}
{{- end -}}

{{/* chart name and version */}}
{{- define "ubt.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* helm required labels */}}
{{- define "ubt.labels" -}}
app.kubernetes.io/name: {{ template "ubt.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ template "ubt.chart" . }}
{{- end -}}

{{/* matchLabels */}}
{{- define "ubt.matchLabels" -}}
app.kubernetes.io/name: {{ template "ubt.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* image url */}}
{{- define "ubt.image" -}}
{{- printf "%s/%s:%s" .Values.image.registry .Values.image.repository .Values.image.tag -}}
{{- end -}}

{{/* app name */}}
{{- define "ubtam.name" -}}
{{- printf "ubt-am" -}}
{{- end -}}

{{/* helm required labels */}}
{{- define "ubtam.labels" -}}
app.kubernetes.io/name: {{ template "ubtam.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ template "ubt.chart" . }}
{{- end -}}

{{/* matchLabels */}}
{{- define "ubtam.matchLabels" -}}
app.kubernetes.io/name: {{ template "ubtam.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* image url */}}
{{- define "ubtam.image" -}}
{{- printf "%s/%s:%s" .Values.am.image.registry .Values.am.image.repository .Values.am.image.tag -}}
{{- end -}}

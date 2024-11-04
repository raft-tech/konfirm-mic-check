{{/*
Expand the name of the chart.
*/}}
{{- define "inspect.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified base name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "inspect.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create a fully qualified http inspection name.
We truncate at 63 chars in the same manner as inspect.fullname above.
*/}}
{{- define "inspect.httpName" }}
{{- printf "%s-%s" (include "inspect.fullname" . | trunc 58 | trimSuffix "-") "http" }}
{{- end }}

{{/*
Create a fully qualified http server name.
We truncate at 63 chars in the same manner as inspect.fullname above.
*/}}
{{- define "inspect.httpServerName" }}
{{- printf "%s-%s" (include "inspect.fullname" . | trunc 52 | trimSuffix "-") "http-server" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "inspect.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "inspect.labels" -}}
helm.sh/chart: {{ include "inspect.chart" . }}
{{ include "inspect.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "inspect.selectorLabels" -}}
app.kubernetes.io/name: {{ include "inspect.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

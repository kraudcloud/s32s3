{{/*
Expand the name of the chart.
*/}}
{{- define "s32s3.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "s32s3.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "s32s3.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "s32s3.labels" -}}
helm.sh/chart: {{ include "s32s3.chart" . }}
{{ include "s32s3.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "s32s3.selectorLabels" -}}
app.kubernetes.io/name: {{ include "s32s3.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "s32s3.envRequired" -}}
{{- $name := index . 0 -}}
{{- $env := index . 1 -}}
{{- $value := index . 2 -}}
- name: {{ $env | quote }}
{{- if $value.value }}
  value: {{ $value.value | quote }}
{{- else if $value.valueFrom }}
  valueFrom:
    {{- toYaml $value.valueFrom | nindent 4 }}
{{- else }}
{{- fail (printf ".%s must have either value or valueFrom specified" $name)}}
{{- end }}
{{- end }}

{{- define "s32s3.env" -}}
{{- $name := index . 0 -}}
{{- $env := index . 1 -}}
{{- $value := index . 2 -}}
{{- if $value.value }}
- name: {{ $env | quote }}
  value: {{ $value.value | quote }}
{{- else if $value.valueFrom }}
- name: {{ $env | quote }}
  valueFrom:
    {{- toYaml $value.valueFrom | nindent 4 }}
{{- else }}
{{- end }}
{{- end }}

{{- define "s32s3.image" -}}
{{- .Values.image.name -}}:{{- .Values.image.tag | default .Chart.AppVersion -}}
{{- end }}
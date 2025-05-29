{{/*
Expand the name of the chart.
*/}}
{{- define "snapshots-quota.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "snapshots-quota.fullname" -}}
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
{{- define "snapshots-quota.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "snapshots-quota.labels" -}}
helm.sh/chart: {{ include "snapshots-quota.chart" . }}
{{ include "snapshots-quota.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "snapshots-quota.selectorLabels" -}}
app.kubernetes.io/name: {{ include "snapshots-quota.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Define containerd socket path
*/}}
{{- define "snapshots-quota.containerd_socket" -}}
{{- if .Values.nri.plugin.containerd_socket }}
{{- .Values.nri.plugin.containerd_socket }}
{{- else }}
{{- "/run/containerd/containerd.sock" }}
{{- end }}
{{- end }}

{{/*
Define containerd host base path
*/}}
{{- define "snapshots-quota.containerd_host_base_path" -}}
{{- if .Values.nri.plugin.containerd_host_base_path }}
{{- .Values.nri.plugin.containerd_host_base_path }}
{{- else }}
{{- "/" }}
{{- end }}
{{- end }}

{{/*
Define containerd container base path
*/}}
{{- define "snapshots-quota.containerd_container_base_path" -}}
{{- if .Values.nri.plugin.containerd_container_base_path }}
{{- .Values.nri.plugin.containerd_container_base_path }}
{{- else }}
{{- "/data" }}
{{- end }}
{{- end }}

{{/*
Define containerd root dir
*/}}
{{- define "snapshots-quota.containerd_root_dir" -}}
{{- if .Values.nri.plugin.containerd_root_dir }}
{{- .Values.nri.plugin.containerd_root_dir }}
{{- else }}
{{- "/var/lib/containerd" }}
{{- end }}
{{- end }}


{{/*
Define containerd state dir
*/}}
{{- define "snapshots-quota.containerd_state_dir" -}}
{{- if .Values.nri.plugin.containerd_state_dir }}
{{- .Values.nri.plugin.containerd_state_dir }}
{{- else }}
{{- "/run/containerd" }}
{{- end }}
{{- end }}
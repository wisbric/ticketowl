{{/*
Expand the name of the chart.
*/}}
{{- define "ticketowl.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "ticketowl.fullname" -}}
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
{{- define "ticketowl.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "ticketowl.labels" -}}
helm.sh/chart: {{ include "ticketowl.chart" . }}
{{ include "ticketowl.selectorLabels" . }}
app.kubernetes.io/version: {{ .Values.image.tag | default .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "ticketowl.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ticketowl.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Service account name
*/}}
{{- define "ticketowl.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "ticketowl.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Secret name — uses existing secret or chart-created secret.
*/}}
{{- define "ticketowl.secretName" -}}
{{- if and (not .Values.secrets.create) .Values.secrets.existingSecret }}
{{- .Values.secrets.existingSecret }}
{{- else }}
{{- include "ticketowl.fullname" . }}
{{- end }}
{{- end }}

{{/*
Image reference
*/}}
{{- define "ticketowl.image" -}}
{{- printf "%s:%s" .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) }}
{{- end }}

{{/*
Common environment variables shared by api and worker deployments.
Avoids duplicating the full env block in both deployment templates.
*/}}
{{- define "ticketowl.commonEnv" -}}
- name: TICKETOWL_DB_URL
  valueFrom:
    secretKeyRef:
      name: {{ include "ticketowl.secretName" . }}
      key: TICKETOWL_DB_URL
- name: TICKETOWL_REDIS_URL
  valueFrom:
    secretKeyRef:
      name: {{ include "ticketowl.secretName" . }}
      key: TICKETOWL_REDIS_URL
- name: TICKETOWL_OIDC_ISSUER
  valueFrom:
    secretKeyRef:
      name: {{ include "ticketowl.secretName" . }}
      key: TICKETOWL_OIDC_ISSUER
- name: TICKETOWL_OIDC_CLIENT_ID
  valueFrom:
    secretKeyRef:
      name: {{ include "ticketowl.secretName" . }}
      key: TICKETOWL_OIDC_CLIENT_ID
- name: TICKETOWL_ENCRYPTION_KEY
  valueFrom:
    secretKeyRef:
      name: {{ include "ticketowl.secretName" . }}
      key: TICKETOWL_ENCRYPTION_KEY
- name: TICKETOWL_NIGHTOWL_API_KEY
  valueFrom:
    secretKeyRef:
      name: {{ include "ticketowl.secretName" . }}
      key: TICKETOWL_NIGHTOWL_API_KEY
- name: TICKETOWL_BOOKOWL_API_KEY
  valueFrom:
    secretKeyRef:
      name: {{ include "ticketowl.secretName" . }}
      key: TICKETOWL_BOOKOWL_API_KEY
- name: TICKETOWL_LOG_LEVEL
  valueFrom:
    configMapKeyRef:
      name: {{ include "ticketowl.fullname" . }}
      key: TICKETOWL_LOG_LEVEL
- name: TICKETOWL_LOG_FORMAT
  valueFrom:
    configMapKeyRef:
      name: {{ include "ticketowl.fullname" . }}
      key: TICKETOWL_LOG_FORMAT
- name: TICKETOWL_PORT
  valueFrom:
    configMapKeyRef:
      name: {{ include "ticketowl.fullname" . }}
      key: TICKETOWL_PORT
- name: TICKETOWL_WORKER_POLL_SECONDS
  valueFrom:
    configMapKeyRef:
      name: {{ include "ticketowl.fullname" . }}
      key: TICKETOWL_WORKER_POLL_SECONDS
{{- if .Values.config.nightowlApiUrl }}
- name: TICKETOWL_NIGHTOWL_API_URL
  valueFrom:
    configMapKeyRef:
      name: {{ include "ticketowl.fullname" . }}
      key: TICKETOWL_NIGHTOWL_API_URL
{{- end }}
{{- if .Values.config.bookowlApiUrl }}
- name: TICKETOWL_BOOKOWL_API_URL
  valueFrom:
    configMapKeyRef:
      name: {{ include "ticketowl.fullname" . }}
      key: TICKETOWL_BOOKOWL_API_URL
{{- end }}
{{- if .Values.config.otelEndpoint }}
- name: TICKETOWL_OTEL_ENDPOINT
  valueFrom:
    configMapKeyRef:
      name: {{ include "ticketowl.fullname" . }}
      key: TICKETOWL_OTEL_ENDPOINT
{{- end }}
{{- end }}

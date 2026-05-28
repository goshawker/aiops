{{/*
Expand the name of the chart.
*/}}
{{- define "aiops.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "aiops.fullname" -}}
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
Common labels
*/}}
{{- define "aiops.labels" -}}
helm.sh/chart: {{ include "aiops.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Go service deployment
*/}}
{{- define "aiops.goService" -}}
{{- $name := index . 0 }}
{{- $root := index . 1 }}
{{- $svc := index $root.Values $name }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "aiops.fullname" $root }}-{{ $name }}
  labels:
    {{- include "aiops.labels" $root | nindent 4 }}
    app.kubernetes.io/component: {{ $name }}
spec:
  replicas: {{ $svc.replicaCount | default 1 }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ $name }}
      app.kubernetes.io/instance: {{ $root.Release.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ $name }}
        app.kubernetes.io/instance: {{ $root.Release.Name }}
    spec:
      containers:
        - name: {{ $name }}
          image: "{{ $svc.image.repository }}:{{ $svc.image.tag }}"
          imagePullPolicy: {{ $svc.image.pullPolicy | default "IfNotPresent" }}
          ports:
            - containerPort: {{ $svc.service.port }}
          livenessProbe:
            httpGet:
              path: /health
              port: {{ $svc.service.port }}
            initialDelaySeconds: 5
          readinessProbe:
            httpGet:
              path: /health
              port: {{ $svc.service.port }}
            initialDelaySeconds: 5
          {{- if $svc.resources }}
          resources:
            {{- toYaml $svc.resources | nindent 12 }}
          {{- end }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ $name }}
  labels:
    {{- include "aiops.labels" $root | nindent 4 }}
spec:
  type: {{ $svc.service.type | default "ClusterIP" }}
  ports:
    - port: {{ $svc.service.port }}
      targetPort: {{ $svc.service.port }}
      protocol: TCP
  selector:
    app.kubernetes.io/name: {{ $name }}
    app.kubernetes.io/instance: {{ $root.Release.Name }}
{{- end }}

{{/*
Python AI service deployment
*/}}
{{- define "aiops.pythonService" -}}
{{- $name := index . 0 }}
{{- $root := index . 1 }}
{{- $svc := index $root.Values $name }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "aiops.fullname" $root }}-{{ $name }}
  labels:
    {{- include "aiops.labels" $root | nindent 4 }}
    app.kubernetes.io/component: {{ $name }}
spec:
  replicas: {{ $svc.replicaCount | default 1 }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ $name }}
      app.kubernetes.io/instance: {{ $root.Release.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ $name }}
        app.kubernetes.io/instance: {{ $root.Release.Name }}
    spec:
      containers:
        - name: {{ $name }}
          image: "{{ $svc.image.repository }}:{{ $svc.image.tag }}"
          imagePullPolicy: {{ $svc.image.pullPolicy | default "IfNotPresent" }}
          ports:
            - containerPort: {{ $svc.service.port }}
          livenessProbe:
            httpGet:
              path: /health
              port: {{ $svc.service.port }}
            initialDelaySeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: {{ $svc.service.port }}
            initialDelaySeconds: 10
          {{- if $svc.resources }}
          resources:
            {{- toYaml $svc.resources | nindent 12 }}
          {{- end }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ $name }}
  labels:
    {{- include "aiops.labels" $root | nindent 4 }}
spec:
  type: {{ $svc.service.type | default "ClusterIP" }}
  ports:
    - port: {{ $svc.service.port }}
      targetPort: {{ $svc.service.port }}
      protocol: TCP
  selector:
    app.kubernetes.io/name: {{ $name }}
    app.kubernetes.io/instance: {{ $root.Release.Name }}
{{- end }}

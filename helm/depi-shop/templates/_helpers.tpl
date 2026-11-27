{{/* <registry>/<service>:<tag>  (one ECR repo per service) */}}
{{/* includeServiceInTag=true  -> tag is <service>-<tag>  (e.g. frontend-abc1234) */}}
{{- define "depi.image" -}}
{{- $tag := required "image.tag is required" (default .root.Values.image.tag .svc.tag) -}}
{{- if .root.Values.image.includeServiceInTag -}}
{{- $tag = printf "%s-%s" .name $tag -}}
{{- end -}}
{{- printf "%s/%s:%s" (required "image.registry is required (e.g. <account>.dkr.ecr.<region>.amazonaws.com/online-boutique)" .root.Values.image.registry) .name $tag -}}
{{- end -}}

{{- define "depi.labels" -}}
app: {{ .name }}
app.kubernetes.io/part-of: depi-shop
app.kubernetes.io/managed-by: {{ .root.Release.Service }}
{{- end -}}

{{- define "depi.ordersSecretName" -}}
{{- default "orders-db" .Values.ordersDb.existingSecret -}}
{{- end -}}

{{- define "depi.probeHandler" -}}
{{- if eq .type "http" -}}
httpGet:
  path: {{ default "/" .path }}
  port: {{ .port }}
{{- else if eq .type "grpc" -}}
grpc:
  port: {{ .port }}
{{- else -}}
tcpSocket:
  port: {{ .port }}
{{- end -}}
{{- end -}}

{{- define "depi.redisAddr" -}}
{{- required "redis.address is required (ElastiCache endpoint:6379)" .Values.redis.address -}}
{{- end -}}

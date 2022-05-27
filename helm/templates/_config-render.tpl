###~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~###
###                jackal configuration file                 ###
###~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~###

logger:
  level: {{ .Values.jackal.config.logger.level }}

http:
  port: {{ .Values.jackal.config.http.port }}

admin:
  port: {{ .Values.jackal.config.admin.port }}

{{ if .Values.jackal.config.domains }}
hosts:
{{ toYaml .Values.jackal.config.domains | indent 6 }}
{{ end }}

{{ if .Values.jackal.config.peppers }}
peppers:
{{ toYaml .Values.jackal.config.peppers | indent 6 }}
{{ end }}

storage:
  type: pgsql
  pgsql:
    host: jackal-postgresql-ha-pgpool.{{ .Release.Namespace }}.svc.cluster.local:5432
    user: jackal
    database: jackal
    max_open_conns: {{ .Values.jackal.config.storage.maxConns }}
    max_idle_conns: {{ .Values.jackal.config.storage.maxIdleConns }}
    conn_max_lifetime: {{ .Values.jackal.config.storage.connMaxLifetime }}
    conn_max_idle_time: {{ .Values.jackal.config.storage.connMaxIdleTime }}

{{ if .Values.redis.enabled }}
  cache:
    type: redis
    redis:
      srv: _redis._tcp.redis-headless.{{ .Release.Namespace }}.svc.cluster.local
{{ end }}

cluster:
  type: kv
  kv:
    type: etcd
    etcd:
      endpoints:
      - http://jackal-etcd.{{ .Release.Namespace }}.svc.cluster.local:{{ .Values.etcd.containerPorts.client }}

  server:
    port: {{ .Values.jackal.config.cluster.server.port }}

{{ if .Values.jackal.config.shapers }}
shapers:
{{ toYaml .Values.jackal.config.shapers | indent 2 }}
{{ end }}

c2s:
{{ toYaml .Values.jackal.config.c2s | indent 2 }}

s2s:
{{ toYaml .Values.jackal.config.s2s | indent 2 }}

{{ if .Values.jackal.config.modules }}
modules:
{{ toYaml .Values.jackal.config.modules | indent 2 }}
{{ end }}

{{ if .Values.jackal.config.components }}
components:
{{ toYaml .Values.jackal.config.components | indent 2 }}
{{ end }}

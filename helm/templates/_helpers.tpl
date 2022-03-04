{{/*
Calculate the config from structured and unstructred text input
*/}}
{{- define "jackal.calculatedConfig" -}}
{{ include (print $.Template.BasePath "/_config-render.tpl") . }}
{{- end -}}

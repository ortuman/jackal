#!/usr/bin/env bash
set -eufo pipefail

command -v kubectl >/dev/null 2>&1 || { echo "kubectl not installed, aborting." >&2; exit 1; }
command -v helm >/dev/null 2>&1 || { echo "helm not installed, aborting." >&2; exit 1; }

if [ $# -eq 0 ] || [ -z $1 ]; then
   echo "A custom values.yaml file must be provided"
   exit 1;
fi

export POSTGRESQL_PASSWORD=$(kubectl get secret --namespace "jackal" jackal-postgresql-ha-postgresql -o jsonpath="{.data.postgresql-password}" | base64 --decode)
export REPMGR_PASSWORD=$(kubectl get secret --namespace "jackal" jackal-postgresql-ha-postgresql -o jsonpath="{.data.repmgr-password}" | base64 --decode)
export ADMIN_PASSWORD=$(kubectl get secret --namespace "jackal" jackal-postgresql-ha-pgpool -o jsonpath="{.data.admin-password}" | base64 --decode)

helm upgrade jackal helm/ --dependency-update \
--set postgresql-ha.postgresql.password=$POSTGRESQL_PASSWORD \
--set postgresql-ha.postgresql.repmgrPassword=$REPMGR_PASSWORD \
--set postgresql-ha.pgpool.adminPassword=$ADMIN_PASSWORD \
--namespace=jackal \
-f "$1"

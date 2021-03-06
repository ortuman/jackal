module github.com/ortuman/jackal

go 1.16

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/Masterminds/squirrel v1.1.0
	github.com/bgentry/speakeasy v0.1.0
	github.com/cockroachdb/errors v1.8.4
	github.com/cockroachdb/sentry-go v0.6.1-cockroachdb.2
	github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.1.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/jackal-xmpp/runqueue/v2 v2.0.0
	github.com/jackal-xmpp/stravaganza/v2 v2.0.0
	github.com/kkyr/fig v0.2.0
	github.com/lib/pq v1.8.0
	github.com/mattn/go-sqlite3 v1.14.5 // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e
	google.golang.org/grpc v1.33.0
	google.golang.org/protobuf v1.26.0
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

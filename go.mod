module github.com/drone/drone

replace github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20200309214505-aa6a9891b09c+incompatible

require (
	github.com/99designs/httpsignatures-go v0.0.0-20170731043157-88528bf4ca7e
	github.com/Azure/azure-storage-blob-go v0.7.0
	github.com/Azure/go-autorest/autorest/adal v0.8.3 // indirect
	github.com/asaskevich/govalidator v0.0.0-20180315120708-ccb8e960c48f
	github.com/aws/aws-sdk-go v1.37.3
	github.com/codegangsta/negroni v1.0.0 // indirect
	github.com/coreos/go-semver v0.2.0
	github.com/dchest/authcookie v0.0.0-20120917135355-fbdef6e99866
	github.com/dchest/uniuri v0.0.0-20160212164326-8902c56451e9
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/drone/drone-go v1.4.1-0.20201109202657-b9e58bbbcf27
	github.com/drone/drone-runtime v1.1.1-0.20200623162453-61e33e2cab5d
	github.com/drone/drone-ui v0.0.0-20210702192143-f81f74399385
	github.com/drone/drone-yaml v1.2.4-0.20200326192514-6f4d6dfb39e4
	github.com/drone/envsubst v1.0.3-0.20200709231038-aa43e1c1a629
	github.com/drone/go-license v1.0.2
	github.com/drone/go-login v1.0.4-0.20190311170324-2a4df4f242a2
	github.com/drone/go-scm v1.15.1
	github.com/drone/signal v1.0.0
	github.com/dustin/go-humanize v1.0.0
	github.com/go-chi/chi v3.3.3+incompatible
	github.com/go-chi/cors v1.0.0
	github.com/go-sql-driver/mysql v1.4.0
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.4 // indirect
	github.com/google/go-cmp v0.4.0
	github.com/google/go-jsonnet v0.17.0
	github.com/google/wire v0.2.1
	github.com/gosimple/slug v1.3.0
	github.com/h2non/gock v1.0.15
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/go-retryablehttp v0.5.4
	github.com/hashicorp/golang-lru v0.5.1
	github.com/jmoiron/sqlx v0.0.0-20180614180643-0dae4fefe7c0
	github.com/joho/godotenv v1.3.0
	github.com/kelseyhightower/envconfig v1.3.0
	github.com/kr/pretty v0.2.0 // indirect
	github.com/lib/pq v1.1.0
	github.com/mattn/go-sqlite3 v1.9.0
	github.com/oxtoacart/bpool v0.0.0-20150712133111-4e1c5567d7c2
	github.com/prometheus/client_golang v0.9.2
	github.com/rainycape/unidecode v0.0.0-20150907023854-cb7f23ec59be // indirect
	github.com/robfig/cron v0.0.0-20180505203441-b41be1df6967
	github.com/segmentio/ksuid v1.0.2
	github.com/sirupsen/logrus v1.6.0
	github.com/unrolled/secure v0.0.0-20181022170031-4b6b7cf51606
	go.starlark.net v0.0.0-20201118183435-e55f603d8c79
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/yaml.v2 v2.2.8
)

replace github.com/h2non/gock => gopkg.in/h2non/gock.v1 v1.0.14

go 1.13

SHA := $(shell git rev-parse --short HEAD)

all: build

deps:
	# which npm && npm -g install uglify-js less
	go get github.com/GeertJohan/go.rice/rice
	go get github.com/franela/goblin
	go list github.com/drone/drone/... | xargs go get -t -v

test:
	# Test with SQLite in memory
	go vet ./...
	DRONE_DATABASE_DRIVER="sqlite3" DRONE_DATABASE_DATASOURCE=":memory:" go test -cover -short ./...

ci_test: test test_psql test_mysql

test_psql:
	# Test with PostreSQL
	DRONE_DATABASE_DRIVER="postgres" DRONE_DATABASE_DATASOURCE="user=postgres host=127.0.0.1 port=5432 dbname=drone sslmode=disable" go test -cover -short ./...

test_mysql:
	# Test with MySQL
	DRONE_DATABASE_DRIVER="mysql" DRONE_DATABASE_DATASOURCE="root@tcp(127.0.0.1:3306)/drone" go test -cover -short ./...

build:
	go build -o debian/drone/usr/local/bin/drone  -ldflags "-X main.revision $(SHA)" github.com/drone/drone/cmd
	go build -o debian/drone/usr/local/bin/droned -ldflags "-X main.revision $(SHA)" github.com/drone/drone/server

install:
	install -t /usr/local/bin debian/drone/usr/local/bin/drone 
	install -t /usr/local/bin debian/drone/usr/local/bin/droned 

run:
	@go run server/main.go

clean:
	find . -name "*.out" -delete
	rm -f debian/drone/usr/local/bin/drone
	rm -f debian/drone/usr/local/bin/droned
	rm -f debian/drone.deb
	rm -f server/server
	rm -f cmd/cmd

lessc:
	lessc --clean-css server/app/styles/drone.less server/app/styles/drone.css

dpkg: build embed deb

# embeds content in go source code so that it is compiled
# and packaged inside the go binary file.
embed:
	rice --import-path="github.com/drone/drone/server" append --exec="debian/drone/usr/local/bin/droned"

# creates a debian package for drone to install
# `sudo dpkg -i drone.deb`
deb:
	mkdir -p debian/drone/usr/local/bin
	mkdir -p debian/drone/var/lib/drone
	dpkg-deb --build debian/drone

# deploys drone to a staging server. this requires the following
# environment variables are set:
#
#   DRONE_STAGING_HOST -- the hostname or ip
#   DRONE_STAGING_USER -- the username used to login
#   DRONE_STAGING_KEY  -- the identity file path (~/.ssh/id_rsa)
deploy:
	scp -i $$DRONE_STAGING_KEY debian/drone.deb $$DRONE_STAGING_USER@$$DRONE_STAGING_HOST:/tmp
	ssh -i $$DRONE_STAGING_KEY $$DRONE_STAGING_USER@$$DRONE_STAGING_HOST -- dpkg -i /tmp/drone.deb
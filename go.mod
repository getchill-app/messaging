module github.com/getchill-app/messaging

go 1.16

require (
	github.com/jmoiron/sqlx v1.3.1
	github.com/keys-pub/keys v0.1.21-0.20210326211358-fb3db764000f
	github.com/keys-pub/vault v0.0.0-20210328154023-333da5d2058d
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/vmihailenco/msgpack/v4 v4.3.12
)

// replace github.com/keys-pub/vault => ../../keys.pub/vault

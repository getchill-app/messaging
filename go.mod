module github.com/getchill-app/messaging

go 1.16

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/getchill-app/http/api v0.0.0-20210516232549-af1f6728a486
	github.com/getchill-app/http/client v0.0.0-00010101000000-000000000000
	github.com/getchill-app/http/server v0.0.0-20210510182642-e681eced1611
	github.com/jmoiron/sqlx v1.3.4
	github.com/keys-pub/keys v0.1.22-0.20210428191820-49dfbda60f85
	github.com/mutecomm/go-sqlcipher/v4 v4.4.2
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
)

replace github.com/mutecomm/go-sqlcipher/v4 => github.com/getchill-app/go-sqlcipher/v4 v4.4.3-0.20210518231725-725caa68982f

replace github.com/getchill-app/http/api => ../http/api

replace github.com/getchill-app/http/client => ../http/client

replace github.com/getchill-app/http/server => ../http/server

replace github.com/getchill-app/ws/api => ../ws/api

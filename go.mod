module github.com/getchill-app/messaging

go 1.16

require (
	github.com/alta/protopatch v0.3.4 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/getchill-app/http/api v0.0.0-20210504010216-724792fd62e1
	github.com/getchill-app/http/client v0.0.0-20210504011100-0d36c616cd37
	github.com/getchill-app/http/server v0.0.0-20210504010821-957671867b63
	github.com/jmoiron/sqlx v1.3.3
	github.com/keys-pub/keys v0.1.22-0.20210428191820-49dfbda60f85
	github.com/mutecomm/go-sqlcipher/v4 v4.4.2
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20210428140749-89ef3d95e781 // indirect
	golang.org/x/sys v0.0.0-20210426230700-d19ff857e887 // indirect
	google.golang.org/genproto v0.0.0-20210427215850-f767ed18ee4d // indirect
	google.golang.org/grpc v1.37.0 // indirect
)

// replace github.com/getchill-app/http/api => ../http/api

// replace github.com/getchill-app/http/client => ../http/client

// replace github.com/getchill-app/http/server => ../http/server

// replace github.com/getchill-app/ws/api => ../ws/api

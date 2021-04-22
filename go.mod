module github.com/getchill-app/messaging

go 1.16

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/jmoiron/sqlx v1.3.1
	github.com/keys-pub/keys v0.1.21-0.20210402011617-28dedbda9f32
	github.com/keys-pub/vault v0.0.0-20210403222024-d7c66fea4997
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/vmihailenco/msgpack/v4 v4.3.12
)

// replace github.com/keys-pub/vault => ../../keys.pub/vault

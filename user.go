package messaging

import "github.com/keys-pub/keys"

type User struct {
	KID      keys.ID `json:"kid"`
	Username string  `json:"username"`
}

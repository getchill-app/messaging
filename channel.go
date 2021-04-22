package messaging

import (
	"github.com/keys-pub/keys"
)

type Channel struct {
	ID              keys.ID `json:"id" msgpack:"id" db:"id"`
	Name            string  `json:"name,omitempty" msgpack:"name,omitempty" db:"name"`
	Description     string  `json:"desc,omitempty" msgpack:"desc,omitempty" db:"desc"`
	Snippet         string  `json:"snippet,omitempty" msgpack:"snippet,omitempty" db:"snippet"`
	Index           int64   `json:"index,omitempty" msgpack:"index,omitempty" db:"index"`
	Timestamp       int64   `json:"ts,omitempty" msgpack:"ts,omitempty" db:"ts"`
	RemoteTimestamp int64   `json:"rts,omitempty" msgpack:"rts,omitempty" db:"rts"`
	ReadIndex       int64   `json:"readIndex,omitempty" msgpack:"readIndex,omitempty" db:"readIndex"`
	Visibility      int     `json:"visibility,omitempty" msgpack:"visibility,omitempty" db:"visibility"`
}

const (
	VisibilityHidden int = 1
)

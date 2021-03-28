package messaging

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/keys-pub/vault"
	"github.com/keys-pub/vault/sync"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v4"
)

// Messenger ...
type Messenger struct {
	vault *vault.Vault
	init  bool
}

func NewMessenger(vault *vault.Vault) *Messenger {
	return &Messenger{vault: vault}
}

func initTables(db *sqlx.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY NOT NULL,
			channel TEXT NOT NULL,			
			text TEXT,
			sender TEXT,
			prev TEXT,
			cmd BLOB,
			ts INTEGER,
			rts INTEGER,
			ridx INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS channelStatus (
			id TEXT PRIMARY KEY NOT NULL,
			name TEXT,			
			desc TEXT,
			snippet TEXT,			
			"index" INTEGER,
			readIndex INTEGER,
			ts INTEGER,
			rts INTEGER			
		);`,
		`CREATE INDEX index_channel 
			ON messages(channel, ridx);`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (m *Messenger) check() error {
	if m.vault.DB() == nil {
		return vault.ErrLocked
	}
	if !m.init {
		if err := initTables(m.vault.DB()); err != nil {
			return err
		}
		m.init = true
	}
	return nil
}

func (m *Messenger) AddChannel(ctx context.Context, channel *keys.EdX25519Key) (*api.Key, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	return m.vault.Register(ctx, channel)
}

func (m *Messenger) AddKey(key *api.Key) error {
	if err := m.check(); err != nil {
		return err
	}
	return m.vault.Keyring().Set(key)
}

// Save a message.
func (m *Messenger) Set(msg *Message) error {
	if err := m.check(); err != nil {
		return err
	}

	channel, err := m.vault.Keyring().Key(msg.Channel)
	if err != nil {
		return err
	}
	if channel == nil {
		return errors.Errorf("channel key not found %s", msg.Channel)
	}

	senderKey, err := m.vault.Keyring().Key(msg.Sender)
	if err != nil {
		return err
	}
	if senderKey == nil {
		return errors.Errorf("sender key not found %s", msg.Sender)
	}

	cipher := NewSenderBox(senderKey.AsEdX25519())
	return sync.Transact(m.vault.DB(), func(tx *sqlx.Tx) error {
		logger.Debugf("Saving msg %s", msg.ID)
		b, err := msgpack.Marshal(msg)
		if err != nil {
			return err
		}
		// For pending message set remote index to max
		msg.RemoteIndex = 9223372036854775807
		if err := sync.AddTx(tx, channel.AsEdX25519(), b, cipher); err != nil {
			return err
		}
		if err := insertMessageTx(tx, msg); err != nil {
			return err
		}
		return nil
	})
}

// Send message.
func (m *Messenger) Send(ctx context.Context, msg *Message) error {
	if err := m.Set(msg); err != nil {
		return err
	}
	return m.SyncChannel(ctx, msg.Channel)
}

func (m *Messenger) Messages(channel keys.ID) ([]*Message, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	return getMessages(m.vault.DB(), channel)
}

// Sync all messages.
// Returns error if sync is not enabled.
func (m *Messenger) Sync(ctx context.Context) error {
	if err := m.check(); err != nil {
		return err
	}

	// Sync keyring
	logger.Infof("Sync keyring...")
	if err := m.vault.Keyring().Sync(ctx); err != nil {
		return err
	}

	// Get changes
	logger.Infof("Get changes...")
	chgs, err := m.vault.Changes(ctx)
	if err != nil {
		return err
	}
	logger.Infof("Found %d change(s)", len(chgs))

	// Sync each changed channel
	s := sync.NewSyncer(m.vault.DB(), m.vault.Client(), m.receive)
	for _, chg := range chgs {
		key, err := m.vault.Keyring().Key(chg.VID)
		if err != nil {
			return err
		}
		if key == nil {
			return keys.NewErrNotFound(chg.VID.String())
		}
		if err := s.Sync(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

func (m *Messenger) SyncChannel(ctx context.Context, channel keys.ID) error {
	if err := m.check(); err != nil {
		return err
	}
	key, err := m.vault.Keyring().Key(channel)
	if err != nil {
		return err
	}
	if key == nil {
		return keys.NewErrNotFound(channel.String())
	}

	s := sync.NewSyncer(m.vault.DB(), m.vault.Client(), m.receive)
	if err := s.Sync(ctx, key); err != nil {
		return err
	}

	return nil
}

func (m *Messenger) receive(ctx *sync.Context, events []*vault.Event) error {
	key, err := m.vault.Keyring().Key(ctx.VID)
	if err != nil {
		return err
	}

	for _, event := range events {
		b, pk, err := DecryptSenderBox(event.Data, key.AsEdX25519())
		if err != nil {
			return err
		}

		var msg Message
		if err := msgpack.Unmarshal(b, &msg); err != nil {
			return err
		}

		if !keys.X25519Match(msg.Sender, pk.ID()) {
			return errors.Errorf("message sender mismatch")
		}

		if err := insertMessageTx(ctx.Tx, &msg); err != nil {
			return err
		}
	}
	return nil
}

func insertMessageTx(tx *sqlx.Tx, msg *Message) error {
	if _, err := tx.Exec(`INSERT OR REPLACE INTO messages (id, channel, sender, ts, prev, text, cmd, ridx, rts) VALUES 
		($1, $2, $3, $4, $5, $6, $7, $8, $9);`,
		msg.ID,
		msg.Channel,
		msg.Sender,
		msg.Timestamp,
		msg.Prev,
		msg.Text,
		msg.Command,
		msg.RemoteIndex,
		msg.RemoteTimestamp); err != nil {
		return errors.Wrapf(err, "failed to insert message")
	}
	return nil
}

func getMessages(db *sqlx.DB, channel keys.ID) ([]*Message, error) {
	var out []*Message
	if err := db.Select(&out, "SELECT * FROM messages WHERE channel = $1 ORDER BY ridx, ts;", channel); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get messages")
	}
	return out, nil
}

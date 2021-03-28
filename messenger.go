package messaging

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/vault"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v4"
)

// Messenger ...
type Messenger struct {
	vlt *vault.Vault
}

func NewMessenger(vlt *vault.Vault) (*Messenger, error) {
	if vlt.DB() == nil {
		return nil, vault.ErrLocked
	}

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
		`CREATE INDEX index_channel 
			ON messages(channel, ridx);`,
		// TODO: Indexes
	}
	for _, stmt := range stmts {
		if _, err := vlt.DB().Exec(stmt); err != nil {
			return nil, err
		}
	}

	return &Messenger{vlt}, nil
}

func (m *Messenger) check() error {
	if m.vlt.DB() == nil {
		return vault.ErrLocked
	}
	return nil
}

func (m *Messenger) Register(ctx context.Context, channel *keys.EdX25519Key) error {
	if err := m.vlt.Register(ctx, channel); err != nil {
		return err
	}
	return nil
}

// Save a message.
func (m *Messenger) Set(msg *Message) error {
	if err := m.check(); err != nil {
		return err
	}
	return vault.TransactDB(m.vlt.DB(), func(tx *sqlx.Tx) error {
		logger.Debugf("Saving msg %s", msg.ID)
		b, err := msgpack.Marshal(msg)
		if err != nil {
			return err
		}
		// For pending message set remote index to max
		msg.RemoteIndex = 9223372036854775807
		if err := vault.Add(tx, msg.Channel, b); err != nil {
			return err
		}
		if err := insertMessageTx(tx, msg); err != nil {
			return err
		}
		return nil
	})
}

func (m *Messenger) Messages(channel keys.ID) ([]*Message, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	return getMessages(m.vlt.DB(), channel)
}

// Sync all messages.
// Returns error if sync is not enabled.
func (m *Messenger) Sync(ctx context.Context) error {
	if err := m.check(); err != nil {
		return err
	}

	// Sync keyring
	logger.Infof("Sync keyring...")
	if err := m.vlt.Keyring().Sync(ctx); err != nil {
		return err
	}

	// Get changes
	logger.Infof("Get changes...")
	chgs, err := m.vlt.Changes(ctx)
	if err != nil {
		return err
	}
	logger.Infof("Found %d change(s)", len(chgs))

	// Sync each changed channel
	s := vault.NewSyncer(m.vlt.DB(), m.vlt.Client(), m.receive)
	for _, chg := range chgs {
		key, err := m.vlt.Keyring().Key(chg.VID)
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
	key, err := m.vlt.Keyring().Key(channel)
	if err != nil {
		return err
	}
	if key == nil {
		return keys.NewErrNotFound(channel.String())
	}

	s := vault.NewSyncer(m.vlt.DB(), m.vlt.Client(), m.receive)
	if err := s.Sync(ctx, key); err != nil {
		return err
	}

	return nil
}

func (m *Messenger) receive(ctx *vault.SyncContext, events []*vault.Event) error {
	for _, event := range events {
		var msg Message
		if err := msgpack.Unmarshal(event.Data, &msg); err != nil {
			return err
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

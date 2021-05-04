package messaging

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/keys-pub/keys"
	"github.com/pkg/errors"
)

type User struct {
	KID      keys.ID `json:"kid"`
	Username string  `json:"username"`
}

func addUserTx(tx *sqlx.Tx, user *User) error {
	if _, err := tx.Exec(`INSERT OR REPLACE INTO users (kid, username) VALUES ($1, $2);`,
		user.KID, user.Username); err != nil {
		return errors.Wrapf(err, "failed to insert user")
	}

	return nil
}

func getUser(db *sqlx.DB, kid keys.ID) (*User, error) {
	var user User
	if err := db.Get(&user, "SELECT * from user WHERE kid = ?", kid.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (m *Messenger) User(kid keys.ID) (*User, error) {
	return getUser(m.db, kid)
}

func (m *Messenger) UserAdd(user *User) error {
	return Transact(m.db, func(tx *sqlx.Tx) error {
		if err := addUserTx(tx, user); err != nil {
			return err
		}
		return nil
	})
}

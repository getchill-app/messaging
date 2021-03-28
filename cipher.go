package messaging

import (
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
)

type SenderBox struct {
	sender *keys.EdX25519Key
}

func NewSenderBox(sender *keys.EdX25519Key) SenderBox {
	return SenderBox{sender: sender}
}

// Encrypt does crypto_box_seal(pk+crypto_box(b)).
func (c SenderBox) Encrypt(b []byte, key *keys.EdX25519Key) ([]byte, error) {
	pk := api.NewKey(key).AsX25519Public()
	if pk == nil {
		return nil, errors.Errorf("invalid recipient")
	}
	sk := c.sender.X25519Key()
	encrypted := keys.BoxSeal(b, pk, sk)
	box := append(sk.Public(), encrypted...)
	anonymized := keys.CryptoBoxSeal(box, pk)
	return anonymized, nil
}

// DecryptSenderBox returning sender public key.
func DecryptSenderBox(b []byte, key *keys.EdX25519Key) ([]byte, *keys.X25519PublicKey, error) {
	if key == nil {
		return nil, nil, errors.Errorf("failed to decrypt: no key")
	}
	box, err := keys.CryptoBoxSealOpen(b, key.X25519Key())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to decrypt")
	}
	if len(box) < 32 {
		return nil, nil, errors.Wrapf(errors.Errorf("not enough bytes"), "failed to decrypt")
	}
	pk := keys.NewX25519PublicKey(keys.Bytes32(box[:32]))
	encrypted := box[32:]

	decrypted, err := keys.BoxOpen(encrypted, pk, key.X25519Key())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to decrypt")
	}

	return decrypted, pk, nil
}

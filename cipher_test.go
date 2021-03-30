package messaging_test

import (
	"testing"

	"github.com/getchill-app/messaging"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/vault/testutil"
	"github.com/stretchr/testify/require"
)

func TestEncrypt(t *testing.T) {
	alice := keys.NewEdX25519KeyFromSeed(testutil.Seed(0x01))
	channel := keys.NewEdX25519KeyFromSeed(testutil.Seed(0xa0))

	cipher := messaging.NewSenderBox(alice)

	in := []byte("testing")
	encrypted, err := cipher.Encrypt(in, channel)
	require.NoError(t, err)

	out, pk, err := messaging.DecryptSenderBox(encrypted, channel)
	require.NoError(t, err)
	require.Equal(t, out, in)
	require.Equal(t, alice.X25519Key().ID(), pk.ID())
}

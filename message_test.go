package messaging_test

import (
	"testing"

	"github.com/getchill-app/messaging"
	"github.com/stretchr/testify/require"
)

func TestMessageCommandDB(t *testing.T) {
	cmd := messaging.MessageCommand{}

	val, err := cmd.Value()
	require.NoError(t, err)
	b := val.([]byte)

	var out messaging.MessageCommand
	out.Scan(b)
	require.Equal(t, cmd, out)
}

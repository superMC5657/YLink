package passwd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashVerify(t *testing.T) {
	h, err := Hash("Passw0rd!")
	assert.NoError(t, err)
	assert.NotEqual(t, "Passw0rd!", h)
	assert.True(t, Verify(h, "Passw0rd!"))
	assert.False(t, Verify(h, "wrong"))
}

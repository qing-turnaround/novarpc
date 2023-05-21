package consul

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	err := Init("43.139.192.217:8600")

	assert.Nil(t, err)
}

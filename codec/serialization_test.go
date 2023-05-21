package codec

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xing-you-ji/novarpc/protocol"
)

func TestPbSerializationMarshal(t *testing.T) {
	pbSer := &pbSerialization{}
	data, err := pbSer.Marshal(nil)
	assert.NotNil(t, err)
	fmt.Println(string(data), err)
	err = pbSer.Unmarshal(data, &protocol.Response{})
	assert.NotNil(t, err)
	err = pbSer.Unmarshal(nil, &protocol.Response{})
	assert.NotNil(t, err)
	fmt.Println(err)
}

package docgenerator

import (
	"testing"

	"github.com/long12310225/pitaya/v2/constants"
	_ "github.com/long12310225/pitaya/v2/protos"
	"github.com/stretchr/testify/assert"
)

func TestProtoDescriptors(t *testing.T) {
	t.Parallel()
	tables := []struct {
		name        string
		messageName string
		err         error
	}{
		{"fail filename", "not_exists.proto", constants.ErrProtodescriptor},
		{"success filename", "kick.proto", nil},
		{"success message", "protos.Push", nil},
		{"fail message", "protos.DoNotExist", constants.ErrProtodescriptor},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			bts, err := ProtoDescriptors(table.messageName)
			if table.err != nil {
				assert.EqualError(t, table.err, err.Error())
				assert.Nil(t, bts)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, bts)
			}
		})
	}
}

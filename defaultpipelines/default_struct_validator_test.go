package defaultpipelines

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestingStruct struct {
	Name         string `json:"name"`
	Email        string `json:"email" validate:"email"`
	SoftCurrency int    `json:"softCurrency" validate:"gte=0,lte=1000"`
	HardCurrency int    `json:"hardCurrency" validate:"gte=0,lte=200"`
}

func TestDefaultValidator(t *testing.T) {
	validator := &DefaultValidator{}

	var tableTest = map[string]struct {
		s          *TestingStruct
		shouldFail bool
	}{
		"validation_error": {
			s: &TestingStruct{
				Email: "notvalid",
			},
			shouldFail: true,
		},
		"validation_success": {
			s: &TestingStruct{
				Name:         "foo",
				Email:        "foo@tfgco.com",
				SoftCurrency: 100,
				HardCurrency: 10,
			},
		},
		"validate_nil_object": {
			s:          nil,
			shouldFail: false,
		},
	}

	for tname, tbl := range tableTest {
		t.Run(tname, func(t *testing.T) {
			var err error
			if tbl.s == nil {
				_, _, err = validator.Validate(context.Background(), nil)
			} else {
				_, _, err = validator.Validate(context.Background(), tbl.s)
			}

			if tbl.shouldFail {
				assert.Error(t, err)
			} else {
				fmt.Println(err)
				assert.NoError(t, err)
			}
		})
	}
}

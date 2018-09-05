package defaultpipelines

import (
	"context"
	"sync"

	validator "gopkg.in/go-playground/validator.v9"
)

// DefaultValidator is the default arguments validator for handlers
// in pitaya
type DefaultValidator struct {
	once     sync.Once
	validate *validator.Validate
}

// Validate is the the function responsible for validating the 'in' parameter
// based on the struct tags the parameter has.
// This function has the pipeline.Handler signature so
// it is possible to use it as a pipeline function
func (v *DefaultValidator) Validate(ctx context.Context, in interface{}) (interface{}, error) {
	if in == nil {
		return in, nil
	}

	v.lazyinit()
	if err := v.validate.Struct(in); err != nil {
		return nil, err
	}

	return in, nil
}

func (v *DefaultValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
	})
}

package defaultpipelines

import (
	"context"
	"sync"

	validator "github.com/go-playground/validator/v10"
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
func (v *DefaultValidator) Validate(ctx context.Context, in interface{}) (context.Context, interface{}, error) {
	if in == nil {
		return ctx, in, nil
	}

	v.lazyinit()
	if err := v.validate.Struct(in); err != nil {
		return ctx, nil, err
	}

	return ctx, in, nil
}

func (v *DefaultValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
	})
}

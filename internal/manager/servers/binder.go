package servers

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

type customBinder struct {
	defaultBinder echo.Binder
	v             *validator.Validate
}

func newBinder() *customBinder {
	newValidator := validator.New()
	return &customBinder{
		defaultBinder: new(echo.DefaultBinder),
		v:             newValidator,
	}
}

// Bind decodes request input and validates the bound value.
func (b *customBinder) Bind(c *echo.Context, i interface{}) error {
	if err := b.defaultBinder.Bind(c, i); err != nil {
		return err
	}
	if err := b.v.Struct(i); err != nil {
		return err
	}
	return nil
}

package utils

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
)

type ginContextKey struct{}

// ContextWithGin stores c in ctx for GinContextFromContext. Use this instead of ad-hoc
// context keys so lookups stay consistent across HTTP handlers and tests.
func ContextWithGin(ctx context.Context, c *gin.Context) context.Context {
	return context.WithValue(ctx, ginContextKey{}, c)
}

// GinContextFromContext returns the gin.Context previously stored with ContextWithGin.
func GinContextFromContext(ctx context.Context) (*gin.Context, error) {
	ginContext := ctx.Value(ginContextKey{})
	if ginContext == nil {
		err := fmt.Errorf("could not retrieve gin.Context")
		return nil, err
	}

	gc, ok := ginContext.(*gin.Context)
	if !ok {
		err := fmt.Errorf("gin.Context has wrong type")
		return nil, err
	}
	return gc, nil
}

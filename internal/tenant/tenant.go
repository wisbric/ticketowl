package tenant

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Info holds the resolved tenant metadata for the current request.
type Info struct {
	ID     uuid.UUID
	Name   string
	Slug   string
	Schema string
}

// SchemaName returns the PostgreSQL schema name for a tenant slug.
func SchemaName(slug string) string {
	return fmt.Sprintf("tenant_%s", slug)
}

type contextKey string

const (
	infoKey contextKey = "tenant_info"
	connKey contextKey = "tenant_conn"
)

// NewContext stores tenant info in the context.
func NewContext(ctx context.Context, info *Info) context.Context {
	return context.WithValue(ctx, infoKey, info)
}

// FromContext extracts the tenant info from the context.
// Returns nil if no tenant is set.
func FromContext(ctx context.Context) *Info {
	v, _ := ctx.Value(infoKey).(*Info)
	return v
}

// NewConnContext stores a tenant-scoped database connection in the context.
func NewConnContext(ctx context.Context, conn *pgxpool.Conn) context.Context {
	return context.WithValue(ctx, connKey, conn)
}

// ConnFromContext extracts the tenant-scoped database connection from the context.
// Returns nil if no connection is set.
func ConnFromContext(ctx context.Context) *pgxpool.Conn {
	v, _ := ctx.Value(connKey).(*pgxpool.Conn)
	return v
}

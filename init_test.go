package migrate

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/url"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

func pgxConnect() (*pgx.Conn, error) {
	rawURL := "postgres://postgres:localdb@127.0.0.1:5432/migrate-test"
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	return pgx.Connect(context.Background(), parsedURL.String())
}

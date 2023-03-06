//go:build go1.16
// +build go1.16

package migrate

import (
	"context"
	"embed"

	. "gopkg.in/check.v1"
)

//go:embed test-migrations/*
var testEmbedFS embed.FS

func (s *SqliteMigrateSuite) TestEmbedSource(c *C) {
	migrations := EmbedFileSystemMigrationSource{
		FileSystem: testEmbedFS,
		Root:       "test-migrations",
	}

	// Executes two migrations
	ctx := context.Background()

	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)

	// Has data
	var id int
	err = s.Db.QueryRow(ctx, "SELECT id FROM people").Scan(&id)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, 1)
}

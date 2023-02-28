package migrate

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/http"

	. "gopkg.in/check.v1"
)

var sqliteMigrations = []*Migration{
	{
		Id:   "123",
		Up:   []string{"CREATE TABLE people (id int);"},
		Down: []string{"DROP TABLE people;"},
	},
	{
		Id:   "124",
		Up:   []string{"ALTER TABLE people ADD COLUMN first_name text;"},
		Down: []string{"SELECT 0;"}, // Not really supported
	},
}

type SqliteMigrateSuite struct {
	Db *pgx.Conn
}

var _ = Suite(&SqliteMigrateSuite{})

func (s *SqliteMigrateSuite) SetUpTest(c *C) {
	var err error
	db, err := pgxConnect()
	c.Assert(err, IsNil)
	SetTable(DefaultMigrationTableName)

	s.Db = db
}

func (s *SqliteMigrateSuite) TearDownTest(c *C) {
	s.Db.Exec(context.Background(), "DROP TABLE IF EXISTS people")
	s.Db.Exec(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS %s", DefaultMigrationTableName))
}

func (s *SqliteMigrateSuite) TestRunMigration(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: sqliteMigrations[:1],
	}

	ctx := context.Background()
	// Executes one migration
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 1)

	// Can use table now
	_, err = s.Db.Exec(ctx, "SELECT * FROM people")
	c.Assert(err, IsNil)

	// Shouldn't apply migration again
	n, err = Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 0)
}

func (s *SqliteMigrateSuite) TestRunMigrationEscapeTable(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: sqliteMigrations[:1],
	}

	SetTable("my migrations")

	ctx := context.Background()
	// Executes one migration
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 1)

	// Tear down
	s.Db.Exec(ctx, `DROP TABLE IF EXISTS "my migrations"`)
}

func (s *SqliteMigrateSuite) TestMigrateMultiple(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: sqliteMigrations[:2],
	}

	ctx := context.Background()
	// Executes two migrations
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)

	// Can use column now
	_, err = s.Db.Exec(ctx, "SELECT first_name FROM people")
	c.Assert(err, IsNil)
}

func (s *SqliteMigrateSuite) TestMigrateIncremental(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: sqliteMigrations[:1],
	}

	ctx := context.Background()
	// Executes one migration
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 1)

	// Execute a new migration
	migrations = &MemoryMigrationSource{
		Migrations: sqliteMigrations[:2],
	}
	n, err = Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 1)

	// Can use column now
	_, err = s.Db.Exec(ctx, "SELECT first_name FROM people")
	c.Assert(err, IsNil)
}

func (s *SqliteMigrateSuite) TestFileMigrate(c *C) {
	migrations := &FileMigrationSource{
		Dir: "test-migrations",
	}

	ctx := context.Background()
	// Executes two migrations
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)

	// Has data
	var id int
	err = s.Db.QueryRow(ctx, "SELECT id FROM people").Scan(&id)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, 1)
}

func (s *SqliteMigrateSuite) TestHttpFileSystemMigrate(c *C) {
	migrations := &HttpFileSystemMigrationSource{
		FileSystem: http.Dir("test-migrations"),
	}

	ctx := context.Background()
	// Executes two migrations
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)

	// Has data
	var id int
	err = s.Db.QueryRow(ctx, "SELECT id FROM people").Scan(&id)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, 1)
}

func (s *SqliteMigrateSuite) TestAssetMigrate(c *C) {
	migrations := &AssetMigrationSource{
		Asset:    Asset,
		AssetDir: AssetDir,
		Dir:      "test-migrations",
	}

	ctx := context.Background()
	// Executes two migrations
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)

	// Has data
	var id int
	err = s.Db.QueryRow(ctx, "SELECT id FROM people").Scan(&id)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, 1)
}

func (s *SqliteMigrateSuite) TestMigrateMax(c *C) {
	migrations := &FileMigrationSource{
		Dir: "test-migrations",
	}

	ctx := context.Background()
	// Executes one migration
	n, err := ExecMax(ctx, s.Db, migrations, Up, 1)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 1)

	var id int
	err = s.Db.QueryRow(ctx, "SELECT COUNT(*) FROM people").Scan(&id)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, 0)
}

func (s *SqliteMigrateSuite) TestMigrateVersionInt(c *C) {
	migrations := &FileMigrationSource{
		Dir: "test-migrations",
	}

	ctx := context.Background()
	// Executes migration with target version 1
	n, err := ExecVersion(ctx, s.Db, migrations, Up, 1)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 1)

	var id int
	err = s.Db.QueryRow(ctx, "SELECT COUNT(*) FROM people").Scan(&id)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, 0)
}

func (s *SqliteMigrateSuite) TestMigrateVersionInt2(c *C) {
	migrations := &FileMigrationSource{
		Dir: "test-migrations",
	}

	ctx := context.Background()
	// Executes migration with target version 2
	n, err := ExecVersion(ctx, s.Db, migrations, Up, 2)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)

	var id int
	err = s.Db.QueryRow(ctx, "SELECT COUNT(*) FROM people").Scan(&id)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, 1)
}

func (s *SqliteMigrateSuite) TestMigrateVersionIntFailedWithNotExistingVerion(c *C) {
	migrations := &FileMigrationSource{
		Dir: "test-migrations",
	}

	ctx := context.Background()
	// Executes migration with not existing version 3
	_, err := ExecVersion(ctx, s.Db, migrations, Up, 3)
	c.Assert(err, NotNil)
}

func (s *SqliteMigrateSuite) TestMigrateVersionIntFailedWithInvalidVerion(c *C) {
	migrations := &FileMigrationSource{
		Dir: "test-migrations",
	}

	ctx := context.Background()
	// Executes migration with invalid version -1
	_, err := ExecVersion(ctx, s.Db, migrations, Up, -1)
	c.Assert(err, NotNil)
}

func (s *SqliteMigrateSuite) TestMigrateDown(c *C) {
	migrations := &FileMigrationSource{
		Dir: "test-migrations",
	}

	ctx := context.Background()
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)

	// Has data
	var id int
	err = s.Db.QueryRow(ctx, "SELECT id FROM people").Scan(&id)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, 1)

	// Undo the last one
	n, err = ExecMax(ctx, s.Db, migrations, Down, 1)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 1)

	// No more data
	err = s.Db.QueryRow(ctx, "SELECT COUNT(*) FROM people").Scan(&id)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, 0)

	// Remove the table.
	n, err = ExecMax(ctx, s.Db, migrations, Down, 1)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 1)

	// Cannot query it anymore
	err = s.Db.QueryRow(ctx, "SELECT COUNT(*) FROM people").Scan(&id)
	c.Assert(err, Not(IsNil))

	// Nothing left to do.
	n, err = ExecMax(ctx, s.Db, migrations, Down, 1)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 0)
}

func (s *SqliteMigrateSuite) TestMigrateDownFull(c *C) {
	migrations := &FileMigrationSource{
		Dir: "test-migrations",
	}

	ctx := context.Background()
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)

	// Has data
	var id int
	err = s.Db.QueryRow(ctx, "SELECT id FROM people").Scan(&id)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, 1)

	// Undo the last one
	n, err = Exec(ctx, s.Db, migrations, Down)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)

	// Cannot query it anymore
	var count int
	err = s.Db.QueryRow(ctx, "SELECT COUNT(*) FROM people").Scan(&count)
	c.Assert(err, Not(IsNil))

	// Nothing left to do.
	n, err = Exec(ctx, s.Db, migrations, Down)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 0)
}

func (s *SqliteMigrateSuite) TestMigrateTransaction(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: []*Migration{
			sqliteMigrations[0],
			sqliteMigrations[1],
			{
				Id:   "125",
				Up:   []string{"INSERT INTO people (id, first_name) VALUES (1, 'Test')", "SELECT fail"},
				Down: []string{}, // Not important here
			},
		},
	}

	ctx := context.Background()
	// Should fail, transaction should roll back the INSERT.
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, Not(IsNil))
	c.Assert(n, Equals, 2)

	// INSERT should be rolled back
	var count int
	err = s.Db.QueryRow(ctx, "SELECT COUNT(*) FROM people").Scan(&count)
	c.Assert(err, IsNil)
	c.Assert(count, Equals, 0)
}

func (s *SqliteMigrateSuite) TestPlanMigration(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: []*Migration{
			{
				Id:   "1_create_table.sql",
				Up:   []string{"CREATE TABLE people (id int)"},
				Down: []string{"DROP TABLE people"},
			},
			{
				Id:   "2_alter_table.sql",
				Up:   []string{"ALTER TABLE people ADD COLUMN first_name text"},
				Down: []string{"SELECT 0"}, // Not really supported
			},
			{
				Id:   "10_add_last_name.sql",
				Up:   []string{"ALTER TABLE people ADD COLUMN last_name text"},
				Down: []string{"ALTER TABLE people DROP COLUMN last_name"},
			},
		},
	}
	ctx := context.Background()
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 3)

	migrations.Migrations = append(migrations.Migrations, &Migration{
		Id:   "11_add_middle_name.sql",
		Up:   []string{"ALTER TABLE people ADD COLUMN middle_name text"},
		Down: []string{"ALTER TABLE people DROP COLUMN middle_name"},
	})

	plannedMigrations, err := PlanMigration(ctx, s.Db, migrations, Up, 0)
	c.Assert(err, IsNil)
	c.Assert(plannedMigrations, HasLen, 1)
	c.Assert(plannedMigrations[0].Migration, Equals, migrations.Migrations[3])

	plannedMigrations, err = PlanMigration(ctx, s.Db, migrations, Down, 0)
	c.Assert(err, IsNil)
	c.Assert(plannedMigrations, HasLen, 3)
	c.Assert(plannedMigrations[0].Migration, Equals, migrations.Migrations[2])
	c.Assert(plannedMigrations[1].Migration, Equals, migrations.Migrations[1])
	c.Assert(plannedMigrations[2].Migration, Equals, migrations.Migrations[0])
}

func (s *SqliteMigrateSuite) TestPlanMigrationWithHoles(c *C) {
	up := "SELECT 0"
	down := "SELECT 1"
	migrations := &MemoryMigrationSource{
		Migrations: []*Migration{
			{
				Id:   "1",
				Up:   []string{up},
				Down: []string{down},
			},
			{
				Id:   "3",
				Up:   []string{up},
				Down: []string{down},
			},
		},
	}
	ctx := context.Background()
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)

	migrations.Migrations = append(migrations.Migrations, &Migration{
		Id:   "2",
		Up:   []string{up},
		Down: []string{down},
	})

	migrations.Migrations = append(migrations.Migrations, &Migration{
		Id:   "4",
		Up:   []string{up},
		Down: []string{down},
	})

	migrations.Migrations = append(migrations.Migrations, &Migration{
		Id:   "5",
		Up:   []string{up},
		Down: []string{down},
	})

	// apply all the missing migrations
	plannedMigrations, err := PlanMigration(ctx, s.Db, migrations, Up, 0)
	c.Assert(err, IsNil)
	c.Assert(plannedMigrations, HasLen, 3)
	c.Assert(plannedMigrations[0].Migration.Id, Equals, "2")
	c.Assert(plannedMigrations[0].Queries[0], Equals, up)
	c.Assert(plannedMigrations[1].Migration.Id, Equals, "4")
	c.Assert(plannedMigrations[1].Queries[0], Equals, up)
	c.Assert(plannedMigrations[2].Migration.Id, Equals, "5")
	c.Assert(plannedMigrations[2].Queries[0], Equals, up)

	// first catch up to current target state 123, then migrate down 1 step to 12
	plannedMigrations, err = PlanMigration(ctx, s.Db, migrations, Down, 1)
	c.Assert(err, IsNil)
	c.Assert(plannedMigrations, HasLen, 2)
	c.Assert(plannedMigrations[0].Migration.Id, Equals, "2")
	c.Assert(plannedMigrations[0].Queries[0], Equals, up)
	c.Assert(plannedMigrations[1].Migration.Id, Equals, "3")
	c.Assert(plannedMigrations[1].Queries[0], Equals, down)

	// first catch up to current target state 123, then migrate down 2 steps to 1
	plannedMigrations, err = PlanMigration(ctx, s.Db, migrations, Down, 2)
	c.Assert(err, IsNil)
	c.Assert(plannedMigrations, HasLen, 3)
	c.Assert(plannedMigrations[0].Migration.Id, Equals, "2")
	c.Assert(plannedMigrations[0].Queries[0], Equals, up)
	c.Assert(plannedMigrations[1].Migration.Id, Equals, "3")
	c.Assert(plannedMigrations[1].Queries[0], Equals, down)
	c.Assert(plannedMigrations[2].Migration.Id, Equals, "2")
	c.Assert(plannedMigrations[2].Queries[0], Equals, down)
}

func (s *SqliteMigrateSuite) TestLess(c *C) {
	c.Assert((Migration{Id: "1"}).Less(&Migration{Id: "2"}), Equals, true)           // 1 less than 2
	c.Assert((Migration{Id: "2"}).Less(&Migration{Id: "1"}), Equals, false)          // 2 not less than 1
	c.Assert((Migration{Id: "1"}).Less(&Migration{Id: "a"}), Equals, true)           // 1 less than a
	c.Assert((Migration{Id: "a"}).Less(&Migration{Id: "1"}), Equals, false)          // a not less than 1
	c.Assert((Migration{Id: "a"}).Less(&Migration{Id: "a"}), Equals, false)          // a not less than a
	c.Assert((Migration{Id: "1-a"}).Less(&Migration{Id: "1-b"}), Equals, true)       // 1-a less than 1-b
	c.Assert((Migration{Id: "1-b"}).Less(&Migration{Id: "1-a"}), Equals, false)      // 1-b not less than 1-a
	c.Assert((Migration{Id: "1"}).Less(&Migration{Id: "10"}), Equals, true)          // 1 less than 10
	c.Assert((Migration{Id: "10"}).Less(&Migration{Id: "1"}), Equals, false)         // 10 not less than 1
	c.Assert((Migration{Id: "1_foo"}).Less(&Migration{Id: "10_bar"}), Equals, true)  // 1_foo not less than 1
	c.Assert((Migration{Id: "10_bar"}).Less(&Migration{Id: "1_foo"}), Equals, false) // 10 not less than 1
	// 20160126_1100 less than 20160126_1200
	c.Assert((Migration{Id: "20160126_1100"}).
		Less(&Migration{Id: "20160126_1200"}), Equals, true)
	// 20160126_1200 not less than 20160126_1100
	c.Assert((Migration{Id: "20160126_1200"}).
		Less(&Migration{Id: "20160126_1100"}), Equals, false)

}

func (s *SqliteMigrateSuite) TestPlanMigrationWithUnknownDatabaseMigrationApplied(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: []*Migration{
			{
				Id:   "1_create_table.sql",
				Up:   []string{"CREATE TABLE people (id int)"},
				Down: []string{"DROP TABLE people"},
			},
			{
				Id:   "2_alter_table.sql",
				Up:   []string{"ALTER TABLE people ADD COLUMN first_name text"},
				Down: []string{"SELECT 0"}, // Not really supported
			},
			{
				Id:   "10_add_last_name.sql",
				Up:   []string{"ALTER TABLE people ADD COLUMN last_name text"},
				Down: []string{"ALTER TABLE people DROP COLUMN last_name"},
			},
		},
	}
	ctx := context.Background()
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 3)

	// Note that migration 10_add_last_name.sql is missing from the new migrations source
	// so it is considered an "unknown" migration for the planner.
	migrations.Migrations = append(migrations.Migrations[:2], &Migration{
		Id:   "10_add_middle_name.sql",
		Up:   []string{"ALTER TABLE people ADD COLUMN middle_name text"},
		Down: []string{"ALTER TABLE people DROP COLUMN middle_name"},
	})

	_, err = PlanMigration(ctx, s.Db, migrations, Up, 0)
	c.Assert(err, NotNil, Commentf("Up migrations should not have been applied when there "+
		"is an unknown migration in the database"))
	c.Assert(err, FitsTypeOf, &PlanError{})

	_, err = PlanMigration(ctx, s.Db, migrations, Down, 0)
	c.Assert(err, NotNil, Commentf("Down migrations should not have been applied when there "+
		"is an unknown migration in the database"))
	c.Assert(err, FitsTypeOf, &PlanError{})
}

func (s *SqliteMigrateSuite) TestPlanMigrationWithIgnoredUnknownDatabaseMigrationApplied(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: []*Migration{
			{
				Id:   "1_create_table.sql",
				Up:   []string{"CREATE TABLE people (id int)"},
				Down: []string{"DROP TABLE people"},
			},
			{
				Id:   "2_alter_table.sql",
				Up:   []string{"ALTER TABLE people ADD COLUMN first_name text"},
				Down: []string{"SELECT 0"}, // Not really supported
			},
			{
				Id:   "10_add_last_name.sql",
				Up:   []string{"ALTER TABLE people ADD COLUMN last_name text"},
				Down: []string{"ALTER TABLE people DROP COLUMN last_name"},
			},
		},
	}
	SetIgnoreUnknown(true)
	ctx := context.Background()
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 3)

	// Note that migration 10_add_last_name.sql is missing from the new migrations source
	// so it is considered an "unknown" migration for the planner.
	migrations.Migrations = append(migrations.Migrations[:2], &Migration{
		Id:   "10_add_middle_name.sql",
		Up:   []string{"ALTER TABLE people ADD COLUMN middle_name text"},
		Down: []string{"ALTER TABLE people DROP COLUMN middle_name"},
	})

	_, err = PlanMigration(ctx, s.Db, migrations, Up, 0)
	c.Assert(err, IsNil)

	_, err = PlanMigration(ctx, s.Db, migrations, Down, 0)
	c.Assert(err, IsNil)
	SetIgnoreUnknown(false) // Make sure we are not breaking other tests as this is globaly set
}

func (s *SqliteMigrateSuite) TestPlanMigrationToVersion(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: []*Migration{
			{
				Id:   "1_create_table.sql",
				Up:   []string{"CREATE TABLE people (id int)"},
				Down: []string{"DROP TABLE people"},
			},
			{
				Id:   "2_alter_table.sql",
				Up:   []string{"ALTER TABLE people ADD COLUMN first_name text"},
				Down: []string{"SELECT 0"}, // Not really supported
			},
			{
				Id:   "10_add_last_name.sql",
				Up:   []string{"ALTER TABLE people ADD COLUMN last_name text"},
				Down: []string{"ALTER TABLE people DROP COLUMN last_name"},
			},
		},
	}
	ctx := context.Background()
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 3)

	migrations.Migrations = append(migrations.Migrations, &Migration{
		Id:   "11_add_middle_name.sql",
		Up:   []string{"ALTER TABLE people ADD COLUMN middle_name text"},
		Down: []string{"ALTER TABLE people DROP COLUMN middle_name"},
	})

	plannedMigrations, err := PlanMigrationToVersion(ctx, s.Db, migrations, Up, 11)
	c.Assert(err, IsNil)
	c.Assert(plannedMigrations, HasLen, 1)
	c.Assert(plannedMigrations[0].Migration, Equals, migrations.Migrations[3])

	plannedMigrations, err = PlanMigrationToVersion(ctx, s.Db, migrations, Down, 1)
	c.Assert(err, IsNil)
	c.Assert(plannedMigrations, HasLen, 3)
	c.Assert(plannedMigrations[0].Migration, Equals, migrations.Migrations[2])
	c.Assert(plannedMigrations[1].Migration, Equals, migrations.Migrations[1])
	c.Assert(plannedMigrations[2].Migration, Equals, migrations.Migrations[0])
}

// TestExecWithUnknownMigrationInDatabase makes sure that problems found with planning the
// migrations are propagated and returned by Exec.
func (s *SqliteMigrateSuite) TestExecWithUnknownMigrationInDatabase(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: sqliteMigrations[:2],
	}

	// Executes two migrations
	ctx := context.Background()
	n, err := Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)

	// Then create a new migration source with one of the migrations missing
	var newSqliteMigrations = []*Migration{
		{
			Id:   "124_other",
			Up:   []string{"ALTER TABLE people ADD COLUMN middle_name text"},
			Down: []string{"ALTER TABLE people DROP COLUMN middle_name"},
		},
		{
			Id:   "125",
			Up:   []string{"ALTER TABLE people ADD COLUMN age int"},
			Down: []string{"ALTER TABLE people DROP COLUMN age"},
		},
	}
	migrations = &MemoryMigrationSource{
		Migrations: append(sqliteMigrations[:1], newSqliteMigrations...),
	}

	n, err = Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, NotNil, Commentf("Migrations should not have been applied when there "+
		"is an unknown migration in the database"))
	c.Assert(err, FitsTypeOf, &PlanError{})
	c.Assert(n, Equals, 0)

	// Make sure the new columns are not actually created
	_, err = s.Db.Exec(ctx, "SELECT middle_name FROM people")
	c.Assert(err, NotNil)
	_, err = s.Db.Exec(ctx, "SELECT age FROM people")
	c.Assert(err, NotNil)
}

func (s *SqliteMigrateSuite) TestRunMigrationObjDefaultTable(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: sqliteMigrations[:1],
	}

	ms := MigrationSet{TableName: DefaultMigrationTableName}
	ctx := context.Background()
	// Executes one migration
	n, err := ms.Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 1)

	// Can use table now
	_, err = s.Db.Exec(ctx, "SELECT * FROM people")
	c.Assert(err, IsNil)

	// Uses default tableName
	_, err = s.Db.Exec(ctx, fmt.Sprintf("SELECT * FROM %s", DefaultMigrationTableName))
	c.Assert(err, IsNil)

	// Shouldn't apply migration again
	n, err = ms.Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 0)
}

func (s *SqliteMigrateSuite) TestRunMigrationObjOtherTable(c *C) {
	migrations := &MemoryMigrationSource{
		Migrations: sqliteMigrations[:1],
	}

	SetTable(DefaultMigrationTableName)

	ms := MigrationSet{TableName: "other_migrations"}
	ctx := context.Background()
	// Executes one migration
	n, err := ms.Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 1)

	// Can use table now
	_, err = s.Db.Exec(ctx, "SELECT * FROM people")
	c.Assert(err, IsNil)

	// Uses default tableName
	_, err = s.Db.Exec(ctx, "SELECT * FROM other_migrations")
	c.Assert(err, IsNil)

	// Shouldn't apply migration again
	n, err = ms.Exec(ctx, s.Db, migrations, Up)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 0)

	// Tear down
	s.Db.Exec(ctx, "DROP TABLE IF EXISTS other_migrations")
}

func (s *SqliteMigrateSuite) TestSetDisableCreateTable(c *C) {
	c.Assert(migSet.DisableCreateTable, Equals, false)

	SetDisableCreateTable(true)
	c.Assert(migSet.DisableCreateTable, Equals, true)

	SetDisableCreateTable(false)
	c.Assert(migSet.DisableCreateTable, Equals, false)
}

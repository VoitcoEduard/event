package postgres

import (
	"io/fs"
	"strconv"
	"strings"
	"testing"
)

func TestEmbeddedMigrationTargetMatchesSchemaVersion(t *testing.T) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	var target int64
	for _, entry := range entries {
		prefix, _, ok := strings.Cut(entry.Name(), "_")
		if !ok || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		if version > target {
			target = version
		}
	}
	if target != schemaVersion {
		t.Fatalf("embedded migrations target %d, application expects %d", target, schemaVersion)
	}
}

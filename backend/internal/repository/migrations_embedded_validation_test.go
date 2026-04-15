package repository

import (
	"io/fs"
	"sort"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestValidateEmbeddedMigrationsExecutionMode(t *testing.T) {
	files, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	sort.Strings(files)

	for _, name := range files {
		contentBytes, err := migrations.FS.ReadFile(name)
		require.NoError(t, err, name)

		content := strings.TrimSpace(string(contentBytes))
		if content == "" {
			continue
		}

		_, err = validateMigrationExecutionMode(name, content)
		require.NoError(t, err, name)
	}
}

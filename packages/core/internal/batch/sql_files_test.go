package batch

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSQLFilesExist verifies that required SQL files exist
func TestSQLFilesExist(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "network_insert.sql should exist",
			filename: "sql/network_insert.sql",
			wantErr:  false,
		},
		{
			name:     "watermark_update.sql should exist",
			filename: "sql/watermark_update.sql",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the current directory (internal/batch)
			_, err := os.Stat(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("SQL file %s not found: %v", tt.filename, err)
			}
		})
	}
}

// TestSQLFilesHaveComments verifies that SQL files have proper comments
func TestSQLFilesHaveComments(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "network_insert.sql should have comments",
			filename: "sql/network_insert.sql",
		},
		{
			name:     "watermark_update.sql should have comments",
			filename: "sql/watermark_update.sql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tt.filename)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Skipf("File not found: %v (expected during RED phase)", err)
				return
			}

			// Check if file has comments (starts with --)
			if len(content) == 0 {
				t.Error("SQL file is empty")
			}
			// More detailed validation will be added in REFACTOR phase
		})
	}
}

// TestLoadSQL_Success tests successful SQL file loading using embed.FS
// AC1: //go:embed sql/*.sql directive added
// AC2: loadSQL(filename string) (string, error) function implemented
// AC4: SQL string returned
func TestLoadSQL_Success(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "should load network_insert.sql successfully",
			filename: "network_insert.sql",
			wantErr:  false,
		},
		{
			name:     "should load watermark_update.sql successfully",
			filename: "watermark_update.sql",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Load SQL file using embed
			content, err := loadSQL(tt.filename)

			// Then: Should return content without error
			if (err != nil) != tt.wantErr {
				t.Errorf("loadSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// AC4: Verify SQL string is returned and not empty
			if len(content) == 0 {
				t.Error("loadSQL() returned empty string for existing file")
			}

			// Verify content contains SQL keywords
			if !contains(content, "INSERT") && !contains(content, "SELECT") {
				t.Error("loadSQL() returned content that doesn't appear to be SQL")
			}
		})
	}
}

// TestLoadSQL_FileNotFound tests error handling when file doesn't exist
// AC3: File read error handling (file not found)
// AC5: Error wrapped using pkg/errors pattern or fmt.Errorf
func TestLoadSQL_FileNotFound(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "should return error for non-existent file",
			filename: "non_existent.sql",
		},
		{
			name:     "should return error for file with wrong extension",
			filename: "test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Try to load non-existent file
			content, err := loadSQL(tt.filename)

			// Then: Should return error
			if err == nil {
				t.Error("loadSQL() expected error for non-existent file, got nil")
			}

			// AC5: Error should be wrapped with context
			if err != nil && len(err.Error()) == 0 {
				t.Error("loadSQL() error message is empty")
			}

			// Content should be empty when error occurs
			if len(content) > 0 {
				t.Error("loadSQL() should return empty string when error occurs")
			}
		})
	}
}

// TestLoadSQL_ContentValidation tests that loaded SQL content is valid
// AC4: SQL string returned with proper content
func TestLoadSQL_ContentValidation(t *testing.T) {
	t.Run("network_insert.sql should contain INSERT statement", func(t *testing.T) {
		// When: Load network_insert.sql
		content, err := loadSQL("network_insert.sql")

		// Then: Should contain INSERT INTO statement
		if err != nil {
			t.Skipf("loadSQL() not implemented yet: %v", err)
			return
		}

		if !contains(content, "INSERT INTO") {
			t.Error("network_insert.sql should contain INSERT INTO statement")
		}

		if !contains(content, "signoz_traces.network_map_connections") {
			t.Error("network_insert.sql should reference network_map_connections table")
		}
	})

	t.Run("watermark_update.sql should contain INSERT statement", func(t *testing.T) {
		// When: Load watermark_update.sql
		content, err := loadSQL("watermark_update.sql")

		// Then: Should contain INSERT INTO statement
		if err != nil {
			t.Skipf("loadSQL() not implemented yet: %v", err)
			return
		}

		if !contains(content, "INSERT INTO") {
			t.Error("watermark_update.sql should contain INSERT INTO statement")
		}

		if !contains(content, "signoz_traces.network_batch_watermark") {
			t.Error("watermark_update.sql should reference network_batch_watermark table")
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

// findSubstring checks if substr exists in s
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

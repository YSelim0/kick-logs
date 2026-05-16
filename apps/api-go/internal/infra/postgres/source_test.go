package postgres

import "testing"

func TestNormalizeDSNConvertsSQLAlchemyAsyncpgScheme(t *testing.T) {
	tests := map[string]string{
		"postgresql+asyncpg://user:pass@localhost:5432/db": "postgresql://user:pass@localhost:5432/db",
		"postgres+asyncpg://user:pass@localhost:5432/db":   "postgres://user:pass@localhost:5432/db",
		"postgresql://user:pass@localhost:5432/db":         "postgresql://user:pass@localhost:5432/db",
	}

	for input, expected := range tests {
		if actual := NormalizeDSN(input); actual != expected {
			t.Fatalf("NormalizeDSN(%q) = %q, want %q", input, actual, expected)
		}
	}
}

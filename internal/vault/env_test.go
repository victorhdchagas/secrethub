package vault

import (
	"reflect"
	"testing"
)

func TestParseEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "simple key value",
			input: "DB_HOST=localhost",
			want:  map[string]string{"DB_HOST": "localhost"},
		},
		{
			name: "multiple keys",
			input: `DB_HOST=localhost
DB_PORT=5432
API_KEY=abc123`,
			want: map[string]string{
				"DB_HOST": "localhost",
				"DB_PORT": "5432",
				"API_KEY": "abc123",
			},
		},
		{
			name: "skip comments and empty lines",
			input: `# database config
DB_HOST=localhost

# api
API_KEY=secret
`,
			want: map[string]string{
				"DB_HOST": "localhost",
				"API_KEY": "secret",
			},
		},
		{
			name: "double-quoted values",
			input: `DB_PASS="my secret password"
JSON='{"a":1}'`,
			want: map[string]string{
				"DB_PASS": "my secret password",
				"JSON":    `{"a":1}`,
			},
		},
		{
			name: "export prefix",
			input: `export NODE_ENV=production
export PATH=/usr/bin`,
			want: map[string]string{
				"NODE_ENV": "production",
				"PATH":     "/usr/bin",
			},
		},
		{
			name: "duplicate keys last wins",
			input: `KEY=first
KEY=second`,
			want: map[string]string{"KEY": "second"},
		},
		{
			name: "empty value",
			input: `EMPTY=
EMPTY2=""`,
			want: map[string]string{
				"EMPTY":  "",
				"EMPTY2": "",
			},
		},
		{
			name:  "empty input",
			input: "",
			want:  map[string]string{},
		},
		{
			name: "only comments",
			input: `# just a comment
# another`,
			want: map[string]string{},
		},
		{
			name: "line without equals ignored",
			input: `INVALID_LINE
KEY=value`,
			want: map[string]string{"KEY": "value"},
		},
		{
			name: "value with equals sign",
			input: `CONN=host=db user=admin password=p@ss=word`,
			want:  map[string]string{"CONN": "host=db user=admin password=p@ss=word"},
		},
		{
			name: "whitespace around key and value",
			input: `  KEY   =   value  
OTHER=value`,
			want: map[string]string{
				"KEY":   "value",
				"OTHER": "value",
			},
		},
		{
			name: "unclosed quote kept as is",
			input: `KEY="unclosed`,
			want:  map[string]string{"KEY": `"unclosed`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseEnv(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseEnvResultIsMutable(t *testing.T) {
	t.Parallel()
	out := ParseEnv("KEY=value")
	out["KEY"] = "modified"
	if out["KEY"] != "modified" {
		t.Error("returned map should be mutable")
	}
}

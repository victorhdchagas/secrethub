package vault

import (
	"bufio"
	"strings"
)

// ParseEnv converte texto no formato .env em um map[string]string.
// Suporta: comentários (#), linhas vazias, aspas duplas/simples, prefixo `export`.
// Chaves duplicadas: último valor vence.
func ParseEnv(input string) map[string]string {
	out := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := splitEnvLine(line)
		if !ok {
			continue
		}
		out[key] = val
	}
	return out
}

// splitEnvLine separa "KEY=VALUE" aplicando regras de export e aspas.
func splitEnvLine(line string) (string, string, bool) {
	if strings.HasPrefix(line, "export ") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
	}
	eq := strings.IndexByte(line, '=')
	if eq <= 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:eq])
	val := strings.TrimSpace(line[eq+1:])
	if key == "" {
		return "", "", false
	}
	val = unquote(val)
	return key, val, true
}

// unquote remove aspas duplas ou simples ao redor do valor, se houver par casado.
func unquote(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

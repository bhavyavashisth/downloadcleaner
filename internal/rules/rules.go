package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Rule struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions"`
}

type Manager struct {
	Rules      []Rule
	configPath string
}

func NewManager() *Manager {
	home, _ := os.UserHomeDir()
	return &Manager{
		Rules:      defaultRules(),
		configPath: filepath.Join(home, ".download-cleaner-rules.json"),
	}
}

// default rules
func defaultRules() []Rule {
	return []Rule{
		{Name: "Images", Extensions: []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".ico"}},
		{Name: "Documents", Extensions: []string{".pdf", ".doc", ".docx", ".txt", ".md", ".csv", ".xls", ".xlsx", ".ppt", ".pptx", ".rtf"}},
		{Name: "Videos", Extensions: []string{".mp4", ".mkv", ".mov", ".webm", ".avi", ".flv", ".wmv", ".m4v"}},
		{Name: "Audio", Extensions: []string{".mp3", ".wav", ".flac", ".m4a", ".ogg", ".aac"}},
		{Name: "Archives", Extensions: []string{".zip", ".7z", ".tar", ".gz", ".rar", ".bz2", ".xz"}},
		{Name: "Setup", Extensions: []string{".dmg", ".pkg", ".exe", ".msi", ".deb", ".rpm"}},
		{Name: "Code", Extensions: []string{".go", ".py", ".rs", ".cpp", ".c", ".js", ".ts", ".html", ".css", ".json", ".yaml", ".toml", ".sh"}},
	}
}

func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no custom rules yet, that's fine
		}
		return err
	}

	var custom []Rule
	if err := json.Unmarshal(data, &custom); err != nil {
		return err
	}

	// merge: start with defaults, override with custom
	m.Rules = defaultRules()
	for _, r := range custom {
		// replace if name matches, otherwise add
		found := false
		for i, existing := range m.Rules {
			if existing.Name == r.Name {
				m.Rules[i] = r
				found = true
				break
			}
		}
		if !found {
			m.Rules = append(m.Rules, r)
		}
	}

	return nil
}

func (m *Manager) Save() error {
	custom := []Rule{}
	for _, r := range m.Rules {
		// only save custom rules (not defaults)
		if !isDefaultRule(r) {
			custom = append(custom, r)
		}
	}

	data, err := json.MarshalIndent(custom, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0644)
}

// check if a rule matches the default set
func isDefaultRule(r Rule) bool {
	for _, d := range defaultRules() {
		if d.Name == r.Name && equalSlices(d.Extensions, r.Extensions) {
			return true
		}
	}
	return false
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Classify a file by its extension
func (m *Manager) Classify(ext string) string {
	ext = strings.ToLower(ext)
	for _, rule := range m.Rules {
		for _, rExt := range rule.Extensions {
			if rExt == ext {
				return rule.Name
			}
		}
	}
	return "" // unknown
}

// GetRule returns a rule by name
func (m *Manager) GetRule(name string) *Rule {
	for i, r := range m.Rules {
		if r.Name == name {
			return &m.Rules[i]
		}
	}
	return nil
}

// Add or update a rule
func (m *Manager) SetRule(name string, extensions []string) {
	// normalize extensions
	normalized := []string{}
	for _, ext := range extensions {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		normalized = append(normalized, ext)
	}

	for i, r := range m.Rules {
		if r.Name == name {
			m.Rules[i].Extensions = normalized
			return
		}
	}

	m.Rules = append(m.Rules, Rule{Name: name, Extensions: normalized})
}

// Delete a rule (can't delete default rules, only custom ones)
func (m *Manager) DeleteRule(name string) bool {
	for i, r := range m.Rules {
		if r.Name == name && !isDefaultRule(r) {
			m.Rules = append(m.Rules[:i], m.Rules[i+1:]...)
			return true
		}
	}
	return false
}
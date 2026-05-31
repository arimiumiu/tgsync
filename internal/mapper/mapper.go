package mapper

import "path/filepath"

type Mapper struct {
	mappings map[string]int
}

func New(mappings map[string]int) *Mapper {
	return &Mapper{mappings: mappings}
}

// TopicID returns the Telegram topic ID for a file based on its parent subfolder.
// Returns (0, false) if the subfolder is not in the config.
func (m *Mapper) TopicID(filePath string) (int, bool) {
	subfolder := filepath.Base(filepath.Dir(filePath))
	id, ok := m.mappings[subfolder]
	return id, ok
}

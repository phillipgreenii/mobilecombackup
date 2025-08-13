package manifest

// FileEntry represents a single file entry in files.yaml
type FileEntry struct {
	Name     string `yaml:"name"`
	Size     int64  `yaml:"size"`
	Checksum string `yaml:"checksum"`
	Modified string `yaml:"modified"`
}

// FileManifest represents the structure of files.yaml
type FileManifest struct {
	Version   string      `yaml:"version"`
	Generated string      `yaml:"generated"`
	Generator string      `yaml:"generator"`
	Files     []FileEntry `yaml:"files"`
}

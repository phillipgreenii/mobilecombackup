package manifest

// FileEntry represents a single file entry in files.yaml
type FileEntry struct {
	File      string `yaml:"file"`
	SHA256    string `yaml:"sha256"`
	SizeBytes int64  `yaml:"size_bytes"`
}

// FileManifest represents the structure of files.yaml
type FileManifest struct {
	Files []FileEntry `yaml:"files"`
}

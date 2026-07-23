package shared

import "os"

// CopyFile is a shared utility for duplicating processed assets across domain pipelines
func CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

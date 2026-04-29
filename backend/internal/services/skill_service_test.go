package services

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"path"
	"sort"
	"testing"
)

func TestHashDirectoryPreservesSingleTopLevelSubdirectory(t *testing.T) {
	files := map[string][]byte{
		"src/main.py": []byte("print('hello')\n"),
	}

	got := hashDirectory(files)
	want := referenceSkillContentMD5(map[string][]byte{
		"src/main.py": []byte("print('hello')\n"),
	})
	if got != want {
		t.Fatalf("hashDirectory() = %s, want %s", got, want)
	}

	flattened := referenceSkillContentMD5(map[string][]byte{
		"main.py": []byte("print('hello')\n"),
	})
	if got == flattened {
		t.Fatalf("hashDirectory stripped the skill's internal src/ directory")
	}
}

func TestExtractSkillDirectoriesStripsArchiveRootOnlyOnce(t *testing.T) {
	archive := buildTestZip(t, map[string][]byte{
		"weather/src/main.py": []byte("print('weather')\n"),
	})

	dirs, err := extractSkillDirectories("weather.zip", archive)
	if err != nil {
		t.Fatalf("extractSkillDirectories() error = %v", err)
	}
	if len(dirs) != 1 {
		t.Fatalf("extractSkillDirectories() returned %d dirs, want 1", len(dirs))
	}
	if _, ok := dirs[0].Files["src/main.py"]; !ok {
		t.Fatalf("expected skill files to preserve src/main.py after stripping archive root once: %#v", dirs[0].Files)
	}

	got := hashDirectory(dirs[0].Files)
	want := referenceSkillContentMD5(map[string][]byte{
		"src/main.py": []byte("print('weather')\n"),
	})
	if got != want {
		t.Fatalf("hashDirectory(extracted files) = %s, want %s", got, want)
	}
}

func TestFlattenSingleTopLevelDirForArchiveRoot(t *testing.T) {
	files := map[string][]byte{
		"weather/src/main.py": []byte("print('weather')\n"),
	}

	got := hashDirectory(flattenSingleTopLevelDir(files))
	want := referenceSkillContentMD5(map[string][]byte{
		"src/main.py": []byte("print('weather')\n"),
	})
	if got != want {
		t.Fatalf("hashDirectory(flattenSingleTopLevelDir(files)) = %s, want %s", got, want)
	}
}

func buildTestZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry, err := writer.Create(key)
		if err != nil {
			t.Fatalf("Create(%q): %v", key, err)
		}
		if _, err := entry.Write(files[key]); err != nil {
			t.Fatalf("Write(%q): %v", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}
	return buffer.Bytes()
}

func referenceSkillContentMD5(files map[string][]byte) string {
	entryKinds := map[string]string{}
	fileMap := map[string][]byte{}
	for key, body := range files {
		clean := path.Clean(key)
		if clean == "." || clean == "" {
			continue
		}
		fileMap[clean] = body
		entryKinds[clean] = "file"
		parts := splitTestPath(clean)
		for i := 1; i < len(parts); i++ {
			entryKinds[path.Join(parts[:i]...)] = "dir"
		}
	}

	keys := make([]string, 0, len(entryKinds))
	for key := range entryKinds {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	digest := md5.New()
	for _, key := range keys {
		_, _ = digest.Write([]byte(key))
		_, _ = digest.Write([]byte("\n"))
		if entryKinds[key] == "dir" {
			_, _ = digest.Write([]byte("dir\n"))
			continue
		}
		_, _ = digest.Write([]byte("file\n"))
		_, _ = digest.Write(fileMap[key])
		_, _ = digest.Write([]byte("\n"))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func splitTestPath(value string) []string {
	result := []string{}
	for _, part := range bytes.Split([]byte(value), []byte("/")) {
		if len(part) > 0 {
			result = append(result, string(part))
		}
	}
	return result
}

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	testCases := []struct {
		name     string
		root     string
		cfg      config
		expected string
	}{
		{"NoFilter", "testdata", config{ext: "", size: 0, list: true}, "testdata/dir.log\ntestdata/dir2/script.sh\n"},
		{"FilterExtensionMatch", "testdata", config{ext: ".log", size: 0, list: true}, "testdata/dir.log\n"},
		{"FilterExtensionNoMatch", "testdata", config{ext: ".gz", size: 0, list: true}, ""},
		{"FilterExtensionSizeMatch", "testdata", config{ext: ".log", size: 10, list: true}, "testdata/dir.log\n"},
		{"FilterExtensionSizeNoMatch", "testdata", config{ext: ".log", size: 20, list: true}, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer

			if err := run(tc.root, &buffer, tc.cfg); err != nil {
				t.Fatal(err)
			}

			res := buffer.String()

			if tc.expected != res {
				t.Errorf("Expected %q, got %q instead\n", tc.expected, res)
			}
		})
	}
}

func TestRunDelExtension(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         config
		extNoDelete string
		nDelete     int
		nNoDelete   int
		expected    string
	}{
		{"DeleteExtensionNoMatch", config{ext: ".log", del: true}, ".gz", 0, 10, ""},
		{"DeleteExtensionMatch", config{ext: ".log", del: true}, "", 10, 10, ""},
		{"DeleteExtensionMixed", config{ext: ".log", del: true}, ".gz", 5, 5, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				buffer    bytes.Buffer
				logBuffer bytes.Buffer
			)
			tc.cfg.wLog = &logBuffer

			tempDir, cleanup := createTempDir(t, map[string]int{
				tc.cfg.ext:     tc.nDelete,
				tc.extNoDelete: tc.nNoDelete,
			})
			defer cleanup()

			if err := run(tempDir, &buffer, tc.cfg); err != nil {
				t.Fatal(err)
			}

			res := buffer.String()

			if tc.expected != res {
				t.Errorf("Expected %q, got %q instead\n", tc.expected, res)
			}

			filesLeft, err := ioutil.ReadDir(tempDir)
			if err != nil {
				t.Error(err)
			}

			if len(filesLeft) != tc.nNoDelete {
				t.Errorf("Expected %d files left, got %d instead\n", tc.nNoDelete, len(filesLeft))
			}

			expLogLines := tc.nDelete + 1
			lines := bytes.Split(logBuffer.Bytes(), []byte("\n"))

			if len(lines) != expLogLines {
				t.Errorf("Expected %d log lines, got %d instead\n", expLogLines, len(lines))
			}
		})
	}
}

func TestRunArchive(t *testing.T) {
	testCases := []struct {
		name         string
		cfg          config
		extNoArchive string
		nArchive     int
		nNoArchive   int
	}{
		{"ArchiveExtensionNoMatch", config{ext: ".log"}, ".gz", 0, 10},
		{"ArchiveExtensionMatch", config{ext: ".log"}, "", 10, 0},
		{"ArchiveExtensionMixed", config{ext: ".log"}, ".gz", 5, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer

			tempDir, cleanup := createTempDir(t, map[string]int{
				tc.cfg.ext:      tc.nArchive,
				tc.extNoArchive: tc.nNoArchive,
			})
			defer cleanup()

			archiveDir, cleanupArchive := createTempDir(t, nil)
			defer cleanupArchive()

			tc.cfg.archive = archiveDir
			if err := run(tempDir, &buffer, tc.cfg); err != nil {
				t.Fatal(err)
			}

			pattern := filepath.Join(tempDir, fmt.Sprintf("*%s", tc.cfg.ext))
			expFiles, err := filepath.Glob(pattern)
			if err != nil {
				t.Fatal(err)
			}

			expOut := strings.Join(expFiles, "\n")
			res := strings.TrimSpace(buffer.String())
			if expOut != res {
				t.Errorf("Expected %q, got %q instead\n", expOut, res)
			}

			filesArchived, err := ioutil.ReadDir(archiveDir)
			if err != nil {
				t.Fatal(err)
			}

			if len(filesArchived) != tc.nArchive {
				t.Errorf("Expected %d files archived, got %d instead\n", tc.nArchive, len(filesArchived))
			}
		})
	}
}

func createTempDir(t *testing.T, files map[string]int) (dirname string, cleanup func()) {
	t.Helper()

	tempDir, err := ioutil.TempDir("", "walktest")
	if err != nil {
		t.Fatal(err)
	}

	for k, n := range files {
		for j := 1; j <= n; j++ {
			fname := fmt.Sprintf("file%d%s", j, k)
			fpath := filepath.Join(tempDir, fname)
			if err := ioutil.WriteFile(fpath, []byte("dummy"), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	return tempDir, func() { os.RemoveAll(tempDir) }
}

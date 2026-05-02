package app

import (
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTmpApp(t *testing.T) (*App, string) {
	t.Helper()
	dir := t.TempDir()
	a := &App{
		Name:         "testapp",
		ConfigFolder: filepath.Join(dir, "config"),
		CacheFolder:  filepath.Join(dir, "cache"),
		Version:      "1.0.0",
	}
	return a, dir
}

// ---------------------------------------------------------------------------
// DefaultConfigFolder
// ---------------------------------------------------------------------------

func TestDefaultConfigFolder(t *testing.T) {
	a := &App{Name: "myapp"}
	folder := a.DefaultConfigFolder()
	assert.Contains(t, folder, "myapp")
	assert.Contains(t, folder, ".config")
}

// ---------------------------------------------------------------------------
// DefaultCacheFolder
// ---------------------------------------------------------------------------

func TestDefaultCacheFolder(t *testing.T) {
	a := &App{Name: "myapp"}
	folder := a.DefaultCacheFolder()
	assert.Contains(t, folder, "myapp")
	assert.Contains(t, folder, ".cache")
}

// ---------------------------------------------------------------------------
// LoadConfig
// ---------------------------------------------------------------------------

type testConfig struct {
	Foo string `yaml:"foo"`
	Bar int    `yaml:"bar"`
}

func TestLoadConfig_NoFile(t *testing.T) {
	a, _ := newTmpApp(t)
	require.NoError(t, os.MkdirAll(a.ConfigFolder, 0700))

	cfg := &testConfig{}
	err := a.LoadConfig(cfg)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Foo)
	assert.Equal(t, 0, cfg.Bar)
}

func TestLoadConfig_WithFile(t *testing.T) {
	a, _ := newTmpApp(t)
	require.NoError(t, os.MkdirAll(a.ConfigFolder, 0700))

	content := "foo: hello\nbar: 42\n"
	cfgPath := filepath.Join(a.ConfigFolder, pathConfig)
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0600))

	cfg := &testConfig{}
	err := a.LoadConfig(cfg)
	require.NoError(t, err)
	assert.Equal(t, "hello", cfg.Foo)
	assert.Equal(t, 42, cfg.Bar)
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	a, _ := newTmpApp(t)
	require.NoError(t, os.MkdirAll(a.ConfigFolder, 0700))

	cfgPath := filepath.Join(a.ConfigFolder, pathConfig)
	require.NoError(t, os.WriteFile(cfgPath, []byte(": invalid: yaml: {"), 0600))

	cfg := &testConfig{}
	err := a.LoadConfig(cfg)
	require.Error(t, err)
}

func TestLoadConfig_FolderInsteadOfFile(t *testing.T) {
	a, _ := newTmpApp(t)
	require.NoError(t, os.MkdirAll(a.ConfigFolder, 0700))

	// Create a directory at the config path
	cfgPath := filepath.Join(a.ConfigFolder, pathConfig)
	require.NoError(t, os.MkdirAll(cfgPath, 0700))

	cfg := &testConfig{}
	err := a.LoadConfig(cfg)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// ensureFolder (tested through Init)
// ---------------------------------------------------------------------------

func TestEnsureFolder_CreatesFolder(t *testing.T) {
	dir := t.TempDir()
	newDir := filepath.Join(dir, "new", "nested")
	err := ensureFolder(newDir)
	require.NoError(t, err)
	_, statErr := os.Stat(newDir)
	assert.NoError(t, statErr)
}

func TestEnsureFolder_ExistingFolder(t *testing.T) {
	dir := t.TempDir()
	err := ensureFolder(dir)
	require.NoError(t, err)
}

func TestEnsureFolder_FileInsteadOfFolder(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "afile")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0600))

	err := ensureFolder(filePath)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func TestInit_Basic(t *testing.T) {
	a, _ := newTmpApp(t)
	err := a.Init(&testConfig{})
	require.NoError(t, err)

	// version file should be written
	versionBytes, err := os.ReadFile(filepath.Join(a.CacheFolder, pathVersion))
	require.NoError(t, err)
	assert.Equal(t, a.Version, string(versionBytes))
}

func TestInit_LoadsConfig(t *testing.T) {
	a, _ := newTmpApp(t)
	require.NoError(t, os.MkdirAll(a.ConfigFolder, 0700))

	content, err := yaml.Marshal(&testConfig{Foo: "world", Bar: 7})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(a.ConfigFolder, pathConfig), content, 0600))

	cfg := &testConfig{}
	require.NoError(t, a.Init(cfg))
	assert.Equal(t, "world", cfg.Foo)
	assert.Equal(t, 7, cfg.Bar)
}

func TestInit_SameVersionSkipsVersionWrite(t *testing.T) {
	a, _ := newTmpApp(t)
	require.NoError(t, a.Init(&testConfig{}))

	info1, err := os.Stat(filepath.Join(a.CacheFolder, pathVersion))
	require.NoError(t, err)

	require.NoError(t, a.Init(&testConfig{}))

	info2, err := os.Stat(filepath.Join(a.CacheFolder, pathVersion))
	require.NoError(t, err)

	// File should not have been re-written (mtime unchanged)
	assert.Equal(t, info1.ModTime(), info2.ModTime())
}

func TestInit_VersionChangedUpdatesVersionFile(t *testing.T) {
	a, _ := newTmpApp(t)
	require.NoError(t, a.Init(&testConfig{}))

	a.Version = "2.0.0"
	require.NoError(t, a.Init(&testConfig{}))

	versionBytes, err := os.ReadFile(filepath.Join(a.CacheFolder, pathVersion))
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", string(versionBytes))
}

// ---------------------------------------------------------------------------
// Init with embedded FS
// ---------------------------------------------------------------------------

//go:embed testdata/embedded
var testEmbedded embed.FS

func TestInit_WithEmbedded(t *testing.T) {
	a, _ := newTmpApp(t)
	a.Embedded = &testEmbedded
	a.EmbeddedPath = "" // will be set by Init

	err := a.Init(&testConfig{})
	require.NoError(t, err)

	// Embedded file should be extracted under EmbeddedPath
	extractedFile := filepath.Join(a.EmbeddedPath, "testdata", "embedded", "hello.txt")
	_, statErr := os.Stat(extractedFile)
	assert.NoError(t, statErr)
}



package config

import (
	"flag"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnv_InvalidFileExtension(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	os.Args = []string{"cmd", "--config=" + tmpFile.Name()}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	err = LoadEnv()
	require.ErrorIs(t, err, ErrFileFormat)
}

func TestLoadEnv_FileNotExists(t *testing.T) {
	os.Args = []string{"cmd", "--config=non_existent.env"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	err := LoadEnv()
	assert.True(t, os.IsNotExist(err))
}

func TestLoadEnv_InvalidLineFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test*.env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("INVALID_LINE"); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	os.Args = []string{"cmd", "--config=" + tmpFile.Name()}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	err = LoadEnv()
	require.ErrorIs(t, err, ErrInvalidString)
}

func TestLoadEnv_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test*.env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := "# Comment\nKEY=value\nANOTHER=123\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	os.Args = []string{"cmd", "--config=" + tmpFile.Name()}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Сохраняем и восстанавливаем переменные окружения
	oldKey := os.Getenv("KEY")
	oldAnother := os.Getenv("ANOTHER")
	defer func() {
		os.Setenv("KEY", oldKey)
		os.Setenv("ANOTHER", oldAnother)
	}()

	err = LoadEnv()
	require.NoError(t, err)
	{
		val := os.Getenv("KEY")
		assert.Equal(t, val, "value")
	}
	{
		val := os.Getenv("ANOTHER")
		assert.Equal(t, val, "123")
	}
}

func TestLoadEnv_ConfigPathPriority(t *testing.T) {
	t.Setenv("CONFIG_PATH", "env_path.env")
	os.Args = []string{"cmd", "--config=flag_path.env"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	filePath := fetchConfigPath()
	assert.Equal(t, filePath, "flag_path.env")
}

func TestLoadEnv_EmptyPath(t *testing.T) {
	os.Args = []string{"cmd"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	t.Setenv("CONFIG_PATH", "")

	err := LoadEnv()
	require.ErrorIs(t, err, ErrFileFormat)
}

func TestLoadEnv_PermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Пропуск теста на Windows")
	}

	tmpFile, err := os.CreateTemp("", "test*.env")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	err = tmpFile.Chmod(0222)
	require.NoError(t, err)
	tmpFile.Close()

	os.Args = []string{"cmd", "--config=" + tmpFile.Name()}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	err = LoadEnv()
	assert.True(t, os.IsPermission(err))
}

package config

import (
	"github.com/juho05/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetSeverity(log.NONE)
	os.Exit(m.Run())
}

func TestLoad(t *testing.T) {
	fullConfig := Config{
		BaseURL:    "https://crossonic.example.com",
		DBUser:     "testuser",
		DBPassword: "testpassword",
		DBName:     "testname",
		DBHost:     "testhost",
		DBPort:     1234,
		MusicDir:   "/test/music",
		DataDir:    "/test/data",
		CacheDir:   "/test/cache",
		EncryptionKey: []byte{0xdd, 0xd5, 0xc1, 0xd3, 0x0c, 0xf8, 0x99, 0x1f, 0xdf, 0x7f, 0xe2,
			0x58, 0x13, 0x8e, 0xda, 0xb0, 0xc0, 0x37, 0xa1, 0x4a, 0xa2, 0x54, 0x5b, 0x86, 0xe6, 0xe4, 0x86, 0x7f, 0x68, 0x27, 0xf4, 0xad},
		ListenAddr:      "test:4321",
		AutoMigrate:     false,
		LogLevel:        log.TRACE,
		StartupScan:     StartupScanFull,
		ListenBrainzURL: "https://listenbrainz.example.com",
		LastFMApiKey:    "lastfmkeytest",
		ScanHidden:      true,
		FrontendDir:     "/test/frontend",
	}

	defaultConfig := Config{
		BaseURL:    "https://crossonic.example.com",
		DBUser:     "testuser",
		DBPassword: "testpassword",
		DBName:     "testname",
		DBHost:     "testhost",
		DBPort:     1234,
		MusicDir:   "/test/music",
		DataDir:    "/test/data",
		CacheDir:   "/test/cache",
		EncryptionKey: []byte{0xdd, 0xd5, 0xc1, 0xd3, 0x0c, 0xf8, 0x99, 0x1f, 0xdf, 0x7f, 0xe2,
			0x58, 0x13, 0x8e, 0xda, 0xb0, 0xc0, 0x37, 0xa1, 0x4a, 0xa2, 0x54, 0x5b, 0x86, 0xe6, 0xe4, 0x86, 0x7f, 0x68, 0x27, 0xf4, 0xad},
		ListenAddr:      "0.0.0.0:8080",
		AutoMigrate:     true,
		LogLevel:        log.INFO,
		StartupScan:     StartupScanQuick,
		ListenBrainzURL: "https://api.listenbrainz.org",
		LastFMApiKey:    "",
		ScanHidden:      false,
		FrontendDir:     "",
	}

	logFileName := filepath.Join(t.TempDir(), "test.log")
	envFull := []string{
		"BASE_URL=" + fullConfig.BaseURL + "/",
		"DB_USER=" + fullConfig.DBUser,
		"DB_PASSWORD=" + fullConfig.DBPassword,
		"DB_NAME=" + fullConfig.DBName,
		"DB_HOST=" + fullConfig.DBHost,
		"DB_PORT=" + strconv.Itoa(fullConfig.DBPort),
		"MUSIC_DIR=" + fullConfig.MusicDir,
		"DATA_DIR=" + fullConfig.DataDir,
		"CACHE_DIR=" + fullConfig.CacheDir,
		"ENCRYPTION_KEY=3dXB0wz4mR/ff+JYE47asMA3oUqiVFuG5uSGf2gn9K0",
		"LISTEN_ADDR=" + fullConfig.ListenAddr,
		"AUTO_MIGRATE=" + strconv.FormatBool(fullConfig.AutoMigrate),
		"LOG_LEVEL=" + strconv.Itoa(int(fullConfig.LogLevel)),
		"LOG_FILE=" + logFileName,
		"LOG_APPEND=true",
		"STARTUP_SCAN=" + string(fullConfig.StartupScan),
		"LISTENBRAINZ_URL=" + fullConfig.ListenBrainzURL,
		"LASTFM_API_KEY=" + fullConfig.LastFMApiKey,
		"SCAN_HIDDEN=" + strconv.FormatBool(fullConfig.ScanHidden),
		"FRONTEND_DIR=" + fullConfig.FrontendDir,
	}

	envRequired := []string{
		"BASE_URL=" + fullConfig.BaseURL + "/",
		"DB_USER=" + fullConfig.DBUser,
		"DB_PASSWORD=" + fullConfig.DBPassword,
		"DB_NAME=" + fullConfig.DBName,
		"DB_HOST=" + fullConfig.DBHost,
		"DB_PORT=" + strconv.Itoa(fullConfig.DBPort),
		"MUSIC_DIR=" + fullConfig.MusicDir,
		"DATA_DIR=" + fullConfig.DataDir,
		"CACHE_DIR=" + fullConfig.CacheDir,
		"ENCRYPTION_KEY=3dXB0wz4mR/ff+JYE47asMA3oUqiVFuG5uSGf2gn9K0",
	}

	tests := []struct {
		name       string
		env        []string
		hasLogFile bool
		config     Config
		wantErrs   bool
	}{
		{"nil environment", nil, false, Config{}, true},
		{"empty environment", make([]string, 0), false, Config{}, true},
		{"only required keys are set", envRequired, false, defaultConfig, false},
		{"all keys are set", envFull, true, fullConfig, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf, errs := Load(tt.env)
			assert.Equal(t, tt.wantErrs, len(errs) > 1) // check for multiple errors
			if errs != nil {
				return
			}
			assert.Equal(t, tt.config.BaseURL, conf.BaseURL)
			assert.Equal(t, tt.config.DBUser, conf.DBUser)
			assert.Equal(t, tt.config.DBPassword, conf.DBPassword)
			assert.Equal(t, tt.config.DBName, conf.DBName)
			assert.Equal(t, tt.config.DBHost, conf.DBHost)
			assert.Equal(t, tt.config.DBPort, conf.DBPort)
			assert.Equal(t, tt.config.MusicDir, conf.MusicDir)
			assert.Equal(t, tt.config.DataDir, conf.DataDir)
			assert.Equal(t, tt.config.CacheDir, conf.CacheDir)
			assert.Equal(t, tt.config.EncryptionKey, conf.EncryptionKey)
			assert.Equal(t, tt.config.ListenAddr, conf.ListenAddr)
			assert.Equal(t, tt.config.AutoMigrate, conf.AutoMigrate)
			assert.Equal(t, tt.config.LogLevel, conf.LogLevel)
			assert.Equal(t, tt.config.StartupScan, conf.StartupScan)
			assert.Equal(t, tt.config.ListenBrainzURL, conf.ListenBrainzURL)
			assert.Equal(t, tt.config.LastFMApiKey, conf.LastFMApiKey)
			assert.Equal(t, tt.config.ScanHidden, conf.ScanHidden)
			assert.Equal(t, tt.config.FrontendDir, conf.FrontendDir)
			if tt.hasLogFile {
				assert.Equal(t, logFileName, conf.LogFile.Name())
				conf.LogFile.Close()
			} else {
				assert.Equal(t, os.Stderr, conf.LogFile)
			}
		})
	}
}

func TestStartupScanOption_Valid(t *testing.T) {
	tests := []struct {
		name string
		s    StartupScanOption
		want bool
	}{
		{"disabled is valid", StartupScanDisabled, true},
		{"quick is valid", StartupScanQuick, true},
		{"full is valid", StartupScanFull, true},
		{"asdf is valid", StartupScanOption("asdf"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.Valid())
		})
	}
}

func Test_boolean(t *testing.T) {
	env := map[string]string{
		"KEY_true":    "true",
		"KEY_1":       "1",
		"KEY_false":   "false",
		"KEY_0":       "0",
		"KEY_empty":   "",
		"KEY_invalid": "asdf",
	}
	tests := []struct {
		name    string
		key     string
		def     bool
		want    bool
		wantErr bool
	}{
		{"'true' works", "KEY_true", false, true, false},
		{"'1' works", "KEY_1", false, true, false},
		{"'false' works", "KEY_false", true, false, false},
		{"'0' works", "KEY_0", true, false, false},
		{"default is returned on empty env", "KEY_empty", true, true, false},
		{"default is returned on non-existing env", "asdf", false, false, false},
		{"invalid value leads to error", "KEY_invalid", false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := boolean(env, tt.key, tt.def)
			assertEqualOrErr(t, tt.key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_optionalString(t *testing.T) {
	env := map[string]string{
		"KEY_empty": "",
		"KEY_asdf":  "asdf",
	}
	tests := []struct {
		name string
		key  string
		def  string
		want string
	}{
		{"missing key", "does not exist", "default", "default"},
		{"empty value", "KEY_empty", "default", "default"},
		{"existing value", "KEY_asdf", "default", "asdf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, optionalString(env, tt.key, tt.def))
		})
	}
}

func Test_requiredString(t *testing.T) {
	env := map[string]string{
		"KEY_empty": "",
		"KEY_asdf":  "asdf",
	}
	tests := []struct {
		name    string
		key     string
		want    string
		wantErr bool
	}{
		{"missing key", "does not exist", "", true},
		{"empty value", "KEY_empty", "", true},
		{"existing value", "KEY_asdf", "asdf", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := requiredString(env, tt.key)
			assertEqualOrErr(t, tt.key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_requiredInt(t *testing.T) {
	env := map[string]string{
		"KEY_empty": "",
		"KEY_asdf":  "asdf",
		"KEY_42":    "42",
		"KEY_42.5":  "42.5",
	}
	tests := []struct {
		name    string
		key     string
		want    int
		wantErr bool
	}{
		{"missing key", "does not exist", 0, true},
		{"empty value", "KEY_empty", 0, true},
		{"invalid value (letters)", "KEY_asdf", 0, true},
		{"invalid value (float)", "KEY_42.5", 0, true},
		{"valid integer value", "KEY_42", 42, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := requiredInt(env, tt.key)
			assertEqualOrErr(t, tt.key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadAutoMigrate(t *testing.T) {
	key := "AUTO_MIGRATE"
	tests := []struct {
		name    string
		value   string
		want    bool
		wantErr bool
	}{
		{"empty value", "", true, false},
		{"invalid value", "asdf", false, true},
		{"valid value (true)", "true", true, false},
		{"valid value (0)", "0", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadAutoMigrate(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadBaseURL(t *testing.T) {
	key := "BASE_URL"
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"empty value", "", "", true},
		{"existing value", "https://crossonic.example.com", "https://crossonic.example.com", false},
		{"existing value (trailing slash)", "https://crossonic.example.com/", "https://crossonic.example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadBaseURL(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadCacheDir(t *testing.T) {
	key := "CACHE_DIR"
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"empty value", "", "", true},
		{"existing value", "/test/cache", "/test/cache", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadCacheDir(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadDBHost(t *testing.T) {
	key := "DB_HOST"
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"empty value", "", "", true},
		{"existing value", "testhost", "testhost", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadDBHost(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadDBName(t *testing.T) {
	key := "DB_NAME"
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"empty value", "", "", true},
		{"existing value", "testdbname", "testdbname", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadDBName(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadDBPassword(t *testing.T) {
	key := "DB_PASSWORD"
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"empty value", "", "", true},
		{"existing value", "testpassword", "testpassword", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadDBPassword(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadDBPort(t *testing.T) {
	key := "DB_PORT"
	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{"empty value", "", 0, true},
		{"invalid value", "asdf", 0, true},
		{"existing value", "8080", 8080, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadDBPort(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadDBUser(t *testing.T) {
	key := "DB_USER"
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"empty value", "", "", true},
		{"existing value", "testuser", "testuser", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadDBUser(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadDataDir(t *testing.T) {
	key := "DATA_DIR"
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"empty value", "", "", true},
		{"existing value", "/test/data", "/test/data", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadDataDir(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadEncryptionKey(t *testing.T) {
	key := "ENCRYPTION_KEY"
	tests := []struct {
		name    string
		value   string
		want    []byte
		wantErr bool
	}{
		{"empty value", "", nil, true},
		{"not base64 encoded", "öäü", nil, true},
		{"too few bytes", "3dXB0wz4mR/ff+JYE47asMA3oUqiVFuG5uSGf2gn9A", nil, true},
		{"too many bytes", "3dXB0wz4mR/ff+JYE47asMA3oUqiVFuG5uSGf2gn9PT0", nil, true},
		{"valid encryption key", "3dXB0wz4mR/ff+JYE47asMA3oUqiVFuG5uSGf2gn9K0", []byte{0xdd, 0xd5, 0xc1, 0xd3, 0x0c, 0xf8, 0x99, 0x1f, 0xdf, 0x7f, 0xe2,
			0x58, 0x13, 0x8e, 0xda, 0xb0, 0xc0, 0x37, 0xa1, 0x4a, 0xa2, 0x54, 0x5b, 0x86, 0xe6, 0xe4, 0x86, 0x7f, 0x68, 0x27, 0xf4, 0xad}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadEncryptionKey(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadFrontendDir(t *testing.T) {
	key := "FRONTEND_DIR"
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"empty value", "", ""},
		{"existing value", "/test/frontend", "/test/frontend"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, loadFrontendDir(map[string]string{
				key: tt.value,
			}))
		})
	}
}

func Test_loadLastFMApiKey(t *testing.T) {
	key := "LASTFM_API_KEY"
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"empty value", "", ""},
		{"existing value", "testkey", "testkey"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, loadLastFMApiKey(map[string]string{
				key: tt.value,
			}))
		})
	}
}

func Test_loadListenAddr(t *testing.T) {
	key := "LISTEN_ADDR"
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"empty value", "", "0.0.0.0:8080"},
		{"existing value", "test:1234", "test:1234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, loadListenAddr(map[string]string{
				key: tt.value,
			}))
		})
	}
}

func Test_loadListenBrainzURL(t *testing.T) {
	key := "LISTENBRAINZ_URL"
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"empty value", "", "https://api.listenbrainz.org"},
		{"existing value", "https://listenbrainz.example.com", "https://listenbrainz.example.com"},
		{"existing value (trailing slash)", "https://listenbrainz.example.com/", "https://listenbrainz.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, loadListenBrainzURL(map[string]string{
				key: tt.value,
			}))
		})
	}
}

func Test_loadLogFile(t *testing.T) {
	key := "LOG_FILE"
	keyAppend := "LOG_APPEND"

	dir := t.TempDir()

	tests := []struct {
		name                 string
		fileValue            string
		appendValue          string
		createFileBeforeTest bool
		wantAppend           bool
	}{
		{"empty value", "", "", false, false},
		{"file does not exist", filepath.Join(dir, "log1.txt"), "", false, false},
		{"file already exists (no append)", filepath.Join(dir, "log2.txt"), "false", true, false},
		{"file already exists (append)", filepath.Join(dir, "log3.txt"), "1", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createFileBeforeTest {
				file, err := os.Create(tt.fileValue)
				require.NoErrorf(t, err, "create log file: %v", err)
				_, err = file.WriteString("TestContent\nbla")
				require.NoErrorf(t, err, "write to log file: %v", err)
				file.Close()
			}

			env := map[string]string{
				key:       tt.fileValue,
				keyAppend: tt.appendValue,
			}

			logFile, err := loadLogFile(env)
			if err == nil && tt.fileValue != "" {
				defer logFile.Close()
			}
			require.NoErrorf(t, err, "load log file: %v", err)

			if tt.fileValue == "" {
				assert.Equal(t, os.Stderr, logFile)
				return
			}

			_, err = logFile.WriteString("log entry1\nlog entry2\n")
			assert.NoErrorf(t, err, "write to config log file: %v", err)
			logFile.Close()

			fileContent, err := os.ReadFile(tt.fileValue)
			require.NoErrorf(t, err, "read log file: %v", err)
			fileContentStr := string(fileContent)
			assert.True(t, strings.Contains(fileContentStr, "log entry1\nlog entry2\n"))
			assert.Equal(t, tt.createFileBeforeTest && tt.wantAppend, strings.HasPrefix(fileContentStr, "TestContent\nbla"))
		})
	}
}

func Test_loadLogLevel(t *testing.T) {
	key := "LOG_LEVEL"
	tests := []struct {
		name    string
		value   string
		want    log.Severity
		wantErr bool
	}{
		{"empty value", "", log.INFO, false},
		{"invalid value (letters)", "asdf", 0, true},
		{"invalid value (>5)", "6", 0, true},
		{"invalid value (<0)", "-1", 0, true},
		{"trace", "5", log.TRACE, false},
		{"info", "4", log.INFO, false},
		{"warn", "3", log.WARNING, false},
		{"error", "2", log.ERROR, false},
		{"fatal", "1", log.FATAL, false},
		{"none", "0", log.NONE, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadLogLevel(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadMusicDir(t *testing.T) {
	key := "MUSIC_DIR"
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"empty value", "", "", true},
		{"existing value", "/test/music", "/test/music", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadMusicDir(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadScanHidden(t *testing.T) {
	key := "SCAN_HIDDEN"
	tests := []struct {
		name    string
		value   string
		want    bool
		wantErr bool
	}{
		{"empty value", "", false, false},
		{"invalid value", "asdf", false, true},
		{"valid value (true)", "true", true, false},
		{"valid value (0)", "0", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadScanHidden(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func Test_loadStartupScan(t *testing.T) {
	key := "STARTUP_SCAN"
	tests := []struct {
		name    string
		value   string
		want    StartupScanOption
		wantErr bool
	}{
		{"empty value", "", StartupScanQuick, false},
		{"invalid value", "asdf", "", true},
		{"disabled", "disabled", StartupScanDisabled, false},
		{"quick", "quick", StartupScanQuick, false},
		{"full", "full", StartupScanFull, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := loadStartupScan(map[string]string{
				key: tt.value,
			})
			assertEqualOrErr(t, key, tt.want, v, tt.wantErr, err)
		})
	}
}

func assertEqualOrErr[T any](t *testing.T, key string, want, got T, wantErr bool, err error) {
	t.Helper()
	if wantErr {
		var configErr Error
		if assert.ErrorAs(t, err, &configErr) {
			assert.Equal(t, key, configErr.Key)
		}
	} else {
		assert.Equal(t, want, got)
	}
}

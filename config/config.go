package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/juho05/log"
)

type StartupScanOption string

type environment map[string]string

var (
	StartupScanDisabled StartupScanOption = "disabled"
	StartupScanQuick    StartupScanOption = "quick"
	StartupScanFull     StartupScanOption = "full"
)

func (s StartupScanOption) Valid() bool {
	return s == StartupScanDisabled || s == StartupScanQuick || s == StartupScanFull
}

const CoverArtPriorityEmbedded = "embedded"

type Config struct {
	BaseURL             string
	DBUser              string
	DBPassword          string
	DBName              string
	DBHost              string
	DBPort              int
	MusicDir            string
	DataDir             string
	CacheDir            string
	EncryptionKey       []byte
	ListenAddr          string
	AutoMigrate         bool
	LogLevel            log.Severity
	LogFile             *os.File
	StartupScan         StartupScanOption
	ListenBrainzURL     string
	LastFMApiKey        string
	ScanHidden          bool
	FrontendDir         string
	CoverArtPriority    []string
	ArtistImagePriority []string
}

// Load loads the configuration from environment variables into Options.
// env should be of the same format as os.Environ()
func Load(environ []string) (Config, []error) {
	env := make(environment, len(environ))
	for _, e := range environ {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			log.Fatalf("invalid environment variable format: %s", e)
		}
		env[parts[0]] = parts[1]
	}

	var errors []error

	var config Config
	var err error

	config.BaseURL, err = loadBaseURL(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.DBUser, err = loadDBUser(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.DBName, err = loadDBName(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.DBPassword, err = loadDBPassword(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.DBHost, err = loadDBHost(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.DBPort, err = loadDBPort(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.MusicDir, err = loadMusicDir(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.DataDir, err = loadDataDir(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.CacheDir, err = loadCacheDir(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.EncryptionKey, err = loadEncryptionKey(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.ListenAddr = loadListenAddr(env)

	config.AutoMigrate, err = loadAutoMigrate(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.LogLevel, err = loadLogLevel(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.LogFile, err = loadLogFile(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.StartupScan, err = loadStartupScan(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.ListenBrainzURL = loadListenBrainzURL(env)
	config.LastFMApiKey = loadLastFMApiKey(env)

	config.ScanHidden, err = loadScanHidden(env)
	if err != nil {
		errors = append(errors, err)
	}

	config.FrontendDir = loadFrontendDir(env)

	config.CoverArtPriority = loadCoverArtPriority(env)

	config.ArtistImagePriority = loadArtistImagePriority(env)

	return config, errors
}

func loadBaseURL(env environment) (string, error) {
	str, err := requiredString(env, "BASE_URL")
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(str, "/"), nil
}

func loadDBUser(env environment) (string, error) {
	return requiredString(env, "DB_USER")
}

func loadDBPassword(env environment) (string, error) {
	return requiredString(env, "DB_PASSWORD")
}

func loadDBHost(env environment) (string, error) {
	return requiredString(env, "DB_HOST")
}

func loadDBName(env environment) (string, error) {
	return requiredString(env, "DB_NAME")
}

func loadDBPort(env environment) (int, error) {
	return requiredInt(env, "DB_PORT")
}

func loadMusicDir(env environment) (string, error) {
	return requiredString(env, "MUSIC_DIR")
}

func loadDataDir(env environment) (string, error) {
	return requiredString(env, "DATA_DIR")
}

func loadCacheDir(env environment) (string, error) {
	return requiredString(env, "CACHE_DIR")
}

func loadEncryptionKey(env environment) ([]byte, error) {
	key := "ENCRYPTION_KEY"
	str := env[key]
	if str == "" {
		return nil, newError(key, "must not be empty")
	}
	k, err := base64.RawStdEncoding.DecodeString(str)
	if err != nil {
		return nil, newError(key, "must be in base64 format")
	}
	if len(k) != 32 {
		return nil, newError(key, "must be a base64 encoded byte array of length 32")
	}
	return k, nil
}

func loadListenAddr(env environment) string {
	return optionalString(env, "LISTEN_ADDR", "0.0.0.0:8080")
}

func loadAutoMigrate(env environment) (bool, error) {
	return boolean(env, "AUTO_MIGRATE", true)
}

func loadLogLevel(env environment) (log.Severity, error) {
	key := "LOG_LEVEL"
	def := log.INFO
	logLevelStr := env[key]
	if logLevelStr == "" {
		return def, nil
	}
	level, err := strconv.Atoi(logLevelStr)
	if err != nil {
		return def, newError(key, "invalid log level: must be an integer")
	}
	if level < int(log.NONE) || level > int(log.TRACE) {
		return def, newError(key, "invalid log level: valid values: 0 (none), 1 (fatal), 2 (error), 3 (warning), 4 (info), 5 (trace)")
	}
	return log.Severity(level), nil
}

// FIXME config should not be responsible for opening log file
func loadLogFile(env environment) (*os.File, error) {
	key := "LOG_FILE"
	def := os.Stderr
	if env[key] == "" {
		return def, nil
	}
	appnd, _ := strconv.ParseBool(env["LOG_APPEND"])
	if appnd {
		file, err := os.OpenFile(env[key], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return def, newError(key, fmt.Sprintf("failed to open log file (append): %s", err))
		}
		return file, nil
	} else {
		file, err := os.Create(env[key])
		if err != nil {
			return def, newError(key, fmt.Sprintf("failed to open log file: %s", err))
		}
		return file, nil
	}
}

func loadStartupScan(env environment) (StartupScanOption, error) {
	key := "STARTUP_SCAN"
	startupScan := StartupScanOption(optionalString(env, key, string(StartupScanQuick)))
	if !startupScan.Valid() {
		return "", newError(key, "invalid startup scan option (valid: disabled, quick, full)")
	}
	return startupScan, nil
}

func loadListenBrainzURL(env environment) string {
	return strings.TrimSuffix(optionalString(env, "LISTENBRAINZ_URL", "https://api.listenbrainz.org"), "/")
}

func loadLastFMApiKey(env environment) string {
	return optionalString(env, "LASTFM_API_KEY", "")
}

func loadScanHidden(env environment) (bool, error) {
	return boolean(env, "SCAN_HIDDEN", false)
}

func loadFrontendDir(env environment) string {
	return optionalString(env, "FRONTEND_DIR", "")
}

func loadCoverArtPriority(env environment) []string {
	list := optionalStringList(env, "COVER_ART_PRIORITY", []string{"cover.*", "folder.*", "front.*", CoverArtPriorityEmbedded})
	for i := range list {
		list[i] = strings.ToLower(list[i])
	}
	return list
}

func loadArtistImagePriority(env environment) []string {
	list := optionalStringList(env, "ARTIST_IMAGE_PRIORITY", []string{"artist.*"})
	for i := range list {
		list[i] = strings.ToLower(list[i])
	}
	return list
}

func optionalString(env environment, key, def string) string {
	str := env[key]
	if str == "" {
		return def
	}
	return str
}

func optionalStringList(env environment, key string, def []string) []string {
	str, ok := env[key]
	if !ok {
		return def
	}
	if str == "" {
		return make([]string, 0)
	}
	list := strings.Split(str, ",")
	newList := make([]string, 0, len(list))
	for _, pattern := range list {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			newList = append(newList, pattern)
		}
	}
	return newList
}

func requiredString(env environment, key string) (string, error) {
	str := env[key]
	if str == "" {
		return "", newError(key, "must not be empty")
	}
	return str, nil
}

func requiredInt(env environment, key string) (int, error) {
	str := env[key]
	if str == "" {
		return 0, newError(key, "must not be empty")
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0, newError(key, "must be an integer")
	}
	return i, nil
}

func boolean(env environment, key string, def bool) (bool, error) {
	str := env[key]
	if str == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(str)
	if err != nil {
		return false, newError(key, "must be a boolean")
	}
	return b, nil
}

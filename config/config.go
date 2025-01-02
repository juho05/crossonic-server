package config

import (
	"encoding/base64"
	"os"
	"strconv"
	"strings"

	"github.com/juho05/log"
)

func LoadAll() {
	BaseURL()
	DBUser()
	DBPassword()
	DBHost()
	DBName()
	DBPort()
	MusicDir()
	DataDir()
	CacheDir()
	PasswordEncryptionKey()
	ListenAddr()
	AutoMigrate()
	LogLevel()
	LogFile()
	DisableStartupScan()
	ListenBrainzURL()
	LastFMApiKey()
	SonosControllerURL()
	FrontendDir()
}

var values = make(map[string]any)

func BaseURL() string {
	return strings.TrimSuffix(requiredString("BASE_URL"), "/")
}

func SonosControllerURL() string {
	return optionalString("SONOS_CONTROLLER_URL", "")
}

func DBUser() string {
	return requiredString("DB_USER")
}

func DBPassword() string {
	return requiredString("DB_PASSWORD")
}

func DBHost() string {
	return requiredString("DB_HOST")
}

func DBName() string {
	return requiredString("DB_NAME")
}

func DBPort() int {
	return requiredInt("DB_PORT")
}

func MusicDir() string {
	return requiredString("MUSIC_DIR")
}

func DataDir() string {
	return requiredString("DATA_DIR")
}

func CacheDir() string {
	return requiredString("CACHE_DIR")
}

func PasswordEncryptionKey() (k []byte) {
	key := "PASSWORD_ENCRYPTION_KEY"
	if s, ok := values[key]; ok {
		return s.([]byte)
	}
	defer func() {
		values[key] = k
	}()
	str := os.Getenv(key)
	if str == "" {
		log.Fatalf("%s must not be empty", key)
	}
	var err error
	k, err = base64.RawStdEncoding.DecodeString(str)
	if err != nil {
		log.Fatalf("%s must be in base64 format", key)
	}
	if len(k) != 32 {
		log.Fatalf("%s must be a base64 encoded byte array of length 32", key)
	}
	return k
}

func ListenAddr() string {
	return optionalString("LISTEN_ADDR", "0.0.0.0:8080")
}

func AutoMigrate() (b bool) {
	return boolean("AUTO_MIGRATE", false)
}

func LogLevel() (sev log.Severity) {
	if l, ok := values["LOG_LEVEL"]; ok {
		return l.(log.Severity)
	}
	defer func() {
		values["LOG_LEVEL"] = sev
	}()
	def := log.INFO
	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr == "" {
		return def
	}
	level, err := strconv.Atoi(logLevelStr)
	if err != nil {
		log.Errorf("Invalid log level '%s': not a number. Using default: %d", logLevelStr, def)
		return def
	}
	if level < int(log.NONE) || level > int(log.TRACE) {
		log.Errorf("Invalid log level. Valid values: 0 (none), 1 (fatal), 2 (error), 3 (warning), 4 (info), 5 (trace). Using default: %d", def)
		return def
	}
	return log.Severity(level)
}

func LogFile() (file *os.File) {
	if f, ok := values["LOG_FILE"]; ok {
		return f.(*os.File)
	}
	defer func() {
		values["LOG_FILE"] = file
	}()
	def := os.Stderr
	if os.Getenv("LOG_FILE") == "" {
		return def
	}
	appnd, _ := strconv.ParseBool(os.Getenv("LOG_APPEND"))
	if appnd {
		file, err := os.Open(os.Getenv("LOG_FILE"))
		if err != nil {
			log.Fatalf("Failed to open log file %s. Using default: STDERR", err)
			return def
		}
		return file
	} else {
		file, err := os.Create(os.Getenv("LOG_FILE"))
		if err != nil {
			log.Fatalf("Failed to create log file %s. Using default: STDERR", err)
			return def
		}
		return file
	}
}

func DisableStartupScan() bool {
	return boolean("DISABLE_STARTUP_SCAN", false)
}

func ListenBrainzURL() string {
	return strings.TrimSuffix(optionalString("LISTENBRAINZ_URL", "https://api.listenbrainz.org"), "/")
}

func LastFMApiKey() string {
	return optionalString("LASTFM_API_KEY", "")
}

func ScanHidden() bool {
	return boolean("SCAN_HIDDEN", false)
}

func FrontendDir() string {
	return optionalString("FRONTEND_DIR", "")
}

func optionalString(key, def string) (str string) {
	if s, ok := values[key]; ok {
		return s.(string)
	}
	defer func() {
		values[key] = str
	}()
	str = os.Getenv(key)
	if str == "" {
		str = def
	}
	return str
}

func requiredString(key string) (str string) {
	if s, ok := values[key]; ok {
		return s.(string)
	}
	defer func() {
		values[key] = str
	}()
	str = os.Getenv(key)
	if str == "" {
		log.Fatalf("%s must not be empty", key)
	}
	return str
}

func requiredInt(key string) (i int) {
	if i, ok := values[key]; ok {
		return i.(int)
	}
	defer func() {
		values[key] = i
	}()
	str := os.Getenv(key)
	if str == "" {
		log.Fatalf("%s must not be empty", key)
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		log.Fatalf("%s must be an integer", key)
	}
	return i
}

func optionalFloat32(key string, def float32) (f float32) {
	if f, ok := values[key]; ok {
		return f.(float32)
	}
	defer func() {
		values[key] = f
	}()
	str := os.Getenv(key)
	if str == "" {
		return def
	}
	f64, err := strconv.ParseFloat(str, 32)
	if err != nil {
		log.Fatalf("%s must be an integer", key)
	}
	return float32(f64)
}

func boolean(key string, def bool) (b bool) {
	if b, ok := values[key]; ok {
		return b.(bool)
	}
	defer func() {
		values[key] = b
	}()
	str := os.Getenv(key)
	if str == "" {
		return def
	}
	b, err := strconv.ParseBool(str)
	if err != nil {
		log.Fatalf("%s must be a boolean", key)
	}
	return b
}

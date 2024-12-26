package local

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/teejays/gokutil/errutil"
)

type Config struct {
	Permanent PermanentConfig `json:"permanent"`
	Temporary TemporaryConfig `json:"temporary"`
}

type PermanentConfig struct {
	Credentials Credentials `json:"credentials"`
}

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TemporaryConfig struct {
	Token string `json:"token"`
}

// InitConfig stores an empty config file in the default location (if it doesn't exist). Create the directory if it doesn't exist.
func InitConfig(ctx context.Context) error {

	dirPath, err := GetDefaultConfigDir(ctx)
	if err != nil {
		return errutil.Wrap(err, "Getting default config dir")
	}

	// Create the directory if it doesn't exist
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		return errutil.Wrap(err, "Creating directory to store config")
	}

	// Create an empty config file if it doesn't exist
	filePath := filepath.Join(dirPath, GetDefaultConfigFileName(ctx))
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err = SaveConfig(ctx, Config{})
		if err != nil {
			return errutil.Wrap(err, "Saving empty config")
		}
	}

	return nil

}

func SaveConfig(ctx context.Context, config Config) error {
	// Where is the config stored?

	// Assume home dir for now + .ongoku directory
	filePath, err := GetDefaultConfigFilePath(ctx)
	if err != nil {
		return errutil.Wrap(err, "Getting default config file path")
	}

	// Create the directory if it doesn't exist
	err = os.MkdirAll(filePath, 0755)
	if err != nil {
		return errutil.Wrap(err, "Creating directory to store config")
	}

	// Write the config to the file (overwrite if it exists)
	configByes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return errutil.Wrap(err, "Marshalling config to json")
	}

	err = os.WriteFile(filePath, configByes, 0644)
	if err != nil {
		return errutil.Wrap(err, "Writing config to file")
	}

	return nil
}

func LoadConfig(ctx context.Context, path string) (Config, error) {
	var ret Config

	// Path is optional
	if path == "" {
		defPath, err := GetDefaultConfigFilePath(ctx)
		if err != nil {
			return ret, errutil.Wrap(err, "Getting default config file path")
		}
		path = defPath
	}

	// Read the file
	configBytes, err := os.ReadFile(path)
	if err != nil {
		return ret, errutil.Wrap(err, "Reading config file")
	}

	// Unmarshal the config
	err = json.Unmarshal(configBytes, &ret)
	if err != nil {
		return ret, errutil.Wrap(err, "Unmarshalling config")
	}

	return ret, nil
}

func GetDefaultConfigFilePath(ctx context.Context) (string, error) {
	dirPath, err := GetDefaultConfigDir(ctx)
	if err != nil {
		return "", errutil.Wrap(err, "Getting default config dir")
	}
	fileName := GetDefaultConfigFileName(ctx)
	return filepath.Join(dirPath, fileName), nil
}

func GetDefaultConfigDir(context.Context) (string, error) {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return "", fmt.Errorf("HOME env variable is not set. Where should we save the config file?")
	}
	return filepath.Join(homeDir, ".ongoku"), nil
}

func GetDefaultConfigFileName(context.Context) string {
	return "ogconfig.json"
}

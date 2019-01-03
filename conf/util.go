package conf

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os/user"
	fpath "path/filepath"
	"strings"
)

const unknownFileExtErrMessage = "Unknown configuration file extension[%s]. Only json and yml types are allowed."

func checkFileExtension(filepath string) error {

	extension := fpath.Ext(strings.ToLower(filepath))

	if extension != ".json" && extension != ".yml" && extension != ".yaml" {
		return errors.Errorf(unknownFileExtErrMessage, extension)
	}
	return nil
}

func readConfigurationContent(filepath string, content io.ReadCloser) (*Configuration, error) {

	configuration := &Configuration{}
	extension := fpath.Ext(strings.ToLower(filepath))

	switch extension {
	case ".json":
		return configuration, json.NewDecoder(content).Decode(configuration)
	case ".yml", "yaml":
		return configuration, yaml.NewDecoder(content).Decode(configuration)
	default:
		return nil, errors.Errorf(unknownFileExtErrMessage, extension)
	}
}

func parseConfigurationFromFile(filepath string) (*Configuration, error) {
	extension := fpath.Ext(strings.ToLower(filepath))

	switch extension {
	case ".json":
		return parseJsonConfiguration(filepath)
	case ".yml", "yaml":
		return parseYmlConfiguration(filepath)
	default:
		return nil, errors.Errorf(unknownFileExtErrMessage, extension)
	}
}

func parseJsonConfiguration(path string) (*Configuration, error) {

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result := &Configuration{}
	err = json.Unmarshal(file, result)

	return result, err
}

func parseYmlConfiguration(path string) (*Configuration, error) {

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result := &Configuration{}
	err = yaml.Unmarshal(file, &result)

	return result, err
}

func getHomePath() (string, error) {

	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}

	return currentUser.HomeDir, nil
}

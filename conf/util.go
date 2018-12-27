package conf

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os/user"
	fpath "path/filepath"
)

const unknownFileExtErrMessage = "Unknown configuration file extension[%s]. Only json and yml types are allowed."

func checkFileExtension(filepath string) error {

	extension := fpath.Ext(filepath)

	if extension != ".json" && extension != ".yml" && extension != ".yaml" {
		return errors.Errorf(unknownFileExtErrMessage, extension)
	}
	return nil
}

func readConfigurationContent(filepath string, content io.ReadCloser) (*Configuration, error) {

	configuration := &Configuration{}
	extension := fpath.Ext(filepath)

	if extension == ".json" {
		return configuration, json.NewDecoder(content).Decode(configuration)
	} else if extension == ".yml" || extension == ".yaml" {
		return configuration, yaml.NewDecoder(content).Decode(configuration)
	} else {
		return nil, errors.Errorf(unknownFileExtErrMessage, extension)
	}
}

func parseConfigurationFromFile(filepath string) (*Configuration, error) {
	extension := fpath.Ext(filepath)

	if extension == ".json" {
		return parseJsonConfiguration(filepath)
	} else if extension == ".yml" || extension == ".yaml" {
		return parseYmlConfiguration(filepath)
	} else {
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

package conf

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os/user"
	"path/filepath"
)

func parseConfiguration(path string) (*Configuration, error) {
	extension := filepath.Ext(path)

	if extension == ".json" {
		return parseJsonConfiguration(path)
	} else if extension == ".yml" || extension == ".yaml" {
		return parseYmlConfiguration(path)
	} else {
		return nil, errors.New("Unknown configuration file extension [" + extension + "]. Only json and yml" +
			" types are allowed.")
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

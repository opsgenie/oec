package conf

import (
	"strings"
	"bufio"
	"os"
	"os/user"
	"encoding/json"
)

func parseConfiguration(path string) (map[string]string, error) {
	var configMap = make(map[string]string)

	file, err := os.Open(path)
	defer file.Close()

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if !strings.HasPrefix(line, "#") && line != "" {
			l := strings.SplitN(line, "=", 2)
			l[0] = strings.TrimSpace(l[0])
			l[1] = strings.TrimSpace(l[1])
			configMap[l[0]] = l[1]
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return configMap, nil
}

func getHomePath() (string, error) {
	currentUser, err := user.Current()

	if err != nil {
		return "", err
	}

	return currentUser.HomeDir, nil
}

func cloneStringMap(original map[string]string) (map[string]string, error) {
	if original == nil {
		return nil, nil
	}

	originalJson, err := json.Marshal(original)

	if err != nil {
		return nil, err
	}

	copiedMap := make(map[string]string)

	err = json.Unmarshal(originalJson, &copiedMap)

	if err != nil {
		return nil, err
	}

	return copiedMap, nil
}

package conf

import (
	"testing"
	"os"
	"github.com/stretchr/testify/assert"
)

var gitCloneCalled = false

func mockGitClone(url string, username string, password string) error {
	gitCloneCalled = true
	var tmpDir = os.TempDir()
	directoryName, err := parseDirectoryNameFromUrl(url)

	if err != nil {
		return err
	}

	os.MkdirAll(tmpDir+string(os.PathSeparator)+directoryName, 0755)
	testFile, err := os.OpenFile(tmpDir+string(os.PathSeparator)+directoryName+string(os.PathSeparator)+
		"marid.conf", os.O_CREATE|os.O_WRONLY, 0755)

	if err != nil {
		return err
	}

	testFile.WriteString("tk1=tv1\ntk2=tv2\nemre=cicek")
	testFile.Close()

	return nil
}

func TestReadConfigurationFromGit(t *testing.T) {
	repoName := "repo"
	url := "https://github.com/someaccount/" + repoName + ".git"
	username := "dummyusername"
	password := "dummypassword"

	oldGitCloneMethod := gitCloneMethod
	defer func() {gitCloneMethod = oldGitCloneMethod}()
	gitCloneMethod = mockGitClone
	config, err := readConfigurationFromGit(url, username, password)

	if err != nil {
		t.Error("Could not read from Marid configuration. Error: " + err.Error())
	}

	assert.True(t, gitCloneCalled, "readConfigurationFromGit function did not call the method gitClone.")

	expectedConfig := map[string]string{
		"tk1": "tv1",
		"tk2": "tv2",
		"emre": "cicek",
	}

	assert.True(t, assert.ObjectsAreEqual(expectedConfig, config),
		"Actual config and expected config are not the same.")
	var repoDir = os.TempDir() + string(os.PathSeparator) + repoName

	if _, err := os.Stat(repoDir + string(os.PathSeparator) + "marid.conf"); os.IsExist(err) {
		t.Error("Marid configuration file still exists.")
	}

	if _, err := os.Stat(repoDir); os.IsExist(err) {
		t.Error("Cloned repository folder still exists.")
	}
}

func TestRemoveLocalRepoEvenIfErrorOccurs(t *testing.T) {
	var repoName = "repo"
	_, err := readConfigurationFromGit("https://github.com/someaccount/" + repoName + ".git",
		"user@name.com", "mypass")
	var repoDir = os.TempDir() + string(os.PathSeparator) + repoName

	if err == nil {
		t.Error("Error should be returned.")
	}

	if _, err := os.Stat(repoDir + string(os.PathSeparator) + "marid.conf"); os.IsExist(err) {
		t.Error("Marid configuration file still exists.")
	}

	if _, err := os.Stat(repoDir); os.IsExist(err) {
		t.Error("Cloned repository folder still exists.")
	}
}

func TestParseDirectoryNameFromUrl(t *testing.T) {
	repoName := "repo"
	url := "https://github.com/abc/" + repoName + ".git"
	actual, err := parseDirectoryNameFromUrl(url)

	if err != nil {
		t.Error("Error occurred while parsing directory name from URL [" + url + "]. Error: " + err.Error())
	}

	if actual != repoName {
		t.Errorf("Parsed repo name wrong. Expected: %s, Actual: %s", repoName, actual)
	}

	url = "sacma_sapan"
	actual, err = parseDirectoryNameFromUrl(url)
	assert.Error(t, err, "Did not throw an error altough URL was wrong.")

	if err.Error() != url + " is not a valid Git URL." {
		t.Error("Error message was wrong.")
	}
}

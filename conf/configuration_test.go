package conf

import (
	"github.com/opsgenie/oec/util"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestHttpFieldsFilledCorrectly(t *testing.T) {

	confPath, err := util.CreateTempTestFile(mockJsonFileContent, ".json")
	assert.Nil(t, err)

	conf, _ := readFileFromLocal(confPath)

	defer os.Remove(confPath)

	assert.Equal(t, conf.ActionMappings["WithHttpAction"].Flags["url"], "https://opsgenie.com")
	assert.Equal(t, conf.ActionMappings["WithHttpAction"].Flags["method"], "PUT")
	assert.Equal(t, conf.ActionMappings["WithHttpAction"].Flags["headers"], "{\"Authentication\":\"Basic JNjDkNsKaMs\"}")
	assert.Equal(t, conf.ActionMappings["WithHttpAction"].Flags["params"], "{\"Key1\":\"Value1\"}")
}

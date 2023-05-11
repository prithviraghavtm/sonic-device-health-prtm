package lib_test

import ( 
	"testing"
	"github.com/stretchr/testify/assert"
        . "lom/src/lib/lomcommon"
	"fmt"
)

/* Validates GetIntConfigurationFromJson for various input json strings and config key */
func Test_GetIntConfigurationFromJson(t *testing.T) {
	jsonString := []string{"{\"abc\":123}", "", "{\"abc\";abc}", "{\"abc\":123}", "{\"abc\":abc}"}
	expectedResult := []int{123, 5, 5, 5, 5}
	configKey := []string{"abc", "abc", "abc", "xyz", "abc"}

	assert := assert.New(t)
	for index := 0; index < len(jsonString); index++ {
		result := GetIntConfigurationFromJson(jsonString[index], configKey[index], 5)
		assert.Equal(expectedResult[index], result, fmt.Sprintf("GetIntConfigurationFromJson Failed for test index : %d", index))
	}
}

/* Validates GetFloatConfigurationFromJson for various input json strings and config key */
func Test_GetFloatConfigurationFromJson1(t *testing.T) {
	jsonString := []string{"{\"abc\":0.0001}", "{\"abc\";abc}", "", "{\"abc\":0.0001}", "{\"abc\":abc}"}
	expectedResult := []float64{0.0001, 0.1, 0.1, 0.1, 0.1}
	configKey := []string{"abc", "abc", "abc", "xyz", "abc"}
	assert := assert.New(t)
	for index := 0; index < len(jsonString); index++ {
		result := GetFloatConfigurationFromJson(jsonString[index], configKey[index], 0.1)
		assert.Equal(expectedResult[index], result, fmt.Sprintf("GetFloatConfigurationFromJson Failed for test index : %d", index))
	}
}


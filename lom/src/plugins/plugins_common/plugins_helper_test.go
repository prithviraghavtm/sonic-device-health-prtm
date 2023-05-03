package plugins_common

import (
	"time"
        "testing"
        "github.com/stretchr/testify/assert"
        "fmt"
)


/* Validate that reportingLimiter reports successfuly for first time for an anomaly key */
func Test_DetectionReportingFreqLimiter_ReportsSuccessfulyForFirstTime(t *testing.T) {
	detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
	//detectionReportingFrequencyLimiter.Initialize()
	shouldReport := detectionReportingFrequencyLimiter.ShouldReport("Ethernet0")
	cache := detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache
	_, ok := cache["Ethernet0"]

	assert := assert.New(t)
	assert.True(shouldReport, "ShouldReport is expected to be true")
	assert.Equal(1, len(cache), "Length of cache is expected to be 1")
	assert.True(ok)
}

/* Validate that reportingLimiter does not report in initial frequency */
func Test_DetectionReportingFreqLimiter_DoesNotReportForInitialFrequency(t *testing.T) {
	detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
	//detectionReportingFrequencyLimiter.Initialize()
	currentTimeMinusTwoMins := time.Now().Add(-2 * time.Minute)
	reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 8}
	detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
	shouldReport := detectionReportingFrequencyLimiter.ShouldReport("Ethernet0")

	assert := assert.New(t)
	assert.False(shouldReport, "ShouldReport is expected to be false")
	assert.False(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to have updated.")
	assert.Equal(9, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 9")
}

/* Validate that reportingLimiter reports in initial freq */
func Test_DetectionReportingFreqLimiter_ReportsInInitialFrequency(t *testing.T) {
	detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
	currentTimeMinusTwoMins := time.Now().Add(-7 * time.Minute)
	reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 8}
	detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
	shouldReport := detectionReportingFrequencyLimiter.ShouldReport("Ethernet0")

	assert := assert.New(t)
	assert.True(shouldReport, "ShouldReport is expected to be True")
	assert.False(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to have updated.")
	assert.Equal(9, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 9")
}

/* Validates that reportingLimiter does not report for subsequent frequency */
func Test_DetectionReportingFreqLimiter_DoesNotReportForSubsequentFrequency(t *testing.T) {
	detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
	currentTimeMinusTwoMins := time.Now().Add(-2 * time.Minute)
	reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 15}
	detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
	shouldReport := detectionReportingFrequencyLimiter.ShouldReport("Ethernet0")

	assert := assert.New(t)
	assert.False(shouldReport, "ShouldReport is expected to be false")
	assert.False(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to have updated.")
	assert.Equal(16, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 16")
}

/* Validates that reportingLimiter does report in subsequent Frequency */
func Test_LimitDetectionReportingFreq_ReportsInSubsequentFrequency(t *testing.T) {
	detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
	currentTimeMinusTwoMins := time.Now().Add(-62 * time.Minute)
	reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 15}
	detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
	shouldReport := detectionReportingFrequencyLimiter.ShouldReport("Ethernet0")

	assert := assert.New(t)
	assert.True(shouldReport, "ShouldReport is expected to be True")
	assert.False(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to have updated.")
	assert.Equal(16, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 16")
}

type MockElement struct {
        key int
}

/* Validates FixedSizeRollingWindow AddElement does not add more than max allowed elements into the rolling window */
func Test_FixedSizeRollingWindow_AddElementDoesNotAddMoreThanMaxElements(t *testing.T) {
        // Mock
        fixedSizeRollingWindow := FixedSizeRollingWindow[MockElement]{}
        fixedSizeRollingWindow.Initialize(4)

        mockElement1 := MockElement{key: 1}
        mockElement2 := MockElement{key: 2}
        mockElement3 := MockElement{key: 3}
        mockElement4 := MockElement{key: 4}
        mockElement5 := MockElement{key: 5}
        mockElement6 := MockElement{key: 6}

        fixedSizeRollingWindow.AddElement(mockElement1)
        fixedSizeRollingWindow.AddElement(mockElement2)
        fixedSizeRollingWindow.AddElement(mockElement3)
        fixedSizeRollingWindow.AddElement(mockElement4)
        fixedSizeRollingWindow.AddElement(mockElement5)
        fixedSizeRollingWindow.AddElement(mockElement6)

        // Act
        list := fixedSizeRollingWindow.GetElements()

        // Assert.
        validator := 3
        assert := assert.New(t)
        for iterator := list.Front(); iterator != nil; iterator = iterator.Next() {
                mockElmnt := iterator.Value.(MockElement)
                assert.Equal(validator, mockElmnt.key, fmt.Sprintf("Key is expected to be %d", validator))
                validator = validator + 1
        }
        // Ensure the elements are as expected while traversing from back to front.
        validator = 6
        for iterator := list.Back(); iterator != nil; iterator = iterator.Prev() {
                mockElmnt := iterator.Value.(MockElement)
                assert.Equal(validator, mockElmnt.key, fmt.Sprintf("Key is expected to be %d", validator))
                validator = validator - 1
        }
}

/* Validates that FixedSizeRollingWindow Initialize returns error for invalid maxSize */
func Test_FixedSizeRollingWindow_InitializeReturnsErrorForInvalidMaxSize(t *testing.T) {
        // Mock
        fixedSizeRollingWindow := FixedSizeRollingWindow[MockElement]{}
        // Act
        err := fixedSizeRollingWindow.Initialize(0)
        // Assert.
        assert := assert.New(t)
        assert.NotEqual(nil, err, "Error is expected to be non nil for input 0")

        // Act
        err = fixedSizeRollingWindow.Initialize(-1)
        // Assert.
        assert.NotEqual(nil, err, "Error is expected to be non nil for input 1")
}

/* Validates that FixedSizeRollingWindow returns empty list for no addition of elements */
func Test_FixedSizeRollingWindow_InitializeReturnsEmptyListForNoAdditionOfElements(t *testing.T) {
        // Mock
        fixedSizeRollingWindow := FixedSizeRollingWindow[MockElement]{}

        // Act
        err := fixedSizeRollingWindow.Initialize(4)
        list := fixedSizeRollingWindow.GetElements()
        countOfElements := 0
        for iterator := list.Front(); iterator != nil; iterator = iterator.Next() {
                countOfElements = countOfElements + 1
        }

        // Assert.
        assert := assert.New(t)
        assert.Equal(nil, err, "Error is expected to be nil")
        assert.NotEqual(nil, fixedSizeRollingWindow.GetElements(), "DoubleyLinkedList expected to be non nil")
        assert.Equal(0, countOfElements, "CountOfElements expected to be 0")
}

package plugins_common

import (
    "context"
    "fmt"
    "github.com/stretchr/testify/assert"
    "lom/src/lib/lomcommon"
    "lom/src/lib/lomipc"
    "testing"
    "time"
)

func init() {
    configFiles := &lomcommon.ConfigFiles_t{}
    configFiles.GlobalFl = "../../lib/lib_test/config/globals.conf.json"
    configFiles.ActionsFl = []string{"../../lib/lib_test/config/actions.confd/dummy_action.conf.json",
        "../../lib/lib_test/config/actions.confd/GenericPluginDetection.conf.json",
        "../../lib/lib_test/config/actions.confd/link_flap.conf.json",
        "../../lib/lib_test/config/actions.confd/plmgr_plugin_2.conf.json",
        "../../lib/lib_test/config/actions.confd/plmgr_plugin_3.conf.json",
        "../../lib/lib_test/config/actions.confd/test_plugin_001_bad.conf.json",
        "../../lib/lib_test/config/actions.confd/test_plugin_001.conf.json",
        "../../lib/lib_test/config/actions.confd/test_plugin_002.conf.json",
        "../../lib/lib_test/config/actions.confd/test_startup_plugin_001.conf.json"}
    configFiles.BindingsFl = "../../lib/lib_test/config/bindings.conf.json"
    configFiles.ProcsFl = "../../lib/lib_test/config/procs.conf.json"
    lomcommon.InitConfigMgr(configFiles)
}

/* Validate that reportingLimiter reports successfuly for first time for an anomaly key */
func Test_DetectionReportingFreqLimiter_ReportsSuccessfulyForFirstTime(t *testing.T) {
    detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
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
    currentTimeMinusTwoMins := time.Now().Add(-2 * time.Minute)
    reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 8}
    detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
    shouldReport := detectionReportingFrequencyLimiter.ShouldReport("Ethernet0")

    assert := assert.New(t)
    assert.False(shouldReport, "ShouldReport is expected to be false")
    assert.True(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to be same.")
    assert.Equal(8, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 8")
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
    assert.True(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to be same.")
    assert.Equal(15, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 15")
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

/* Validate that ResetCache does not delete entry  in initial freq */
func Test_DetectionReportingFreqLimiter_ResetsCacheDoesNotRemoveEntryInInitialFrequency(t *testing.T) {
    detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
    currentTimeMinusTwoMins := time.Now().Add(-2 * time.Minute)
    reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 8}
    detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
    detectionReportingFrequencyLimiter.ResetCache("Ethernet0")

    assert := assert.New(t)
    assert.NotNil(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"], "Cache entry is expected to be removed")
    assert.True(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to be same.")
    assert.Equal(8, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 8")
}

/* Validate that ResetCache deletes entry in initial freq */
func Test_DetectionReportingFreqLimiter_ResetsCacheSucceedsInInitialFrequency(t *testing.T) {
    detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
    currentTimeMinusTwoMins := time.Now().Add(-7 * time.Minute)
    reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 8}
    detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
    detectionReportingFrequencyLimiter.ResetCache("Ethernet0")

    assert := assert.New(t)
    assert.Nil(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"], "Cache entry is expected to be removed")
}

/* Validate that ResetCache does not delete entry  in subsequent freq */
func Test_DetectionReportingFreqLimiter_ResetCacheDoesNotRemoveEntryInSubsequentFrequency(t *testing.T) {
    detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
    currentTimeMinusTwoMins := time.Now().Add(-2 * time.Minute)
    reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 15}
    detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
    detectionReportingFrequencyLimiter.ResetCache("Ethernet0")

    assert := assert.New(t)
    assert.NotNil(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"], "Cache entry is expected to be removed")
    assert.True(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to be same.")
    assert.Equal(15, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 15")
}

/* Validate that ResetCache deletes entry in subsequent freq */
func Test_DetectionReportingFreqLimiter_ResetCacheSucceedsInSubsequentFrequency(t *testing.T) {
    detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
    currentTimeMinusTwoMins := time.Now().Add(-62 * time.Minute)
    reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 15}
    detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
    detectionReportingFrequencyLimiter.ResetCache("Ethernet0")

    assert := assert.New(t)
    assert.Nil(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"], "Cache entry is expected to be removed")
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

const (
    pluginName = "DummyPlugin"
    version    = "1.0.0.0"
)

var testRequestFrequency int

type DummyPlugin struct {
    testValue1 int
    testValue2 int
    PeriodicDetectionPluginUtil
    iteration            int
    firstTimeInvocation  time.Time
    secondTimeInvocation time.Time
}

func (dummyPlugin *DummyPlugin) Init(actionConfig *lomcommon.ActionCfg_t) error {
    dummyPlugin.testValue1 = 1
    err := dummyPlugin.PeriodicDetectionPluginUtil.Init(actionConfig.Name, testRequestFrequency, actionConfig, dummyPlugin.executeRequest, dummyPlugin.executeShutdown)
    if err != nil {
        return err
    }

    return nil
}

func (dummyPlugin *DummyPlugin) executeRequest(request *lomipc.ActionRequestData, isHealthy *bool, ctx context.Context) *lomipc.ActionResponseData {
    dummyPlugin.testValue1 = 2
    *isHealthy = true
    if request.Action == "ReturnNilScenario" {
        return nil
    }

    if request.Action == "ReturnNilScenarioWithError" {
        *isHealthy = false
        return nil
    }

    if request.Action == "ReturnNilAfterLongSleep" {
        if dummyPlugin.iteration == 0 {
            dummyPlugin.firstTimeInvocation = time.Now()
            dummyPlugin.iteration = dummyPlugin.iteration + 1
            time.Sleep(5 * time.Second)
        } else if dummyPlugin.iteration == 1 {
            dummyPlugin.secondTimeInvocation = time.Now()
            dummyPlugin.iteration = dummyPlugin.iteration + 1
            time.Sleep(5 * time.Second)
        }
        return nil
    }

    if request.Action == "Sleep" {
        time.Sleep(5 * time.Second)
    }
    return &lomipc.ActionResponseData{ResultCode: -1}
}

func (dummyPlugin *DummyPlugin) executeShutdown() error {
    dummyPlugin.testValue2 = 3
    return nil
}

func (dummyPlugin *DummyPlugin) GetPluginID() PluginId {
    return PluginId{Name: pluginName, Version: version}
}

/* Validates that Init returns error for invalid arguments */
func Test_PeriodicDetectionPluginUtil_InitReturnsErrorForInvalidArgument(t *testing.T) {
    // Mock
    periodicDetectionPluginUtil := &PeriodicDetectionPluginUtil{}
    testRequestFrequency = 1
    actionConfig := lomcommon.ActionCfg_t{Name: pluginName, HeartbeatInt: -1}
    err := periodicDetectionPluginUtil.Init("dummyName", 10, &actionConfig, nil, nil)

    // Assert.
    assert := assert.New(t)
    assert.NotNil(err, "err is expected to be non nil")

    actionConfig = lomcommon.ActionCfg_t{Name: pluginName, HeartbeatInt: 1}
    err = periodicDetectionPluginUtil.Init("dummyName", -10, &actionConfig, nil, nil)

    // Assert.
    assert.NotNil(err, "err is expected to be non nil")

    actionConfig = lomcommon.ActionCfg_t{Name: pluginName, HeartbeatInt: 1}
    err = periodicDetectionPluginUtil.Init("dummyName", 10, &actionConfig, nil, nil)

    // Assert.
    assert.NotNil(err, "err is expected to be non nil")

    actionConfig = lomcommon.ActionCfg_t{HeartbeatInt: 1}
    dummyPlugin := &DummyPlugin{}
    err = periodicDetectionPluginUtil.Init("dummyName", 10, &actionConfig, nil, dummyPlugin.executeShutdown)

    // Assert.
    assert.NotNil(err, "err is expected to be non nil")
    err = periodicDetectionPluginUtil.Init("dummyName", 10, &actionConfig, dummyPlugin.executeRequest, nil)

    // Assert.
    assert.NotNil(err, "err is expected to be non nil")
    actionConfig = lomcommon.ActionCfg_t{HeartbeatInt: 10}
    err = periodicDetectionPluginUtil.Init("", 10, &actionConfig, dummyPlugin.executeRequest, dummyPlugin.executeShutdown)

    // Assert.
    assert.NotNil(err, "err is expected to be non nil")
}

/* Validates that Request returns failure when actionRequest has timeout */
func Test_PeriodicDetectionPluginUtil_ReturnsErrorWhenTimeoutIsInvalid(t *testing.T) {
    // Mock
    periodicDetectionPluginUtil := &PeriodicDetectionPluginUtil{}
    pluginHBChan := make(chan PluginHeartBeat, 2)
    request := &lomipc.ActionRequestData{Timeout: 10}

    // Act
    response := periodicDetectionPluginUtil.Request(pluginHBChan, request)

    // Assert.
    assert := assert.New(t)
    assert.NotNil(response, "response is expected to be non nil")
    assert.Equal(response.ResultCode, ResultCodeInvalidArgument, "resultCode is expected to be ResultCodeInvalidArgument")
}

/* Validates that Request returns successfully when ActionResponseData is non nil */
func Test_PeriodicDetectionPluginUtil_RequestDetectsSuccessfulyAndStopsHeartBeat(t *testing.T) {
    // Mock
    var dummyPlugin Plugin
    testRequestFrequency = 5
    dummyPlugin = &DummyPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "pluginName1", HeartbeatInt: 2}
    dummyPlugin.Init(&actionConfig)
    request := &lomipc.ActionRequestData{}
    pluginHBChan := make(chan PluginHeartBeat)
    totalHbReceived := 0
    go func() {
        for i := 0; i < 10; i++ {
            <-pluginHBChan
            totalHbReceived++
        }
    }()

    // Act
    response := dummyPlugin.Request(pluginHBChan, request)

    // Assert.
    assert := assert.New(t)
    assert.NotNil(response, "response is expected to be non nil")
    assert.Equal(-1, response.ResultCode, "ResultCode is expected to be as sent by dummy plugin")
    time.Sleep(3 * time.Second)
    assert.Equal(1, totalHbReceived, "Hb received should be 1")
    assert.Equal(2, dummyPlugin.(*DummyPlugin).testValue1, "someValue is expected to be 2")
}

/* Validates that the request and heartbeat is aborted on shutdown */
func Test_PeriodicDetectionPluginUtil_EnsureRequestAndHeartbeatAbortedOnShutdown(t *testing.T) {
    var dummyPlugin Plugin
    testRequestFrequency = 2
    dummyPlugin = &DummyPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "pluginName2", HeartbeatInt: 2}
    dummyPlugin.Init(&actionConfig)
    request := &lomipc.ActionRequestData{Action: "ReturnNilScenario"}
    go func() {
        time.Sleep(3 * time.Second)
        dummyPlugin.Shutdown()
    }()
    pluginHBChan := make(chan PluginHeartBeat)
    totalHbReceived := 0
    go func() {
        for i := 0; i < 10; i++ {
            <-pluginHBChan
            totalHbReceived++
        }
    }()
    response := dummyPlugin.Request(pluginHBChan, request)
    assert := assert.New(t)
    assert.NotNil(response, "response is expected to be non nil")
    // Give shutdown 2 seconds to finish its complete execution and also heartbeat to get stopped.
    time.Sleep(1 * time.Second)
    assert.Equal(2, dummyPlugin.(*DummyPlugin).testValue1, "someValue is expected to be 2")
    assert.Equal(3, dummyPlugin.(*DummyPlugin).testValue2, "otherValue is expected to be 3")
    assert.Equal(2, totalHbReceived, "Hb received should be 2")
    assert.Equal(2, response.ResultCode, "ResultCode is expected to be aborted")
}

/* Validates that the heartbeat stops for long running request */
func Test_PeriodicDetectionPluginUtil_EnsureHeartBeatStopsForLongRunningRequest(t *testing.T) {
    var dummyPlugin Plugin
    testRequestFrequency = 2
    dummyPlugin = &DummyPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "pluginName3", HeartbeatInt: 3}
    dummyPlugin.Init(&actionConfig)
    request := &lomipc.ActionRequestData{Action: "Sleep"}
    pluginHBChan := make(chan PluginHeartBeat)
    totalHbReceived := 0
    go func() {
        for i := 0; i < 10; i++ {
            <-pluginHBChan
            totalHbReceived++
        }
    }()
    response := dummyPlugin.Request(pluginHBChan, request)
    assert := assert.New(t)
    assert.NotNil(response, "response is expected to be non nil")
    // Give shutdown few seconds to finish its complete execution and also heartbeat to get stopped.
    assert.Equal(2, dummyPlugin.(*DummyPlugin).testValue1, "someValue is expected to be 2")
    assert.Equal(0, dummyPlugin.(*DummyPlugin).testValue2, "otherValue is expected to be 3")
    assert.Equal(1, totalHbReceived, "Hb received should be 1")
    assert.Equal(-1, response.ResultCode, "ResultCode is expected to be as returned by dummy plugin")
}

/* Validates that the heartbeat is skipped for consecutive errors */
func Test_PeriodicDetectionPluginUtil_HandleHeartBeatSkipsHeartBeatForConsecutiveErrors(t *testing.T) {
    dummyPlugin := &DummyPlugin{}
    dummyPlugin.requestFrequencyInSecs = 5
    dummyPlugin.detectionRunInfo = DetectionRunInfo{durationOfLatestRunInSeconds: 2}
    dummyPlugin.numOfConsecutiveErrors.Store(3)
    dummyPlugin.heartBeatIntervalInSecs = 2
    pluginHBChan := make(chan PluginHeartBeat)
    totalHbReceived := 0
    go func() {
        for i := 0; i < 10; i++ {
            <-pluginHBChan
            totalHbReceived++
        }
    }()

    dummyPlugin.publishHeartBeat(pluginHBChan)
    assert := assert.New(t)
    // Ensure heartbeat is not published.
    time.Sleep(1 * time.Second)
    assert.Equal(0, totalHbReceived, "Hb received should be 0")
}

/* Validates that the heartbeat is skipped after a completion of long run request */
func Test_PeriodicDetectionPluginUtil_HandleHeartBeatSkipsHeartBeatAfterLongRunCompletion(t *testing.T) {
    dummyPlugin := &DummyPlugin{}
    dummyPlugin.requestFrequencyInSecs = 5
    timeNowInUtc := time.Now().UTC().Add(-1 * time.Millisecond)
    dummyPlugin.detectionRunInfo = DetectionRunInfo{durationOfLatestRunInSeconds: 10, currentRunStartTimeInUtc: &timeNowInUtc}
    dummyPlugin.numOfConsecutiveErrors.Store(0)
    dummyPlugin.heartBeatIntervalInSecs = 2
    pluginHBChan := make(chan PluginHeartBeat)
    totalHbReceived := 0
    go func() {
        for i := 0; i < 10; i++ {
            <-pluginHBChan
            totalHbReceived++
        }
    }()

    dummyPlugin.publishHeartBeat(pluginHBChan)
    assert := assert.New(t)
    // Ensure heartbeat is not published
    time.Sleep(1 * time.Second)
    assert.Equal(0, totalHbReceived, "Hb received should be 0")
}

/* Validates that the heartbeat is skipped for long running request */
func Test_PeriodicDetectionPluginUtil_HandleHeartBeatSkipsHeartBeatForLongRunningRequest(t *testing.T) {
    dummyPlugin := &DummyPlugin{}
    dummyPlugin.requestFrequencyInSecs = 2
    timeNowInUtc := time.Now().UTC().Add(-3 * time.Second)
    dummyPlugin.detectionRunInfo = DetectionRunInfo{durationOfLatestRunInSeconds: 1, currentRunStartTimeInUtc: &timeNowInUtc}
    dummyPlugin.numOfConsecutiveErrors.Store(0)
    dummyPlugin.heartBeatIntervalInSecs = 2
    pluginHBChan := make(chan PluginHeartBeat)
    totalHbReceived := 0
    go func() {
        for i := 0; i < 10; i++ {
            <-pluginHBChan
            totalHbReceived++
        }
    }()

    dummyPlugin.publishHeartBeat(pluginHBChan)
    assert := assert.New(t)
    // Ensure heartbeat is not published
    time.Sleep(1 * time.Second)
    assert.Equal(0, totalHbReceived, "Hb received should be 0")
}

/* Validates that the error count is increases for unhealthy execution. */
func Test_PeriodicDetectionPluginUtil_RequestIncrementsErrorCountForUnhealthyExecution(t *testing.T) {
    // Mock
    testRequestFrequency = 30
    dummyPlugin := &DummyPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "pluginName4", HeartbeatInt: 3600}
    dummyPlugin.Init(&actionConfig)
    request := &lomipc.ActionRequestData{Action: "ReturnNilScenarioWithError"}
    pluginHBChan := make(chan PluginHeartBeat)
    dummyPlugin.detectionRunInfo = DetectionRunInfo{durationOfLatestRunInSeconds: 2}
    dummyPlugin.numOfConsecutiveErrors.Store(2)

    go func() {
        time.Sleep(2 * time.Second)
        dummyPlugin.Shutdown()
    }()

    go func() {
        /* read first heartbeat */
        <-pluginHBChan
    }()

    // Act
    response := dummyPlugin.Request(pluginHBChan, request)

    // Assert.
    assert := assert.New(t)
    assert.NotNil(response, "response is expected to be non nil")
    assert.Equal(uint64(3), dummyPlugin.numOfConsecutiveErrors.Load(), "NumOfConsecutiveErrors is expected to be 3")
    assert.Nil(dummyPlugin.detectionRunInfo.currentRunStartTimeInUtc, "currentRunStartTimeInUtc is expected to be nil")
    assert.Equal(2, response.ResultCode, "ResultCode is expected to be 2")
    assert.Equal(2, dummyPlugin.testValue1, "someValue is expected to be 2")
}

/* Validates that the error count is reset after healthy execution. */
func Test_PeriodicDetectionPluginUtil_RequestResetsErrorCountAfterhealthyExecution(t *testing.T) {
    // Mock
    testRequestFrequency = 30
    dummyPlugin := &DummyPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "pluginName5", HeartbeatInt: 3600}
    dummyPlugin.Init(&actionConfig)
    request := &lomipc.ActionRequestData{Action: "ReturnNilScenario"}
    pluginHBChan := make(chan PluginHeartBeat)
    dummyPlugin.detectionRunInfo = DetectionRunInfo{durationOfLatestRunInSeconds: 2}
    dummyPlugin.numOfConsecutiveErrors.Store(2)

    go func() {
        time.Sleep(2 * time.Second)
        dummyPlugin.cancelCtxFunc()
    }()

    go func() {
        /* read first heartbeat */
        <-pluginHBChan
    }()

    // Act
    response := dummyPlugin.Request(pluginHBChan, request)

    // Assert.
    assert := assert.New(t)
    assert.NotNil(response, "response is expected to be non nil")
    assert.Equal(uint64(0), dummyPlugin.numOfConsecutiveErrors.Load(), "NumOfConsecutiveErrors is expected to be 0")
    assert.Nil(dummyPlugin.detectionRunInfo.currentRunStartTimeInUtc, "currentRunStartTimeInUtc is expected to be nil")
    assert.Equal(2, response.ResultCode, "ResultCode is expected to be 2")
    assert.Equal(2, dummyPlugin.testValue1, "someValue is expected to be 2")
}

/* Functional test -> Validates that the detection is performed immediately after a long run execution. */
func Test_PeriodicDetectionPluginUtil_DetectionIsPerformedImmediatelyAfterALongExecution(t *testing.T) {
    // Mock
    testRequestFrequency = 4
    dummyPlugin := &DummyPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "DummyPlugin10", HeartbeatInt: 3600}
    dummyPlugin.Init(&actionConfig)
    request := &lomipc.ActionRequestData{Action: "ReturnNilAfterLongSleep"}
    pluginHBChan := make(chan PluginHeartBeat)

    go func() {
        time.Sleep(7 * time.Second)
        dummyPlugin.cancelCtxFunc()
    }()

    go func() {
        /* read first heartbeat */
        <-pluginHBChan
    }()

    // Act
    response := dummyPlugin.Request(pluginHBChan, request)

    // Assert.
    assert := assert.New(t)
    assert.NotNil(response, "response is expected to be non nil")
    assert.Equal(uint64(0), dummyPlugin.numOfConsecutiveErrors.Load(), "NumOfConsecutiveErrors is expected to be 0")
    assert.NotNil(dummyPlugin.detectionRunInfo.currentRunStartTimeInUtc, "currentRunStartTimeInUtc is expected to be nil")
    assert.True(dummyPlugin.secondTimeInvocation.Sub(dummyPlugin.firstTimeInvocation).Seconds() <= 6, "The invocation after a long running job is expected to be immediate")
    fmt.Println(dummyPlugin.detectionRunInfo.durationOfLatestRunInSeconds)
    assert.True(dummyPlugin.detectionRunInfo.durationOfLatestRunInSeconds >= 5, "The duration of latest run is expected to be greater than or equal to 5 seconds")
}

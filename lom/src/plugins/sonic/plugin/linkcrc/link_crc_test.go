package linkcrc

import (
    "context"
    "errors"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "lom/src/lib/lomcommon"
    "lom/src/lib/lomipc"
    "lom/src/plugins/plugins_common"
    "lom/src/plugins/sonic/client/dbclient"
    "testing"
    "time"
)

func init() {
    configFiles := &lomcommon.ConfigFiles_t{}
    configFiles.GlobalFl = "../../../../lib/lib_test/config/globals.conf.json"
    configFiles.ActionsFl = "../../../../lib/lib_test/config/actions.conf.json"
    configFiles.BindingsFl = "../../../../lib/lib_test/config/bindings.conf.json"
    configFiles.ProcsFl = "../../../../lib/lib_test/config/procs.conf.json"
    lomcommon.InitConfigMgr(configFiles)
}

type MockCounterRepository struct {
    mock.Mock
}

func (mockCounterRepository *MockCounterRepository) GetCountersForAllInterfaces(ctx context.Context) (dbclient.InterfaceCountersMap, error) {
    args := mockCounterRepository.Called(ctx)
    return args.Get(0).(dbclient.InterfaceCountersMap), args.Error(1)
}

func (mockCounterRepository *MockCounterRepository) GetInterfaceStatus(interfaceName string) (bool, bool, error) {
    args := mockCounterRepository.Called(interfaceName)
    return args.Get(0).(bool), args.Get(1).(bool), args.Error(2)
}

/* Validate AddInterfaceCounter detects crc successfuly */
func Test_LinkCrcDetector_AddInterfaceCountersDetectsSuccessfuly(t *testing.T) {
    // Mock
    ifInErrorsDiffMinValue = if_in_errors_diff_min_value_default
    inUnicastPacketsMinValue = in_unicast_packets_min_value_default
    outUnicastPacketsMinValue = out_unicast_packets_min_value_default
    outlierRollingWindowSize = outlier_rolling_window_size_default
    minCrcError = min_crc_error_default
    minOutliersForDetection = min_outliers_for_detection_default
    lookBackPeriodInSecs = look_back_period_in_secs_default

    rollingWindowCrcDetector := RollingWindowLinkCrcDetector{}
    rollingWindowCrcDetector.Initialize()

    map1 := map[string]uint64{"IfInErrors": 100, "InUnicastPackets": 101, "OutUnicastPackets": 1100, "IfOutErrors": 1}
    map2 := map[string]uint64{"IfInErrors": 450, "InUnicastPackets": 222, "OutUnicastPackets": 2100, "IfOutErrors": 2}
    map3 := map[string]uint64{"IfInErrors": 850, "InUnicastPackets": 333, "OutUnicastPackets": 3100000000, "IfOutErrors": 3}
    map4 := map[string]uint64{"IfInErrors": 1220, "InUnicastPackets": 444, "OutUnicastPackets": 4100000000, "IfOutErrors": 4}
    map5 := map[string]uint64{"IfInErrors": 1650, "InUnicastPackets": 555, "OutUnicastPackets": 4100004000, "IfOutErrors": 5}

    // Assert
    assert := assert.New(t)

    isDetected := rollingWindowCrcDetector.AddInterfaceCountersAndDetectCrc(map1, time.Now())
    assert.False(isDetected, "isDetected is expected to be false for first call")

    // This is an outlier.
    isDetected = rollingWindowCrcDetector.AddInterfaceCountersAndDetectCrc(map2, time.Now())
    assert.False(isDetected, "isDetected is expected to be false for second call")

    isDetected = rollingWindowCrcDetector.AddInterfaceCountersAndDetectCrc(map3, time.Now())
    assert.False(isDetected, "isDetected is expected to be false for third call")

    isDetected = rollingWindowCrcDetector.AddInterfaceCountersAndDetectCrc(map4, time.Now())
    assert.False(isDetected, "isDetected is expected to be false for fourth call")

    // This is an outlier.
    isDetected = rollingWindowCrcDetector.AddInterfaceCountersAndDetectCrc(map5, time.Now())
    assert.True(isDetected, "isDetected is expected to be True for fifth call")

    outlierRollingWindow := rollingWindowCrcDetector.outlierRollingWindow
    assert.Equal(2, outlierRollingWindow.GetElements().Len(), "Length of rolling window is expected to be 2")
}

/* Validate AddInterfaceCountersAndDetectCrc returns false for nil counters */
func Test_LinkCrcDetector_AddInterfaceCountersReturnsFalseForNilCounters(t *testing.T) {
    // Mock
    ifInErrorsDiffMinValue = if_in_errors_diff_min_value_default
    inUnicastPacketsMinValue = in_unicast_packets_min_value_default
    outUnicastPacketsMinValue = out_unicast_packets_min_value_default
    outlierRollingWindowSize = outlier_rolling_window_size_default
    minCrcError = min_crc_error_default
    minOutliersForDetection = min_outliers_for_detection_default
    lookBackPeriodInSecs = look_back_period_in_secs_default
    // Act
    rollingWindowCrcDetector := RollingWindowLinkCrcDetector{}
    rollingWindowCrcDetector.Initialize()
    // Assert
    assert := assert.New(t)

    isDetected := rollingWindowCrcDetector.AddInterfaceCountersAndDetectCrc(nil, time.Now())
    assert.False(isDetected, "isDetected is expected to be false for nil interface coutners")
}

/* Validate AddInterfaceCountersAndDetectCrc returns false for invalid diff counters */
func Test_LinkCrcDetector_AddInterfaceCountersReturnsFalseForInvalidCountersDiff(t *testing.T) {
    // Mock
    ifInErrorsDiffMinValue = if_in_errors_diff_min_value_default
    inUnicastPacketsMinValue = in_unicast_packets_min_value_default
    outUnicastPacketsMinValue = out_unicast_packets_min_value_default
    outlierRollingWindowSize = outlier_rolling_window_size_default
    minCrcError = min_crc_error_default
    minOutliersForDetection = min_outliers_for_detection_default
    lookBackPeriodInSecs = look_back_period_in_secs_default
    // Act
    rollingWindowCrcDetector := RollingWindowLinkCrcDetector{}
    rollingWindowCrcDetector.Initialize()
    // Assert
    assert := assert.New(t)

    rollingWindowCrcDetector.latestCounters = map[string]uint64{"IfInErrors": 150, "IfOutErrors": 151, "InUnicastPackets": 152, "OutUnicastPackets": 153}
    currentCounters := map[string]uint64{"IfInErrors": 120, "IfOutErrors": 1051, "InUnicastPackets": 1052, "OutUnicastPackets": 1053}

    isDetected := rollingWindowCrcDetector.AddInterfaceCountersAndDetectCrc(currentCounters, time.Now())
    assert.False(isDetected, "isDetected is expected to be false for when IfInErrors is less than previous value")

    rollingWindowCrcDetector.latestCounters = map[string]uint64{"IfInErrors": 150, "IfOutErrors": 151, "InUnicastPackets": 152, "OutUnicastPackets": 153}
    currentCounters = map[string]uint64{"IfInErrors": 1050, "IfOutErrors": 121, "InUnicastPackets": 1052, "OutUnicastPackets": 1053}

    isDetected = rollingWindowCrcDetector.AddInterfaceCountersAndDetectCrc(currentCounters, time.Now())
    assert.False(isDetected, "isDetected is expected to be false for when IfOutErrors is less than previous value")

    rollingWindowCrcDetector.latestCounters = map[string]uint64{"IfInErrors": 150, "IfOutErrors": 151, "InUnicastPackets": 152, "OutUnicastPackets": 153}
    currentCounters = map[string]uint64{"IfInErrors": 1050, "IfOutErrors": 1051, "InUnicastPackets": 122, "OutUnicastPackets": 1053}

    isDetected = rollingWindowCrcDetector.AddInterfaceCountersAndDetectCrc(currentCounters, time.Now())
    assert.False(isDetected, "isDetected is expected to be false for when InUnicastPackets is less than previous value")

    rollingWindowCrcDetector.latestCounters = map[string]uint64{"IfInErrors": 150, "IfOutErrors": 151, "InUnicastPackets": 152, "OutUnicastPackets": 153}
    currentCounters = map[string]uint64{"IfInErrors": 1050, "IfOutErrors": 1051, "InUnicastPackets": 1052, "OutUnicastPackets": 123}

    isDetected = rollingWindowCrcDetector.AddInterfaceCountersAndDetectCrc(currentCounters, time.Now())
    assert.False(isDetected, "isDetected is expected to be false for when InUnicastPackets is less than previous value")
}

/* Validates Link crc plugin initialized with actions knobs */
func Test_LinkCrcDetectionPlugin_InitializesWithActionsKnobs(t *testing.T) {
    linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
    actionKnobs := `{
    "DetectionFreqInSecs": 35,
    "IfInErrorsDiffMinValue": 5,
    "InUnicastPacketsMinValue": 105,
    "OutUnicastPacketsMinValue": 105,
    "OutlierRollingWindowSize": 6,
    "MinCrcError": 0.000002,
    "MinOutliersForDetection": 3,
    "LookBackPeriodInSecs": 127
    }`

    actionConfig := lomcommon.ActionCfg_t{HeartbeatInt: 10, ActionKnobs: actionKnobs}
    linkCRCDetectionPlugin.Init(&actionConfig)

    assert := assert.New(t)
    assert.Equal(5, ifInErrorsDiffMinValue, "IfInErrorsDiffMinValue is expected to be 5")
    assert.Equal(105, inUnicastPacketsMinValue, "InUnicastPacketsMinValue is expected to be 105")
    assert.Equal(105, outUnicastPacketsMinValue, "OutUnicastPacketsMinValue is expected to be 105")
    assert.Equal(6, outlierRollingWindowSize, "OutlierRollingWindowSize is expected to be 6")
    assert.Equal(0.000002, minCrcError, "MinCrcError is expected to be 0.000002")
    assert.Equal(3, minOutliersForDetection, "MinOutliersForDetection is expected to be 3")
    assert.Equal(127, lookBackPeriodInSecs, "LookBackPeriodInSecs is expected to be 127")
}

/* Validates DetectCrc returns nil for error */
func Test_LinkCrcDetectionPlugin_DetectCrcReturnsNilForError(t *testing.T) {
    // Mock
    linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "linkCrc", HeartbeatInt: 10}
    linkCRCDetectionPlugin.Init(&actionConfig)
    mockCounterRepository := new(MockCounterRepository)
    ctx, _ := context.WithCancel(context.Background())
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(dbclient.InterfaceCountersMap(nil), errors.New("Some Error"))
    linkCRCDetectionPlugin.counterRepository = mockCounterRepository
    request := &lomipc.ActionRequestData{Action: "action", InstanceId: "InstanceId", AnomalyInstanceId: "AnmlyInstId", Timeout: 0}
    isHealthy := true
    // Act
    response := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    assert := assert.New(t)
    // Assert
    assert.Nil(response, "response is expected to be nil")
    assert.False(isHealthy, "isHealthy is expected to be False")
    mockCounterRepository.AssertNumberOfCalls(t, "GetCountersForAllInterfaces", 1)
    mockCounterRepository.AssertExpectations(t)
}

/* Validates executeCrcDetection returns nil for empty interfaces from redis  */
func Test_LinkCrcDetectionPlugin_DetectCrcReturnsNilForEmptyInterfacesFromRedis(t *testing.T) {
    // Mock
    linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "linkCrc", HeartbeatInt: 10}
    linkCRCDetectionPlugin.Init(&actionConfig)
    mockCounterRepository := new(MockCounterRepository)
    interfaceCountersMap := new(dbclient.InterfaceCountersMap)
    ctx, _ := context.WithCancel(context.Background())
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(*interfaceCountersMap, nil)
    linkCRCDetectionPlugin.counterRepository = mockCounterRepository
    // Act
    request := &lomipc.ActionRequestData{Action: "action", InstanceId: "InstanceId", AnomalyInstanceId: "AnmlyInstId", Timeout: 0}
    isHealthy := true
    response := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    // Assert
    assert := assert.New(t)
    assert.Nil(response, "response is expected to be nil")
    assert.False(isHealthy, "isHealthy is expected to be False")
    mockCounterRepository.AssertNumberOfCalls(t, "GetCountersForAllInterfaces", 1)
    mockCounterRepository.AssertExpectations(t)
}

/* Validates executeCrcDetection ignores empty interface counters from redis  */
func Test_LinkCrcDetectionPlugin_DetectCrcReturnsNilForEmptyInterfaceCountersFromRedis(t *testing.T) {
    // Mock
    linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "LinkCrc", HeartbeatInt: 10}
    linkCRCDetectionPlugin.Init(&actionConfig)
    mockCounterRepository := new(MockCounterRepository)
    interfaceCountersMap := dbclient.InterfaceCountersMap{"ethernet0": nil}
    ctx, _ := context.WithCancel(context.Background())
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(interfaceCountersMap, nil)
    linkCRCDetectionPlugin.counterRepository = mockCounterRepository
    request := &lomipc.ActionRequestData{Action: "action", InstanceId: "InstanceId", AnomalyInstanceId: "AnmlyInstId", Timeout: 0}
    isHealthy := true
    // Act
    response := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    // Assert
    assert := assert.New(t)
    assert.Nil(response, "response is expected to be nil")
    assert.False(isHealthy, "isHealthy is expected to be False")
    assert.Equal(0, len(linkCRCDetectionPlugin.currentMonitoredInterfaces), "Monitored interfaces length is expected to be 0")
    mockCounterRepository.AssertNumberOfCalls(t, "GetCountersForAllInterfaces", 1)
    mockCounterRepository.AssertExpectations(t)
}

/* Validates GetPluginId returns plugin details. */
func Test_LinkCrcDetectionPlugin_GetPluginIdReturnsPluginDetails(t *testing.T) {
    // Mock
    linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "LinkCrc", HeartbeatInt: 10}
    linkCRCDetectionPlugin.Init(&actionConfig)
    // Act
    pluginId := linkCRCDetectionPlugin.GetPluginId()
    // Assert
    assert := assert.New(t)
    assert.NotNil(pluginId, "pluginId is expected to be non nil")
    assert.Equal("LinkCrc", pluginId.Name, "PluginId.Name is expected to be LinkCrc")
    assert.Equal("1.0.0.0", pluginId.Version, "PluginId.version  is expected to be 1.0.0.0")
}

/* Validates executeShutdown returns successfully */
func Test_LinkCrcDetectionPlugin_ExecuteShutdownReturnsSuccessfuly(t *testing.T) {
    // Mock
    linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "LinkCrc", HeartbeatInt: 10}
    initErr := linkCRCDetectionPlugin.Init(&actionConfig)
    // Act
    shutDownErr := linkCRCDetectionPlugin.executeShutdown()
    // Assert
    assert := assert.New(t)
    assert.Nil(initErr, "initErr is expected to be nil")
    assert.Nil(shutDownErr, "shutDownErr is expected to be nil")
}

/* Validates DetectCrc returns error when ActionConfig is invalid */
func Test_LinkCrcDetectionPlugin_InitReturnsErrorForInvalidActionConfig(t *testing.T) {
    // Mock
    linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
    actionConfig := lomcommon.ActionCfg_t{HeartbeatInt: 10}

    // Act
    err := linkCRCDetectionPlugin.Init(&actionConfig)

    // Assert
    assert := assert.New(t)
    assert.NotNil(err, "response is expected to be non nil")
}

/* Validates executeCrcDetection detects successfuly */
func Test_LinkCrcDetectionPlugin_CrcDetectionDetectsSuccessfuly(t *testing.T) {
    // Mock
    linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "linkCrc", HeartbeatInt: 10}
    linkCRCDetectionPlugin.Init(&actionConfig)

    map1 := map[string]uint64{"IfInErrors": 100, "InUnicastPackets": 101, "OutUnicastPackets": 1100, "IfOutErrors": 1}
    map2 := map[string]uint64{"IfInErrors": 450, "InUnicastPackets": 222, "OutUnicastPackets": 2100, "IfOutErrors": 2}
    map3 := map[string]uint64{"IfInErrors": 850, "InUnicastPackets": 333, "OutUnicastPackets": 3100000000, "IfOutErrors": 3}
    map4 := map[string]uint64{"IfInErrors": 1220, "InUnicastPackets": 444, "OutUnicastPackets": 4100000000, "IfOutErrors": 4}
    map5 := map[string]uint64{"IfInErrors": 1650, "InUnicastPackets": 555, "OutUnicastPackets": 4100004000, "IfOutErrors": 5}

    counterMap1 := dbclient.InterfaceCountersMap{"Ethernet1": map1, "Ethernet2": map1}
    counterMap2 := dbclient.InterfaceCountersMap{"Ethernet1": map2, "Ethernet2": map2}
    counterMap3 := dbclient.InterfaceCountersMap{"Ethernet1": map3, "Ethernet2": map3}
    counterMap4 := dbclient.InterfaceCountersMap{"Ethernet1": map4, "Ethernet2": map4}
    counterMap5 := dbclient.InterfaceCountersMap{"Ethernet1": map5, "Ethernet2": map5}
    mockCounterRepository := new(MockCounterRepository)
    ctx, _ := context.WithCancel(context.Background())
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap1, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap2, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap3, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap4, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap5, nil).Once()

    mockCounterRepository.On("GetInterfaceStatus", "Ethernet1").Return(true, true, nil).Once()
    mockCounterRepository.On("GetInterfaceStatus", "Ethernet2").Return(true, true, nil).Once()
    linkCRCDetectionPlugin.counterRepository = mockCounterRepository

    request := &lomipc.ActionRequestData{Action: "action", InstanceId: "InstanceId", AnomalyInstanceId: "AnmlyInstId", Timeout: 0}
    isHealthy := true
    response1 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response2 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response3 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response4 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response5 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    // Assert
    assert := assert.New(t)
    assert.Nil(response1, "response is expected to be nil")
    assert.Nil(response2, "response is expected to be nil")
    assert.Nil(response3, "response is expected to be nil")
    assert.Nil(response4, "response is expected to be nil")
    assert.NotNil(response5, "response is expected to be non nil")
    assert.Contains(response5.AnomalyKey, "Ethernet1", "AnomalyKey is expected to be Ethernet0,Ethernet1")
    assert.Contains(response5.AnomalyKey, "Ethernet2", "AnomalyKey is expected to be Ethernet0,Ethernet1")
}

type MockLimitDetectionReportingFrequency struct {
    mock.Mock
}

func (mockLimitDetectionReportingFrequency *MockLimitDetectionReportingFrequency) Initialize(initialReportingFreqInMins int, subsequentReportingFreqInMins int, initialReportingMaxCount int) {
    mockLimitDetectionReportingFrequency.Called(initialReportingFreqInMins, subsequentReportingFreqInMins, initialReportingMaxCount)
}

func (mockLimitDetectionReportingFrequency *MockLimitDetectionReportingFrequency) ShouldReport(anomalyKey string) bool {
    args := mockLimitDetectionReportingFrequency.Called(anomalyKey)
    return args.Get(0).(bool)
}

func (mockLimitDetectionReportingFrequency *MockLimitDetectionReportingFrequency) ResetCache(anomalyKey string) {
    mockLimitDetectionReportingFrequency.Called(anomalyKey)
}

func (mockLimitDetectionReportingFrequency *MockLimitDetectionReportingFrequency) IsNotWithinFrequency(reportingDetails plugins_common.ReportingDetails) bool {
    args := mockLimitDetectionReportingFrequency.Called(reportingDetails)
    return args.Get(0).(bool)
}

/* Validates executeCrcDetection reports only for one interface */
func Test_LinkCrcDetectionPlugin_CrcDetectionReportsForOnlyOneInterface(t *testing.T) {
    // Mock
    linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "linkCrc", HeartbeatInt: 10}
    mockLimitDetectionReportingFrequency := new(MockLimitDetectionReportingFrequency)
    mockLimitDetectionReportingFrequency.On("ShouldReport", "Ethernet1").Return(true)
    mockLimitDetectionReportingFrequency.On("ShouldReport", "Ethernet2").Return(false)
    mockLimitDetectionReportingFrequency.On("ResetCache", "Ethernet1").Return()
    mockLimitDetectionReportingFrequency.On("ResetCache", "Ethernet2").Return()
    linkCRCDetectionPlugin.Init(&actionConfig)
    linkCRCDetectionPlugin.reportingFreqLimiter = mockLimitDetectionReportingFrequency

    map1 := map[string]uint64{"IfInErrors": 100, "InUnicastPackets": 101, "OutUnicastPackets": 1100, "IfOutErrors": 1}
    map2 := map[string]uint64{"IfInErrors": 450, "InUnicastPackets": 222, "OutUnicastPackets": 2100, "IfOutErrors": 2}
    map3 := map[string]uint64{"IfInErrors": 850, "InUnicastPackets": 333, "OutUnicastPackets": 3100000000, "IfOutErrors": 3}
    map4 := map[string]uint64{"IfInErrors": 1220, "InUnicastPackets": 444, "OutUnicastPackets": 4100000000, "IfOutErrors": 4}
    map5 := map[string]uint64{"IfInErrors": 1650, "InUnicastPackets": 555, "OutUnicastPackets": 4100004000, "IfOutErrors": 5}

    counterMap1 := dbclient.InterfaceCountersMap{"Ethernet1": map1, "Ethernet2": map1}
    counterMap2 := dbclient.InterfaceCountersMap{"Ethernet1": map2, "Ethernet2": map2}
    counterMap3 := dbclient.InterfaceCountersMap{"Ethernet1": map3, "Ethernet2": map3}
    counterMap4 := dbclient.InterfaceCountersMap{"Ethernet1": map4, "Ethernet2": map4}
    counterMap5 := dbclient.InterfaceCountersMap{"Ethernet1": map5, "Ethernet2": map5}
    mockCounterRepository := new(MockCounterRepository)
    ctx, _ := context.WithCancel(context.Background())
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap1, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap2, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap3, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap4, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap5, nil).Once()
    mockCounterRepository.On("GetInterfaceStatus", "Ethernet1").Return(true, true, nil).Once()
    linkCRCDetectionPlugin.counterRepository = mockCounterRepository
    // Act
    request := &lomipc.ActionRequestData{Action: "action", InstanceId: "InstanceId", AnomalyInstanceId: "AnmlyInstId", Timeout: 0}
    isHealthy := true
    response1 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response2 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response3 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response4 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response5 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    // Assert
    assert := assert.New(t)
    assert.Nil(response1, "response is expected to be nil")
    assert.Nil(response2, "response is expected to be nil")
    assert.Nil(response3, "response is expected to be nil")
    assert.Nil(response4, "response is expected to be nil")
    assert.NotNil(response5, "response is expected to be non nil")
    assert.Equal("Ethernet1", response5.AnomalyKey, "AnomalyKey is expected to be Ethernet0")
}

/* Validates executeCrcDetection reports none */
func Test_LinkCrcDetectionPlugin_CrcDetectionReportsNone(t *testing.T) {
    // Mock
    linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
    actionConfig := lomcommon.ActionCfg_t{Name: "linkCrc", HeartbeatInt: 10}
    mockLimitDetectionReportingFrequency := new(MockLimitDetectionReportingFrequency)
    mockLimitDetectionReportingFrequency.On("ShouldReport", "Ethernet1").Return(false)
    mockLimitDetectionReportingFrequency.On("ShouldReport", "Ethernet2").Return(false)
    mockLimitDetectionReportingFrequency.On("ResetCache", "Ethernet1").Return()
    mockLimitDetectionReportingFrequency.On("ResetCache", "Ethernet2").Return()
    linkCRCDetectionPlugin.Init(&actionConfig)
    linkCRCDetectionPlugin.reportingFreqLimiter = mockLimitDetectionReportingFrequency

    map1 := map[string]uint64{"IfInErrors": 100, "InUnicastPackets": 101, "OutUnicastPackets": 1100, "IfOutErrors": 1}
    map2 := map[string]uint64{"IfInErrors": 450, "InUnicastPackets": 222, "OutUnicastPackets": 2100, "IfOutErrors": 2}
    map3 := map[string]uint64{"IfInErrors": 850, "InUnicastPackets": 333, "OutUnicastPackets": 3100000000, "IfOutErrors": 3}
    map4 := map[string]uint64{"IfInErrors": 1220, "InUnicastPackets": 444, "OutUnicastPackets": 4100000000, "IfOutErrors": 4}
    map5 := map[string]uint64{"IfInErrors": 1650, "InUnicastPackets": 555, "OutUnicastPackets": 4100004000, "IfOutErrors": 5}

    counterMap1 := dbclient.InterfaceCountersMap{"Ethernet1": map1, "Ethernet2": map1}
    counterMap2 := dbclient.InterfaceCountersMap{"Ethernet1": map2, "Ethernet2": map2}
    counterMap3 := dbclient.InterfaceCountersMap{"Ethernet1": map3, "Ethernet2": map3}
    counterMap4 := dbclient.InterfaceCountersMap{"Ethernet1": map4, "Ethernet2": map4}
    counterMap5 := dbclient.InterfaceCountersMap{"Ethernet1": map5, "Ethernet2": map5}
    mockCounterRepository := new(MockCounterRepository)
    ctx, _ := context.WithCancel(context.Background())
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap1, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap2, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap3, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap4, nil).Once()
    mockCounterRepository.On("GetCountersForAllInterfaces", ctx).Return(counterMap5, nil).Once()
    linkCRCDetectionPlugin.counterRepository = mockCounterRepository

    request := &lomipc.ActionRequestData{Action: "action", InstanceId: "InstanceId", AnomalyInstanceId: "AnmlyInstId", Timeout: 0}
    isHealthy := true
    // Act
    response1 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response2 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response3 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response4 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    response5 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy, ctx)
    // Assert
    assert := assert.New(t)
    assert.Nil(response1, "response is expected to be nil")
    assert.Nil(response2, "response is expected to be nil")
    assert.Nil(response3, "response is expected to be nil")
    assert.Nil(response4, "response is expected to be nil")
    assert.Nil(response5, "response is expected to be nil")
}

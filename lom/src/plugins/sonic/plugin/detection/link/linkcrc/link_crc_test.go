package linkcrc

import (
	"errors"
	"testing"
        "time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
        "lom/src/lib/lomcommon"
        "lom/src/lib/lomipc"
	"lom/src/plugins/sonic/client/dbclient"
)

type MockCounterRepository struct {
	mock.Mock
}

func (mockCounterRepository *MockCounterRepository) GetCountersForActiveInterfaces() (dbclient.InterfaceCountersMap, error) {
	args := mockCounterRepository.Called()
	return args.Get(0).(dbclient.InterfaceCountersMap), args.Error(1)
}

func (mockCounterRepository *MockCounterRepository) IsInterfaceActive(interfaceName string) (bool, error) {
	args := mockCounterRepository.Called(interfaceName)
	return args.Get(0).(bool), args.Error(1)
}

/* Validate AddInterfaceCounter detects crc successfuly */
func Test_LinkCrcDetector_AddInterfaceCountersDetectsSuccessfuly(t *testing.T) {
	ifInErrorsDiffMinValue = if_in_errors_diff_min_value_default
	inUnicastPacketsMinValue = in_unicast_packets_min_value_default
	outUnicastPacketsMinValue = out_unicast_packets_min_value_default
	outlierRollingWindowSize = outlier_rolling_window_size_default
	minCrcError = min_crc_error_default
	minOutliersForDetection = min_outliers_for_detection_default
	lookBackPeriodInSecs = look_back_period_in_secs_default

	rollingWindowCrcDetector := LinkCrcDetector{}
	rollingWindowCrcDetector.Initialize()

	map1 := map[string]uint64{"IfInErrors": 100, "InUnicastPackets": 101, "OutUnicastPackets": 1100, "IfOutErrors": 1}
	map2 := map[string]uint64{"IfInErrors": 450, "InUnicastPackets": 222, "OutUnicastPackets": 2100, "IfOutErrors": 2}
	map3 := map[string]uint64{"IfInErrors": 850, "InUnicastPackets": 333, "OutUnicastPackets": 3100000000, "IfOutErrors": 3}
	map4 := map[string]uint64{"IfInErrors": 1220, "InUnicastPackets": 444, "OutUnicastPackets": 4100000000, "IfOutErrors": 4}
	map5 := map[string]uint64{"IfInErrors": 1650, "InUnicastPackets": 555, "OutUnicastPackets": 4100004000, "IfOutErrors": 5}

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

/* Validates DetectCrc returns nil for error */
func Test_LinkCrcDetectionPlugin_DetectCrcReturnsNilForError(t *testing.T) {
	linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
	actionConfig := lomcommon.ActionCfg_t{HeartbeatInt: 10}
	linkCRCDetectionPlugin.Init(&actionConfig)

	mockCounterRepository := new(MockCounterRepository)
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(dbclient.InterfaceCountersMap(nil), errors.New("Some Error"))
	linkCRCDetectionPlugin.counterRepository = mockCounterRepository

	request := &lomipc.ActionRequestData{Action: "action", InstanceId: "InstanceId", AnomalyInstanceId: "AnmlyInstId", Timeout: 0}
	isHealthy := true
	response := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	assert := assert.New(t)
	assert.Nil(response, "response is expected to be nil")
}

/* Validates executeCrcDetection detects successfuly */
func Test_LinkCrcDetectionPlugin_CrcDetectionDetectsSuccessfuly(t *testing.T) {
	linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
	actionConfig := lomcommon.ActionCfg_t{HeartbeatInt: 10}
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
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap1, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap2, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap3, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap4, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap5, nil).Once()
	linkCRCDetectionPlugin.counterRepository = mockCounterRepository

	request := &lomipc.ActionRequestData{Action: "action", InstanceId: "InstanceId", AnomalyInstanceId: "AnmlyInstId", Timeout: 0}
	isHealthy := true
	response1 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response2 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response3 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response4 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response5 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
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

/* Validates executeCrcDetection reports only for one interface */
func Test_LinkCrcDetectionPlugin_CrcDetectionReportsForOnlyOneInterface(t *testing.T) {
	linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
	actionConfig := lomcommon.ActionCfg_t{HeartbeatInt: 10}
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
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap1, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap2, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap3, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap4, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap5, nil).Once()
	linkCRCDetectionPlugin.counterRepository = mockCounterRepository

	request := &lomipc.ActionRequestData{Action: "action", InstanceId: "InstanceId", AnomalyInstanceId: "AnmlyInstId", Timeout: 0}
	isHealthy := true
	response1 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response2 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response3 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response4 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response5 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
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
	linkCRCDetectionPlugin := LinkCRCDetectionPlugin{}
	actionConfig := lomcommon.ActionCfg_t{HeartbeatInt: 10}
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
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap1, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap2, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap3, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap4, nil).Once()
	mockCounterRepository.On("GetCountersForActiveInterfaces").Return(counterMap5, nil).Once()
	linkCRCDetectionPlugin.counterRepository = mockCounterRepository

	request := &lomipc.ActionRequestData{Action: "action", InstanceId: "InstanceId", AnomalyInstanceId: "AnmlyInstId", Timeout: 0}
	isHealthy := true
	response1 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response2 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response3 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response4 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	response5 := linkCRCDetectionPlugin.executeCrcDetection(request, &isHealthy)
	assert := assert.New(t)
	assert.Nil(response1, "response is expected to be nil")
	assert.Nil(response2, "response is expected to be nil")
	assert.Nil(response3, "response is expected to be nil")
	assert.Nil(response4, "response is expected to be nil")
	assert.Nil(response5, "response is expected to be nil")
}


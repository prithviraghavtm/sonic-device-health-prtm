/* This file contains logic for Link CRC anomaly detection. It detects CRC anomlies for all the interfaces on the device.*/
package linkcrc

import (
	"fmt"
	"strings"
	"time"
        "lom/src/plugins/sonic/client/dbclient"
        "lom/src/plugins/plugins_common"
        "lom/src/lib/lomcommon"
        "lom/src/lib/lomipc"
)

const (
	/* Default values to be used for the detection, in case configuration could not be read */
	detection_freq_in_secs_default        = 30
	if_in_errors_diff_min_value_default   = 0
	in_unicast_packets_min_value_default  = 100
	out_unicast_packets_min_value_default = 100
	outlier_rolling_window_size_default   = 5
	min_crc_error_default                 = 0.000001
	min_outliers_for_detection_default    = 2
	look_back_period_in_secs_default      = 125

	/* Config Keys for accessing cfg file */
	detection_freq_in_secs_config_key        = "DetectionFreqInSecs"
	if_in_errors_diff_min_value_config_key   = "IfInErrorsDiffMinValue"
	in_unicast_packets_min_value_config_key  = "InUnicastPacketsMinValue"
	out_unicast_packets_min_value_config_key = "OutUnicastPacketsMinValue"
	outlier_rolling_window_size_config_key   = "OutlierRollingWindowSize"
	min_crc_error_config_key                 = "MinCrcError"
	min_outliers_for_detection_config_key    = "MinOutliersForDetection"
	look_back_period_in_secs_config_key      = "LookBackPeriodInSecs"
	plugin_version                           = "1.0.0.0"
)

var ifInErrorsDiffMinValue int
var inUnicastPacketsMinValue int
var outUnicastPacketsMinValue int
var outlierRollingWindowSize int
var minCrcError float64
var minOutliersForDetection int
var lookBackPeriodInSecs int

type LinkCRCDetectionPlugin struct {
	counterRepository    dbclient.CounterRepositoryInterface
	monitoredInterfaces  map[string]LinkCrcDetectorInterface
	reportingFreqLimiter plugins_common.PluginReportingFrequencyLimiterInterface
	plugins_common.PeriodicDetectionPluginUtil
}

/* Inheritied from Plugin */
func (linkCrcDetectionPlugin *LinkCRCDetectionPlugin) Init(actionConfig *lomcommon.ActionCfg_t) error {
	// Get config settings or assign default values.
	actionKnobsJsonString := actionConfig.ActionKnobs
	detectionFreqInSecs := lomcommon.GetIntConfigurationFromJson(actionKnobsJsonString, detection_freq_in_secs_config_key, detection_freq_in_secs_default)
	ifInErrorsDiffMinValue = lomcommon.GetIntConfigurationFromJson(actionKnobsJsonString, if_in_errors_diff_min_value_config_key, if_in_errors_diff_min_value_default)
	inUnicastPacketsMinValue = lomcommon.GetIntConfigurationFromJson(actionKnobsJsonString, in_unicast_packets_min_value_config_key, in_unicast_packets_min_value_default)
	outUnicastPacketsMinValue = lomcommon.GetIntConfigurationFromJson(actionKnobsJsonString, out_unicast_packets_min_value_config_key, out_unicast_packets_min_value_default)
	outlierRollingWindowSize = lomcommon.GetIntConfigurationFromJson(actionKnobsJsonString, outlier_rolling_window_size_config_key, outlier_rolling_window_size_default)
	minCrcError = lomcommon.GetFloatConfigurationFromJson(actionKnobsJsonString, min_crc_error_config_key, min_crc_error_default)
	minOutliersForDetection = lomcommon.GetIntConfigurationFromJson(actionKnobsJsonString, min_outliers_for_detection_config_key, min_outliers_for_detection_default)
	lookBackPeriodInSecs = lomcommon.GetIntConfigurationFromJson(actionKnobsJsonString, look_back_period_in_secs_config_key, look_back_period_in_secs_default)

	// Initialize values.
	linkCrcDetectionPlugin.counterRepository = &dbclient.CounterRepository{RedisProvider: &dbclient.RedisProvider{}}
	linkCrcDetectionPlugin.monitoredInterfaces = map[string]LinkCrcDetectorInterface{}
	linkCrcDetectionPlugin.reportingFreqLimiter = plugins_common.GetDefaultDetectionFrequencyLimiter()
	err := linkCrcDetectionPlugin.PeriodicDetectionPluginUtil.Init(actionConfig.Name, detectionFreqInSecs, actionConfig, linkCrcDetectionPlugin.executeCrcDetection, linkCrcDetectionPlugin.executeShutdown)
	if err != nil {
		lomcommon.LogError(fmt.Sprintf("Plugin initialization failed. (%s), err: (%v)", actionConfig.Name, err))
		return err
	}
	return nil
}

func (linkCrcDetectionPlugin *LinkCRCDetectionPlugin) executeCrcDetection(request *lomipc.ActionRequestData, isExecutionHealthy *bool) *lomipc.ActionResponseData {
	lomcommon.LogInfo(fmt.Sprintf("executeCrcDetection Starting"))
	ifAnyInterfaceHasCrcError := false
	var listOfInterfacesWithCrcError strings.Builder
	currentInterfaceCounters, err := linkCrcDetectionPlugin.counterRepository.GetCountersForActiveInterfaces()
	if err != nil {
		lomcommon.LogError("Error fetching interface counters for LinkCrc detection")
		*isExecutionHealthy = false
		return nil
	}

	if len(currentInterfaceCounters) == 0 {
		*isExecutionHealthy = false
		lomcommon.LogError("interface counters is 0")
		return nil
	}

	for interfaceName, interfaceCounters := range currentInterfaceCounters {
		if interfaceCounters == nil {
			lomcommon.LogError(fmt.Sprintf("Nil interface Counters for %s", interfaceName))
			continue
		}

		linkCrcDetector, ok := linkCrcDetectionPlugin.monitoredInterfaces[interfaceName]
		if !ok {
			linkCrcDetector := &LinkCrcDetector{}
			linkCrcDetectionPlugin.monitoredInterfaces[interfaceName] = linkCrcDetector
			linkCrcDetector.Initialize()
			linkCrcDetector.AddInterfaceCountersAndDetectCrc(interfaceCounters, time.Now().UTC())
		} else {
			if linkCrcDetector.AddInterfaceCountersAndDetectCrc(interfaceCounters, time.Now().UTC()) {
				if linkCrcDetectionPlugin.reportingFreqLimiter.ShouldReport(interfaceName) {
					ifAnyInterfaceHasCrcError = true
					listOfInterfacesWithCrcError.WriteString(interfaceName)
					listOfInterfacesWithCrcError.WriteString(",")
				}
			} else {
				// reset limiter freq when detection is false.
				linkCrcDetectionPlugin.reportingFreqLimiter.ResetCache(interfaceName)
			}
		}
	}
	*isExecutionHealthy = true
	if ifAnyInterfaceHasCrcError {
		lomcommon.LogInfo("executeCrcDetection Anomaly Detected")
		return plugins_common.GetResponse(request, strings.TrimSuffix(listOfInterfacesWithCrcError.String(), ","), "Detected Crc", plugins_common.ResultCodeSuccess, plugins_common.ResultStringSuccess)
	}
	return nil
}

func (linkCrcDetectionPlugin *LinkCRCDetectionPlugin) executeShutdown() error {
	linkCrcDetectionPlugin.monitoredInterfaces = nil
	return nil
}

func (linkCrcDetectionPlugin *LinkCRCDetectionPlugin) GetPluginId() *plugins_common.PluginId {
	return &plugins_common.PluginId{Name: linkCrcDetectionPlugin.PluginName, Version: plugin_version}
}

type LinkCrcDetectorInterface interface {
	Initialize()
	AddInterfaceCountersAndDetectCrc(currentCounters map[string]uint64, localTimeStampUtc time.Time) bool
	validateCountersDiff(previousCounter map[string]uint64, currentCounters map[string]uint64) bool
}

/*
Contains logic for detecting CRC on an interface using a rolling window based approach.
It uses same algorithm that is currently used by Event hub pipelines today
*/
type LinkCrcDetector struct {
	latestCounters       map[string]uint64 // This will be nil for the very first time.
	outlierRollingWindow plugins_common.FixedSizeRollingWindow[CrcOutlierInfo]
}

/* Initializes the detector instance with config values */
func (linkCrcDetector *LinkCrcDetector) Initialize() {
	linkCrcDetector.outlierRollingWindow = plugins_common.FixedSizeRollingWindow[CrcOutlierInfo]{}
	linkCrcDetector.outlierRollingWindow.Initialize(outlierRollingWindowSize)
}

/* Adds CRC based interface counters, computes outlier and detects CRC utilizing the rollowing window outlier details */
func (linkCrcDetector *LinkCrcDetector) AddInterfaceCountersAndDetectCrc(currentCounters map[string]uint64, localTimeStampUtc time.Time) bool {
	if currentCounters == nil {
		return false
	}

	defer func() {
		linkCrcDetector.latestCounters = currentCounters
	}()

	// For the very first time, initialize latestCounters to counters and return false, as diff can not be calculated.
	if linkCrcDetector.latestCounters == nil {
		return false
	}

	// validate if all diff counters are valid.
	if !linkCrcDetector.validateCountersDiff(linkCrcDetector.latestCounters, currentCounters) {
		// Make this alertable.
		// LogError(fmt.Sprintf("Invalid counters"))
		// TODO: should we reset latestCounters ? Also should we reset latestCounters when the its stale ?
		return false
	}

	// Check if current counter w.r.t previous counter evaluates to an outlier.
	ifInErrorsDiff := currentCounters[dbclient.IF_IN_ERRORS_COUNTER_KEY] - linkCrcDetector.latestCounters[dbclient.IF_IN_ERRORS_COUNTER_KEY]
	ifOutErrorsDiff := currentCounters[dbclient.IF_OUT_ERRORS_COUNTER_KEY] - linkCrcDetector.latestCounters[dbclient.IF_OUT_ERRORS_COUNTER_KEY]
	inUnicastPacketsDiff := currentCounters[dbclient.IN_UNICAST_PACKETS_COUNTER_KEY] - linkCrcDetector.latestCounters[dbclient.IN_UNICAST_PACKETS_COUNTER_KEY]
	outUnicastPacketsDiff := currentCounters[dbclient.OUT_UNICAST_PACKETS_COUNTER_KEY] - linkCrcDetector.latestCounters[dbclient.OUT_UNICAST_PACKETS_COUNTER_KEY]

	// Start evaluating the outliers and detect CRC anomaly.
	if ifInErrorsDiff > uint64(ifInErrorsDiffMinValue) && (inUnicastPacketsDiff > uint64(inUnicastPacketsMinValue) || outUnicastPacketsDiff > uint64(outUnicastPacketsMinValue)) {
		errorMetric := float64(ifInErrorsDiff) / (float64(inUnicastPacketsDiff) + float64(outUnicastPacketsDiff))
		if errorMetric > minCrcError {
			if inUnicastPacketsDiff > 0 {
				totalLinkErrors := ifInErrorsDiff - ifOutErrorsDiff
				fcsErrorRate := totalLinkErrors / inUnicastPacketsDiff
				if fcsErrorRate > 0 {
					crcOutlier := CrcOutlierInfo{TimeStamp: localTimeStampUtc}
					linkCrcDetector.outlierRollingWindow.AddElement(crcOutlier)

					// Check if outlier occured twice in past 125 seconds.
					outliersCount := 0
					crcOutliers := linkCrcDetector.outlierRollingWindow.GetElements()
					for iterator := crcOutliers.Back(); iterator != nil; iterator = iterator.Prev() {
						outlier := iterator.Value.(CrcOutlierInfo)
						if localTimeStampUtc.Sub(outlier.TimeStamp).Seconds() < float64(lookBackPeriodInSecs) {
							outliersCount = outliersCount + 1
							if outliersCount == minOutliersForDetection {
								return true
							}
						}
					}
				}
			}
		}
	}

	return false
}

/* Validates if counters are valid */
func (linkCrcDetector *LinkCrcDetector) validateCountersDiff(previousCounter map[string]uint64, currentCounters map[string]uint64) bool {
	if previousCounter[dbclient.IF_IN_ERRORS_COUNTER_KEY] > currentCounters[dbclient.IF_IN_ERRORS_COUNTER_KEY] {
		return false
	}

	if previousCounter[dbclient.IF_OUT_ERRORS_COUNTER_KEY] > currentCounters[dbclient.IF_OUT_ERRORS_COUNTER_KEY] {
		return false
	}

	if previousCounter[dbclient.IN_UNICAST_PACKETS_COUNTER_KEY] > currentCounters[dbclient.IN_UNICAST_PACKETS_COUNTER_KEY] {
		return false
	}

	if previousCounter[dbclient.OUT_UNICAST_PACKETS_COUNTER_KEY] > currentCounters[dbclient.OUT_UNICAST_PACKETS_COUNTER_KEY] {
		return false
	}

	return true
}

/* Contains details if counter diff is outlier or not */
type CrcOutlierInfo struct {
	TimeStamp time.Time
}

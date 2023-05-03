package plugins_common
import (
	"time"
	"container/list"
	"errors"
	"fmt"
        "lom/src/lib/lomcommon"
        "lom/src/lib/lomipc"
)

/* Interface for limiting reporting frequency of plugin */
type PluginReportingFrequencyLimiterInterface interface {
	ShouldReport(anomalyKey string) bool
	ResetCache(anomalyKey string)
	Initialize(initialReportingFreqInMins int, subsequentReportingFreqInMins int, initialReportingMaxCount int)
}

/* Contains when detection was last reported and the count of reports so far */
type ReportingDetails struct {
	lastReported         time.Time
	countOfTimesReported int
}

const (
	initial_detection_reporting_freq_in_mins    int = 5
	subsequent_detection_reporting_freq_in_mins int = 60
	initial_detection_reporting_max_count       int = 12
)

type PluginReportingFrequencyLimiter struct {
	cache                         map[string]*ReportingDetails
	initialReportingFreqInMins    int
	SubsequentReportingFreqInMins int
	initialReportingMaxCount      int
}

/* Initializes values with detection frequencies */
func (pluginReportingFrequencyLimiter *PluginReportingFrequencyLimiter) Initialize(initialReportingFreqInMins int, subsequentReportingFreqInMins int, initialReportingMaxCount int) {
	pluginReportingFrequencyLimiter.cache = make(map[string]*ReportingDetails)
	pluginReportingFrequencyLimiter.initialReportingFreqInMins = initialReportingFreqInMins
	pluginReportingFrequencyLimiter.SubsequentReportingFreqInMins = subsequentReportingFreqInMins
	pluginReportingFrequencyLimiter.initialReportingMaxCount = initialReportingMaxCount
}

/* Determines if detection can be reported now for an anomalyKey. True if it can be reported else false.*/
func (pluginReportingFrequencyLimiter *PluginReportingFrequencyLimiter) ShouldReport(anomalyKey string) bool {
	reportingDetails, ok := pluginReportingFrequencyLimiter.cache[anomalyKey]

	if !ok {
		reportingDetails := ReportingDetails{lastReported: time.Now(), countOfTimesReported: 1}
		pluginReportingFrequencyLimiter.cache[anomalyKey] = &reportingDetails
		return true
	} else {
		defer func() {
			reportingDetails.countOfTimesReported = reportingDetails.countOfTimesReported + 1
			reportingDetails.lastReported = time.Now()
		}()

		if reportingDetails.countOfTimesReported <= pluginReportingFrequencyLimiter.initialReportingMaxCount {
			if time.Since(reportingDetails.lastReported).Minutes() > float64(pluginReportingFrequencyLimiter.initialReportingFreqInMins) {
				return true
			}
		} else if reportingDetails.countOfTimesReported > pluginReportingFrequencyLimiter.initialReportingMaxCount {
			if time.Since(reportingDetails.lastReported).Minutes() > float64(pluginReportingFrequencyLimiter.SubsequentReportingFreqInMins) {
				return true
			}
		}
		return false
	}
}

/* Resets cache for anomaly Key. This needs to be used when anomaly is not detected for an anomaly key */
func (pluginReportingFrequencyLimiter *PluginReportingFrequencyLimiter) ResetCache(anomalyKey string) {
	delete(pluginReportingFrequencyLimiter.cache, anomalyKey)
}

/* Factory method to get default detection reporting limiter instance */
func GetDefaultDetectionFrequencyLimiter() PluginReportingFrequencyLimiterInterface {
	detectionFreqLimiter := &PluginReportingFrequencyLimiter{}
	detectionFreqLimiter.Initialize(initial_detection_reporting_freq_in_mins, subsequent_detection_reporting_freq_in_mins, initial_detection_reporting_max_count)
	return detectionFreqLimiter
}

/* A generic rolling window data structure with fixed size */
type FixedSizeRollingWindow[T any] struct {
        doublyLinkedList     *list.List
        maxRollingWindowSize int
}

/* Initalizes the datastructure with size */
func (fxdSizeRollingWindow *FixedSizeRollingWindow[T]) Initialize(maxSize int) error {
        if maxSize <= 0 {
                return errors.New(fmt.Sprintf("%d Invalid size for fxd size rolling window", maxSize))
        }
        fxdSizeRollingWindow.maxRollingWindowSize = maxSize
        fxdSizeRollingWindow.doublyLinkedList = list.New()
        return nil
}

/* Adds element to rolling window */
func (fxdSizeRollingWindow *FixedSizeRollingWindow[T]) AddElement(value T) {
        if fxdSizeRollingWindow.doublyLinkedList.Len() == 0 || fxdSizeRollingWindow.doublyLinkedList.Len() < fxdSizeRollingWindow.maxRollingWindowSize {
                fxdSizeRollingWindow.doublyLinkedList.PushBack(value)
        } else if fxdSizeRollingWindow.doublyLinkedList.Len() == fxdSizeRollingWindow.maxRollingWindowSize {
                // Remove first element.
                element := fxdSizeRollingWindow.doublyLinkedList.Front()
                fxdSizeRollingWindow.doublyLinkedList.Remove(element)
                // Add the input element into the back.
                fxdSizeRollingWindow.doublyLinkedList.PushBack(value)
        }
}

/* Gets all current elements as list */
func (fxdSizeRollingWindow *FixedSizeRollingWindow[T]) GetElements() *list.List {
        return fxdSizeRollingWindow.doublyLinkedList
}

const (
	ResultCodeSuccess int = iota
	ResultCodeInvalidArgument
	ResultCodeAborted
)

const (
	ResultStringSuccess = "Success"
	ResultStringFailure = "Failure"
)

/*
This util can be used by detection plugins which needs to detect anomalies periodically and send heartbeat to plugin manager.
This util takes care of executing detection logic periodically and shutting down the request when shutdown is invoked on the plugin.
If detection plugin uses this Util as a field in its struct, Request and Shutdown methods from this util get promoted to the plugin.
Guidence for requestFunc
  - Return nil if periodic detection needs to be continued.
  - Return action response if Request needs to return to the caller.
  - isExecutionHealthy needs to be marked false when there is any issue in Request method that needs to be reported.
*/
type PeriodicDetectionPluginUtil struct {
	requestFrequencyInSecs  int
	heartBeatIntervalInSecs int
	shutDownChannel         chan interface{}
	requestAborted          bool
	requestFunc             func(*lomipc.ActionRequestData, *bool) *lomipc.ActionResponseData
	shutdownFunc            func() error
	PluginName              string
}

/* This method needs to be called to initialize fields present in PeriodicDetectionPluginUtil struct */
func (periodicDetectionPluginUtil *PeriodicDetectionPluginUtil) Init(pluginName string, requestFrequencyInSecs int, actionConfig *lomcommon.ActionCfg_t, requestFunction func(*lomipc.ActionRequestData, *bool) *lomipc.ActionResponseData, shutDownFunction func() error) error {
	if actionConfig.HeartbeatInt <= 0 {
		// Do not use a default heartbeat interval. Validate and honor the one passed from plugin manager.
		return errors.New(fmt.Sprintf("Invalid heartbeat interval %d", actionConfig.HeartbeatInt))
	}
	if requestFrequencyInSecs <= 0 {
		return errors.New(fmt.Sprintf("Invalid requestFreq %d", requestFrequencyInSecs))
	}
	if requestFunction == nil || shutDownFunction == nil {
		return errors.New(fmt.Sprintf("requestFunction or shutDownFunction is not initialized"))
	}
	if pluginName == "" {
		return errors.New(fmt.Sprintf("PluginName invalid"))
	}
	periodicDetectionPluginUtil.requestFrequencyInSecs = requestFrequencyInSecs
	periodicDetectionPluginUtil.heartBeatIntervalInSecs = actionConfig.HeartbeatInt
	periodicDetectionPluginUtil.requestAborted = false
	periodicDetectionPluginUtil.shutDownChannel = make(chan interface{})
	periodicDetectionPluginUtil.requestFunc = requestFunction
	periodicDetectionPluginUtil.shutdownFunc = shutDownFunction
	periodicDetectionPluginUtil.PluginName = pluginName
	lomcommon.LogInfo(fmt.Sprintf("Initialized periodicDetectionPluginUtil successfuly for (%s)",pluginName))
	return nil
}

/* This method is promoted to the plugin Periodically invokes detection logic and send heartbeat as well. Honors shutdown when shutdown is invoked on plugin */
func (periodicDetectionPluginUtil *PeriodicDetectionPluginUtil) Request(
	hbchan chan PluginHeartBeat,
	request *lomipc.ActionRequestData) *lomipc.ActionResponseData {

	if request.Timeout > 0 {
		return GetResponse(request, "", "", ResultCodeInvalidArgument, "Invalid Timeout value for detection plugin")
	}

	detectionTicker := time.NewTicker(time.Duration(periodicDetectionPluginUtil.requestFrequencyInSecs) * time.Second)
	heartBeatTicker := time.NewTicker(time.Duration(periodicDetectionPluginUtil.heartBeatIntervalInSecs) * time.Second)
	lomcommon.LogInfo(fmt.Sprintf("Timers initialized for plugin (%s)", periodicDetectionPluginUtil.PluginName))
        isExecutionHealthy := false
	for {
		select {
		case <-heartBeatTicker.C:
		     if isExecutionHealthy {
			pluginHeartBeat := PluginHeartBeat{PluginName: periodicDetectionPluginUtil.PluginName, EpochTime: time.Now().Unix()}
			hbchan <- pluginHeartBeat
                     }
		case <-detectionTicker.C:
			if !periodicDetectionPluginUtil.requestAborted {
				response := periodicDetectionPluginUtil.requestFunc(request, &isExecutionHealthy)
				if response != nil {
					return response
				}
			}

		case <-periodicDetectionPluginUtil.shutDownChannel:
			lomcommon.LogInfo(fmt.Sprintf("Aborting Request for (%s)", periodicDetectionPluginUtil.PluginName))
			responseData := GetResponse(request, "", "", ResultCodeAborted, ResultStringFailure)
			periodicDetectionPluginUtil.requestAborted = true
			return responseData
		}
	}
}

/* Shutdown that aborts the request. It also cleans up plugin defined cleanUp at the end */
func (periodicDetectionPluginUtil *PeriodicDetectionPluginUtil) Shutdown() error {
	lomcommon.LogInfo(fmt.Sprintf("Shutdown called for plugin (%s)", periodicDetectionPluginUtil.PluginName))
	close(periodicDetectionPluginUtil.shutDownChannel)
	startTime := time.Now()
	for {
		if periodicDetectionPluginUtil.requestAborted {
			break
		}

		elapsedTimeInSeconds := time.Since(startTime).Seconds()
		if elapsedTimeInSeconds > float64(periodicDetectionPluginUtil.requestFrequencyInSecs) {
			return errors.New(fmt.Sprintf("Request could not be aborted"))
		}
	}

	periodicDetectionPluginUtil.shutdownFunc()
	lomcommon.LogInfo(fmt.Sprintf("Shutdown successful for plugin (%s)", periodicDetectionPluginUtil.PluginName))
	return nil
}

func GetResponse(request *lomipc.ActionRequestData, anomalyKey string, response string, resultCode int, resultString string) *lomipc.ActionResponseData {
	responseData := lomipc.ActionResponseData{Action: request.Action,
		InstanceId:        request.InstanceId,
		AnomalyInstanceId: request.AnomalyInstanceId,
		AnomalyKey:        anomalyKey,
		Response:          response,
		ResultCode:        resultCode,
		ResultStr:         resultString}
	return &responseData
}

/* This file contains helper utils that plugins can use to perform their actions/tasks */
package plugins_common

import (
    "container/list"
    "context"
    "lom/src/lib/lomcommon"
    "lom/src/lib/lomipc"
    "sync"
    "sync/atomic"
    "time"
)

/* Interface for limiting reporting frequency of plugin */
type PluginReportingFrequencyLimiterInterface interface {
    ShouldReport(anomalyKey string) bool
    ResetCache(anomalyKey string)
    Initialize(initialReportingFreqInMins int, subsequentReportingFreqInMins int, initialReportingMaxCount int)
    IsNotWithinFrequency(reportingDetails ReportingDetails) bool
}

/* Contains when detection was last reported and the count of reports so far */
type ReportingDetails struct {
    lastReported         time.Time
    countOfTimesReported int
}

const (
    initial_detection_reporting_freq_in_mins    = "INITIAL_DETECTION_REPORTING_FREQ_IN_MINS"
    subsequent_detection_reporting_freq_in_mins = "SUBSEQUENT_DETECTION_REPORTING_FREQ_IN_MINS"
    initial_detection_reporting_max_count       = "INITIAL_DETECTION_REPORTING_MAX_COUNT"
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
        if pluginReportingFrequencyLimiter.IsNotWithinFrequency(*reportingDetails) {
            defer func() {
                reportingDetails.countOfTimesReported = reportingDetails.countOfTimesReported + 1
                reportingDetails.lastReported = time.Now()
            }()
            return true
        }
        return false
    }
}

/* Resets cache for anomaly Key. This needs to be used when anomaly is not detected for an anomaly key */
func (pluginReportingFrequencyLimiter *PluginReportingFrequencyLimiter) ResetCache(anomalyKey string) {
    reportingDetails, ok := pluginReportingFrequencyLimiter.cache[anomalyKey]

    if ok {
        if pluginReportingFrequencyLimiter.IsNotWithinFrequency(*reportingDetails) {
            delete(pluginReportingFrequencyLimiter.cache, anomalyKey)
        }
    }
}

func (pluginReportingFrequencyLimiter *PluginReportingFrequencyLimiter) IsNotWithinFrequency(reportingDetails ReportingDetails) bool {
    if reportingDetails.countOfTimesReported <= pluginReportingFrequencyLimiter.initialReportingMaxCount {
        if time.Since(reportingDetails.lastReported).Minutes() > float64(pluginReportingFrequencyLimiter.initialReportingFreqInMins) {
            return true
        }
    } else {
        if time.Since(reportingDetails.lastReported).Minutes() > float64(pluginReportingFrequencyLimiter.SubsequentReportingFreqInMins) {
            return true
        }
    }
    return false
}

/* Factory method to get default detection reporting limiter instance */
func GetDefaultDetectionFrequencyLimiter() PluginReportingFrequencyLimiterInterface {
    detectionFreqLimiter := &PluginReportingFrequencyLimiter{}
    detectionFreqLimiter.Initialize(lomcommon.GetConfigMgr().GetGlobalCfgInt(initial_detection_reporting_freq_in_mins), lomcommon.GetConfigMgr().GetGlobalCfgInt(subsequent_detection_reporting_freq_in_mins), lomcommon.GetConfigMgr().GetGlobalCfgInt(initial_detection_reporting_max_count))
    return detectionFreqLimiter
}

/* A generic rolling window data structure with fixed size */
type FixedSizeRollingWindow[T any] struct {
    orderedDataPoints    *list.List
    maxRollingWindowSize int
}

/* Initalizes the datastructure with size */
func (fxdSizeRollingWindow *FixedSizeRollingWindow[T]) Initialize(maxSize int) error {
    if maxSize <= 0 {
        return lomcommon.LogError("%d Invalid size for fxd size rolling window", maxSize)
    }
    fxdSizeRollingWindow.maxRollingWindowSize = maxSize
    fxdSizeRollingWindow.orderedDataPoints = list.New()
    return nil
}

/* Adds element to rolling window */
func (fxdSizeRollingWindow *FixedSizeRollingWindow[T]) AddElement(value T) {
    if fxdSizeRollingWindow.orderedDataPoints.Len() < fxdSizeRollingWindow.maxRollingWindowSize {
        fxdSizeRollingWindow.orderedDataPoints.PushBack(value)
        return
    }
    // Remove first element.
    element := fxdSizeRollingWindow.orderedDataPoints.Front()
    fxdSizeRollingWindow.orderedDataPoints.Remove(element)
    // Add the input element into the back.
    fxdSizeRollingWindow.orderedDataPoints.PushBack(value)
}

/* Gets all current elements as list */
func (fxdSizeRollingWindow *FixedSizeRollingWindow[T]) GetElements() *list.List {
    return fxdSizeRollingWindow.orderedDataPoints
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

const (
    min_err_cnt_to_skip_hb_key = "PLUGIN_MIN_ERR_CNT_TO_SKIP_HEARTBEAT"
    plugin_prefix              = "plugin_"
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
    requestFunc             func(*lomipc.ActionRequestData, *bool, context.Context) *lomipc.ActionResponseData
    shutdownFunc            func() error
    PluginName              string
    shutDownInitiated       atomic.Bool
    detectionRunInfo        DetectionRunInfo
    numOfConsecutiveErrors  atomic.Uint64
    responseChannel         chan *lomipc.ActionResponseData
    ctx                     context.Context
    cancelCtxFunc           context.CancelFunc
}

type DetectionRunInfo struct {
    /* If this value is nil, it indicates there is no current run in execution. Non-nil value signifies a current run in execution. */
    currentRunStartTimeInUtc *time.Time
    /* Duration of the latest completed run in seconds */
    durationOfLatestRunInSeconds int64
    mutex                        sync.Mutex
}

/* This method needs to be called to initialize fields present in PeriodicDetectionPluginUtil struct */
func (periodicDetectionPluginUtil *PeriodicDetectionPluginUtil) Init(pluginName string, requestFrequencyInSecs int, actionConfig *lomcommon.ActionCfg_t, requestFunction func(*lomipc.ActionRequestData, *bool, context.Context) *lomipc.ActionResponseData, shutDownFunction func() error) error {
    if actionConfig.HeartbeatInt <= 0 {
        // Do not use a default heartbeat interval. Validate and honor the one passed from plugin manager.
        return lomcommon.LogError("Invalid heartbeat interval %d", actionConfig.HeartbeatInt)
    }
    if requestFrequencyInSecs <= 0 {
        return lomcommon.LogError("Invalid requestFreq %d", requestFrequencyInSecs)
    }
    if requestFunction == nil || shutDownFunction == nil {
        return lomcommon.LogError("requestFunction or shutDownFunction is not initialized")
    }
    if pluginName == "" {
        return lomcommon.LogError("PluginName invalid")
    }
    periodicDetectionPluginUtil.requestFrequencyInSecs = requestFrequencyInSecs
    periodicDetectionPluginUtil.heartBeatIntervalInSecs = actionConfig.HeartbeatInt
    /* Size of responseChannel should be 2, so that the go routine handling request can be terminated on shutdown if the Request method has already terminated. */
    periodicDetectionPluginUtil.responseChannel = make(chan *lomipc.ActionResponseData, 2)
    periodicDetectionPluginUtil.requestFunc = requestFunction
    periodicDetectionPluginUtil.shutdownFunc = shutDownFunction
    periodicDetectionPluginUtil.PluginName = pluginName
    periodicDetectionPluginUtil.detectionRunInfo = DetectionRunInfo{}
    periodicDetectionPluginUtil.ctx, periodicDetectionPluginUtil.cancelCtxFunc = context.WithCancel(context.Background())
    lomcommon.LogInfo("Initialized periodicDetectionPluginUtil successfuly for (%s)", pluginName)
    return nil
}

/*
This method immediately starts heartbeat and request execution.
This method is promoted to the plugin. Honors shutdown when shutdown is invoked on plugin.
*/
func (periodicDetectionPluginUtil *PeriodicDetectionPluginUtil) Request(
    hbchan chan PluginHeartBeat,
    request *lomipc.ActionRequestData) *lomipc.ActionResponseData {
    if request.Timeout > 0 {
        return GetResponse(request, "", "", ResultCodeInvalidArgument, "Invalid Timeout value for detection plugin")
    }

    // Publish a heartbeat immediately.
    pluginHeartBeat := PluginHeartBeat{PluginName: periodicDetectionPluginUtil.PluginName, EpochTime: time.Now().Unix()}
    hbchan <- pluginHeartBeat

    lomcommon.GetGoroutineTracker().Start(plugin_prefix+periodicDetectionPluginUtil.PluginName, periodicDetectionPluginUtil.handleRequest, request)
    heartBeatTicker := time.NewTicker(time.Duration(periodicDetectionPluginUtil.heartBeatIntervalInSecs) * time.Second)
    defer heartBeatTicker.Stop()

    for {
        select {

        case <-heartBeatTicker.C:
            periodicDetectionPluginUtil.publishHeartBeat(hbchan)

        case resp := <-periodicDetectionPluginUtil.responseChannel:
            return resp

        case <-periodicDetectionPluginUtil.ctx.Done():
            /* Shutdown stops the periodic detection */
            lomcommon.LogInfo("Aborting Request for (%s)", periodicDetectionPluginUtil.PluginName)
            responseData := GetResponse(request, "", "", ResultCodeAborted, ResultStringFailure)
            return responseData
        }
    }
}

/* Publishes heartbeat after performing validations */
func (periodicDetectionPluginUtil *PeriodicDetectionPluginUtil) publishHeartBeat(hbchan chan PluginHeartBeat) {

    numConsecutiveErrors := periodicDetectionPluginUtil.numOfConsecutiveErrors.Load()
    periodicDetectionPluginUtil.detectionRunInfo.mutex.Lock()
    durationOfLatestRunInSecs := periodicDetectionPluginUtil.detectionRunInfo.durationOfLatestRunInSeconds
    var currentRunStartTimeInUtc time.Time
    if periodicDetectionPluginUtil.detectionRunInfo.currentRunStartTimeInUtc != nil {
        currentRunStartTimeInUtc = *periodicDetectionPluginUtil.detectionRunInfo.currentRunStartTimeInUtc
    }
    periodicDetectionPluginUtil.detectionRunInfo.mutex.Unlock()

    if numConsecutiveErrors >= uint64(lomcommon.GetConfigMgr().GetGlobalCfgInt(min_err_cnt_to_skip_hb_key)) {
        lomcommon.LogError("Skipping heartbeat for %s. numConsecutiveErrors %d", periodicDetectionPluginUtil.PluginName, numConsecutiveErrors)
        return
    } else if durationOfLatestRunInSecs > int64(periodicDetectionPluginUtil.requestFrequencyInSecs) {
        lomcommon.LogError("Skipping heartbeat for %s. DurationOfLatestRunInSecs %d", periodicDetectionPluginUtil.PluginName, durationOfLatestRunInSecs)
        return
    } else if !currentRunStartTimeInUtc.IsZero() { /* Indicates a request is running currently */
        durationTillNow := int64(time.Since(currentRunStartTimeInUtc).Seconds())
        if durationTillNow > int64(periodicDetectionPluginUtil.requestFrequencyInSecs) {
            lomcommon.LogError("Skipping heartbeat for %s. Duration of current execution in secs %d", periodicDetectionPluginUtil.PluginName, durationTillNow)
            return
        }
    }

    /* Publish heartbeat only after above validations pass.*/
    pluginHeartBeat := PluginHeartBeat{PluginName: periodicDetectionPluginUtil.PluginName, EpochTime: time.Now().Unix()}
    hbchan <- pluginHeartBeat
}

/* Hanldes detection logic execution and honors shutdown as well. This is called in a goRoutine in the Request method */
func (periodicDetectionPluginUtil *PeriodicDetectionPluginUtil) handleRequest(request *lomipc.ActionRequestData) {

    detectionTicker := time.NewTicker(time.Duration(periodicDetectionPluginUtil.requestFrequencyInSecs) * time.Second)
    defer detectionTicker.Stop()
    lomcommon.LogInfo("Detection Timer initialized for plugin (%s)", periodicDetectionPluginUtil.PluginName)
    isExecutionHealthy := false
loop:
    for {
        // Start immediately before the select.
        if !periodicDetectionPluginUtil.shutDownInitiated.Load() {
            periodicDetectionPluginUtil.detectionRunInfo.mutex.Lock()
            startTimeInUtc := time.Now().UTC()
            periodicDetectionPluginUtil.detectionRunInfo.currentRunStartTimeInUtc = &startTimeInUtc
            periodicDetectionPluginUtil.detectionRunInfo.mutex.Unlock()

            /* Perform detection logic periodically */
            response := periodicDetectionPluginUtil.requestFunc(request, &isExecutionHealthy, periodicDetectionPluginUtil.ctx)
            if response != nil {
                periodicDetectionPluginUtil.responseChannel <- response
                return
            }

            if !isExecutionHealthy {
                periodicDetectionPluginUtil.numOfConsecutiveErrors.Add(1)
                lomcommon.LogError("Incremented consecutiveError count for plugin (%s)", periodicDetectionPluginUtil.PluginName)
            } else {
                periodicDetectionPluginUtil.numOfConsecutiveErrors.Store(0)
            }

            elapsedTime := int64(time.Since(startTimeInUtc).Seconds())
            periodicDetectionPluginUtil.detectionRunInfo.mutex.Lock()
            periodicDetectionPluginUtil.detectionRunInfo.currentRunStartTimeInUtc = nil
            periodicDetectionPluginUtil.detectionRunInfo.durationOfLatestRunInSeconds = elapsedTime
            periodicDetectionPluginUtil.detectionRunInfo.mutex.Unlock()

            if elapsedTime > int64(periodicDetectionPluginUtil.requestFrequencyInSecs) {
                // Reset the timer.
                lomcommon.LogInfo("Resetting timer for plugin (%s)", periodicDetectionPluginUtil.PluginName)
                detectionTicker.Reset(time.Duration(periodicDetectionPluginUtil.requestFrequencyInSecs) * time.Second)
            }
        }

        select {
        case <-detectionTicker.C:
            continue

        case <-periodicDetectionPluginUtil.ctx.Done():
            /* Shutdown stops the periodic detection */
            lomcommon.LogInfo("Aborting handleRequest for (%s)", periodicDetectionPluginUtil.PluginName)
            break loop
        }
    }
}

/* Shutdown that aborts the request. It also cleans up plugin defined cleanUp at the end */
func (periodicDetectionPluginUtil *PeriodicDetectionPluginUtil) Shutdown() error {
    lomcommon.LogInfo("Shutdown called for plugin (%s)", periodicDetectionPluginUtil.PluginName)
    periodicDetectionPluginUtil.cancelCtxFunc()
    periodicDetectionPluginUtil.shutDownInitiated.Store(true)
    periodicDetectionPluginUtil.shutdownFunc()
    lomcommon.LogInfo("Shutdown successful for plugin (%s)", periodicDetectionPluginUtil.PluginName)
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

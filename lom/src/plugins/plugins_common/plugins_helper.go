package plugins_common
import (
	"time"
	"container/list"
	"errors"
	"fmt"
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

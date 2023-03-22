package lomcommon

import (
    "errors"
    "fmt"
    "log"
    "log/syslog"
    "os"
    "os/exec"
    "sort"
    "time"
)

var writers = make(map[syslog.Priority]*syslog.Writer)

var log_level = syslog.LOG_DEBUG

var FmtFprintf = fmt.Fprintf
var OSExit = os.Exit

func init() {

    for i := syslog.LOG_EMERG; i <= syslog.LOG_DEBUG; i++ {
        writer, err := syslog.Dial("", "", (i|syslog.LOG_LOCAL7), "")
        if err != nil {
            log.Fatal(err)
        }
        writers[i] = writer
    }

    /*
     * Samples:
     *  fmt.Fprintf(writers[syslog.LOG_WARNING], "This is a daemon warning message")
     *  fmt.Fprintf(writers[syslog.LOG_ERR], "This is a daemon ERROR message")
     *  fmt.Fprintf(writers[syslog.LOG_INFO], "This is a daemon INFO message")
     */
}

/* Return currently set log level */
func GetLogLevel() syslog.Priority {
    return log_level
}


/* Set current log level */
func SetLogLevel(lvl syslog.Priority) {
    log_level = lvl
}


/*
 * Log this message for given log level, if this level <= current log level
 *
 * Input:
 *  lvl - Log level for this message
 *  s - Message string, with var args as required by format in s
 *
 * Output:
 *  None
 *
 * Return:
 *  None
 */
func LogMessage(lvl syslog.Priority, s string, a ...interface{})  {
    ct_lvl := GetLogLevel()
    if lvl <= ct_lvl {
        FmtFprintf(writers[lvl], s, a...)
        if ct_lvl >= syslog.LOG_DEBUG {
            /* Debug messages gets printed out to STDOUT */
            fmt.Printf(s, a...)
            fmt.Println("")
        }
    }
}


/* Log this message for panic level and exit */
func LogPanic(s string, a ...interface{})  {
    LogMessage(syslog.LOG_CRIT, s, a...)
    LogMessage(syslog.LOG_CRIT, "LoM exiting ...")
    OSExit(-1)
}


/* Log this message at error level */
func LogError(s string, a ...interface{}) error {
    e := fmt.Sprintf(s, a...)
    LogMessage(syslog.LOG_ERR, e)
    return errors.New(e)
}


/* Log this message at warning level */
func LogWarning(s string, a ...interface{})  {
    LogMessage(syslog.LOG_WARNING, s, a...)
}


/* Log this message at info level */
func LogInfo(s string, a ...interface{})  {
    LogMessage(syslog.LOG_INFO, s, a...)
}


/* Log this message at debug level */
func LogDebug(s string, a ...interface{})  {
    LogMessage(syslog.LOG_DEBUG, s, a...)
}

var uuid_suffix = 0
var UUID_BIN = "uuidgen"

/* Helper to get UUID as string */
func GetUUID() string {
    if newUUID, err := exec.Command(UUID_BIN).Output(); err != nil {
        LogError("Internal failed uuidgen. (%s)", err)
        uuid_suffix++
        return fmt.Sprintf("%v-%d", time.Now().Unix(), uuid_suffix)
    } else {
        return string(newUUID)[:36]
    }
}


const A_DAY_IN_SECS = int64(24 * 60 * 60)

/* Info related to logging periodically */
type LogPeriodicEntry_t struct {
    ID      string          /* Identifier provided by caller */
    Message string          /* string to log */
    Lvl     syslog.Priority /* Log Level to use */
    Period  int             /* period in seconds */
    /* TODO: Change period to list of {period, cnt}
     * after finishing cnt writes, move to next entry in list.
     * Caller may send [ {5, 100}, {3600, 0} ], implying
     * Write first 100 logs for every 5 seconds. After that
     * write logs for every hour forever (cnt = 0) 
     */
}

type logPeriodicEntryInt_t struct {
    LogPeriodicEntry_t
    Due     int64           /* Next due epoch time point */
    index   uint64          /* Add a sequential index to the message */
                            /* This can help identify repeated logs with index */
                            /* indicating set to count of logs written so far */
}

/* Log Periodic module */
type LogPeriodic_t struct {
    /* Channel to communicate to logging routine */
    logCh               chan *LogPeriodicEntry_t

    logPeriodicList     map[string]*logPeriodicEntryInt_t
    logPeriodicSorted   []*logPeriodicEntryInt_t

    /* TODO: Any entry after logging repeatedly at set period 
     * for a day or two, reduce the period to every hour
     * No point in polluting logs, as we have screamed enough
     */
}


var logPeriodic *LogPeriodic_t

/* Get Log Periodic instance */
func GetlogPeriodic() *LogPeriodic_t {
    return logPeriodic
}

/* Initialize the singleton instance for log periodic */
func LogPeriodicInit(chAbort chan interface{}) {
    logPeriodic = &LogPeriodic_t {
        logCh: make( chan *LogPeriodicEntry_t),
        logPeriodicList: make(map[string]*logPeriodicEntryInt_t),
        logPeriodicSorted: nil,
    }

    go logPeriodic.run(chAbort)
}

/* Helper to add a log periodic entry */
func (p *LogPeriodic_t) AddLogPeriodic(l *LogPeriodicEntry_t) error {
    if ((len(l.ID) == 0) || (len(l.Message) == 0)) {
        return LogError("LogPeriodicEntry ID or message is empty")
    }
    min := GetConfigMgr().GetGlobalCfgInt("MIN_PERIODIC_LOG_PERIOD_SECS")
    if l.Period < min {
        return LogError("LogPeriodicEntry Period(%v) < min(%v)", l.Period, min)
    }
    p.logCh  <- l
    return nil
}

/* Helper to remove a previouslu added log periodic entry */
func (p *LogPeriodic_t) DropLogPeriodic(ID string) {
    if len(ID) > 0 {
        /* Emtpy Message implies drop */
        p.logCh  <- &LogPeriodicEntry_t {ID: ID }
    }
}


func (p *LogPeriodic_t) run(chAbort chan interface{}) {
    tout := A_DAY_IN_SECS           /* Just a init value; Once per day */

    for {
        upd := false
        select {
        case l := <- p.logCh:
            upd = p.update(l)

        case <- time.After(time.Duration(tout) * time.Second):
            upd = p.writeLogs()

        case <- chAbort:
            LogDebug("Terminating LogPeriodic upon explicit abort")
            return
        }

        if upd {
            sort.Slice(p.logPeriodicSorted, func(i, j int) bool {
                return p.logPeriodicSorted[i].Due < p.logPeriodicSorted[j].Due
            })
        }
        /* Recompute tout */
        if len(p.logPeriodicSorted) > 0 {
            due := p.logPeriodicSorted[0].Due
            now := time.Now().Unix()
            if now >= due {
                tout = 0
            } else {
                tout = due - now
            }
        } else {
            /* No data to print */
            tout = A_DAY_IN_SECS
        }
    }
}


func (p *LogPeriodic_t) update(l *LogPeriodicEntry_t) bool {
    upd := false
    v, ok := p.logPeriodicList[l.ID]
    if len(l.Message) > 0 {
        if !ok || ((*v).LogPeriodicEntry_t != *l) {
            p.logPeriodicList[l.ID] = &logPeriodicEntryInt_t{*l, 0, 0} /* Set Due immediate */
            upd = true
        }
    } else if ok {
        delete (p.logPeriodicList, l.ID)
        upd = true
    }
    if upd {
        p.logPeriodicSorted = make([]*logPeriodicEntryInt_t, len(p.logPeriodicList))

        i := 0
        for _, v := range p.logPeriodicList {
            p.logPeriodicSorted[i] = v
            i++
        }
    }
    return upd
}


func (p *LogPeriodic_t) writeLogs() bool {
    now := time.Now().Unix()
    upd := false 

    for _, v := range p.logPeriodicSorted {
        if now >= v.Due {
            LogMessage(v.Lvl, "periodic:%v (%s)", v.index, v.Message)
            v.Due = now + int64(v.Period)
            v.index++
            upd = true
        } else {
            break
        }
    }

    return upd
}


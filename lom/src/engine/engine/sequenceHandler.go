package engine

import (
    "fmt"
    . "lom/src/lib/lomcommon"
    . "lom/src/lib/lomipc"
    "sort"
    "time"
)

/*
 * Core functionality
 *
 *  1.  Raise requests for all actions right upon registration, that are first
 *      in any sequence. This is mostly anomaly detection.
 *
 *  2.  Track all active requests.
 *
 *  3.  Upon response from first action create sequence, if all actions of that
 *      sequence are registered.
 *
 *  TODO: timeout should be given to all actions except first, irrespective
 *        of result
 *  4.  Send requests with timeout to rest of the actions in sequence, provided
 *      the response of last called is good.
 *
 *  5.  Upon completion of processing all actions, the sequence is removed
 *
 *  Note:
 *      a.  Every action response is published, even if stale.
 *      b.  First action of the sequence is published with state='init' and upon
 *          sequence completion, it is re-publsihed with state="complete" and
 *          appropriate result code.
 *
 *  Failure scenarios:
 *  A.  Sequence config could be missing data, like action w/o config, the sequence
 *      config is removed, one or more actions in the sequence is marked 'disabled'
 *      and more
 *
 *  B.  An action in the sequence could have failed result_code != 0
 *
 *  C.  Request or sequence could have timedout
 *
 *  D.  Any other fatal failure ?
 *
 *  On any failure:
 *      i.  Seq is processed as complete with possible error codes
 *      ii. TODO: Honor *mandatory* flag, as call
 */

const (
    ANOMALY_PUB_STATUS_INIT     = "init"
    ANOMALY_PUB_STATUS_COMPLETE = "complete"
)

/*
 * Active request - Requests sent out awaiting response with or w/o timeout
 */
type activeRequest_t struct {
    req         *ServerRequestData
    anomalyID   string
    reqExpEpoch int64 /* Expiry time. A value of 0 means no expiry */
}

/*
 * Action vs request -- All active requests
 *  A request may or may not have a timeout.
 *
 * NOTE: Only one outstanding request per Action
 *       Hence can be keyed off by action.
 *
 * NOTE: Engine may abort a sequence when its current request does not
 *       respond within its timeout. But it does *not* remove request from
 *       the active list, until the action/client sends the response. This
 *       is because, it implies that the plugin/action is still busy with
 *       last given request and we can't raise another request, as there
 *       can be atmost one active request per action.
 *
 *       In case of any fatal failure, a de-register action/client or
 *       re-register by client will drop the requests associated with
 *       de-registered actions.
 */
type activeRequestsMap_t map[string]*activeRequest_t

/*
 * Sequence is kicked off when first request of a sequence responds.
 *
 * There can be many sequences. But only one sequence can be active.
 *
 * SequenceInfo and related data is read & filled upon sequence creation
 * which is when the first action of the sequence responds.
 *
 * Only one sequence can be active. Any sequence created when one is active
 * is added to pending Q. Going active implies invoking other actions of the
 * sequence, sequentially. Sequence remains active until all actions of the
 * sequence are complete or timeout, whichever comes earlier.
 *
 * When there are multiple sequences ready to go active, they are first sorted
 * by Priority and then by order of arrival within same priority
 *
 * All sequences have timeout. Some sequence may timeout, even before going
 * active.
 *
 */

type sequenceStatus_t int

const (
    sequenceStatus_pending = sequenceStatus_t(iota)
    sequenceStatus_running
    sequenceStatus_complete
)

type sequenceState_t struct {
    sequenceStatus    sequenceStatus_t
    anomalyInstanceId string                /* Id referred in all requests & responses for this seq */
    sequence          *BindingSequence_t    /* Read from config; Cache as config may change. */
    seqStartEpoch     int64                 /* Start of the sequence */
    seqExpEpoch       int64                 /* Timepoint of expiry for sequence */
    ctIndex           int                   /* Index of action in sequence in-progress */
    context           []*ActionResponseData /* Ordered responses as received, so far */
    currentRequest    *activeRequest_t      /* Current request in progress */
    expTimer          *OneShotEntry_t       /* One shot timer fired to watch timeout  */
}

func (p *sequenceState_t) ExpiryEpoch() int64 {
    /* If current request expire early, send it; else send seq expiry */
    if p.currentRequest != nil {
        if p.currentRequest.reqExpEpoch < p.seqExpEpoch {
            return p.currentRequest.reqExpEpoch
        }
    }
    return p.seqExpEpoch
}

func (p *sequenceState_t) validate() bool {
    if (len(p.anomalyInstanceId) == 0) ||
        (p.sequence == nil) ||
        (p.seqStartEpoch == 0) ||
        (p.seqExpEpoch < p.seqStartEpoch) ||
        (p.ctIndex < 1) ||
        (len(p.context) == 0) ||
        (p.expTimer == nil) {
        LogDebug("sequenceState_t: validate: anId:%v seq:%v epoch(%v/%v) ct(%d) ctx(%v) status(%d) cur(%v) exp(%v)",
            len(p.anomalyInstanceId) == 0, p.sequence == nil, p.seqStartEpoch,
            p.seqExpEpoch < p.seqStartEpoch, p.ctIndex, len(p.context) == 0, p.sequenceStatus,
            p.currentRequest == nil, p.expTimer == nil)

        return false
    }
    return true
}

/*
 * map[anomaly id]sequence_state_t
 *
 * Sequences are keyed off of anomaly instance Id
 */
type Sequences_t map[string]*sequenceState_t

/*
 * timeout:
 * Two possible candidates:
 *      1. Currently active request for currently active sequence
 *      2. All pending sequences, who may timeout before getting chance to execute sequence.
 */
type SortedSequences_t []*sequenceState_t

type SeqHandler_t struct {
    /* All Active requests by action name */
    activeRequests activeRequestsMap_t

    /* Collected sequences by anomaly */
    sequencesByAnomaly Sequences_t

    /* collected sequences by first action */
    sequencesByFirstAction Sequences_t

    sortedSequencesByDue SortedSequences_t
    sortedSequencesByPri SortedSequences_t

    chTimer chan<- int64 /* Channel to convey earliest timeout */

    currentSequence *sequenceState_t
}

var seqHandler *SeqHandler_t = nil

func GetSeqHandler() *SeqHandler_t {
    if seqHandler == nil {
        LogError("InitSeqHandler is not done yet")
    }
    return seqHandler
}

func InitSeqHandler(chTimer chan<- int64) {
    if chTimer == nil {
        LogError("Internal error: Nil chan")
    } else if seqHandler != nil {
        LogError("Duplicate init seq handler")
    } else {
        seqHandler = &SeqHandler_t{
            activeRequests:         make(activeRequestsMap_t),
            sequencesByAnomaly:     make(Sequences_t),
            sequencesByFirstAction: make(Sequences_t),
            chTimer:                chTimer}
    }
}

/* Called asynchronously from a one shot timer */
func (p *SeqHandler_t) FireTimer() {
    if len(p.chTimer) < cap(p.chTimer) {
        p.chTimer <- 0
    }
    /* Being an alert, as long as there is any outstanding, good enough */
}

/*
 * Engine calls from main loop upon signal from FireTimer which is
 * triggered from OneShotTimer fired for seq expiry.
 * Hence only single routine calls ProcessResponse via
 * response from action, or processTimeout via chTimer signal
 * or dropRequest via de-register action process.
 *
 * The timeout could be from current active request or overall seq
 * timeout, whichever expected to fire early.
 *
 * In either case, abort sequence.
 * NOTE: The pending request, if any is *not* dropped from active requests
 * until response arrives, as Engine is not expected to send a request to
 * a plugin, while last req is still pending. Hence entry in active requests
 * only cleared upon action response via client or upon de-register action
 * by client,
 */
func (p *SeqHandler_t) processTimeout() {
    /*
     * Complete seq will remove from p.sortedSequencesByDue. Hence always use
     * current first entry
     */
    for len(p.sortedSequencesByDue) > 0 {
        seq := p.sortedSequencesByDue[0]
        if time.Now().Unix() >= seq.ExpiryEpoch() {
            p.completeSequence(seq, LoMSequenceTimeout, "From Process timeout")
            p.resumeNextSequence()
        } else {
            break /* List is sorted. Hence break */
        }
    }

}

/*
 * Called upon action registration by client or upon sequence completion
 *
 * Raise request, if this is first action in any configured sequence
 *
 * In case of mis-behaving request, PluginMgr will watch & disable the
 * action, if it returns too frequently.
 *
 * Engine need not care.
 */
func (p *SeqHandler_t) RaiseRequestForFirstAction(action string) error {
    regF := GetRegistrations()

    /* Is action registered */
    if regF.GetActiveActionInfo(action) == nil {
        return LogError("Internal: Failing to get active action info (%s)", action)
    }

    /* Is any active request for this action */
    if _, ok := p.activeRequests[action]; ok {
        return LogError("Internal: An active request exists (%s)", action)
    }

    cfg := GetConfigMgr()
    /* Is this action start of a sequence */
    if !cfg.IsStartSequenceAction(action) {
        /* Not an initial action of any sequence */
        return nil
    }

    /* Is there an active/pending sequence for this action */
    if s, ok := p.sequencesByFirstAction[action]; ok {
        return LogError(
            "Internal: An active/pending sequence is in progress ID(%s) index(%d) due(%v)seconds",
            s.anomalyInstanceId, s.ctIndex, time.Now().Unix()-s.seqExpEpoch)
    }

    /* All clear. Fire request for the first action of a sequence */
    uuid := GetUUID()
    /* Make request. No timeout for first action. Add to client & active */
    req := &ServerRequestData{
        TypeServerRequestAction,
        &ActionRequestData{
            Action:            action,
            InstanceId:        uuid,
            AnomalyInstanceId: uuid,
        },
    }

    /* Add to client's pending Q  to send to client upon client asking for a request. */
    /* Stays in this Q until client reads it */
    if err := regF.AddServerRequest(action, req); err != nil {
        return LogError("Internal error: Failed to AddServerRequest (%s)", action)
    }

    /* Track it in our active requests;  Waits here till response */
    p.activeRequests[action] = &activeRequest_t{req, uuid, 0}

    /*
     * NOTE: We only run one timer for seq timeout.
     *  So a req coming after its timeout, *could* still get consumed.
     */
    return nil
}

/* Called upon action de-registration */
func (p *SeqHandler_t) DropRequest(action string) {
    if r, ok := p.activeRequests[action]; ok {
        delete(p.activeRequests, action)
        if s, ok := p.sequencesByAnomaly[r.anomalyID]; ok {
            p.completeSequence(s, LoMActionDeregistered, "Action ("+action+") Deregistered")
            p.resumeNextSequence()
        }
    }
}

/* Sort sorted sequences */
func (p *SeqHandler_t) sortSequences() {
    sort.Slice(p.sortedSequencesByDue, func(i, j int) bool {
        return p.sortedSequencesByDue[i].ExpiryEpoch() <
            p.sortedSequencesByDue[j].ExpiryEpoch()
    })

    sort.Slice(p.sortedSequencesByPri, func(i, j int) bool {
        return (p.sortedSequencesByPri[i].sequence.Priority <
            p.sortedSequencesByPri[j].sequence.Priority)
    })
}

/*
 * addSequence
 *
 * Add this sequence to handler.
 * Enable timer for seq expiry.
 * Re-create sorted sequences
 */
func (p *SeqHandler_t) addSequence(seq *sequenceState_t, tout int) {
    if (seq == nil) || (len(seq.context) == 0) {
        LogError("Expect atleast one context (%v)", seq)
    } else {
        p.sequencesByAnomaly[seq.anomalyInstanceId] = seq
        p.sequencesByFirstAction[seq.context[0].Action] = seq
        p.sortedSequencesByDue = append(p.sortedSequencesByDue, seq)
        p.sortedSequencesByPri = append(p.sortedSequencesByPri, seq)
        m := "Seq: " + seq.sequence.SequenceName + " timeout"
        seq.expTimer = AddOneShotTimer(int64(tout), m, p.FireTimer)
        p.sortSequences()
    }
}

/*
 * dropSequence
 *
 * Remove this sequence from handler.
 * Disable timer.
 * Re-create sorted sequences
 */
func (p *SeqHandler_t) dropSequence(seq *sequenceState_t) {
    if (seq == nil) || (len(seq.context) == 0) {
        LogError("Expect atleast one context (%v)", seq)
    } else {
        seq.expTimer.Disable()
        if p.currentSequence == seq {
            p.currentSequence = nil
        }
        delete(p.sequencesByAnomaly, seq.anomalyInstanceId)
        delete(p.sequencesByFirstAction, seq.context[0].Action)
        p.sortedSequencesByDue = make(SortedSequences_t, len(p.sequencesByAnomaly))
        p.sortedSequencesByPri = make(SortedSequences_t, len(p.sequencesByAnomaly))
        i := 0
        for _, seq = range p.sequencesByAnomaly {
            p.sortedSequencesByDue[i] = seq
            p.sortedSequencesByPri[i] = seq
            i++
        }
        p.sortSequences()
    }
}

type pubAction_t struct {
    LoM_Action *ActionResponseData
    State      string
}

func (p *SeqHandler_t) publishResponse(res *ActionResponseData, complete bool) {
    if res == nil {
        LogError("Expect non null ActionResponseData")
    } else {
        m := pubAction_t{LoM_Action: res}
        if res.InstanceId == res.AnomalyInstanceId {
            if !complete {
                m.State = ANOMALY_PUB_STATUS_INIT
            } else {
                m.State = ANOMALY_PUB_STATUS_COMPLETE
            }
        }
        PublishEvent(m)
    }
}

func (p *SeqHandler_t) ProcessResponse(msg *MsgSendServerResponse) {
    /* TODO: Honor BindingActionCfg_t:Mandatory */
    if msg == nil {
        LogError("Expect non null MsgSendServerResponse")
    } else if msg.ReqType != TypeServerRequestAction {
        LogError("Unexpected response req type (%d)/(%s)",
            msg.ReqType, ServerReqTypeToStr[msg.ReqType])
    } else {
        m := msg.ResData
        if data, ok := m.(ActionResponseData); !ok {
            LogError("Unexpected response res data (ActionResponseData)/(%T)", m)
        } else {
            p.processActionResponse(&data)
        }
    }
}

/* Validate & drop from active requests */
func (p *SeqHandler_t) validate_with_active(data *ActionResponseData, seq *sequenceState_t) error {
    if r, ok := p.activeRequests[data.Action]; ok {
        if (seq != nil) && (seq.currentRequest != r) {
            return LogError("Response's req (%v) != current(%v)", r, seq.currentRequest)
        }
        if r.req.ReqType != TypeServerRequestAction {
            return LogError("Active requests type (%v) != TypeServerRequestAction", r)
        }
        rd := r.req.ReqData
        if ar, ok1 := rd.(*ActionRequestData); !ok1 {
            return LogError("Active requests data (%T) != ActionRequestData (%v)", rd, r)
        } else if ar.InstanceId != data.InstanceId {
            /* stale response */
            return LogError("Active req vs res instance ID mismatch (%v) (%v)", ar, data)
        }
    } else {
        return LogError("No matching req for res (%v)", data)
    }
    return nil
}

/*
 * Called upon response received from client for an action.
 *
 * Validate response; publish it;
 * validate corresponding request; Upon match, drop from active requests; else bail out.
 *
 * Find corresponding sequence by anomaly ID.
 * If not found, validate it for first action in sequence; else if found, validate
 * against current request for this sequence.
 *
 * if first action, but failed, re-publish with state as complete & skip creating
 * sequence. Else if not first action, try resume current seq as this response could
 * have been blocking it from running.
 *
 * If first action and succeeded, create a new sequence and add to pending seq list.
 * Else if seq found, save the response in context. If response is success, call
 * resumeSequence to kick off next request. Else call complete sequence.
 *
 * For any error, the anomaly is re-published with state=complete and seq is completed,
 * if appropriate.
 * Try resumeNextSequence.
 */
func (p *SeqHandler_t) processActionResponse(data *ActionResponseData) {
    if data == nil {
        LogError("Internal error: Nil ActionResponseData")
        return
    }
    if !data.Validate() {
        LogError("Invalid ActionResponseData (%v)", *data)
        return
    }

    anomalyID := data.AnomalyInstanceId
    errCode := LoMResponseOk
    errStr := ""
    var err error = nil
    seq, ok := p.sequencesByAnomaly[anomalyID]
    if !ok {
        seq = nil
    } else if !seq.validate() {
        LogError("Internal error. seq status incorrect (%v) res(%v)", seq, data)
        return
    }

    defer func() {
        /* If sequence is complete / failed, re-publish anomaly */

        if (len(errStr) == 0) && (err != nil) {
            errStr = fmt.Sprintf("%v", err)
        }

        if seq != nil {
            switch seq.sequenceStatus {
            case sequenceStatus_pending:
                /* Only possibility: Just added; Fall below to resume */

            case sequenceStatus_running:
                if errCode == LoMResponseOk {
                    /* Sequence in progress. Nothing todo bail out */
                    return
                }
                /* Complete seq with error. */
                p.completeSequence(seq, errCode, errStr)

            case sequenceStatus_complete:
                /* resume sequence mark it complete upon last response. */
                /* Complete seq with error. */
                p.completeSequence(seq, errCode, errStr)
            }

        } else if anomalyID == data.InstanceId {
            /* This is anomaly/first action. Failed/skipped to create sequence. Re-publish. */
            data.ResultCode = int(errCode)
            data.ResultStr = errStr
            p.publishResponse(data, true)
            p.RaiseRequestForFirstAction(data.Action)
        } else {
            /* Stale resp; Do nothing */
        }
        /*
         * Last seq is dropped or New seq got added or an action responded, which
         * may be stale, but can allow a seq to resume.
         * Try resume
         */

        p.resumeNextSequence()
    }()

    /*
     * Publish received response, even if stale/unexpected.
     * A failed first request, carry no real switch state data, yet publish
     * as it exposes internal error/failure of plugin.
     *
     * In any case it is result posted by plugin, hence publish/record
     */
    p.publishResponse(data, false)

    /* Validate & drop from active requests */
    if err = p.validate_with_active(data, seq); err != nil {
        return
    }

    /* Response matched active request */
    delete(p.activeRequests, data.Action)

    /* check sequence */
    if seq == nil {
        if anomalyID == data.InstanceId {
            /* Result from first action. Create sequence */
        } else {
            /*
             * Matched a valid request that engine raise. But no seq found.
             * Possibly stale response. As non-first action, likely seq timedout.
             * The current seq could be a pending for this request. Try resume it.
             */
            LogInfo("Stale response (%v)", data)
            if p.currentSequence != nil {
                /* Just a try; So don't track any error */
                p.resumeSequence(p.currentSequence)
            }
            return
        }
    } else {
        if anomalyID == data.InstanceId {
            /* Anomaly response creates sequence. Hence unexpected for existing seq */
            /* Duplicate post by plugin / pluginMgr ? */
            err = LogError("First action response for existing seq (%v) res(%v)", seq, data)
            return
        }
    }

    if seq == nil {
        /* No existing sequence found. Already validated to be first action. */
        if data.ResultCode != 0 {
            /*
             * No point in creating sequence. Call Raise request, which will succeed
             * if first action in sequence.
             *
             * NOTE: An action may go crazy returning too frequently. This is guarded
             * as below.
             * 1. A plugin's conf has min interval between two callbacks for same anomaly key.
             * 2. PluginMgr has a moving window of last 100 instances for a key of action+anomalyKey
             *    If that crosses an average of N seconds between 2, it disables the action and
             *    de-register it.
             *
             * If first action, re-publish is not done, as detection has failed.
             */
            errCode = LoMResponseCode(data.ResultCode)
            errStr = data.ResultStr
            return
        }
        bs, err := GetConfigMgr().GetSequence(data.Action)
        if err != nil {
            /* No sequence for this action. Likely stale non-first action or config changed */
            errCode = LoMMissingSequence
            errStr = "No sequence found"
            return
        }
        tnow := time.Now().Unix()
        ctx := make([]*ActionResponseData, 0, len(bs.Actions))
        ctx = append(ctx, data)

        seq = &sequenceState_t{
            anomalyInstanceId: anomalyID,
            sequence:          bs,
            seqStartEpoch:     tnow,
            seqExpEpoch:       tnow + int64(bs.Timeout),
            ctIndex:           1,
            context:           ctx,
        }
        p.addSequence(seq, bs.Timeout)
        return
    }

    /* reset current request */
    seq.currentRequest = nil

    /* Move ahead with next action */
    seq.context = append(seq.context, data)
    seq.ctIndex++

    /* Save last result */
    errCode = LoMResponseCode(data.ResultCode)
    errStr = data.ResultStr

    /* Process the sequence */
    if data.ResultCode != 0 {
        /* Need to abort the sequence */
        return
    }

    if seq.ctIndex >= len(seq.sequence.Actions) {
        /* WooHoo we are really complete  */
        seq.sequenceStatus = sequenceStatus_complete
        return
    }

    errCode, errStr, err = p.resumeSequence(seq)
    if errCode == LoMActionActive {
        /*
         * May be a prior seq left behind an active request. We got to wait
         * till it responds or this sequence timeout, whichever earlier.
         */
        errCode = LoMResponseOk
    }
    return
}

/*
 * resumeSequence
 *
 * Validate ctIndex. Attempt to raise request per ctIndex. If failed, return
 * appropriate error. If succeeded, add it to corresponding client's pending
 * server requests and list of active requests. Set it as current request of
 * this sequence.
 */
func (p *SeqHandler_t) resumeSequence(seq *sequenceState_t) (errCode LoMResponseCode, errStr string, err error) {
    regF := GetRegistrations()

    errCode = LoMResponseOk
    errStr = ""
    err = nil

    if seq == nil {
        errCode = LoMSequenceIncorrect
        errStr = "Resume called on nil sequence"
        return
    }
    if !seq.validate() {
        errCode = LoMSequenceIncorrect
        errStr = fmt.Sprintf("Resume called on invalid seq (%v)", *seq)
        return
    }
    if seq.currentRequest != nil {
        LogInfo("current req is active. Nothing to resume (%v)", *seq)
        return
    }
    if seq.ctIndex >= len(seq.sequence.Actions) {
        /*
         * This would be true only for empty sequence as ProcessActionResponse
         * would terminate a sequence upon last action.
         */
        if seq.ctIndex > 1 {
            errCode = LoMSequenceIncorrect
            errStr = fmt.Sprintf("Expect to hit here for empty seq only (%v)", *seq)
        } else {
            errCode = LoMSequenceEmpty
            errStr = fmt.Sprintf("Configured as empty (%v)", *seq)
        }
        return
    }
    nextAction := seq.sequence.Actions[seq.ctIndex]

    /* validate action */
    if regF.GetActiveActionInfo(nextAction.Name) == nil {
        errCode = LoMActionNotRegistered
        errStr = fmt.Sprintf("%v", LogError("%s: %s not registered. Abort", seq.sequence.SequenceName,
            nextAction.Name))
        return
    } else if _, ok := p.activeRequests[nextAction.Name]; ok {
        errCode = LoMActionActive
        errStr = fmt.Sprintf("%v", LogError("%s: %s (%s). Abort", seq.sequence.SequenceName,
            nextAction.Name, GetLoMResponseStr(LoMActionActive)))
        return
    }

    /* All clear. Fire request for the next action in sequence */
    req := &ServerRequestData{
        TypeServerRequestAction,
        &ActionRequestData{
            Action:            nextAction.Name,
            InstanceId:        GetUUID(),
            AnomalyInstanceId: seq.anomalyInstanceId,
            AnomalyKey:        seq.context[0].AnomalyKey,
            Timeout:           nextAction.Timeout,
            Context:           seq.context,
        },
    }

    /* Add to client's pending Q  to send to client upon client asking for a request. */
    /* Stays in this Q until client reads it */
    if err := regF.AddServerRequest(nextAction.Name, req); err != nil {
        errCode = LoMInternalError
        errStr = fmt.Sprintf("%v", err)
    } else {
        /* Track it in our active requests;  Waits here till response */
        tout := time.Now().Unix() + int64(nextAction.Timeout)
        act := &activeRequest_t{req, seq.anomalyInstanceId, tout}
        p.activeRequests[nextAction.Name] = act

        seq.currentRequest = act
    }
    return
}

/*
 * Complete a sequence for given errCode, which can be 0 for successful completion.
 *
 * Re-publish anomaly; Re-raise req to anomaly; delete seq; Resume any pending sequence
 */
func (p *SeqHandler_t) completeSequence(seq *sequenceState_t, errCode LoMResponseCode, errStr string) {
    if seq == nil {
        LogError("Internal error: Nil sequenceState_t")
    } else {

        /* Mark failed seq as complete */
        seq.sequenceStatus = sequenceStatus_complete
        anomalyResp := seq.context[0]

        if anomalyResp == nil {
            LogError("Expect non null anomaly resp seq (%v)", *seq)
        } else {
            anomalyResp.ResultCode = int(errCode)
            anomalyResp.ResultStr = errStr

            p.publishResponse(anomalyResp, true)

            p.dropSequence(seq)
            p.RaiseRequestForFirstAction(anomalyResp.Action)
        }
    }
}

/*
 * resumeNextSequence
 *
 * Upon completion of a sequence, resume next from pending by Pri
 *
 * Validate that there is no current running sequence.
 * Loop across all pending until one succeeds or list becomes empty.
 *
 * Call resumeSequence to resume. On success set its status as running and set it as
 * the current sequence. On failure, call completeSequence with error code and
 * the looping will auto attempt next, till no more. But if failure is because the
 * next req of this seq, leave it and try next in loop.
 */

func (p *SeqHandler_t) resumeNextSequence() {
    errCode := LoMResponseOk
    errStr := ""
    var err error = nil

    /* Walk until next sequence by Pri is resumed */
    for {
        if p.currentSequence != nil {
            /* There is a sequence in progress */
            if p.currentSequence.sequenceStatus != sequenceStatus_running {
                LogError("Current seq is not in running state (%v)", *p.currentSequence)
            }
            /* Possibly blocked by next req being active. Try again */
            return
        }
        if len(p.sortedSequencesByPri) == 0 {
            /* Nothing to resume */
            break
        }

        seq := p.sortedSequencesByPri[0]
        if seq.sequenceStatus != sequenceStatus_pending {
            LogError("seq is not in pending state (%v)", seq)
            return
        }

        errCode, errStr, err = p.resumeSequence(seq)
        if errCode == 0 {
            p.currentSequence = seq
            seq.sequenceStatus = sequenceStatus_running
            break
        } else if errCode != LoMActionActive {
            if (len(errStr) == 0) && (err != nil) {
                errStr = fmt.Sprintf("%v", err)
            }
            p.completeSequence(seq, errCode, errStr)
            /* Try next sequence */
        } else {
            /*
             * May be a prior seq left behind the next request of this sequence as active.
             * We got to wait till it responds or this sequence timeout, whichever earlier.
             *
             * Don;t
             */
            break
        }
    }
}

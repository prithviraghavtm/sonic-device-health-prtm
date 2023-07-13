package engine

import (
    . "lom/src/lib/lomcommon"
    . "lom/src/lib/lomipc"
    "os"
    "testing"
    "time"
)

/*
 * UT for crafted requests to exercise codes that are not easily possible
 * via functional tests
 */

/* Test context.go APIs for corner cases. */
func testContext(t *testing.T) {

    clientRegistrations = nil
    reg := GetRegistrations()
    {
        clientName := "Foo"

        defer func() {
            reg.DeregisterClient(clientName)
            delete(reg.activeActions, "Detect-0")
        }()

        /* simulate clients */
        if err := reg.RegisterClient(clientName); err != nil {
            t.Fatalf("****TEST FAILED: RegisterClient faied (%v)", err)
        }

        /* Simulate one action */
        reg.activeActions["Detect-0"] = &ActiveActionInfo_t{"Detect-0", clientName, 0}

        /* Test request timeout */
        /*
         * send testCnt requests with timeout.
         * verify the requests time out.
         */

        testCnt := 3
        ch := make(chan interface{}, testCnt)

        /* Send testCnt requests with different timeouts */
        for i := 0; i < testCnt; i++ {
            req := &LoMRequestInt{&LoMRequest{Client: clientName, TimeoutSecs: testCnt - i}, ch}
            if err := reg.PendServerRequest(req); err != nil {
                t.Fatalf("****TEST FAILED: failed in PendServerRequest (%d)/(%d) (%v)",
                    i, testCnt, err)
            }
        }
        /* Add One server request */
        req := &ServerRequestData{}
        if err := reg.AddServerRequest("Detect-0", req); err != nil {
            t.Fatalf("****TEST FAILED: Failed to fail for missing action's client.")
        }

        /* Wait for them to return. They would return in 3 seconds */
        cnt := testCnt + 2       /* Wait 2 seconds more before aborting */
        errCode := LoMResponseOk /* Expect first req to succeed */

        /* Wait till all requests completes or timeout */
        for i := 0; i < testCnt; {
            select {
            case r := <-ch:
                /* Validate timeout error */
                if res, ok := r.(*LoMResponse); !ok {
                    t.Fatalf("****TEST FAILED: PendServerRequest (%d/%d) (%T) != *LoMResponse",
                        i, testCnt, r)
                } else if res.ResultCode != int(errCode) {
                    t.Fatalf("****TEST FAILED: PendServerRequest (%d/%d) res (%d) != (%d)",
                        i, testCnt, res.ResultCode, errCode)
                } else if (errCode == LoMReqTimeout) && (res.ResultStr != GetLoMResponseStr(LoMReqTimeout)) {
                    t.Fatalf("****TEST FAILED: PendServerRequest (%d/%d) res (%s) != (%s)",
                        i, testCnt, res.ResultStr, GetLoMResponseStr(LoMReqTimeout))
                }
                i++
                errCode = LoMReqTimeout

            case <-time.After(time.Second):
                cnt--
                if cnt == 0 {
                    t.Fatalf("****TEST FAILED: PendServerRequest Aborting due to test timeout")
                }
            }
        }
    }
    {
        clientName := "client-Foo"
        clientName_new := "xxxx"
        /* Re-simulate client, but don't call registerClient as that kicks off processSendRequests */

        reg.activeClients[clientName] = &ActiveClientInfo_t{
            ClientName:           clientName,
            Actions:              make(map[string]struct{}),
            pendingWriteRequests: make(chan *ServerRequestData, CHAN_REQ_SIZE),
            pendingReadRequests:  make(chan *LoMRequestInt, CHAN_REQ_SIZE),
            abortCh:              make(chan interface{}, 2)}

        /* Simulate one action */
        reg.activeActions["Detect-0"] = &ActiveActionInfo_t{"Detect-0", clientName, 0}

        /* Random corner cases */

        /* Corner cases: Register/de-reg action & client */
        if err := reg.RegisterAction(nil); err == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for nil arg")
        }

        act := &ActiveActionInfo_t{clientName_new, "bar", 0}
        if err := reg.RegisterAction(act); err == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for non-existing client")
        }

        if info := reg.GetActiveActionInfo("foo"); info != nil {
            t.Fatalf("****TEST FAILED: Failed to fail for non-existing action")
        }
        ResetLastError()
        reg.DeregisterAction(clientName_new, "Detect-0")
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for incorrect client")
        }

        ResetLastError()
        reg.DeregisterClient("")
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for empty client")
        }

        /* Corner test cases for AddServerRequest */

        if err := reg.AddServerRequest("", nil); err == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for empty action")
        }
        if err := reg.AddServerRequest("xxx", nil); err == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for nil req")
        }
        req := &ServerRequestData{}
        if err := reg.AddServerRequest("xxx", req); err == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for non existing action")
        }

        /* Corrupt action's client name and test failing AddServerRequest */
        reg.activeActions["Detect-0"].Client = clientName_new
        if err := reg.AddServerRequest("Detect-0", req); err == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for missing action's client.")
        }
        reg.activeActions["Detect-0"].Client = clientName

        /*
         * Test server pending req channel overflow.
         */

        for i := 0; i < CHAN_REQ_SIZE; i++ {
            if err := reg.AddServerRequest("Detect-0", req); err != nil {
                t.Fatalf("****TEST FAILED: AddServerRequest Failed to write %d/%d (%v)",
                    i, CHAN_REQ_SIZE, err)
            }
        }
        if err := reg.AddServerRequest("Detect-0", req); err == nil {
            t.Fatalf("****TEST FAILED: AddServerRequest Failed to fail to write %d/%d",
                CHAN_REQ_SIZE+1, CHAN_REQ_SIZE)
        }

        /* Corner test cases for PendServerRequest */
        if err := reg.PendServerRequest(nil); err == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for nil req")
        }

        lomReq := &LoMRequest{Client: clientName_new}
        lreq := &LoMRequestInt{Req: lomReq}
        if err := reg.PendServerRequest(lreq); err == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for non-existing client")
        }

        /*
         * Test client pending req channel overflow
         */
        lreq.Req.Client = clientName
        for i := 0; i < CHAN_REQ_SIZE; i++ {
            if err := reg.PendServerRequest(lreq); err != nil {
                t.Fatalf("****TEST FAILED: PendServerRequest Failed to write %d/%d (%v)",
                    i, CHAN_REQ_SIZE, err)
            }
        }
        if err := reg.PendServerRequest(lreq); err == nil {
            t.Fatalf("****TEST FAILED: PendServerRequest Failed to write %d/%d",
                CHAN_REQ_SIZE+1, CHAN_REQ_SIZE)
        }
    }
}

/* Test serverReqHandler.go APIs for corner cases. */
func testserverReqHandler(t *testing.T) {
    /* Verify LoMResponseStr size not matching error code count */
    LoMResponseStr = append(LoMResponseStr, "rerere")
    if ok, err := LoMResponseValidate(); ok {
        t.Fatalf("LoMResponseValidate not failing as expected")
    } else if err == nil {
        t.Fatalf("LoMResponseValidate empty error message")
    }
    /* Verify responses for few codes */
    if m := GetLoMResponseStr(LoMResponseOk); m != LoMResponseOkStr {
        t.Fatalf("LoMResponseStr (%s) != (%s)", m, LoMResponseOkStr)
    }
    if m := GetLoMResponseStr(LoMResponseCode(LoMErrorCnt + 2)); m != LoMResponseUnknownStr {
        t.Fatalf("LoMResponseStr (%s) != (%s)", m, LoMResponseUnknownStr)
    }
    if m := GetLoMResponseStr(LoMResponseCode(LOM_RESP_CODE_START)); m != LoMResponseStr[0] {
        t.Fatalf("LoMResponseStr (%s) != (%s)", m, LoMResponseStr[0])
    }

    /* Test createLoMResponse */
    {
        ResetLastError()
        createLoMResponse(LoMErrorCnt, "")
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for invalid code")
        }

        res := createLoMResponse(LoMReqFailed, "")
        if len(res.ResultStr) == 0 {
            t.Fatalf("****TEST FAILED: Failed to construct error message")
        }
    }

    /* Test processRequest corner cases - only failure */
    {
        handler := GetServerReqHandler()
        m := ""
        lreq := &LoMRequest{}
        var req *LoMRequestInt
        for i := 0; i < 5; i++ {
            failed := true
            switch i {
            case 0:
                /* req is nil */
                m = "nil req"
                req = nil
            case 1:
                /* Nil members */
                req = &LoMRequestInt{}
                m = "nil req members"
            case 2:
                /* Nil chResponse */
                req.Req = lreq
                m = "nil chResponse"
            case 3:
                /* ChResponse with len == cap */
                req.ChResponse = make(chan interface{})
                m = "no space in chResponse"
            case 4:
                /* Hope to succeed for unknown req type*/
                req.ChResponse = make(chan interface{}, 1)
                req.Req.ReqType = TypeNotifyActionHeartbeat + 5
                m = "Unknown req type reflects in reponse result"
                failed = false
            default:
                t.Fatalf("Test Error: Unexpected case{")
            }

            ResetLastError()
            handler.processRequest(req)
            if failed {
                if GetLastError() == nil {
                    t.Fatalf("****TEST FAILED: Failed to fail for (%s)", m)
                }
            } else if GetLastError() != nil {
                t.Fatalf("****TEST FAILED: Failed to succeed for (%s)", m)
            }
        }
    }

    /* Test for incorrect message type for various requests.*/
    {
        handler := GetServerReqHandler()
        lreq := &LoMRequest{}

        if res := handler.registerClient(lreq); res.ResultCode != int(LoMIncorrectReqData) {
            t.Fatalf("****TEST FAILED: Failed res.ResultCode != %d", int(LoMIncorrectReqData))
        }

        if res := handler.deregisterClient(lreq); res.ResultCode != int(LoMIncorrectReqData) {
            t.Fatalf("****TEST FAILED: Failed res.ResultCode != %d", int(LoMIncorrectReqData))
        }

        if res := handler.registerAction(lreq); res.ResultCode != int(LoMIncorrectReqData) {
            t.Fatalf("****TEST FAILED: Failed res.ResultCode != %d", int(LoMIncorrectReqData))
        }

        if res := handler.deregisterAction(lreq); res.ResultCode != int(LoMIncorrectReqData) {
            t.Fatalf("****TEST FAILED: Failed res.ResultCode != %d", int(LoMIncorrectReqData))
        }

        if res := handler.notifyHeartbeat(lreq); res.ResultCode != int(LoMIncorrectReqData) {
            t.Fatalf("****TEST FAILED: Failed res.ResultCode != %d", int(LoMIncorrectReqData))
        }

        if res := handler.sendServerResponse(lreq); res.ResultCode != int(LoMIncorrectReqData) {
            t.Fatalf("****TEST FAILED: Failed res.ResultCode != %d", int(LoMIncorrectReqData))
        }

        req := &LoMRequestInt{Req: lreq}
        if res := handler.recvServerRequest(req); res.ResultCode != int(LoMIncorrectReqData) {
            t.Fatalf("****TEST FAILED: Failed res.ResultCode != %d", int(LoMIncorrectReqData))
        }

        lreq.Client = "XYZ"
        lreq.ReqData = MsgRecvServerRequest{}
        if res := handler.recvServerRequest(req); res.ResultCode != int(LoMReqFailed) {
            t.Fatalf("****TEST FAILED: Failed res.ResultCode != %d", LoMReqFailed)
        }
    }
}

/* Test sequenceHandler.go APIs for corner cases. */
func testSequenceHandler(t *testing.T) {
    {
        /* Sequence validate */
        bs := &BindingSequence_t{}
        res := &ActionResponseData{}
        seq := &sequenceState_t{}
        tmr := &OneShotEntry_t{}

        for i := 0; i < 9; i++ {
            m := ""
            failed := true
            switch i {
            case 0:
                m = "Empty sequence"
            case 1:
                seq.anomalyInstanceId = "erere"
                m = "No Binding sequence"
            case 2:
                seq.sequence = bs
                m = "Missing start epoch"
            case 3:
                seq.seqStartEpoch = 10
                m = "Missing expEpoch is smaller"
            case 4:
                seq.seqExpEpoch = seq.seqStartEpoch + 10
                m = "Invalid ct Index"
            case 5:
                seq.ctIndex = 1
                m = "Missing context"
            case 6:
                seq.context = make([]*ActionResponseData, 2)
                seq.context[0] = res
                seq.sequenceStatus = sequenceStatus_running
                m = "Missing current req"
            case 7:
                res.Action = "hello"
                m = "Missing exp timer"
            case 8:
                seq.expTimer = tmr
                m = "Expected to pass"
                failed = false
            }

            if failed == seq.validate() {
                t.Fatalf("****TEST FAILED: failed:%v m:%s", failed, m)
            }
        }
    }

    {
        /* Test Get/Init handler */
        seqHandler = nil

        ResetLastError()
        if GetSeqHandler() != nil {
            t.Fatalf("****TEST FAILED: Expect nil handler")
        } else if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to get error logged.")
        }

        ResetLastError()
        InitSeqHandler(nil)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for nil chTimer")
        }

        InitSeqHandler(make(chan int64, 100))
        handler := GetSeqHandler()
        if handler == nil {
            t.Fatalf("****TEST FAILED: failed to init seqHandler")
        }

        ResetLastError()
        InitSeqHandler(make(chan int64))
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for duplicate init")
        }
    }

    {
        /* Raise request corner cases */
        regF := GetRegistrations()
        handler := GetSeqHandler()

        /* Non existing active action */
        if err := handler.RaiseRequestForFirstAction("bar"); err == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for non-existing action")
        }

        /* Simulate active action for Detect-0 */
        regF.activeActions["Detect-0"] = &ActiveActionInfo_t{}

        /* Pre-existing active request */
        handler.activeRequests["Detect-0"] = &activeRequest_t{}
        if err := handler.RaiseRequestForFirstAction("Detect-0"); err == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for pre-existing req")
        }
        delete(handler.activeRequests, "Detect-0")

        /* Pre-existing sequence for this action */
        handler.sequencesByFirstAction["Detect-0"] = &sequenceState_t{}
        if err := handler.RaiseRequestForFirstAction("Detect-0"); err == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for pre-existing seq")
        }
        delete(handler.sequencesByFirstAction, "Detect-0")

        /* Fail AddServerRequst due to missing active client */
        if err := handler.RaiseRequestForFirstAction("Detect-0"); err == nil {
            t.Fatalf("****TEST FAILED: Failed to AddServerRequest for inactive client")
        }
    }

    {
        /* Drop req for in-prog sequence */
        handler := GetSeqHandler()

        if len(handler.activeRequests) != 0 {
            t.Fatalf("****TEST FAILED: Failed to remove active req")
        }
        handler.activeRequests["Detect-0"] = &activeRequest_t{anomalyID: "id"}
        handler.sequencesByAnomaly["id"] = nil

        handler.DropRequest("Detect-0")
        if len(handler.activeRequests) != 0 {
            t.Fatalf("****TEST FAILED: Failed to remove active req")
        }
        delete(handler.sequencesByAnomaly, "id")
    }

    {
        handler := GetSeqHandler()

        /* Add Sequence */
        ResetLastError()
        handler.addSequence(nil, 0)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for nil seq")
        }

        seq := &sequenceState_t{}
        ResetLastError()
        handler.addSequence(seq, 0)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for empty seq context")
        }

        /* Drop Sequence */
        ResetLastError()
        handler.dropSequence(nil)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for nil seq")
        }
    }

    {
        /* Publish/process response */

        handler := GetSeqHandler()

        /* Nil response */
        ResetLastError()
        handler.publishResponse(nil, false)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail publishResponse for nil response")
        }

        /* Nil response */
        ResetLastError()
        handler.ProcessResponse(nil)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail processResponse for nil response")
        }

        /* Set incorrect ReqType */
        msg := &MsgSendServerResponse{ReqType: TypeServerRequestCount}
        ResetLastError()
        handler.ProcessResponse(msg)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for incorrect ServerReqDataType")
        }

        /* Leave incorrect ReqData type */
        msg.ReqType = TypeServerRequestAction
        ResetLastError()
        handler.ProcessResponse(msg)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail for incorrect ReqData")
        }

    }

    {
        /* Process Action Response */

        actionName := "Foo"
        handler := GetSeqHandler()

        /* Nil response */
        ResetLastError()
        handler.processActionResponse(nil)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail ProcessActionResponse for nil response")
        }

        /* Invalid ActionResponseData */
        data := &ActionResponseData{}
        ResetLastError()
        handler.processActionResponse(data)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail ProcessActionResponse for invalid response")
        }

        /* Invalid sequence */
        handler.sequencesByAnomaly["123"] = &sequenceState_t{}
        data = &ActionResponseData{actionName, "123", "123", "key", "fff", 0, ""}
        ResetLastError()
        handler.processActionResponse(data)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail ProcessActionResponse for invalid seq")
        }

        /* Setup for a good ActionResponse in steps */
        bs := &BindingSequence_t{}
        reqTmp := &activeRequest_t{}
        req := &activeRequest_t{}
        res := &ActionResponseData{}
        handler.activeRequests[actionName] = reqTmp
        seq := &sequenceState_t{sequenceStatus_running,
            "123", bs, 10, 20, 1, []*ActionResponseData{res}, req, &OneShotEntry_t{}}
        handler.sequencesByAnomaly["123"] = seq

        /* seq.currentRequest != active request found for this action */
        ResetLastError()
        handler.processActionResponse(data)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail ProcessActionResponse req mismatch")
        }

        /* Active & seq in sync */
        handler.activeRequests[actionName] = req
        sreq := &ServerRequestData{ReqType: TypeServerRequestCount}
        req.req = sreq

        /* Fail for incorrect ReqType */
        ResetLastError()
        handler.processActionResponse(data)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail ProcessActionResponse ReqType mismatch")
        }

        sreq.ReqType = TypeServerRequestAction
        /* Fail for incorrect ReqData type */
        ResetLastError()
        handler.processActionResponse(data)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail ProcessActionResponse ReqData type mismatch")
        }

        /* Instance Id between active request & response mismatch */
        sreq.ReqData = &ActionRequestData{InstanceId: "999"}
        ResetLastError()
        handler.processActionResponse(data)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail ProcessActionResponse instance id mismatch with active req")
        }

        /* Seq exists, but response is for first action. InstanceId == AnomalyId. Fail */
        sreq.ReqData = &ActionRequestData{InstanceId: "123"}
        ResetLastError()
        handler.processActionResponse(data)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail ProcessActionResponse first action resp for existing seq")
        }

        /* Simulate no active request for this response's action */
        delete(handler.activeRequests, actionName)
        ResetLastError()
        handler.processActionResponse(data)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail ProcessActionResponse missing active req")
        }

        /* Simulate no seq by changing anomaly Id and response not for first action */
        handler.activeRequests[actionName] = req         /* active req - Yes */
        data.AnomalyInstanceId = data.InstanceId + "abc" /* Not first action */
        handler.currentSequence = &sequenceState_t{sequenceStatus: sequenceStatus_running}
        ResetLastError()
        handler.processActionResponse(data)
        if GetLastError() != nil {
            /* No error expected */
            t.Fatalf("****TEST FAILED: Failed to pass ProcessActionResponse delayed response")
        }

        /* No seq; response is for first action; But GetSequence fails */
        handler.activeRequests[actionName] = req /* active req - Yes */
        data.AnomalyInstanceId = data.InstanceId /* first action */
        delete(handler.sequencesByAnomaly, "123")
        ResetLastError()
        handler.processActionResponse(data)
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: Failed to fail ProcessActionResponse no binding seq.")
        }
    }
    {
        /* Corner test cases for resumeSequence */
        regF := GetRegistrations()
        handler := GetSeqHandler()

        /* Fail for nil seq */
        if c, _, _ := handler.resumeSequence(nil); c != LoMSequenceIncorrect {
            t.Fatalf("****TEST FAILED: resumeSequence for nil %d != %d",
                c, LoMSequenceIncorrect)
        }

        /* Fail for invalid seq */
        seq := &sequenceState_t{}
        if c, _, _ := handler.resumeSequence(seq); c != LoMSequenceIncorrect {
            t.Fatalf("****TEST FAILED: resumeSequence for invalid seq %d != %d",
                c, LoMSequenceIncorrect)
        }

        actionName := "xyz"
        bs := &BindingSequence_t{"ee", 0, 0, []*BindingActionCfg_t{
            {}, {Name: actionName}}}
        // reqTmp := &activeRequest_t{}
        req := &activeRequest_t{}
        res := &ActionResponseData{Action: "foo"}
        // handler.activeRequests[actionName] = reqTmp
        seq = &sequenceState_t{sequenceStatus_running,
            "123", bs, 10, 20, 1, []*ActionResponseData{res}, req, &OneShotEntry_t{}}

        /* Expect to succeed as current req is non nil */
        if c, _, _ := handler.resumeSequence(seq); c != LoMResponseOk {
            t.Fatalf("****TEST FAILED: resumeSequence: non null current req to succeed %d != %d",
                c, LoMResponseOk)
        }

        /* Fail to resume as this action is not active. */
        seq.currentRequest = nil
        if c, _, _ := handler.resumeSequence(seq); c != LoMActionNotRegistered {
            t.Fatalf("****TEST FAILED: resumeSequence: non active action. %d != %d",
                c, LoMActionNotRegistered)
        }

        /* Ensure active action, but fail due to missing client */
        regF.activeActions[actionName] = &ActiveActionInfo_t{}
        if c, _, _ := handler.resumeSequence(seq); c != LoMInternalError {
            t.Fatalf("****TEST FAILED: resumeSequence: non active client. %d != %d",
                c, LoMInternalError)
        }

        /* Fail complete sequence due to nil context for first action */
        seq.context[0] = nil
        ResetLastError()
        handler.completeSequence(seq, 0, "")
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: completeSequence missing context to report")
        }

        /* resumeNextSequence to complain when current seq is not in running state */
        handler.currentSequence = seq
        seq.sequenceStatus = sequenceStatus_pending
        ResetLastError()
        handler.resumeNextSequence()
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: resumeNextSequence fail on pending")
        }

        /* Simulate non-pending seq in Q */
        handler.currentSequence = nil
        handler.sortedSequencesByPri = append(handler.sortedSequencesByPri, seq)
        LogDebug("-------- p.sortedSequencesByPri(%d): [%v]", len(handler.sortedSequencesByPri), handler.sortedSequencesByPri)
        seq.sequenceStatus = sequenceStatus_running
        ResetLastError()
        handler.resumeNextSequence()
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: resumeNextSequence fail on running in Q")
        }

        /* Simulate failed resume of next pending */
        handler.sortedSequencesByPri = append(handler.sortedSequencesByPri, seq)
        LogDebug("-------- p.sortedSequencesByPri(%d): [%v]", len(handler.sortedSequencesByPri), handler.sortedSequencesByPri)
        seq.sequenceStatus = sequenceStatus_pending
        seq.anomalyInstanceId = "" /* Failing to validate will fail resume */
        ResetLastError()
        handler.resumeNextSequence()
        if GetLastError() == nil {
            t.Fatalf("****TEST FAILED: resumeNextSequence fail on running in Q")
        }
    }
}

/* Test engine.go APIs for corner cases. */
func testEngine(t *testing.T) {
    if _, err := startUp("ddd", []string{"-xxx", "rrr"}); err == nil {
        t.Fatalf("****TEST FAILED: engine::startup incorrect args")
    }
    if _, err := startUp("ddd", []string{"-path", ""}); err == nil {
        t.Fatalf("****TEST FAILED: engine::startup Incorrect path")
    }
    if _, err := startUp("ddd", []string{"-path", "ddd"}); err == nil {
        t.Fatalf("****TEST FAILED: engine::startup Incorrect path")
    }

    os.Args = []string{"-user=bla"}
    ResetLastError()
    Main()
    if GetLastError() == nil {
        t.Fatalf("****TEST FAILED: engine: main to fail for invalid args")
    }
}

var utList = []func(t *testing.T){
    testContext,
    testserverReqHandler,
    testSequenceHandler,
    testEngine,
}

var xutList = []func(t *testing.T){
    testEngine,
}

func TestAll(t *testing.T) {
    initConfig(t)
    cfgPath = CFGPATH
    os.Setenv("LOM_CONF_LOCATION", "./config/")
    if err := InitConfigPath(cfgPath); err != nil {
        t.Fatalf("Failed to init configMgr")
    }

    for _, f := range utList {
        f(t)
    }
    cleanConfigFiles()
}

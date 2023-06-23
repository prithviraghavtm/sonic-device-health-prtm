package engine

/*
 *  Mock PublishEventAPI
 *  This test code combines unit test & functional test - Two in one shot
 *
 *  Sample Scenarios created by test collections:
 *  NOTE:   Take this as example only. I wrote this at the start of the test as how
 *          I wanted to shape up. So final collections much better revised.
 *          Still this is useful to get the basic insight.
 *
 *      Register/de-register:
 *          1.  register empty client - Fails
 *          2.  register client CLIENT_0 - Succeeds
 *          3.  re-register client CLIENT_0 - fails
 *          4.  register action with empty name ("") under CLIENT_0 client - fails
 *          5.  register action "Detect-0" under CLIENT_0 client - Succeeds
 *          6.  re-register action "Detect-0" under CLIENT_0 client - Succeeds
 *          7.  register client CLIENT_1
 *          8.  re-register action "Detect-0" under CLIENT_1 client. De-register from
                client0 & re-register - succeeds
 *          9.  register "Safety-chk-0", "Mitigate-0", "Mitigate-2" under CLIENT_0
 *          10. register ""Detect-1", "Safety-chk-1", "Mitigate-1", "Detect-2" & "Mitigate-2" under CLIENT_1
 *          11. register "Disabled-0" nder CLIENT_0 client - fails
 *          12. verify all registrations
 *
 *      Scenarios:
 *      Initial requests
 *          1.  Expect requests from engine for "Detect-0", "Detect-1" & "Detect-2"
 *
 *      One proper sequence
 *          2. "Detect-0" returns good. Expect "Safety-chk-0"; return good; expect"Mitigate-0"; return good
 *              verify publish responses
 *          3. Expect request for detect-0
 *          4. "Detect-0" returns good. Expect "Safety-chk-0"; return good; expect"Mitigate-0"; return fail
 *              verify publish responses
 *          5. "Detect-0" returns good. Expect "Safety-chk-0"; return fail
 *              verify publish responses
 *          6. "Detect-0" returns fail.
 *              verify publish responses
 *          7. "Detect-0" returns good. Expect "Safety-chk-0"; return good; expect"Mitigate-0"; sleep 3s; mmitigate-0  responds; seq timeout
 *              verify publish responses
 *          8. "Detect-0" returns good. Expect "Safety-chk-0"; Sleep forever; req expect to timeout
 *              verify publish responses
 *          9. "Detect-2" & "Detect-1" returns good; But "Safety-chk-0" busy. bind-2 timesout.
 *          10.Expect "Safety-chk-1" call; return good; expect "Mitigate-1"; return good
 *              verify publish responses
 *          11.Trigger "Safety-chk-0" respond
 *          12."Detect-2" return good; "Safety-chk-0"; good; "Safety-chk-2"; good; "Mitigate-2"; good; seq complete
 *              verify publish responses
 *
 *          13."Detect-0" good; safety-check-0 sleeps; bind-0 timesout.
 *          14."Detect-0" good; safety-check-0 not called; bind-0 timesout.
 *          15.De-register safety-check-0 & re-register
 *          16."Detect-0" good; safety-check-0 good; mitigate-0 good; bind-0 good.
 *              verify publish responses
 *          17. NotifyHearbeat for "Detect-0"
 *              Verify responnse
 *          18. NotifyHearbeat for "xyz" non-existing
 *              Verify responnse
 *
*/

import (
    "encoding/json"
    "fmt"
    . "lom/src/lib/lomcommon"
    . "lom/src/lib/lomipc"
    "net"
    // "net/rpc/jsonrpc"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "testing"
    "time"
)

const EMPTY_STR = ""
const CLIENT_0 = "client-0"
const CLIENT_1 = "client-1"
const CLIENT_2 = "client-2"

/*
 * During test run, test code keep this chan active. An idle channel for timeout
 * seconds will abort the test
 */
var chTestHeartbeat = make(chan string)

/*
 * Globals.conf
 */
var globals_conf = `{"ENGINE_HB_INTERVAL_SECS" : 3 }`

var procs_conf = `{}`

/*
 *  Actions.conf
 */
var actions_conf = `{
        "Detect-0" : { "name": "Detect-0" },
        "Safety-chk-0" : { "name": "Safety-chk-0", "Timeout": 1},
        "Mitigate-0" : { "name": "Mitigate-0", "Timeout": 6},
        "Detect-1" : { "name": "Detect-1" },
        "Safety-chk-1" : { "name": "Safety-chk-1", "Timeout": 7},
        "Mitigate-1" : { "name": "Mitigate-1", "Timeout": 8},
        "Detect-2" : { "name": "Detect-2" },
        "Safety-chk-2" : { "name": "Safety-chk-2", "Timeout": 1},
        "Mitigate-2" : { "name": "Mitigate-2", "Timeout": 6},
        "Disabled-0" : { "name": "Disabled-0", "Disable": true},
        "Detect-3" : { "name": "Detect-3" }
        }`

var bindings_conf = `{ "bindings": [
    {
        "SequenceName": "bind-0", 
        "Priority": 0,
        "Timeout": 2,
        "Actions": [
            {"name": "Detect-0" },
            {"name": "Safety-chk-0", "sequence": 1 },
            {"name": "Mitigate-0", "sequence": 2 }
        ]
    },
    {
        "SequenceName": "bind-1", 
        "Priority": 1,
        "Timeout": 19,
        "Actions": [
            {"name": "Detect-1" },
            {"name": "Safety-chk-1", "sequence": 1 },
            {"name": "Mitigate-1", "sequence": 2 }
        ]
    },
    {
        "SequenceName": "bind-2", 
        "Priority": 0,
        "Timeout": 8,
        "Actions": [
            {"name": "Detect-2" },
            {"name": "Safety-chk-2", "sequence": 1 },
            {"name": "Safety-chk-0", "sequence": 2 },
            {"name": "Mitigate-2", "sequence": 3 }
        ]
    },
    {
        "SequenceName": "bind-3", 
        "Priority": 0,
        "Timeout": 8,
        "Actions": [
            {"name": "Detect-3" }
        ]
    }
]}`

/*
 * A bunch of APIs from client transport or internal to engine to be called with varying
 * args and expected results
 */

type clientAPIID int

const (
    REG_CLIENT = clientAPIID(iota)
    REG_ACTION
    DEREG_CLIENT
    DEREG_ACTION
    RECV_REQ
    SEND_RES
    SEQ_COMPLETE
    SHUTDOWN
    NOTIFY_HB
    CHK_ACTIV_REQ
    CHK_REG_ACTIONS
    PAUSE
)

/*
 * Req / Resp received/sent will need to be saved for proper
 * verification of subsequent req/resp as these share context
 *
 * These APIs provide a way to save/restore/reset
 */
type savedResults_t map[int][]any

var saveResults = make(savedResults_t)

func printResultAny(entire bool) string {
    if !entire {
        ret := make([]int, len(saveResults))
        i := 0
        for k, _ := range saveResults {
            ret[i] = k
            i++
        }
        return fmt.Sprintf("%v", ret)
    }
    return fmt.Sprintf("%v", saveResults)
}

func saveResultAny(seq int, data any) {
    if _, ok := saveResults[seq]; !ok {
        saveResults[seq] = make([]any, 0, 5) /* 5 - init size to minimize realloc */
    }
    saveResults[seq] = append(saveResults[seq], data)
}

func restoreResultAny(seq int, index int) (any, error) {
    /* negative index walk back */
    if v, ok := saveResults[seq]; !ok {
        return nil, LogError("No saved results for seq(%d)", seq)
    } else {
        i := index
        if i < 0 {
            i = len(v) + index
            if i < 0 {
                return nil, LogError("Incorrect index=%d len=%d", index, len(v))
            }
        } else if i >= len(v) {
            return nil, LogError("Incorrect index=%d len=%d", index, len(v))
        }
        return v[i], nil
    }
}

func resetResultAny(seq int) {
    delete(saveResults, seq)
}

func resetResultAll() {
    saveResults = make(savedResults_t)
}

/* Mock publish fn given to engine for any publish calls */
var publishCh = make(chan string, 10)

func testPublish(s string) string {

    /* Write to channel if there is space */
    if len(publishCh) < cap(publishCh) {
        publishCh <- s
    } else {
        LogError("ERROR: publishCh too full. Publish skipped ")
    }
    LogDebug("testPublish: (%s)", s)
    return s
}

const CFGPATH = "/tmp"

/* Helper API */
func createFile(t *testing.T, name string, s string) {
    fl := filepath.Join(CFGPATH, name)

    if len(s) == 0 {
        s = "{}"
    }
    if f, err := os.Create(fl); err != nil {
        t.Fatalf("Failed to create file (%s)", fl)
    } else {
        if _, err := f.WriteString(s); err != nil {
            t.Fatalf("Failed to write file (%s)", fl)
        }
        f.Close()
    }
}

/*
 * Starts the engine
 *
 * Engine starts RPC listeners
 */
func initServer(t *testing.T) (engine *engine_t) {
    chTestHeartbeat <- "Start: initServer"
    defer func() {
        chTestHeartbeat <- "End: initServer"
    }()

    engine = nil
    if e, err := startUp("test", []string{"-path", CFGPATH}); err != nil {
        t.Fatalf("initServer failed in startup")
        return
    } else {
        engine = e
    }

    chTestHeartbeat <- "Complete: initServer"
    return
}

/*
 * Holder of client transport & test instance. All wrapper APIs are
 * methods on this object.
 */
type callArgs struct {
    t     *testing.T
    lstTx map[string]*ClientTx
}

func (p *callArgs) getTxWithTimeout(cl string, tout int) *ClientTx {
    tx, ok := p.lstTx[cl]
    if !ok {
        tx = GetClientTx(tout)
        if tx != nil {
            p.lstTx[cl] = tx
        } else {
            p.t.Fatalf("Failed to get client")
        }
    }
    return tx
}

func (p *callArgs) getTx(cl string) *ClientTx {
    return p.getTxWithTimeout(cl, 0)
}

/*
 * Implementation for clientAPIID - REG_CLIENT
 *
 * Invoke tx.RegisterClient as requested in collection.
 *
 * Refer test collection for sample usage.
 */
func (p *callArgs) call_register_client(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_register_client"
    defer func() {
        chTestHeartbeat <- "End: call_register_client"
    }()

    if len(te.args) < 1 {
        p.t.Fatalf("Test index %v: Expect only one arg len(%d)", ti, len(te.args))
    }
    a := te.args[0]
    clName, ok := a.(string)
    if !ok {
        p.t.Fatalf("Test index %v: Expect string as arg for client name (%T)", ti, a)
    }
    tout := 0
    if len(te.args) > 1 {
        if t, ok := te.args[1].(int); !ok {
            p.t.Fatalf("Test index %v: Expect int as second arg. (%T)", ti, te.args[1])
        } else {
            tout = t
        }
    }
    tx := p.getTxWithTimeout(te.clTx, tout)
    err := tx.RegisterClient(clName)
    if te.failed != (err != nil) {
        p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
            ti, te.toStr(), err)
    }
}

/*
 * Implementation for clientAPIID - REG_ACTION
 *
 * Invoke tx.RegisterAction as requested in collection.
 *
 * Refer test collection for sample usage.
 */
func (p *callArgs) call_register_action(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_register_action"
    defer func() {
        chTestHeartbeat <- "End: call_register_action"
    }()

    if len(te.args) != 1 {
        p.t.Fatalf("Test index %v: Expect only one arg len(%d)", ti, len(te.args))
    }
    a := te.args[0]
    actName, ok := a.(string)
    if !ok {
        p.t.Fatalf("Test index %v: Expect string as arg for action name (%T)", ti, a)
    }
    tx := p.getTx(te.clTx)
    err := tx.RegisterAction(actName)
    if te.failed != (err != nil) {
        p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
            ti, te.toStr(), err)
    }
}

/*
 * Implementation for clientAPIID - DEREG_ACTION
 *
 * Invoke tx.DeregisterAction as requested in collection.
 *
 * Refer test collection for sample usage.
 */
func (p *callArgs) call_deregister_action(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_deregister_action"
    defer func() {
        chTestHeartbeat <- "End: call_deregister_action"
    }()

    if len(te.args) != 1 {
        p.t.Fatalf("Test index %v: Expect only one arg len(%d)", ti, len(te.args))
    }
    a := te.args[0]
    actName, ok := a.(string)
    if !ok {
        p.t.Fatalf("Test index %v: Expect string as arg for action name (%T)", ti, a)
    }
    tx := p.getTx(te.clTx)
    err := tx.DeregisterAction(actName)
    if te.failed != (err != nil) {
        p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
            ti, te.toStr(), err)
    }
}

/*
 * Implementation for clientAPIID - DEREG_CLIENT
 *
 * Invoke tx.DeregisterClient as requested in collection.
 *
 * Refer test collection for sample usage.
 */
func (p *callArgs) call_deregister_client(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_deregister_client"
    defer func() {
        chTestHeartbeat <- "End: call_deregister_client"
    }()

    if te.args != nil {
        p.t.Fatalf("Test index %v: Expect nil arg len(%d)", ti, len(te.args))
    }
    tx := p.getTx(te.clTx)
    err := tx.DeregisterClient()
    if te.failed != (err != nil) {
        p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
            ti, te.toStr(), err)
    }
}

/* Helper API */
func compStr(msg, rcv, tst string) string {
    if len(rcv) == 0 {
        return fmt.Sprintf("%s empty", msg)
    }
    if (len(tst) != 0) && (tst != rcv) {
        return fmt.Sprintf("%s mismatch (%s) != (%s)", msg, rcv, tst)
    }
    return ""
}

/* Helper API */
func compActResData(rcv *ActionResponseData, tst *ActionResponseData) string {
    if s := compStr("Action", rcv.Action, tst.Action); len(s) > 0 {
        return s
    }
    if s := compStr("InstanceId", rcv.InstanceId, tst.InstanceId); len(s) > 0 {
        return s
    }
    if s := compStr("AnomalyInstanceId", rcv.AnomalyInstanceId,
        tst.AnomalyInstanceId); len(s) > 0 {
        return s
    }
    if s := compStr("AnomalyKey", rcv.AnomalyKey, tst.AnomalyKey); len(s) > 0 {
        return s
    }
    if s := compStr("Response", rcv.Response, tst.Response); len(s) > 0 {
        return s
    }
    if (tst.ResultCode != -1) && (tst.ResultCode != rcv.ResultCode) {
        return fmt.Sprintf("ResultCode mismatch (%d) != (%d)", rcv.ResultCode, tst.ResultCode)
    }
    if (len(tst.ResultStr) != 0) && (len(rcv.ResultStr) == 0) {
        return fmt.Sprintf("Expect non empty result string")
    }
    return ""
}

/* Helper API */
func compActReqData(rcv *ActionRequestData, tst *ActionRequestData) string {
    if s := compStr("Action", rcv.Action, tst.Action); len(s) > 0 {
        return s
    }
    if s := compStr("InstanceId", rcv.InstanceId, tst.InstanceId); len(s) > 0 {
        return s
    }
    if s := compStr("AnomalyInstanceId", rcv.AnomalyInstanceId,
        tst.AnomalyInstanceId); len(s) > 0 {
        return s
    }
    if (tst.Timeout != -1) && (tst.Timeout != rcv.Timeout) {
        return fmt.Sprintf("Timeout mismatch (%d) != (%d)", rcv.Timeout, tst.Timeout)
    }
    if rcv.InstanceId != rcv.AnomalyInstanceId {
        if s := compStr("AnomalyKey", rcv.AnomalyKey, tst.AnomalyKey); len(s) > 0 {
            return s
        }

        if len(tst.Context) == 0 {
            return fmt.Sprintf("Context: Expect non-empty")
        }
        if tst.Context != nil {
            if len(tst.Context) != len(rcv.Context) {
                return fmt.Sprintf("Context: len mismatch (%d) != (%d)",
                    len(rcv.Context), len(tst.Context))
            }
            for i, t := range tst.Context {
                if s := compActResData(rcv.Context[i], t); len(s) > 0 {
                    return s
                }
            }
        }
    } else {
        if len(rcv.AnomalyKey) != 0 {
            return fmt.Sprintf("AnomalyKey: Expect empty")
        }
        if len(tst.Context) != 0 {
            return fmt.Sprintf("Context: Expect empty (%d)", len(tst.Context))
        }
    }
    return ""
}

/* Helper API */
func buildReq(exp *ActionRequestData, seq int) (*ActionRequestData, error) {
    /*
     * Test code data can at most carry action name & timeout
     * as rest are dynamic and set by engine.
     * But if you are not first request and has a reference to last
     * you could get anomaly instance id & key. Plus context if any.
     *
     * But to get full set of context, we need last response sent
     * Append last response sent to context.
     *
     * Now verify the incoming request against this.
     *
     */

    /* Update from restored */
    if r, err := restoreResultAny(seq, -2); err == nil {
        if req, ok := r.(*ActionRequestData); !ok {
            return nil, LogError("Restored data type (%T) != *ActionRequestData", r)
        } else if rs, err := restoreResultAny(seq, -1); err != nil {
            /* Restore last response */
            return nil, LogError("Failed to restore last res %v", err)
        } else if res, ok := rs.(*ActionResponseData); !ok {
            return nil, LogError("Restored data type (%T) != *ActionResponseData", rs)
        } else {
            ret := &ActionRequestData{
                Action:            exp.Action,
                Timeout:           exp.Timeout,
                AnomalyInstanceId: req.AnomalyInstanceId,
                AnomalyKey:        res.AnomalyKey,
                Context:           make([]*ActionResponseData, len(req.Context)+1),
            }
            for i, v := range req.Context {
                ret.Context[i] = v
            }
            ret.Context[len(req.Context)] = res
            return ret, nil
        }
    } else {
        /* possible if first in sequence */
        ret := &ActionRequestData{
            Action:  exp.Action,
            Timeout: exp.Timeout,
        }
        return ret, nil
    }
}

/* Helper API */
func buildRes(exp *ActionResponseData, seq int) (*ActionResponseData, error) {
    /*
     * Test code data can at most carry action name & timeout
     * as rest are dynamic and set by engine.
     * But if you are not first request and has a reference to last
     * you could get anomaly instance id & key. Plus context if any.
     *
     * But to get full set of context, we need last response sent
     * Append last response sent to context.
     *
     * Now verify the incoming request against this.
     *
     */

    if r, err := restoreResultAny(seq, -1); err != nil {
        return nil, LogError("Require last req to coin response (%v)", err)
    } else if req, ok := r.(*ActionRequestData); !ok {
        return nil, LogError("Restored data type (%T) != *ActionRequestData", r)
    } else {
        key := exp.AnomalyKey
        if len(key) == 0 {
            key = req.AnomalyKey
        }
        ret := &ActionResponseData{
            Action:            exp.Action,
            InstanceId:        req.InstanceId,
            AnomalyInstanceId: req.AnomalyInstanceId,
            AnomalyKey:        key,
            Response:          exp.Response,
            ResultCode:        exp.ResultCode,
            ResultStr:         exp.ResultStr,
        }
        return ret, nil
    }
}

/*
 * Implementation for clientAPIID - RECV_REQ
 *
 * Invoke tx.RecvServerRequest and verify against expected from test entry.
 * Saves rcvd data in local sequence which could be needed for verifications.
 * For e.g. we need to know anomaly & instance-id to verify/construct future
 * responses/requests.
 *
 * Refer test collection for sample usage.
 */
func (p *callArgs) call_receive_req(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_receive_req"
    defer func() {
        chTestHeartbeat <- "End: call_receive_req"
    }()

    tx := p.getTx(te.clTx)
    rcv, err := tx.RecvServerRequest()
    if te.failed != (err != nil) {
        p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
            ti, te.toStr(), err)
    }
    if err == nil {
        if len(te.result) != 1 {
            p.t.Fatalf("test index %v: Expect only one result len(%d)", ti, len(te.args))
        } else if rcv.ReqType != TypeServerRequestAction {
            p.t.Fatalf("Test index %v: Mismatch ReqType rcv(%v) != exp(%v)", ti,
                rcv.ReqType, TypeServerRequestAction)
        } else if rcvd, ok := rcv.ReqData.(ActionRequestData); !ok {
            p.t.Fatalf("Test index %v: reqData type (%T) != ActionRequestData",
                ti, rcv.ReqData)
        } else if exp, ok := te.result[0].(*ActionRequestData); !ok {
            p.t.Fatalf("Test index %v: Test error result (%T) != *ActionRequestData",
                ti, te.result[0])
        } else if expUpd, err := buildReq(exp, te.seqId); err != nil {
            p.t.Fatalf("Test index %v: buildReq failed (%v)", ti, err)
        } else if res := compActReqData(&rcvd, expUpd); len(res) > 0 {
            p.t.Fatalf("Test index %v: Data mismatch (%s) (%v)", ti, res, rcvd)
        } else {
            saveResultAny(te.seqId, &rcvd)
        }
    }
}

/*
 * Internal helper API to verify engine publishing against expected.
 * Receives the engine publishing via local channel through mock publish fn.
 * Wait till LomAction & verify.
 * No worries about wait forever as test will be terminated if it takes
 * too long.
 */
func verifyPublish(exp *ActionResponseData, complete bool) error {
    pubRes := pubAction_t{}
    s := ""

    for {
        /* It is OK to block. If no data for 5 seconds, test will terminate */
        s = <-publishCh

        if err := json.Unmarshal([]byte(s), &pubRes); err != nil {
            return LogError("Unmarshal failed (%s)", s)
        }
        if pubRes.LoM_Action != nil {
            /* action published */
            break
        }
        /* Likely HB; Wait till action */
    }

    if *pubRes.LoM_Action != *exp {
        return LogError("published(%v) != exp (%v)", *pubRes.LoM_Action, exp)
    }
    if exp.InstanceId == exp.AnomalyInstanceId {
        var m map[string]any

        json.Unmarshal([]byte(s), &m)
        if st, ok := m["State"]; !ok {
            return LogError("Failed to find state (%v)", m)
        } else if s, ok := st.(string); !ok {
            return LogError("state val not string (%v)", m)
        } else if !complete && (s != "init") {
            return LogError("state val != init (%v)", m)
        } else if complete && (s != "complete") {
            return LogError("state val != complete (%v)", m)
        }
    }
    return nil
}

/*
 * Implementation for clientAPIID - SEND_RES
 *
 * Invoke tx.SendServerResponse. Save expected result in cache and also
 * verify that engine publish o/p matches.
 *
 * Refer test collection for sample usage.
 */
func (p *callArgs) call_send_res(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_send_res"
    defer func() {
        chTestHeartbeat <- "End: call_send_res"
    }()

    if len(te.args) != 1 {
        p.t.Fatalf("test index %v: Expect only one result len(%d)", ti, len(te.args))
    } else if exp, ok := te.args[0].(*ActionResponseData); !ok {
        p.t.Fatalf("Test index %v: Test error args (%T) != *ActionResponseData",
            ti, te.args[0])
    } else if expUpd, err := buildRes(exp, te.seqId); err != nil {
        p.t.Fatalf("Test index %v: Test error (%v)", ti, err)
    } else {
        res := &MsgSendServerResponse{TypeServerRequestAction, expUpd}
        if te.failed {
            res.ReqType = TypeServerRequestCount /* To induce failure */
        }

        tx := p.getTx(te.clTx)
        err := tx.SendServerResponse(res)
        if te.failed != (err != nil) {
            p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
                ti, te.toStr(), err)
        } else if err == nil {
            saveResultAny(te.seqId, expUpd)

            if err = verifyPublish(expUpd, false); err != nil {
                p.t.Fatalf("Test index %v: verifyPublish failed (%v)", ti, err)
            }
        }
    }
}

/*
 * Implementation for clientAPIID - CHK_REG_ACTIONS
 *
 * Internal engine state verifications for expected active actions.
 * Refer test collection for sample usage.
 */
func (p *callArgs) call_verify_registrations(ti int, te *testEntry_t) {
    reg := GetRegistrations()

    if len(te.args) != 1 {
        p.t.Fatalf("test index %v: Expect 2 args. len(%d)", ti, len(te.args))
    }
    index, ok := te.args[0].(int)
    if !ok {
        p.t.Fatalf("%d: args is not type int (%T)", ti, te.args[0])
    }
    expReg := expRegistrations[index]
    expAct := expActiveActions[index]

    if len(expReg) != len(reg.activeClients) {
        p.t.Fatalf("%d: len mismatch. expReg(%d) active(%d)", ti, len(expReg), len(reg.activeClients))
    }
    for k, v := range expReg {
        info, ok := reg.activeClients[k]
        if !ok {
            p.t.Fatalf("%d: Missing client (%s) in active list", ti, k)
        }
        if len(v) != len(info.Actions) {
            p.t.Fatalf("%d: len mismatch for client(%s) exp(%d) active(%d)", ti,
                k, len(v), len(info.Actions))
        }
        for _, a := range v {
            if _, ok1 := info.Actions[a]; !ok1 {
                p.t.Fatalf("%d: Missing action. client(%s) exp(%v) active(%v)",
                    ti, k, v, info.Actions)
            }
        }
    }
    if len(expAct) != len(reg.activeActions) {
        p.t.Fatalf("%d: len mismatch. exp(%d) active(%d)", ti,
            len(expAct), len(reg.activeActions))
    }

    for k, v := range expAct {
        if v1, ok := reg.activeActions[k]; !ok {
            p.t.Fatalf("%d: Missing active action (%s)", ti, k)
        } else if v != *v1 {
            p.t.Fatalf("%d: Value mismatch (%v) != (%v)", ti, v, *v1)
        }
    }
}

/*
 * Implementation for clientAPIID - NOTIFY_HB
 *
 * Call tx.NotifyHeartbeat for each in test entry.
 * As engine resp async, call testHeartbeat to verify
 */
func (p *callArgs) call_notify_hb(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_notify_hb"
    defer func() {
        chTestHeartbeat <- "End: call_notify_hb"
    }()
    tx := p.getTx(te.clTx)

    /* Call NotifyHeartbeat for each entry in args */
    for i, v := range te.args {
        if s, ok := v.(string); !ok {
            p.t.Fatalf("%d: Test error. arg (%T) not string", ti, v)
        } else {
            /* For now engine ignores tstamp */
            if err := tx.NotifyHeartbeat(s, 0); err != nil {
                p.t.Fatalf("Test index %v: Unexpected failure (%v)", ti, err)
                return
            }
        }
        chTestHeartbeat <- fmt.Sprintf("call_notify_hb ti=%d i=%d", ti, i)
    }

    /* Expect actions in published HB per result only */
    res := make([]string, len(te.result))
    for i, v := range te.result {
        if s, ok := v.(string); !ok {
            p.t.Fatalf("%d: Test error. arg (%T) not string", ti, v)
        } else {
            res[i] = s
        }
    }
    /* Engine publishes heartbeat asynchronously at its own HB loop */
    if err := testHeartbeat(res); err != nil {
        p.t.Fatalf("%d: testHeartbeat failed. (%v)", ti, err)
    }
}

/*
 * Implementation for clientAPIID - CHK_ACTIV_REQ
 *
 * Internal engine state verifications for expected active requests.
 * Refer test collection for sample usage.
 */
func (p *callArgs) call_verify_active_requests(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_verify_active_requests"
    defer func() {
        chTestHeartbeat <- "End: call_verify_active_requests"
    }()

    handler := GetSeqHandler()
    exp := make([]string, len(te.args))
    for i, v := range te.args {
        if s, ok := v.(string); !ok {
            p.t.Fatalf("%d: Test error. arg (%T) not string", ti, v)
        } else {
            exp[i] = s
        }
    }

    if len(exp) != len(handler.activeRequests) {
        p.t.Fatalf("%d: exp(%v) != active(%v)", ti, exp, handler.activeRequests)
    } else {
        for _, a := range exp {
            if _, ok := handler.activeRequests[a]; !ok {
                p.t.Fatalf("%d: active request missing for (%s)", ti, a)
                break
            }
        }
    }
}

/*
 * A helper just to introduce pause time where needed.
 * It literally sleeps with test heartbeats.
 * Look at test collection to see where & when pause is needed.
 */
func (p *callArgs) call_pause(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_pause"
    defer func() {
        chTestHeartbeat <- "End: call_pause"
    }()

    if len(te.args) != 1 {
        p.t.Fatalf("%d: Test error. Expect one arg (%v)", ti, te.args)
    } else if tout, ok := te.args[0].(int); !ok || tout <= 0 {
        p.t.Fatalf("%d: Test error. arg (%T) not int or 0 (%v)", ti, te.args[0], tout)
    } else {
        for tout > 0 {
            chTestHeartbeat <- "Run: call_pause: " + strconv.Itoa(tout)
            time.Sleep(time.Second)
            tout--
        }
    }
}

/* Simple helper to construct first action's final resp */
func margeRes(p *ActionResponseData, q *ActionResponseData) (*ActionResponseData, error) {
    p.ResultCode = q.ResultCode
    p.ResultStr = q.ResultStr
    return p, nil
}

/*
 * Implementation for clientAPIID - SEQ_COMPLETE
 *
 * first received result + test i/p == Last publish result.
 * Refer test collection for sample usage.
 */
/* Verify seq complete publish o/p */
func (p *callArgs) call_seq_complete(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_seq_complete"
    defer func() {
        chTestHeartbeat <- "End: call_seq_complete"
    }()

    rArgs := &ActionResponseData{}
    if len(te.args) > 0 {
        if rtmp, ok := te.args[0].(*ActionResponseData); !ok {
            p.t.Fatalf("%d: Test error. arg (%T) not *ActionResponseData", ti, te.args[0])
            return
        } else {
            rArgs = rtmp
        }
    }
    /*
     * All results from a seq are saved. Get first res, which is anomaly's
     * Merge it with expected res codes ... from test entry
     * verify the last publish which is anomaly re-published against this
     */
    if rs, err := restoreResultAny(te.seqId, 1); err != nil {
        /* Restore first response */
        p.t.Fatalf("%d: Failed to get first res (%v)", ti, err)
    } else if res, ok := rs.(*ActionResponseData); !ok {
        p.t.Fatalf("%d: Restored data type (%T) != *ActionResponseData", ti, rs)
    } else if resUpd, err := margeRes(res, rArgs); err != nil {
        p.t.Fatalf("%d: margeRes failed (%v)", ti, err)
    } else if err = verifyPublish(resUpd, true); err != nil {
        p.t.Fatalf("Test index %v: verifyPublish failed (%v)", ti, err)
    }
}

/*
 * All test runs are watched via chTestHeartbeat.
 * Every test is expected to keep this channel alive.
 * An idle channel for too long is watched and terminated
 */
func terminate(t *testing.T, tout int, chEnd chan int) {
    for {
        select {
        case m := <-chTestHeartbeat:
            LogDebug("Test HB: (%s)", m)

        case <-time.After(time.Duration(tout) * time.Second):
            LogPanic("Terminating test for no heartbeats for tout=%d", tout)
        case <-chEnd:
            return
        }
    }
}

/*
 * Engine collects all actions reported and include the actions
 * list in its next HB publish message. Wait till all actions are
 * received and possibly one more HB from engine.
 *
 * Local publishAPI which engine made to call for any publishing
 * send engine's publisings via publishCh. Listen to it until
 * all expected actions are reported or on error.
 *
 * No worries about blocked for ever in cases of bug in engine.
 * Test's terminate will abort.
 *
 * For details refer comments in caller function - testHeartbeat
 */
func testHeartbeatCh(m map[string]struct{}, ch chan string) {
    hb := HBData_t{}
    done := false
    for !done {
        /* It is OK to block. If no data for long, test will terminate */
        s := <-publishCh

        if err := json.Unmarshal([]byte(s), &hb); err != nil {
            ch <- "Unmarshal failed: " + s
            return
        }
        if hb.LoM_Heartbeat.Timestamp != 0 {
            /* This is HB */
            done = len(m) == 0 /* Do one check after m is empty */
            p := &hb.LoM_Heartbeat
            for _, v := range p.Actions {
                if _, ok := m[v]; !ok {
                    ch <- "Unexpected action present: " + v
                    return
                } else {
                    /* remove reported */
                    delete(m, v)
                }
            }
            LogDebug("Processed HB: (%s)", s)
        } else {
            LogDebug("Skipped non HB: (%s)", s)
            /* Likely notHB; Wait till action */
        }
    }
    ch <- ""
}

/*
 * The test calls couple of NotifyHeartbeat.
 *
 * Engine collects them and send the collected list in its own HB.
 * The list is transient and gets cleared upon engine sending HB.
 * So the test sent notify for multiple actions could come in one / two
 * engine heartbeats.
 *
 * Kick off testHeartbeatCh to receive engine HBs and remove from expected
 * list. Return when list become empty or on error.
 *
 * This routine waits for that to return within HB_WAIT period.
 * Till HB_WAIT period, it keeps chTestHeartbeat (channel to watch for
 * test timeout) happy. After HB_WAIT it does not update chTestHeartbeat
 * allowing terminate to abort the testing.
 */
const HB_WAIT = 10

func testHeartbeat(actions []string) error {
    ch := make(chan string)
    cnt := HB_WAIT

    /* Collect expected actions as key in map to help locate easy */
    m := map[string]struct{}{}
    for _, v := range actions {
        m[v] = struct{}{}
    }

    /* Run until all expected actions reported or on error */
    go testHeartbeatCh(m, ch)

    /* Return on error or completion. Stop sending test HB after HB_WAIT */
    for {
        select {
        case err := <-ch:
            if len(err) != 0 {
                return LogError(err)
            } else {
                return nil
            }

        case <-time.After(1 * time.Second):
            if cnt > 0 {
                cnt--
                chTestHeartbeat <- fmt.Sprintf("Waiting for HB %d/%d", cnt, HB_WAIT)
            } else {
                LogError("testHeartbeat timed after %d seconds", HB_WAIT)
                /* Don't send HB and let test terminate with no test heartbeats. */
            }
        }
    }
}

/*
 * Run individual test entry
 * Use the callArgs to use the right transport.
 * Call appropriate API per type, which is merely a wrapper for clientTransport
 * API. It takes data from test entry and pass to the corresponding client API as args.
 */
func runTestEntries(cArgs *callArgs, collPath string, lst testEntriesList_t) {

    ordered := make([]int, len(lst))
    {
        i := 0
        for t_i, _ := range lst {
            ordered[i] = t_i
            i++
        }
        sort.Ints(ordered)
    }

    for _, t_i := range ordered {
        t_e := lst[t_i]

        LogDebug("---------------- coll: %v tid: %v START (%s) ----------", collPath, t_i, t_e.desc)
        switch t_e.id {
        case REG_CLIENT:
            cArgs.call_register_client(t_i, &t_e)
        case REG_ACTION:
            cArgs.call_register_action(t_i, &t_e)
        case DEREG_ACTION:
            cArgs.call_deregister_action(t_i, &t_e)
        case DEREG_CLIENT:
            cArgs.call_deregister_client(t_i, &t_e)
        case CHK_REG_ACTIONS:
            cArgs.call_verify_registrations(t_i, &t_e)
        case RECV_REQ:
            cArgs.call_receive_req(t_i, &t_e)
        case SEND_RES:
            cArgs.call_send_res(t_i, &t_e)
        case SEQ_COMPLETE:
            cArgs.call_seq_complete(t_i, &t_e)
        case NOTIFY_HB:
            cArgs.call_notify_hb(t_i, &t_e)
        case CHK_ACTIV_REQ:
            cArgs.call_verify_active_requests(t_i, &t_e)
        case PAUSE:
            cArgs.call_pause(t_i, &t_e)
        default:
            cArgs.t.Fatalf("Unhandled API ID (%v)", t_e.id)
        }
        LogDebug("---------------- coll: %v tid: %v  END  (%s) ----------", collPath, t_i, t_e.desc)
    }
}

/*
 * Run a single collection, in the order pre, <this coll> & post
 * Pre & Post are by themselves list of collections, which can be empty
 */
func runColl(cArgs *callArgs, collPath string, te *testCollectionEntry_t) {
    LogDebug("**************** coll: %s START (%s) **********", collPath, te.desc)
    for _, pre := range te.preSetup {
        runColl(cArgs, collPath+"/"+string(pre), testCollections[pre])
    }
    LogDebug("**************** coll: %s  Run  (%s) **********", collPath, te.desc)
    runTestEntries(cArgs, collPath, te.testEntries)
    for _, post := range te.postCleanup {
        runColl(cArgs, collPath+"/"+string(post), testCollections[post])
    }
    LogDebug("**************** coll: %s  END  (%s) **********", collPath, te.desc)
}

func testRPCListener(t *testing.T) {
    /*
     * LibTest extensively test JSON-RPC APIs.
     * The RPC APIs use SendToServer for redirecting requests to server as
     * done in HTTP APIs
     *
     * All the tests above are HTTP based and extensively test engine via
     * SendToServer.
     *
     * All we need to test is to ensure that engine indeed started RPC listener
     * or not. So a sample register would do.
     */

    servAddr := "localhost:" + strconv.Itoa(RPC_JSON_PORT)
    rpcResp := map[string]any{}
    resp := LoMResponse{}
    received := make([]byte, 1024)

    type RPCReq struct {
        Id     int
        Method string
        Params []string
    }
    req := &LoMRequest{TypeRegClient, "test", 0, struct{}{}}
    if out, err := json.Marshal(req); err != nil {
        t.Fatalf("Failed to marshal LoMRequest (%v)", err)
    } else if out, err := json.Marshal(RPCReq{1, "LoMTransport.LoMRPCRequest", []string{string(out)}}); err != nil {
        t.Fatalf("Failed to marshal LoMRequest (%v)", err)
    } else if tcpServer, err := net.ResolveTCPAddr("tcp", servAddr); err != nil {
        t.Fatalf("Failed to resolve (%s) (%v)", servAddr, err)
    } else if conn, err := net.DialTCP("tcp", nil, tcpServer); err != nil {
        t.Fatalf("Dial failed: (%v)", err)
    } else if _, err := conn.Write([]byte(string(out))); err != nil {
        t.Fatalf("Failed to write req")
    } else if n, err := conn.Read(received); err != nil {
        t.Fatalf("Failed to read reply")
    } else if err := json.Unmarshal(received[:n], &rpcResp); err != nil {
        t.Fatalf("Failed to unmarshal reply (%s) (%v)", string(received[:n]), err)
    } else if strRes, ok := rpcResp["result"].(string); !ok {
        t.Fatalf("Failed to get result as string result (%T)/(%s)", rpcResp["result"], rpcResp["result"])
    } else if err := json.Unmarshal([]byte(strRes), &resp); err != nil {
        t.Fatalf("Failed to unmarshal reply (%s)", strRes)
    } else if resp.ResultCode != 0 {
        t.Fatalf("register Client failed (%v)", strRes)
    }

    LogDebug("testRPCListener COMPLETE")
}

/* Creates the conf files as per data in this code */
var initConfigDone = false

func initConfig(t *testing.T) {
    if !initConfigDone {
        createFile(t, "globals.conf.json", globals_conf)
        createFile(t, "actions.conf.json", actions_conf)
        createFile(t, "procs.conf.json", procs_conf)
        createFile(t, "bindings.conf.json", bindings_conf)

        initConfigDone = true
    }
}

/*
 * Executes all test collections
 *
 * initServer starts the engine and terminated only the end.
 * engine starts listening for client connections.
 *
 * For each collection, a new client transport is created via CallArgs
 *
 * The terminate helps abort the entire test run when any test blocks for too long.
 */
func TestRun(t *testing.T) {
    chEnd := make(chan int, 1)
    go terminate(t, 5, chEnd)

    initConfig(t)
    engine := initServer(t)

    /* Init local list for test data */
    initActive()

    SetPublishAPI(testPublish)

    for _, collId := range testRunList {
        /* Create new transports for a collection */
        cArgs := &callArgs{t: t, lstTx: make(map[string]*ClientTx)}
        runColl(cArgs, string(collId), testCollections[collId])
        resetResultAll() /* Reset all saved results */
    }

    testRPCListener(t)
    chEnd <- 0
    engine.close() /* Close the engine */
}

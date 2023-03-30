package engine

/*
 *  Mock PublishEventAPI 
 *  This test code combines unit test & functional test - Two in one shot
 *
 *  Scenarios:
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
    . "lib/lomcommon"
    . "lib/lomipc"
    "os"
    "path/filepath"
    "sort"
    "testing"
    "time"
)

const EMPTY_STR= ""
const CLIENT_0 = "client-0"
const CLIENT_1 = "client-1"
const CLIENT_2 = "client-2"

/*
 * During test run, test code keep this chan active. An idle channel for timeout
 * seconds will abort the test
 */
var chTestHeartbeat = make(chan string)

/*
 *  Actions.conf
 */
 var actions_conf = `{ "actions": [
        { "name": "Detect-0" },
        { "name": "Safety-chk-0", "Timeout": 1},
        { "name": "Mitigate-0", "Timeout": 6},
        { "name": "Detect-1" },
        { "name": "Safety-chk-1", "Timeout": 7},
        { "name": "Mitigate-1", "Timeout": 8},
        { "name": "Detect-2" },
        { "name": "Safety-chk-2", "Timeout": 1},
        { "name": "Mitigate-2", "Timeout": 6},
        { "name": "Disabled-0", "Disable": true}
        ] }`


var bindings_conf = `{ "bindings": [
    {
        "name": "bind-0", 
        "priority": 0,
        "Timeout": 2,
        "actions": [
            {"name": "Detect-0" },
            {"name": "Safety-chk-0", "sequence": 1 },
            {"name": "Mitigate-0", "sequence": 2 }
        ]
    },
    {
        "name": "bind-1", 
        "priority": 1,
        "Timeout": 19,
        "actions": [
            {"name": "Detect-1" },
            {"name": "Safety-chk-1", "sequence": 1 },
            {"name": "Mitigate-1", "sequence": 2 }
        ]
    },
    {
        "name": "bind-2", 
        "priority": 0,
        "Timeout": 1,
        "actions": [
            {"name": "Detect-2" },
            {"name": "Safety-chk-0", "sequence": 1 },
            {"name": "Safety-chk-2", "sequence": 2 },
            {"name": "Mitigate-2", "sequence": 3 }
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
)

type testEntry_t struct {
    id          clientAPIID
    clTx        string          /* Which Tx to use*/
    seqId       int             /* The context to use for save/restore results per seq */
    args        []any
    result      []any
    failed      bool            /* True if expected to fail. */
    desc        string
}

func (p *testEntry_t) toStr() string {
    s := ""
    switch p.id {
    case REG_CLIENT:
        s = "REG_CLIENT"
    case REG_ACTION:
        s = "REG_ACTION"
    case DEREG_CLIENT:
        s = "DEREG_CLIENT"
    case DEREG_ACTION:
        s = "DEREG_ACTION"
    case RECV_REQ:
        s = "RECV_REQ"
    case SEND_RES:
        s = "SEND_RES"
    case SHUTDOWN:
        s = "SHUTDOWN"
    case NOTIFY_HB:
        s = "NOTIFY_HB"
    case CHK_ACTIV_REQ:
        s = "CHK_ACTIV_REQ"
    case CHK_REG_ACTIONS:
        s = "CHK_REG_ACTIONS"
    default:
        s = fmt.Sprintf("UNK(%d)", p.id)
    }
    return fmt.Sprintf("%s:%s: args:(%v) res(%v) failed(%v)",
            p.clTx, s, p.args, p.result, p.failed)
}


type registrations_t map[string][]string

/* Test scenario expectations */
var expRegistrations = []registrations_t {
    {    /* Map of client vs actions */
        CLIENT_0: []string { "Detect-0", "Safety-chk-0", "Mitigate-0", "Mitigate-2" },
        CLIENT_1: []string { "Detect-1", "Safety-chk-1", "Mitigate-1", "Detect-2", "Safety-chk-2" },
    },
    {    /* Map of client vs actions */
        CLIENT_0: []string { "Detect-0", "Safety-chk-0" },
        CLIENT_1: []string { "Detect-1", "Safety-chk-1", "Mitigate-1" },
    },
    {    /* Map of client vs actions */
        CLIENT_1: []string { "Detect-1", "Safety-chk-1", "Mitigate-1" },
    },
    {    /* Map of client vs actions */
},
}

type activeActionsList_t map[string]ActiveActionInfo_t
var expActiveActions = make([]activeActionsList_t, len(expRegistrations))

func initActive() {
    if  len(expActiveActions[0]) > 0 {
        return
    }

    cfg := GetConfigMgr()

    for i, rl := range expRegistrations {
        expActiveActions[i] = make(activeActionsList_t)
        lst := expActiveActions[i]
        for cl, v := range rl {
            for _, a := range v {
                if _, ok := lst[a]; ok {
                    LogPanic("Duplicate action in expRegistrations[%d] cl(%s) a(%s)", i, cl, a)
                }
                if c, e := cfg.GetActionConfig(a); e != nil {
                    LogPanic("Failed to get action config for (%s)", a)
                } else {
                    lst[a] = ActiveActionInfo_t {
                        Action: a, Client: cl, Timeout: c.Timeout, }
                }
            }
        }
    }
}

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

type testEntriesList_t  map[int]testEntry_t

var xtestEntriesList = testEntriesList_t {
    0: {
        id: REG_ACTION,
        clTx: "",
        args: []any{"xyz"},
        failed: true,
        desc: "RegisterAction: Fail as before register client",
    },
    1: {
        id: REG_CLIENT,
        clTx: "iX",
        args: []any{EMPTY_STR},
        failed: true,
        desc: "RegisterClient: Fail for empty name",
    },
    2: {
        id: REG_CLIENT,
        clTx: "Bogus",
        args: []any{CLIENT_0},
        failed: false,
        desc: "RegisterClient to succeed",
    },
    3: {
        id: REG_CLIENT,
        clTx: "Bogus",
        args: []any{CLIENT_0},
        failed: true,
        desc: "register-client: Fail duplicate on same transport",
    },
    4: {
        id: REG_CLIENT,
        clTx: CLIENT_0,             /* re-reg under new Tx. So succeed" */
        args: []any{CLIENT_0},
        failed: false,
        desc: "RegClient re-reg on new tx to succeed",
    },
    5: {
        id: REG_ACTION,
        clTx: CLIENT_0,
        args: []any{""},
        failed: true,
        desc: "RegisterAction fail for empty name",
    },
    6: {
        id: REG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Detect-0"},
        failed: false,
        desc: "RegisterAction client-0/detect-0 succeeds",
    },
    7: {
        id: REG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Detect-0"},
        failed: false,
        desc: "Re-registerAction succeeds",
    },
    8: {
        id: REG_CLIENT,
        clTx: CLIENT_1,
        args: []any{CLIENT_1},
        failed: false,
        desc: "second Client reg to succeed",
    },
    9: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Detect-0"},
        failed: false,
        desc: "RegAction: Succeed duplicate register on different client",
    },
    10: {
        id: REG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Detect-0"},
        failed: false,
        desc: "Duplicate action register on different client",
    },
    11: {
        id: REG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Mitigate-0"},
        failed: false,
        desc: "action register succeed",
    },
    12: {
        id: REG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Mitigate-2"},
        failed: false,
        desc: "action register succeed",
    },
    13: {
        id: REG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Safety-chk-0"},
        failed: false,
        desc: "action register succeed",
    },
    14: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Detect-1"},
        failed: false,
        desc: "action register succeed",
    },
    15: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Safety-chk-1"},
        failed: false,
        desc: "action register succeed",
    },
    16: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Mitigate-1"},
        failed: false,
        desc: "action register succeed",
    },
    17: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Detect-2"},
        failed: false,
        desc: "action register succeed",
    },
    18: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Safety-chk-2"},
        failed: false,
        desc: "action register succeed",
    },
    19: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Disabled-0"},
        failed: true,
        desc: "action register fail for disabled",
    },
    20: {
        id: CHK_REG_ACTIONS,
        clTx: "",               /* Local verification */
        args: []any{0},
        desc: "Verify local cache to succeed",
    },
    21: {
        id: DEREG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Detect-2"},
        failed: false,
        desc: "action deregister succeed",
    },
    22: {
        id: DEREG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Safety-chk-2"},
        failed: false,
        desc: "action deregister succeed",
    },
    23: {
        id: DEREG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Mitigate-0"},
        failed: false,
        desc: "action deregister succeed",
    },
    24: {
        id: DEREG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Mitigate-2"},
        failed: false,
        desc: "action deregister succeed",
    },
    25: {
        id: DEREG_ACTION,
        clTx: CLIENT_0,
        args: []any{""},
        desc: "action deregister succeed for empty",
    },
    26: {
        id: DEREG_ACTION,
        clTx: CLIENT_0,
        args: []any{"XXX"},
        desc: "action deregister succeed for non-existing",
    },
    27: {
        id: CHK_REG_ACTIONS,
        clTx: "",               /* Local verification */
        args: []any{1},
        desc: "Verify local cache to succeed",
    },
    28: {
        id: DEREG_CLIENT,
        clTx: CLIENT_0,
        desc: "action deregister client succeed",
    },
    29: {
        id: CHK_REG_ACTIONS,
        clTx: "",               /* Local verification */
        args: []any{2},
        desc: "Verify local cache to succeed",
    },
    30: {
        id: DEREG_CLIENT,
        clTx: CLIENT_1,
        desc: "action deregister client succeed",
    },
    31: {
        id: CHK_REG_ACTIONS,
        clTx: "",               /* Local verification */
        args: []any{3},
        desc: "Verify local cache to succeed",
    },
}

var testEntriesList = testEntriesList_t {
    100: {
        id: REG_CLIENT,
        clTx: CLIENT_0,             /* re-reg under new Tx. So succeed" */
        args: []any{CLIENT_0},
        failed: false,
        desc: "RegClient to succeed",
    },
    102: {
        id: REG_CLIENT,
        clTx: CLIENT_1,
        args: []any{CLIENT_1},
        failed: false,
        desc: "second Client reg to succeed",
    },
    104: {
        id: REG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Detect-0"},
        failed: false,
        desc: "Reg Action to succeed",
    },
    106: {
        id: REG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Mitigate-0"},
        failed: false,
        desc: "action register succeed",
    },
    108: {
        id: REG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Mitigate-2"},
        failed: false,
        desc: "action register succeed",
    },
    110: {
        id: REG_ACTION,
        clTx: CLIENT_0,
        args: []any{"Safety-chk-0"},
        failed: false,
        desc: "action register succeed",
    },
    114: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Detect-1"},
        failed: false,
        desc: "action register succeed",
    },
    116: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Safety-chk-1"},
        failed: false,
        desc: "action register succeed",
    },
    118: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Mitigate-1"},
        failed: false,
        desc: "action register succeed",
    },
    120: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Detect-2"},
        failed: false,
        desc: "action register succeed",
    },
    122: {
        id: REG_ACTION,
        clTx: CLIENT_1,
        args: []any{"Safety-chk-2"},
        failed: false,
        desc: "action register succeed",
    },
    124: {
        id: CHK_REG_ACTIONS,
        clTx: "",               /* Local verification */
        args: []any{0},
        desc: "Verify local cache to succeed",
    },
    /* Requests are expected in the same order as registration */
    140: {
        id: RECV_REQ,
        clTx: CLIENT_0,
        seqId: 1,               /* Use non-zero, default is 0. Make it explicit */
        result: []any { &ActionRequestData { Action: "Detect-0"} },
        desc: "Read server request for Detect-0",
    },
    142: {
        id: RECV_REQ,
        clTx: CLIENT_1,
        seqId: 2,
        result: []any { &ActionRequestData { Action: "Detect-1"} },
        desc: "Read server request for Detect-1",
    },
    144: {
        id: RECV_REQ,
        clTx: CLIENT_1,
        seqId: 3,
        result: []any { &ActionRequestData { Action: "Detect-2"} },
        desc: "Read server request for Detect-2",
    },
    150: {
        id: SEND_RES,
        clTx: CLIENT_0,
        seqId: 1,
        args: []any {&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-0", Response: "Detect-0 detected",}},
        desc: "Send res for detect0",
    },
    152: {
        id: RECV_REQ,
        clTx: CLIENT_0,
        seqId: 1,
        result: []any { &ActionRequestData { Action: "Safety-chk-0", Timeout: 1} },
        desc: "Read server request for Safety-check-0",
    },
    154: {
        id: SEND_RES,
        clTx: CLIENT_0,
        seqId: 1,
        args: []any {&ActionResponseData{Action: "Safety-chk-0", Response: "Safety-chk-0 passed",}},
        desc: "Send res for safety-chk-0",
    },
    156: {
        id: RECV_REQ,
        clTx: CLIENT_0,
        seqId: 1,
        result: []any { &ActionRequestData { Action: "Mitigate-0", Timeout: -1} },
        desc: "Read server request for Safety-check-0",
    },
    158: {
        id: SEND_RES,
        clTx: CLIENT_0,
        seqId: 1,
        args: []any {&ActionResponseData{Action: "Mitigate-0", Response: "Mitigate-0 passed",}},
        desc: "Send res for Mitigate-0",
    },
    160: {
        id: SEQ_COMPLETE,
        seqId: 1,
        desc: "Verify seq complete",
    },
}

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
    chTestHeartbeat <- "createFile: " + name
}

func initServer(t *testing.T) chan int {
    chTestHeartbeat <- "Start: initServer"
    defer func() {
        chTestHeartbeat <- "End: initServer"
    }()

    ch := make(chan int, 2)     /* Two to take start & end of loop w/o blocking */
    
    startUp("test", []string { "-path", CFGPATH }, ch)
    chTestHeartbeat <- "Waiting: initServer"

    select {
    case <- ch:
        break

    case <- time.After(2 * time.Second):
        t.Fatalf("initServer failed")
    }
    return ch
}

type callArgs struct {
    t   *testing.T
    lstTx   map[string]*ClientTx
}


func (p *callArgs) getTx(cl string) *ClientTx {
    tx, ok := p.lstTx[cl];
    if !ok {
        tx = GetClientTx(0)
        if tx != nil {
            p.lstTx[cl] = tx
        } else {
            p.t.Fatalf("Failed to get client")
        }
    }
    return tx
}


func (p *callArgs) call_register_client(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_register_client"
    defer func() {
        chTestHeartbeat <- "End: call_register_client"
    }()

    if len(te.args) != 1 {
        p.t.Fatalf("Test index %v: Expect only one arg len(%d)", ti, len(te.args))
    }
    a := te.args[0]
    clName, ok := a.(string)
    if !ok {
        p.t.Fatalf("Test index %v: Expect string as arg for client name (%T)", ti, a)
    }
    tx := p.getTx(te.clTx)
    err := tx.RegisterClient(clName)
    if te.failed != (err != nil) {
        p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
                ti, te.toStr(), err)
    }
}

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

func compStr(msg, rcv, tst string) string {
    if (len(rcv) == 0) {
        return fmt.Sprintf("%s empty", msg)
    }
    if (len(tst) != 0) && (tst != rcv) {
        return fmt.Sprintf("%s mismatch (%s) != (%s)", msg, rcv, tst)
    }
    return ""
}

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
                Action: exp.Action,
                Timeout:exp.Timeout,
                AnomalyInstanceId: req.AnomalyInstanceId,
                AnomalyKey: res.AnomalyKey,
                Context: make([]*ActionResponseData, len(req.Context) + 1),
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
            Action: exp.Action,
            Timeout:exp.Timeout,
        }
        return ret, nil
    }
}


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
            Action: exp.Action,
            InstanceId: req.InstanceId,
            AnomalyInstanceId: req.AnomalyInstanceId,
            AnomalyKey: key,
            Response: exp.Response,
            ResultCode: exp.ResultCode,
            ResultStr: exp.ResultStr,
        }
        return ret, nil
    }
}

func (p *callArgs) call_receive_req(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_receive_req"
    defer func() {
        chTestHeartbeat <- "End: call_receive_req"
    }()

    if len(te.result) != 1 {
        p.t.Fatalf("test index %v: Expect only one result len(%d)", ti, len(te.args))
    }
    tx := p.getTx(te.clTx)
    rcv, err := tx.RecvServerRequest()
    if te.failed != (err != nil) {
        p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
                ti, te.toStr(), err)
    }
    if (err == nil) {
        if rcv.ReqType != TypeServerRequestAction {
            p.t.Fatalf("Test index %v: Mismatch ReqType rcv(%v) != exp(%v)", ti,
                    rcv.ReqType, TypeServerRequestAction)
        }
        if rcvd, ok := rcv.ReqData.(ActionRequestData); !ok {
            p.t.Fatalf("Test index %v: reqData type (%T) != ActionRequestData",
                    ti, rcv.ReqData)
        } else if exp, ok:= te.result[0].(*ActionRequestData); !ok {
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


func verifyPublish(exp *ActionResponseData, complete bool) error {
    pubRes := pubAction_t{}
    s := ""

    for {
        /* It is OK to block. If no data for 5 seconds, test will terminate */
        s = <- publishCh

        if err := json.Unmarshal([]byte(s), &pubRes); err != nil {
            return LogError("Unmarshal failed (%s)", s)
        }
        if len(pubRes.LoM_Action.Action) != 0 {
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

func (p *callArgs) call_send_res(ti int, te *testEntry_t) {
    chTestHeartbeat <- "Start: call_send_res"
    defer func() {
        chTestHeartbeat <- "End: call_send_res"
    }()

    if len(te.args) != 1 {
        p.t.Fatalf("test index %v: Expect only one result len(%d)", ti, len(te.args))
    } else if exp, ok:= te.args[0].(*ActionResponseData); !ok {
        p.t.Fatalf("Test index %v: Test error args (%T) != *ActionResponseData",
                ti, te.args[0])
    } else if expUpd, err := buildRes(exp, te.seqId); err != nil {
        p.t.Fatalf("Test index %v: Test error (%v)", ti, err)
    } else {
        res := &MsgSendServerResponse { TypeServerRequestAction, expUpd }
        if te.failed {
            res.ReqType = TypeServerRequestCount /* To induce failure */
        }

        tx := p.getTx(te.clTx)
        err := tx.SendServerResponse(res)
        if te.failed != (err != nil) {
            p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
                    ti, te.toStr(), err)
        } else if (err == nil) {
            saveResultAny(te.seqId, expUpd)
                
            if err = verifyPublish(expUpd, false); err != nil {
                p.t.Fatalf("Test index %v: verifyPublish failed (%v)", ti, err)
            }
        }
    }
}


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
            

func (p *callArgs) call_seq_complete(ti int, te *testEntry_t) {
    if rs, err := restoreResultAny(te.seqId, 1); err != nil {
        /* Restore first response */
        p.t.Fatalf("%d: Failed to get first res (%v)", ti, err)
    } else if res, ok := rs.(*ActionResponseData); !ok {
        p.t.Fatalf("%d: Restored data type (%T) != *ActionResponseData", ti, rs)
    } else if err = verifyPublish(res, true); err != nil {
        p.t.Fatalf("Test index %v: verifyPublish failed (%v)", ti, err)
    }
    resetResultAny(te.seqId)
    LogDebug("saved: (%s)", printResultAny(false))
}

func terminate(t *testing.T, tout int) {
    for {
        select {
        case m := <- chTestHeartbeat:
            LogDebug("Test HB: (%s)", m)

        case <- time.After(time.Duration(tout) * time.Second):
            LogPanic("Terminating test for no heartbeats for tout=%d", tout)
        }
    }
}

    
func TestRun(t *testing.T) {
    go terminate(t, 5)

    createFile(t, "globals.conf.json", "")
    createFile(t, "actions.conf.json", actions_conf)
    createFile(t, "bindings.conf.json", bindings_conf)

    ch := initServer(t)

    cArgs := &callArgs{t: t, lstTx: make(map[string]*ClientTx) }
    ordered := make([]int, len(testEntriesList))
    {
        i := 0
        for t_i, _ := range testEntriesList {
            ordered[i] = t_i
            i++
        }
        sort.Ints(ordered)
    }

    /* Init local list for test data */
    initActive()

    SetPublishAPI(testPublish)

    for _, t_i := range ordered {
        t_e := testEntriesList[t_i]

        if len(ch) > 0 {
            t.Fatalf("Server loop exited")
        }
        LogDebug ("---------------- tid: %v START (%s) ----------", t_i, t_e.desc)
        switch (t_e.id) {
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
        default:
            t.Fatalf("Unhandled API ID (%v)", t_e.id)
        }
        LogDebug ("---------------- tid: %v  END  (%s) ----------", t_i, t_e.desc)
    }
}


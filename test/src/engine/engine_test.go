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
 *          1.  Expect requests for "Detect-0", "Detect-1" & "Detect-2"
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
    SHUTDOWN
    NOTIFY_HB
    CHK_ACTIV_REQ
    CHK_REG_ACTIONS
)

type testEntry_t struct {
    id          clientAPIID
    clTx        string          /* Which Tx to use*/
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
var expRegistrations = registrations_t {    /* Map of client vs actions */
    CLIENT_0: []string { "Detect-0", "Safety-chk-0", "Mitigate-0", "Mitigate-2" },
    CLIENT_1: []string { "Detect-1", "Safety-chk-1", "Mitigate-1", "Detect-2", "Safety-chk-2" },
}
type activeActionsList_t map[string]ActiveActionInfo_t
var expActiveActions = make(activeActionsList_t)

func initActive() {
    if  len(expActiveActions) > 0 {
        return
    }

    cfg := GetConfigMgr()

    for cl, v := range expRegistrations {
        for _, a := range v {
            if _, ok := expActiveActions[a]; ok {
                LogPanic("Duplicate action in expRegistrations cl(%s) a(%s)", cl, a)
            }
            if c, e := cfg.GetActionConfig(a); e != nil {
                LogPanic("Failed to get action config for (%s)", a)
            } else {
                expActiveActions[a] = ActiveActionInfo_t {
                    Action: a, Client: cl, Timeout: c.Timeout, }
            }
        }
    }
}

type testEntriesList_t  map[int]testEntry_t

var testEntriesList = testEntriesList_t {
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
        args: []any{expRegistrations},
        desc: "Verify local cache to succeed",
    },
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
        p.t.Fatalf("Expect only one arg len(%d)", len(te.args))
    }
    a := te.args[0]
    clName, ok := a.(string)
    if !ok {
        p.t.Fatalf("Expect string as arg for client name (%T)", a)
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
        p.t.Fatalf("Expect only one arg len(%d)", len(te.args))
    }
    a := te.args[0]
    actName, ok := a.(string)
    if !ok {
        p.t.Fatalf("Expect string as arg for action name (%T)", a)
    }
    tx := p.getTx(te.clTx)
    err := tx.RegisterAction(actName)
    if te.failed != (err != nil) {
        p.t.Fatalf("Test index %v: Unexpected behavior. te(%v) err(%v)",
                ti, te.toStr(), err)
    }
}

func (p *callArgs) call_verify_registrations(ti int, te *testEntry_t) {
    initActive()
    reg := GetRegistrations()

    exp, ok := te.args[0].(registrations_t)
    if !ok {
        p.t.Fatalf("%d: args is not type registrations_t (%T)", ti, te.args)
    }
    if len(exp) != len(reg.activeClients) {
        p.t.Fatalf("%d: len mismatch. exp(%d) active(%d)", ti, len(exp), len(reg.activeClients))
    }
    for k, v := range exp {
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
    if len(expActiveActions) != len(reg.activeActions) {
        p.t.Fatalf("%d: len mismatch. exp(%d) active(%d)", ti,
                len(expActiveActions), len(reg.activeActions))
    }

    for k, v := range expActiveActions {
        if v1, ok := reg.activeActions[k]; !ok {
            p.t.Fatalf("%d: Missing active action (%s)", ti, k)
        } else if v != *v1 {
            p.t.Fatalf("%d: Value mismatch (%v) != (%v)", ti, v, *v1)
        }
    }
}
            


func terminate(t *testing.T, tout int) {
    LogDebug("DROP: Terminate guard called tout=%d", tout)
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
        case CHK_REG_ACTIONS:
            cArgs.call_verify_registrations(t_i, &t_e)
        default:
            t.Fatalf("Unhandled API ID (%v)", t_e.id)
        }
        LogDebug ("---------------- tid: %v  END  (%s) ----------", t_i, t_e.desc)
    }
}

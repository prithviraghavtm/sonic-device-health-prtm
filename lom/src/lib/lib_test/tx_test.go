package lib_test

/*
 * Info
 *
 * To run test
 * localadmin@remanava-dev-1:~/source/fork/Device-Health/go-main/src/lib$ clear; ./build.sh t
 *  Use ./build.sh v for verbose.
 *
 * To create HTML page
 * localadmin@remanava-dev-1:~/source/fork/Device-Health/go-main/src/lib$ GOPATH=$(pwd) go tool cover -html=coverprofile.out -o /tmp/coverage.html
 *
 * Edge shows uncovered lines by Red color
 *
 * Current
 *    ok      txlib_test      1.017s  coverage: 98.5% of statements in lomipc, lomcommon
 *
 * ./build.sh v <-- to run tests
 */

import (
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log/syslog"
    . "lom/src/lib/lomcommon"
    . "lom/src/lib/lomipc"
    "net/rpc"
    "os"
    "path/filepath"
    "strconv"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
)

type TestClientData struct {
    ReqType  ReqDataType /* Req type to call */
    Args     []string    /* Args needed for the call */
    DataArgs interface{}
    Failed   bool        /* Expect to fail or succeed */
    ExpResp  interface{} /* Differs per request */
}

type TestServerData struct {
    Req LoMRequest
    Res LoMResponse /* LoMResponse to send back */
}

type TestData struct {
    TestClientData /* Simulated client raises call per this data */
    TestServerData /* TestMain code validates incoming req against */
    /* TestServerData:Req and send back TestServerData:Res */
}

type TestDataForJson struct {
    JsonReq        string /* Req to send as string */
    JsonRes        string /* Expected JSON res */
    TestServerData        /* TestMain code validates incoming req against */
    /* TestServerData:Req and send back TestServerData:Res */
}

const TEST_CL_NAME = "Foo"
const TEST_ACTION_NAME = "Detect-0"

var ActReqData = ServerRequestData{TypeServerRequestAction,
    ActionRequestData{"Bar", "inst_1", "an_inst_0", "an_key", 10,
        []*ActionResponseData{
            {TEST_ACTION_NAME, "an_inst_0", "an_inst_0", "an_key", "res_anomaly", 0, ""},
            {"Foo-safety", "inst_0", "an_inst_0", "an_key", "res_foo_check", 2, "some failure"},
        }}}

var ActResData = MsgSendServerResponse{TypeServerRequestAction, ActionResponseData{
    "Foo", "Inst-0", "AN-Inst-0", "an-key", "some resp", 9, "Failure Data"}}

var ShutReqData = ServerRequestData{TypeServerRequestShutdown, ShutdownRequestData{}}

var ClTimeout = 2

/* For clientAPI testing */
var testData = []TestData{
    // Reg Client
    {TestClientData{TypeRegClient, []string{TEST_CL_NAME}, nil, false, MsgEmptyResp{}},
        TestServerData{LoMRequest{TypeRegClient, TEST_CL_NAME, ClTimeout, MsgRegClient{}},
            LoMResponse{0, "Succeeded", MsgEmptyResp{}}}},

    // Reg Action - test failure
    {TestClientData{TypeRegAction, []string{TEST_ACTION_NAME}, nil, true, MsgEmptyResp{}},
        TestServerData{LoMRequest{TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction{TEST_ACTION_NAME}},
            LoMResponse{1, "failed by design", MsgEmptyResp{}}}},

    // Reg Action - test failure
    {TestClientData{TypeRegAction, []string{TEST_ACTION_NAME}, nil, true, MsgEmptyResp{}},
        TestServerData{LoMRequest{TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction{TEST_ACTION_NAME}},
            LoMResponse{1, "SKIP", MsgEmptyResp{}}}},

    // Register action
    {TestClientData{TypeRegAction, []string{TEST_ACTION_NAME}, nil, false, MsgEmptyResp{}},
        TestServerData{LoMRequest{TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction{TEST_ACTION_NAME}},
            LoMResponse{0, "Succeeded", MsgEmptyResp{}}}},

    // Request for request and server sends Action request
    {TestClientData{TypeRecvServerRequest, []string{}, nil, false, ActReqData},
        TestServerData{LoMRequest{TypeRecvServerRequest, TEST_CL_NAME, ClTimeout, MsgRecvServerRequest{}},
            LoMResponse{0, "Succeeded", ActReqData}}},

    // Send Action response to server
    {TestClientData{TypeSendServerResponse, []string{}, ActResData, false, MsgEmptyResp{}},
        TestServerData{LoMRequest{TypeSendServerResponse, TEST_CL_NAME, ClTimeout, ActResData},
            LoMResponse{0, "Succeeded", MsgEmptyResp{}}}},

    // Send Action heartbeat to server
    {TestClientData{TypeNotifyActionHeartbeat, []string{TEST_ACTION_NAME, "100"}, nil, false, MsgEmptyResp{}},
        TestServerData{LoMRequest{TypeNotifyActionHeartbeat, TEST_CL_NAME, ClTimeout,
            MsgNotifyHeartbeat{TEST_ACTION_NAME, 100}},
            LoMResponse{0, "Good", MsgEmptyResp{}}}},

    // Request for request and server sends shutdown request
    {TestClientData{TypeRecvServerRequest, []string{}, nil, false, ShutReqData},
        TestServerData{LoMRequest{TypeRecvServerRequest, TEST_CL_NAME, ClTimeout, MsgRecvServerRequest{}},
            LoMResponse{0, "Succeeded", ShutReqData}}},

    // Send Dereg action
    {TestClientData{TypeDeregAction, []string{TEST_ACTION_NAME}, nil, false, MsgEmptyResp{}},
        TestServerData{LoMRequest{TypeDeregAction, TEST_CL_NAME, ClTimeout, MsgDeregAction{TEST_ACTION_NAME}},
            LoMResponse{0, "Succeeded", MsgEmptyResp{}}}},

    // Send Dereg client
    {TestClientData{TypeDeregClient, []string{}, nil, false, MsgEmptyResp{}},
        TestServerData{LoMRequest{TypeDeregClient, TEST_CL_NAME, ClTimeout, MsgDeregClient{}},
            LoMResponse{0, "Succeeded", MsgEmptyResp{}}}},

    // Send duplicate Dereg client which is expected to fail
    {TestClientData{TypeDeregClient, []string{}, nil, true, MsgEmptyResp{}},
        TestServerData{LoMRequest{}, LoMResponse{}}},
}

/* For clientAPI via JSON testing */
var testDataForJson = []TestDataForJson{
    // Reg Client
    {`{"ReqType":1,"Client":"Foo","TimeoutSecs":2,"ReqData":{}}`,
        `{"ResultCode":0,"ResultStr":"Succeeded","RespData":{}}`,
        TestServerData{LoMRequest{TypeRegClient, TEST_CL_NAME, ClTimeout, MsgRegClient{}},
            LoMResponse{0, "Succeeded", MsgEmptyResp{}}}},

    // Reg Action - test failure - as we set server msg too
    {`{"ReqType":3,"Client":"Foo","TimeoutSecs":2,"ReqData":{"Action":"Detect-0"}}`,
        `{"ResultCode":1,"ResultStr":"failed by design","RespData":{}}`,
        TestServerData{LoMRequest{TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction{TEST_ACTION_NAME}},
            LoMResponse{1, "failed by design", MsgEmptyResp{}}}},

    // Register action failed
    {`{"ReqType":3,"Client":"Foo","TimeoutSecs":2,"ReqData":{"Action":"Detect-0"}}`,
        `{"ResultCode":1,"ResultStr":"Failed","RespData":{}}`,
        TestServerData{LoMRequest{TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction{TEST_ACTION_NAME}},
            LoMResponse{1, "Failed", MsgEmptyResp{}}}},

    // Register action succeeded.
    {`{"ReqType":3,"Client":"Foo","TimeoutSecs":2,"ReqData":{"Action":"Detect-0"}}`,
        `{"ResultCode":0,"ResultStr":"Succeeded","RespData":{}}`,
        TestServerData{LoMRequest{TypeRegAction, TEST_CL_NAME, ClTimeout, MsgRegAction{TEST_ACTION_NAME}},
            LoMResponse{0, "Succeeded", MsgEmptyResp{}}}},

    // Send recv server req to server
    {`{"ReqType":5,"Client":"Foo","TimeoutSecs":2,"ReqData":{}}`,
        `{"ResultCode":0,"ResultStr":"Succeeded","RespData":{"ReqType":1,"ReqData":{"Action":"Bar","InstanceId":"inst_1","AnomalyInstanceId":"an_inst_0","AnomalyKey":"an_key","Timeout":10,"Context":[{"Action":"Detect-0","InstanceId":"an_inst_0","AnomalyInstanceId":"an_inst_0","AnomalyKey":"an_key","Response":"res_anomaly","ResultCode":0,"ResultStr":""},{"Action":"Foo-safety","InstanceId":"inst_0","AnomalyInstanceId":"an_inst_0","AnomalyKey":"an_key","Response":"res_foo_check","ResultCode":2,"ResultStr":"some failure"}]}}}`,
        TestServerData{
            LoMRequest{TypeRecvServerRequest, TEST_CL_NAME, ClTimeout, MsgEmptyResp{}},
            LoMResponse{0, "Succeeded", ActReqData}}},

    // Send Server response
    {`{"ReqType":6,"Client":"Foo","TimeoutSecs":2,"ReqData":{"ReqType":1,"ResData":{"Action":"Foo","InstanceId":"Inst-0","AnomalyInstanceId":"AN-Inst-0","AnomalyKey":"an-key","Response":"some resp","ResultCode":9,"ResultStr":"Failure Data"}}}`,
        `{"ResultCode":0,"ResultStr":"Succeeded","RespData":{}}`,
        TestServerData{LoMRequest{TypeSendServerResponse, TEST_CL_NAME, ClTimeout,
            MsgSendServerResponse{TypeServerRequestAction, ActionResponseData{
                "Foo", "Inst-0", "AN-Inst-0", "an-key", "some resp", 9, "Failure Data"}}},
            LoMResponse{0, "Succeeded", MsgEmptyResp{}}}},

    // Request for heartbeat
    {`{"ReqType":7,"Client":"Foo","TimeoutSecs":2,"ReqData":{"Action":"Detect-0","Timestamp":100}}`,
        `{"ResultCode":0,"ResultStr":"Succeeded","RespData":{}}`,
        TestServerData{LoMRequest{TypeNotifyActionHeartbeat, TEST_CL_NAME, ClTimeout,
            MsgNotifyHeartbeat{"Detect-0", 100}},
            LoMResponse{0, "Succeeded", MsgEmptyResp{}}}},

    // Send Dereg action
    {`{"ReqType":4,"Client":"Foo","TimeoutSecs":2,"ReqData":{"Action":"Detect-0"}}`,
        `{"ResultCode":0,"ResultStr":"Succeeded","RespData":{}}`,
        TestServerData{LoMRequest{TypeDeregAction, TEST_CL_NAME, ClTimeout, MsgDeregAction{TEST_ACTION_NAME}},
            LoMResponse{0, "Succeeded", MsgEmptyResp{}}}},

    // Send Dereg client
    {`{"ReqType":2,"Client":"Foo","TimeoutSecs":2,"ReqData":{}}`,
        `{"ResultCode":0,"ResultStr":"Succeeded","RespData":{}}`,
        TestServerData{LoMRequest{TypeDeregClient, TEST_CL_NAME, ClTimeout, MsgDeregClient{}},
            LoMResponse{0, "Succeeded", MsgEmptyResp{}}}},
}

/*
 * This is *not* a test Function -- Note it is not exported
 *
 * This is used to simulate concurrent client.
 * It walks the testData array and simulates request per TestClientData
 * The test code will act as server end and verify the incoming request
 * against TestServerData:Req and send back TestServerData:Res as response
 *
 * As requests are synchronous, they walk in sync via channel sync.
 */
func testClient(chRes chan interface{}, chComplete chan interface{}) {
    txClient := GetClientTx(ClTimeout)

    for i := 0; i < len(testData); i++ {
        tdata := &testData[i]
        var err error
        var reqData *ServerRequestData = nil

        switch tdata.ReqType {
        case TypeRegClient:
            if len(tdata.Args) != 1 {
                LogPanic("client: tid:%d: Expect 1 args for register client len=%d", i, len(tdata.Args))
            }
            err = txClient.RegisterClient(tdata.Args[0])
        case TypeDeregClient:
            if len(tdata.Args) != 0 {
                LogPanic("client: tid:%d: Expect No args for register client len=%d", i, len(tdata.Args[1]))
            }
            err = txClient.DeregisterClient()
        case TypeRegAction:
            if len(tdata.Args) != 1 {
                LogPanic("client: tid:%d: Expect 1 args for register action len=%d", i, len(tdata.Args))
            }
            err = txClient.RegisterAction(tdata.Args[0])
        case TypeDeregAction:
            if len(tdata.Args) != 1 {
                LogPanic("client: tid:%d: Expect 1 args for deregister action len=%d", i, len(tdata.Args))
            }
            err = txClient.DeregisterAction(tdata.Args[0])
        case TypeRecvServerRequest:
            if len(tdata.Args) != 0 {
                LogPanic("client: tid:%d: Expect No args for RecvServerRequest len=%d", i, len(tdata.Args))
            }
            reqData, err = txClient.RecvServerRequest()
        case TypeSendServerResponse:
            if len(tdata.Args) != 0 {
                LogPanic("client: tid:%d: Expect No args for SendServerResponse len=%d", i, len(tdata.Args))
            }
            p := tdata.DataArgs
            res, ok := p.(MsgSendServerResponse)
            if !ok {
                LogPanic("client: tid:%d: Expect MsgSendServerResponse as DataArgs (%T)/(%v)", i, p, p)
            }
            err = txClient.SendServerResponse(&res)
        case TypeNotifyActionHeartbeat:
            if len(tdata.Args) != 2 {
                LogPanic("client: tid:%d: Expect 2 args for register action len=%d", i, len(tdata.Args))
            }
            t, e := strconv.ParseInt(tdata.Args[1], 10, 64)
            if e != nil {
                LogPanic("client: tid:%d: Expect int64 val as second arg (%v)", i, tdata.Args[1])
            }
            err = txClient.NotifyHeartbeat(tdata.Args[0], t)
        default:
            LogPanic("client: tid:%d TODO - Not yet implemented (%d)", i, tdata.ReqType)
        }
        if (err != nil) != tdata.Failed {
            LogPanic("client: tid:%d type(%d/%s) err=%v failed=%v", i, tdata.ReqType,
                ReqTypeToStr[tdata.ReqType], err, tdata.Failed)
        }

        p := tdata.ExpResp
        if reqData != nil {
            if expData, ok := p.(ServerRequestData); ok {
                if !reqData.Equal(&expData) {
                    LogPanic("Client: tid:%d ReqData (%v) != expData(%v)", i, *reqData, expData)
                }
            } else {
                LogPanic("Client: tid:%d Type mismatch Rcvd:(%T) exp(%T)", i, reqData, p)
            }
        } else if x, ok := p.(MsgEmptyResp); !ok {
            LogPanic("Client: tid:%d Received None. Exp:(%T)", i, x)
        }

        LogDebug("client: tid=%d succeeded", i)
        chRes <- struct{}{}
    }
    LogDebug("client: Complete")
    chComplete <- struct{}{}
}

func TestMain(t *testing.T) {
    tx, err := ServerInit()
    if err != nil {
        t.Errorf("Failed to init server")
    }
    chResult := make(chan interface{})
    chComplete := make(chan interface{})

    /*
     * Run client in separate routine.
     * Walks testData array. On completion of each, send a signal via
     * chRes. This helps the following for loop that simulates server
     * to keep in sync.
     */
    go testClient(chResult, chComplete)

    /*
     * In a loop, read a client request and send response as in
     * testData entry per index. Go to next iteration upon client
     * simulation signalling the completion of that index via chResult
     */
    for i := 0; i < len(testData); i++ {
        if len(chComplete) != 0 {
            t.Errorf("Server tid:%d But client complete", i)
        }

        tdata := &testData[i]
        LogDebug("Server: Running: tid=%d", i)

        if (tdata.Req != LoMRequest{}) {
            p, _ := tx.ReadClientRequest(chComplete)
            if p == nil {
                t.Errorf("Server: tid:%d ReadClientRequest returned nil", i)
            }
            if *p.Req != tdata.Req {
                t.Errorf("Server: tid:%d: Type(%d) Failed to match msg(%v) != exp(%v)",
                    i, tdata.ReqType, *p.Req, tdata.Req)
            }
            /* Response to remote client -- done via clientTx */
            if tdata.Res.ResultStr == "SKIP" {
                p.ChResponse <- struct{}{}
            } else {
                p.ChResponse <- &tdata.Res
            }
        }
        /* Wait for client to complete */
        <-chResult

    }
    LogDebug("Server Complete. Waiting on client to complete...")
    <-chComplete
    LogDebug("SUCCEEDED")
}

func testJSONClient(t *testing.T, tx *LoMTransport, chRes, chComplete chan interface{}) {
    for ti := 0; ti < len(testDataForJson); ti++ {
        tdata := &testDataForJson[ti]
        req := tdata.JsonReq
        res := ""
        if err := tx.LoMRPCRequest(&req, &res); err != nil {
            t.Errorf("ERR: testJSONClient: %d: failed (%v)", ti, err)
        }
        if res != tdata.JsonRes {
            t.Errorf("ERR: testJSONClient: %d: res(%s) != exp(%s)", ti, res, tdata.JsonRes)
        }
        if ti == 4 {
            LogDebug("res:(%v)", res)
        }
        chRes <- struct{}{} /* Indicate completion to server */
    }
    {
        /* Failure cases */
        req := `{"ReqType":1999,"Client":"Foo","TimeoutSecs":2,"ReqData":{}}`
        res := ""
        if err := tx.LoMRPCRequest(nil, &res); err == nil { 
            t.Errorf("ERR: testJSONClient: not failing for nil req")
        }
        if err := tx.LoMRPCRequest(&req, nil); err == nil { 
            t.Errorf("ERR: testJSONClient: not failing for nil res")
        }
        if err := tx.LoMRPCRequest(&req, &res); err == nil { 
            t.Errorf("ERR: testJSONClient: not failing for invalid req type")
        }
    }
    LogDebug("testJSONClient: complete")
    chComplete <- struct{}{}
}

func TestJSONServer(t *testing.T) {
    tx := GetLoMTransport()

    chResult := make(chan interface{})
    chComplete := make(chan interface{})

    go testJSONClient(t, tx, chResult, chComplete)

    for ti := 0; ti < len(testDataForJson); ti++ {
        if len(chComplete) != 0 {
            t.Errorf("ERR: Server tid:%d But client complete", ti)
        }
        tdata := &testDataForJson[ti]
        LogDebug("JSON Server: Running: tid=%d", ti)
        LogDebug("JSON Server: Running: tdata(%v)", *tdata)

        p, err := tx.ReadClientRequest(chComplete)
        if p == nil {
            t.Errorf("ERR: ServerJson: tid:%d ReadClientRequest returned nil err(%v)", ti, err)
        }
        if recv, err := json.Marshal(*p.Req); err != nil {
            t.Errorf("ERR: testJSONServer: %d: err(%v) is not valid Json (%v)", ti, err, *p.Req)
        } else if exp, err := json.Marshal(tdata.Req); err != nil {
            t.Errorf("ERR: testJSONServer: %d Failed to marshal test data err(%v) data(%v)", ti, err, tdata.Req)
        } else if string(recv) != string(exp) {
            t.Errorf("ERR: testJSONServer: %d req mismatch recv(%s) != exp(%s)", ti, string(recv), string(exp))
        }
        p.ChResponse <- &tdata.Res
        /* Wait for client to complete */
        <-chResult
    }
    LogDebug("TestJSONServer complete. Waiting on client ...")
    <-chComplete
    LogDebug("TestJSONServer complete. SUCCESS")
}

func TestClientFail(t *testing.T) {
    txClient := GetClientTx(ClTimeout)

    {
        retE := errors.New("rerer")
        retC := errors.New("irerrwe")
        resCode := -1

        /* Save & override */
        dial := RPCDialHttp
        RPCDialHttp = func(s1 string, s2 string) (*rpc.Client, error) {
            return nil, retE
        }

        clCall := ClientCall
        ClientCall = func(tx *ClientTx, serviceMethod string, args any, reply any) error {
            if retC != nil {
                return retC
            }
            x, ok := reply.(*LoMResponse)
            if !ok {
                t.Errorf("Cient call reply not map to LomResponse (%T)", x)
            }
            x.ResultCode = resCode
            x.RespData = struct{}{}
            return nil
        }

        {
            err := txClient.RegisterClient("")
            if err != retE {
                t.Errorf("Failed to fail HTTP call")
            }

        }

        /* Don't fail HTTP */
        retE = nil
        {
            if err := txClient.RegisterClient(""); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            if err := txClient.DeregisterClient(); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            if err := txClient.RegisterAction(""); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            if err := txClient.DeregisterAction(""); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            if _, err := txClient.RecvServerRequest(); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            d := MsgSendServerResponse{}
            if err := txClient.SendServerResponse(&d); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
            if err := txClient.NotifyHeartbeat("", 0); err != retC {
                t.Errorf("Failed to fail in RPC call")
            }
        }

        /* Don't fail call, but return non zero result */
        retC = nil
        {
            if err := txClient.RegisterClient(""); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            if err := txClient.DeregisterClient(); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            if err := txClient.RegisterAction(""); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            if err := txClient.DeregisterAction(""); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            if _, err := txClient.RecvServerRequest(); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            d := MsgSendServerResponse{}
            if err := txClient.SendServerResponse(&d); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
            if err := txClient.NotifyHeartbeat("", 0); err == nil {
                t.Errorf("Failed to handle non zero response")
            }
        }

        /* Fail in respData */
        resCode = 0
        {
            if err := txClient.RegisterClient(""); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            if err := txClient.DeregisterClient(); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            if err := txClient.RegisterAction(""); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            if err := txClient.DeregisterAction(""); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            if _, err := txClient.RecvServerRequest(); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            d := MsgSendServerResponse{}
            if err := txClient.SendServerResponse(&d); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
            if err := txClient.NotifyHeartbeat("", 0); err == nil {
                t.Errorf("Failed to handle non Empty response")
            }
        }

        /* Restore overrides */
        RPCDialHttp = dial
        ClientCall = clCall
    }
}

func cmpMap(s map[string]string, d map[string]string) bool {
    if len(s) != len(d) {
        LogDebug("len mismatch %d != %d", len(s), len(d))
        return false
    }
    for k, v := range s {
        v1, ok := d[k]
        if !ok || (v1 != v) {
            LogDebug("ok(%v) v1(%v) v(%v)\n", ok, v1, v)
            return false
        }
    }
    LogDebug("MAtched\n")
    return true
}

func TestServerFail(t *testing.T) {
    {
        p1 := []*ActionResponseData{{}, {}}
        p2 := []*ActionResponseData{{}}

        if false != SlicesComp(p1, p2) {
            t.Errorf("SlicesComp Failed to fail")
        }

        p2 = []*ActionResponseData{{}, {}}
        p2[0].Action = "foo"
        if false != SlicesComp(p1, p2) {
            t.Errorf("SlicesComp same len Failed to fail")
        }
    }
    {
        s1 := (*ServerRequestData)(nil)
        s2 := (*ServerRequestData)(nil)
        if true != s1.Equal(s2) {
            t.Errorf("Failed to match nil pointers")
        }

        s1 = &ServerRequestData{TypeServerRequestAction, struct{}{}}
        if false != s1.Equal(s2) {
            t.Errorf("Failed to mismatch non nil vs nil")
        }

        s2 = &ServerRequestData{TypeServerRequestShutdown,
            ActionRequestData{"foo", "", "", "", 9, []*ActionResponseData{}}}
        if false != s1.Equal(s2) {
            t.Errorf("Failed to find mismatched req type")
        }

        s2.ReqType = TypeServerRequestAction
        if false != s1.Equal(s2) {
            t.Errorf("Failed to find mismatched reqData type")
        }

        s1.ReqData = ActionRequestData{"bar", "", "", "", 9, []*ActionResponseData{}}
        if false != s1.Equal(s2) {
            t.Errorf("Failed to find mismatched reqData value type")
        }

        s2.ReqData = ActionRequestData{"Bar", "", "", "", 9, []*ActionResponseData{}}
        if false != s1.Equal(s2) {
            t.Errorf("Failed to find mismatched reqData value")
        }

        s2.ReqData = ActionRequestData{"bar", "", "", "", 9, []*ActionResponseData{}}
        if true != s1.Equal(s2) {
            t.Errorf("Failed to find match reqData value")
        }

        s1.ReqData = struct{}{}
        s2.ReqData = struct{}{}
        if false != s1.Equal(s2) {
            t.Errorf("Failed to find Unexpected ReqData type")
        }
    }
    {
        s1 := (*ActionRequestData)(nil)
        s2 := (*ActionRequestData)(nil)
        if true != s1.Equal(s2) {
            t.Errorf("Failed to match nil pointers")
        }

        s1 = &ActionRequestData{"bar", "", "", "", 9, []*ActionResponseData{}}
        if false != s1.Equal(s2) {
            t.Errorf("Failed to mismatch non nil vs nil")
        }

        s2 = &ActionRequestData{"bar", "rrr", "", "", 9, []*ActionResponseData{}}
        if false != s1.Equal(s2) {
            t.Errorf("Failed to find mismatched value")
        }

        s2 = &ActionRequestData{"bar", "", "", "", 9, []*ActionResponseData{}}
        if true != s1.Equal(s2) {
            t.Errorf("Failed to find matched value")
        }
    }

    {
        tx := LoMTransport{make(chan interface{}, 1)}
        chAbort := make(chan interface{}, 1)

        /* Send incorrect data type */
        {
            {
                t := &struct{}{}
                tx.ServerCh <- t
            }
            if p, e := tx.ReadClientRequest(chAbort); e == nil || p != nil {
                t.Errorf("Failed to fail for incorrect Req data type to server")
            }
        }

        /* explicit Abort */
        chAbort <- "Abort"
        if p, e := tx.ReadClientRequest(chAbort); e == nil || p != nil {
            t.Errorf("Failed to fail for abort")
        }
    }
    {
        p := &ActionResponseData{
            Action:            "aaa",
            InstanceId:        "rerew-erere",
            AnomalyInstanceId: "fgfg-gfgg-453",
            AnomalyKey:        "Blah-Blah",
            Response:          "All good",
            ResultCode:        77,
            ResultStr:         "Seventy Seven",
        }
        m := map[string]string{
            "action":            p.Action,
            "instanceId":        p.InstanceId,
            "anomalyInstanceId": p.AnomalyInstanceId,
            "anomalyKey":        p.AnomalyKey,
            "response":          p.Response,
            "resultCode":        "77",
            "resultStr":         p.ResultStr,
        }
        if !cmpMap(p.ToMap(false), m) {
            t.Errorf("1: Failed cmp (%v) != (%v)", p.ToMap(false), m)
        }

        /* Mark it as first action to get state */
        p.AnomalyInstanceId = p.InstanceId
        m["state"] = "init"
        m["anomalyInstanceId"] = p.AnomalyInstanceId
        if !cmpMap(p.ToMap(false), m) {
            t.Errorf("2: Failed cmp (%v) != (%v)", p.ToMap(false), m)
        }

        m["state"] = "complete"
        if !cmpMap(p.ToMap(true), m) {
            t.Errorf("2: Failed cmp (%v) != (%v)", p.ToMap(true), m)
        }

        lst := map[string]ActionResponseData{
            "missing Action":            ActionResponseData{},
            "missing Instanceid":        ActionResponseData{Action: "foo"},
            "missing AnomalyInstanceid": ActionResponseData{Action: "foo", InstanceId: "ddd"},
            "missing AnomalyKey for non anomaly": ActionResponseData{Action: "foo", InstanceId: "ddd",
                AnomalyInstanceId: "eee"},
            "missing AnomalyKey for anomaly": ActionResponseData{Action: "foo", InstanceId: "ddd",
                AnomalyInstanceId: "ddd"},
        }

        /* Anomaly with key */
        good := ActionResponseData{Action: "foo", InstanceId: "ddd",
            AnomalyInstanceId: "ddd", AnomalyKey: "erere", Response: "rr"}
        /* Failed anonaly w/o key */
        good1 := ActionResponseData{Action: "foo", InstanceId: "ddd",
            AnomalyInstanceId: "ddd", ResultCode: 77}

        for k, v := range lst {
            if v.Validate() != false {
                t.Errorf("Expect to fail (%s)(%v)", k, v)
            }
        }
        if good.Validate() != true {
            t.Errorf("Expect to succeed (%v)", good)
        }
        if good1.Validate() != true {
            t.Errorf("Expect to succeed (%v)", good1)
        }
    }
}

func TestHelper(t *testing.T) {
    {
        /* Test logger helper */
        FmtFprintfCnt := 0

        v := FmtFprintf
        FmtFprintf = func(w io.Writer, s string, a ...any) (int, error) {
            FmtFprintfCnt++
            return 0, nil
        }

        LogWarning("LoM: Lib Test WARNING messsage")
        if FmtFprintfCnt != 1 {
            t.Errorf("FmtFprintf not called")
        }

        lvl := GetLogLevel()
        SetLogLevel(syslog.LOG_ERR)
        if syslog.LOG_ERR != GetLogLevel() {
            t.Errorf("Failed tp set/get log level")
        }

        LogWarning("LoM: Lib Test WARNING messsage")
        if FmtFprintfCnt != 1 {
            t.Errorf("FmtFprintf is called when not expected")
        }

        SetLogLevel(syslog.LOG_DEBUG)

        FmtFprintf = v
        LogWarning("LoM: Lib Test WARNING messsage")
        if FmtFprintfCnt != 1 {
            t.Errorf("FmtFprintf is called when not expected")
        }
        SetLogLevel(lvl)
    }

    {
        /* Test log_panic to exit */
        ExitCnt := 0
        e := OSExit
        OSExit = func(v int) {
            ExitCnt++
        }
        LogPanic("Hitting Panic")
        if ExitCnt != 1 {
            t.Errorf("Panic test failed")
        }
        OSExit = e
    }

}

type ConfigData_t struct {
    GlobalStr   string
    ActionStr   string
    BindStr     string
    ProcStr     string
    Failed      bool
    testMode    bool
    Reason      string
}

var testConfigData = []ConfigData_t {
        {
            "",
            "",
            "",
            "",
            true,
            false,
            "Missing global file",
        },
        {
            "{}",
            "",
            "",
            "",
            true,
            false,
            "Missing actions file",
        },
        {
            "{}",
            "{}",
            "",
            "",
            true,
            false,
            "Missing bindings file",
        },
        {
            "{}",
            "{}",
            "{}",
            "",
            true,
            false,
            "Missing procs file",
        },
        {
            "eee",
            "",
            "",
            "",
            true,
            false,
            "Invalid global Json data",
        },
        {
            "{}",
            "eee",
            "",
            "",
            true,
            false,
            "Invalid actions Json data",
        },
        {
            "{}",
            "{}",
            "eee",
            "",
            true,
            false,
            "Invalid bindings Json data",
        },
        {
            "{}",
            "{}",
            "{}",
            "eee",
            true,
            false,
            "Invalid procs Json data",
        },
        {
            `{}`,
            `{ "actions": [ { "name": "xxx" } ] }`,
            `{ "bindings": [ { "name": "Test", "actions": [ {"name": "YYY"} ] } ] }`,
            `{}`,
            true,
            false,
            "Action name YYY not in actions",
        },
        {
            `{}`,
            `{ "actions": [ { "name": "xxx" }, { "name": "yyy" } ] }`,
            `{ "bindings": [ { "name": "Test", "actions": [ {"name": "xxx", "sequence": 0 }, {"name": "yyy"}] } ] }`,
            `{}`,
            true,
            false,
            "Duplicate sequence",
        },
        {
            `{ "foo": "bar", "ENGINE_HB_INTERVAL_SECS": 11, "list": [ "hello", "world" ], "MAX_SEQ_TIMEOUT_SECS":"77"}`,
            `{ "actions": [ { "name": "xxx" }, { "name": "yyy" } ] }`,
            `{ "bindings": [ { "name": "Test", "actions": [ ] } ] }`,
            `{}`,
            true,
            false,
            "No actions in sequence",
        },
        {
            `{ "foo": "bar", "ENGINE_HB_INTERVAL_SECS": 11, "list": [ "hello", "world" ], "MAX_SEQ_TIMEOUT_SECS":"77"}`,
            `{ "actions": [ { "name": "xxx" }, { "name": "yyy" } ] }`,
            `{ "bindings": [ { "name": "Test", "actions": [ {"name": "xxx", "sequence": 1 }, {"name": "yyy"}] } ] }`,
            `{}`,
            false,
            false,
            "",
        },
        {
            `{ "foo": "bar", "ENGINE_HB_INTERVAL_SECS": 11, "list": [ "hello", "world" ], "MAX_SEQ_TIMEOUT_SECS":"77"}`,
            `{ "actions": [ { "name": "xxx" }, { "name": "yyy" } ] }`,
            `{ "bindings": [ { "name": "Test", "actions": [ {"name": "xxx", "sequence": 1 }, {"name": "yyy"}] } ] }`,
            `{}`,
            false,
            true,
            "",
        },
    }

type testAPIData_t struct {
    GlobalStr       string
    ActionStr       string
    BindStr         string
    ProcStr         string
    Seq             map[string]bool
    Sequence        BindingSequence_t
    ActionsCfg      map[string]ActionCfg_t
}

var testApiData = testAPIData_t{
    `{ "foo": "bar", "ENGINE_HB_INTERVAL_SECS": 22, "list": [ "hello", "world" ], "MAX_SEQ_TIMEOUT_SECS":"77"}`,
    `{ "actions": [ { "name": "foo", "timeout":77 }, { "name": "bar" } ] }`,
    `{ "bindings": [ { "sequencename": "TestFoo", "timeout": 60, "actions": [ {"name": "foo", "sequence": 1 }, {"name": "bar"}] } ] }`,
    `{}`,
    map[string]bool {
        "foo": false,
        "bar": true,
    },
    BindingSequence_t{
        "TestFoo",
        60,
        0,
        []*BindingActionCfg_t{
            {
                "bar",
                false,
                0,
                0,
            },
            {
                "foo",
                false,
                77,
                1,
            },
        },
    },
    map[string]ActionCfg_t{
        "foo": {
            "foo",
            "",
            77,
            0,
            false,
            false,
            "",
        },
        "bar": {
            "bar",
            "",
            0,
            0,
            false,
            false,
            "",
        },
    },
}

func createFile(name string, s string) (string, error) {
    fl := ""
    defer func() {
        LogDebug("name:(%s) file(%s)", name, fl)
        if len(fl) == 9 {
            LogPanic("Failed to create file")
        }
    }()

    if len(s) == 0 {
        return "", nil
    }
    fl = "/tmp/" + name
    if f, err := os.Create(fl); err != nil {
        return "", err
    } else {
        _, err := f.WriteString(s)
        f.Close()
        return fl, err
    }
}

func getConfigMgrTest(t *testing.T, gl, ac, bi, pr string, testMode bool) (*ConfigMgr_t, error) {
    if flG, err := createFile(GLOBALS_CONF_FILE, gl); err != nil {
        t.Errorf("TestConfig: Failed to create Global file")
    } else if _, err := createFile(ACTIONS_CONF_FILE, ac); err != nil {
        t.Errorf("TestConfig: Failed to create Action file")
    } else if _, err := createFile(BINDINGS_CONF_FILE, bi); err != nil {
        t.Errorf("TestConfig: Failed to create Binding file")
    } else if _, err := createFile(PROCS_CONF_FILE, pr); err != nil {
        t.Errorf("TestConfig: Failed to create Procs file")
    } else {
        if testMode {
            if _, err := createFile(LOM_TESTMODE_NAME, "zz"); err != nil {
                t.Errorf("TestConfig: Failed to create TestMode (%s) file", LOM_TESTMODE_NAME)
            }
        }
        if err := InitConfigPath(filepath.Dir(flG)); err != nil {
            return nil, err
        } else {
            return GetConfigMgr(), nil
        }
    }
    return nil, LogError("Failed to init Cfg Manager")
}

func cleanConfigFiles() {
    for _, v := range []string { GLOBALS_CONF_FILE, ACTIONS_CONF_FILE,
                    BINDINGS_CONF_FILE, PROCS_CONF_FILE, LOM_TESTMODE_NAME } {
        fl := filepath.Join("/tmp/", v)
        if _, err := os.Stat(fl); err == nil {
            os.Remove(fl)
        }
    }
}


func TestConfig(t *testing.T) {
    for ti, d := range testConfigData {
        ResetLoMMode()
        ctTestMode := GetLoMRunMode()
        if ctTestMode != LoMRunMode_NotSet {
            t.Errorf("%d: Expect run mode NotSet(%d) != ct(%d)", ti, LoMRunMode_NotSet, ctTestMode)
        }
        _, err := getConfigMgrTest(t, d.GlobalStr, d.ActionStr, d.BindStr, d.ProcStr, d.testMode)
        if d.Failed == (err == nil) {
            t.Errorf("%d: Expect to fail(%v) but result:(%v): (%s)",
                    ti, d.Failed, err, d.Reason)
        }
        LogDebug("err=(%v)", err)
        if !d.Failed {
            expTestMode := LoMRunMode_Prod
            if d.testMode {
                expTestMode = LoMRunMode_Test
            }
            ctTestMode = GetLoMRunMode()
            if expTestMode != ctTestMode {
                t.Errorf("%d: mode mismatch exp(%d) != ct(%d) d.testMode(%v)", ti, expTestMode, ctTestMode, d.testMode)
            }
        }
        cleanConfigFiles()
    }
    {
        {
            lst := []struct{
                    val string
                    mode LoMRunMode_t
                } {{"yes", LoMRunMode_Test }, {"xyz", LoMRunMode_Prod}}

            for i, v := range lst {
                ResetLoMMode()
                os.Setenv(LOM_TESTMODE_NAME, v.val)
                ctMode := GetLoMRunMode()
                if ctMode != v.mode {
                    t.Errorf("%d: mode mismatch exp(%d) != ct(%d)", i, v.mode, ctMode)
                }
            }
            os.Unsetenv(LOM_TESTMODE_NAME)
        }
        {
            ResetLoMMode()
            for i, v := range []struct {
                    mode    LoMRunMode_t
                    fail    bool
                } { {LoMRunMode_NotSet, true},
                    {LoMRunMode_Test, false},
                    {LoMRunMode_Prod, true}} {
                if err := SetLoMRunMode(v.mode); (err != nil) != v.fail {
                    t.Errorf("%d: Expect to fail(%v) err(%v)", i, v.fail, err)
                }
                LogDebug("---------- i=%d v=(%v) -------------", i, v)
            }
            ResetLoMMode()
        }
    }

    {
        mgr, err := getConfigMgrTest(t, testApiData.GlobalStr, testApiData.ActionStr, testApiData.BindStr, testApiData.ProcStr, false)
        if err != nil {
            t.Errorf("Unexpected error: (%v) (%v)", err, mgr == nil)
        }

        if v := mgr.GetGlobalCfgStr("foo"); v != "bar" {
            t.Errorf("Global foo: bar != (%s)", v)
        } else if v := mgr.GetGlobalCfgStr("Foo"); v != "" {
            t.Errorf("Global Foo <empty> != (%s)", v)
        } else if v := mgr.GetGlobalCfgInt("ENGINE_HB_INTERVAL_SECS"); v != 22 {
            t.Errorf("Global ENGINE_HB_INTERVAL_SECS: 22 != (%v) (%T)", v, v)
        } else if v := mgr.GetGlobalCfgInt("MIN_PERIODIC_LOG_PERIOD_SECS"); v != 15 {
            t.Errorf("Global MIN_PERIODIC_LOG_PERIOD_SECS: Default: 15 != (%v) (%T)", v, v)
        } else if v := mgr.GetGlobalCfgInt("XXN_PERIODIC_LOG_PERIOD"); v != 0 {
            t.Errorf("Global XXN_PERIODIC_LOG_PERIOD: Non-existing: 0 != (%v) (%T)", v, v)
        } else if v := mgr.GetGlobalCfgAny("List"); v != nil {
            t.Errorf("Global List: not expected but exist (%T) (%v)", v, v)
        } else if v := mgr.GetGlobalCfgAny("list"); v == nil {
            t.Errorf("Global list: expected to exist")
        } else {
            if l, ok := v.([]interface{}); !ok {
                t.Errorf("Global list: Not interface list")
            } else if len(l) != 2 {
                t.Errorf("Global list: len: 2 != (%d)", len(l))
            } else {
                lst := [2]string{"hello", "world"}
                for i, ac := range l {
                    if s, ok := ac.(string); !ok {
                        t.Errorf("Global list: entry type string != (%T) (%v)", s, s)
                    } else if s != lst[i] {
                        t.Errorf("Global list[%d] %s != %s", i, s, lst[i])
                    }
                }
            }
        }

        startSeqAct := ""

        lst := mgr.GetActionsList()
        for k, b := range testApiData.Seq {
            if b != mgr.IsStartSequenceAction(k) {
                t.Errorf("%v != IsStartSequenceAction(%s)", b, k)
            }
            if v, ok := lst[k]; !ok {
                t.Errorf("%s missing in GetActionsList", k)
            } else if v.IsAnomaly != b {
                t.Errorf("%s isAnomaly (%v) != (%v)", k, v.IsAnomaly, b)
            }
            if b {
                startSeqAct = k
            }
        }

        bsNil := (*BindingSequence_t)(nil)
        if bs, err1 := mgr.GetSequence(startSeqAct); err1 != nil {
            t.Errorf("Failed to get seq (%s) err(%v)", startSeqAct, err1)
        } else if !bs.Compare(&testApiData.Sequence) {
            t.Errorf("%s: sequence mismatch (%v) != (%v)", startSeqAct, *bs, testApiData.Sequence)
        } else if bs.Compare(bsNil) {
            t.Errorf("BindingSequence_t:Compare Failed to mismatch non-nil & nil")
        } else if !bsNil.Compare(bsNil) {
            t.Errorf("BindingSequence_t:Compare Failed to match nil & nil")
        } else {
            bs.Actions[0].Name = "xxx"
            if bs.Compare(&testApiData.Sequence) {
                t.Errorf("%s: sequence Failed to mismatch (%v) != (%v)", startSeqAct, *bs, testApiData.Sequence)
            }
            bs.SequenceName = "XXXX"
            if bs.Compare(&testApiData.Sequence) {
                t.Errorf("%s: sequence Failed to mismatch (%v) != (%v)", startSeqAct, *bs, testApiData.Sequence)
            }
        }

        if _, err1 := mgr.GetSequence("xyz"); err1 == nil {
            t.Errorf("Failed to fail for missing seq xyz")
        }

        for k, v := range testApiData.ActionsCfg {
            if a, e := mgr.GetActionConfig(k); e != nil {
                t.Errorf("%s: Failed to get action cfg", k)
            } else if *a != v {
                t.Errorf("%s: config mismatch (%v) != (%v)", k, a, v)
            }
        }

        if _, e := mgr.GetActionConfig("zyy"); e == nil {
            t.Errorf("Failed to fail for nin existing action cfg")
        }
    }
}

func TestPeriodic(t *testing.T) {
    s := GetUUID()
    if len(s) != 36 {
        t.Errorf("Expect 36 chars long != (%d) (%s)", len(s), s)
    }

    UUID_BIN = "xxx"
    s = GetUUID()
    if len(s) == 36 {
        t.Errorf("Expect custom string not 36. (%d) (%s)", len(s), s)
    }
    _, err := getConfigMgrTest(t, `{ "MIN_PERIODIC_LOG_PERIOD_SECS": 1 }`,"{}", "{}", "{}", false)
    if err != nil {
        t.Errorf("Unexpected error: (%v)", err)
    }

    {
        lg := GetlogPeriodic()

        d := &LogPeriodicEntry_t{}

        if err := lg.AddLogPeriodic(d); err == nil {
            t.Errorf("LogPerodic: Expect to fail for empty ID")
        }

        Ids := []string{"ID_0", "ID_1", "ID_2"}
        d.ID = Ids[0]
        if err := lg.AddLogPeriodic(d); err == nil {
            t.Errorf("LogPerodic: Expect to fail for empty message")
        }

        d.Message = "Message"
        if err := lg.AddLogPeriodic(d); err == nil {
            t.Errorf("LogPerodic: Expect to fail for too small period:%d", d.Period)
        }

        d.Period = 1
        d.Lvl = syslog.LOG_DEBUG
        if err := lg.AddLogPeriodic(d); err != nil {
            t.Errorf("LogPerodic: Expect to succeed (%v)", d)
        }

        d.ID = Ids[1]
        d.Period = 5
        if err := lg.AddLogPeriodic(d); err != nil {
            t.Errorf("LogPerodic: Expect to succeed (%v)", d)
        }

        d.ID = Ids[2]
        d.Period = 2
        if err := lg.AddLogPeriodic(d); err != nil {
            t.Errorf("LogPerodic: Expect to succeed (%v)", d)
        }

        // Sleep to ensure run method, all cases executed
        time.Sleep(2 * time.Second)
        for _, k := range Ids {
            lg.DropLogPeriodic(k)
        }
        time.Sleep(2 * time.Second)

    }

    {
        m := map[string]string{
            "foo":  "bar",
            "val":  "42",
            "data": "xxx",
        }  
        SetPublishAPI(PublishString)
        s := PublishEvent(m)
        exp := `{"data":"xxx","foo":"bar","val":"42"}`
        if s != exp {
            t.Errorf("Incorrect publish string (%s) != (%s)", s, exp)
        }
    }
    {
        err := LogError("Blah Blah")
        if e := GetLastError(); e != err {
            t.Errorf("GetLastError mismatch read(%v) != exp(%v)", e, err)
        }
        ResetLastError()
        if e := GetLastError(); e != nil {
            t.Errorf("GetLastError not nil after reset. read(%v)", e)
        }
    }
}

func TestOneShot(t *testing.T) {
    exp := []int{1, 2, 3}
    rcv := make([]int, 0, len(exp))
    ch := make(chan int, len(exp))

    f0 := func() { ch <- exp[0] }
    f1 := func() { ch <- exp[1] }
    f2 := func() { t.Errorf("f2 is not expected to be called") } /* Disabled */
    f3 := func() { ch <- exp[2] }

    tmr0 := AddOneShotTimer(-2, "f0", f0) /* 2 secs before */
    AddOneShotTimer(1, "f1", f1)
    tmr2 := AddOneShotTimer(1, "f2", f2) /* Two for same time */
    tmr2.Disable()
    tmr3 := AddOneShotTimer(3, "f3", f3) /* one later */

    for {
        select {
        case v := <-ch:
            rcv = append(rcv, v)
            if len(rcv) == len(exp) {
                for i, e := range rcv {
                    if e != exp[i] {
                        t.Errorf("Oneshot slice mismatch (%v) != (%v)", rcv, exp)
                        break
                    }
                }
                /* Test complete */
                if !tmr0.IsDone() || !tmr3.IsDone() || tmr2.IsDone() {
                    t.Errorf("One shot timer IsDone not set (%v) (%v) (%v)",
                        tmr0.IsDone(), tmr2.IsDone(), tmr3.IsDone())
                }
                if tmr0.IsDisabled() || !tmr2.IsDisabled() || tmr3.IsDisabled() {
                    t.Errorf("One shot timer  IsDisabled incorrect (%v) (%v) (%v)",
                        tmr0.IsDisabled(), tmr2.IsDisabled(), tmr3.IsDisabled())
                }
                return
            }

        case <-time.After(4 * time.Second):
            /* test expected to complete before this timeout */
            t.Errorf("Oneshot failed (%v) != (%v)", rcv, exp)
            return
        }
    }
}

func TestShutdown(t *testing.T) {
    cnt := 3
    chStarted := make(chan int, cnt)
    chDone := make(chan int, cnt+1)
    
    /*
     * start cnt routines; Each register; Then indicate their start via chStarted
     * Wait for signal from shutdown.
     * On Signal, indicate their exit via chDone and *then* de-register
     * Last de-register will make DoSysShutdown exit immediately
     * So call de-register at the end.
     */
    for i := 0; i < cnt; i++ {

        go func(index int) {
            s := fmt.Sprintf("ShutTestClient_%d", index)
            // LogDebug("Started %s", s)
            ch := RegisterForSysShutdown(s)
            chStarted <- 0
            <- ch    /* Wait for signal to exit */
            if p := RegisterForSysShutdown("xyx"); p != nil {
                t.Errorf("Expect register to fail after shutdown started")
            }
            if index == 0 {
                /* Induce some delay */
                time.Sleep(3 * time.Second)
            }
            chDone <- 0
            DeregisterForSysShutdown(s)
            // LogDebug("Ended %s", s)
        }(i)
    }

    for i := 0; i < cnt; i++ {
        <- chStarted
    }

    /* Wait for all routines to start */
    // LogDebug("All started")

    /* Kick off shutdown process */
    go func() {
        // LogDebug("Started shutdown")
        DoSysShutdown(0)
        chDone <- 1
        // LogDebug("Shutdown Ended")
    }()

    /* Verify all go routines ended via reading their done signature */
    for i := 0; i < cnt; i++ {
        j := <- chDone
        if j != 0 {
            t.Errorf("Expect 0 value j=%d i=%d cnt=%d", j, i, cnt)
        }
    }

    /* Verify DoSysShutdown ended too */
    j := <- chDone
    if j != 1 {
        t.Errorf("Expect 1 value j=%d cnt=%d", j, cnt)
    }

    DoSysShutdown(0)
    //LogDebug("---------- COMPLETE ----------------")
}

/* Validates if configuration is returned from a valid json */
func Test_GetIntConfigurationFromJson_ReturnsSuccessfully(t *testing.T) {
	jsonString := "{\"abc\":123}"
	result := GetIntConfigurationFromJson(jsonString, "abc", 5)
	assert := assert.New(t)
	assert.Equal(123, result)
}

/* Validates if default configuration is returned for empty json */
func Test_GetIntConfigurationFromJson_ReturnsDefaultValueForEmptyJson(t *testing.T) {
	jsonString := ""
	result := GetIntConfigurationFromJson(jsonString, "abc", 5)
	assert := assert.New(t)
	assert.Equal(5, result)
}

/* Validates if default configuration is returned for invalid json */
func Test_GetIntConfigurationFromJson_ReturnsDefaultValueForInvalidJson(t *testing.T) {
	jsonString := "{\"abc\";abc}"
	result := GetIntConfigurationFromJson(jsonString, "abc", 5)
	assert := assert.New(t)
	assert.Equal(5, result)
}

/* Validates if default value is returned when key not found */
func Test_GetIntConfigurationFromJson_ReturnsDefaultValueForMissingConfigKey(t *testing.T) {
	jsonString := "{\"abc\":123}"
	result := GetIntConfigurationFromJson(jsonString, "xyz", 5)
	assert := assert.New(t)
	assert.Equal(5, result)
}

/* Validates if default value is returned for invalid configuration value type */
func Test_GetIntConfigurationFromJson_ReturnsDefaultValueForInvalidConfigValueType(t *testing.T) {
	jsonString := "{\"abc\":abc}"
	result := GetIntConfigurationFromJson(jsonString, "abc", 5)
	assert := assert.New(t)
	assert.Equal(5, result)
}

/* Validates configuration is returned successfuly for a valid json */
func Test_GetFloatConfigurationFromJson_ReturnsSuccessfully(t *testing.T) {
	jsonString := "{\"abc\":0.0001}"
	result := GetFloatConfigurationFromJson(jsonString, "abc", 0.1)
	assert := assert.New(t)
	assert.Equal(0.0001, result)
}

/* Validates default config is returned for invalid json */
func Test_GetFloatConfigurationFromJson_ReturnsDefaultValueForInvalidJson(t *testing.T) {
	jsonString := "{\"abc\";abc}"
	result := GetFloatConfigurationFromJson(jsonString, "abc", 0.1)
	assert := assert.New(t)
	assert.Equal(0.1, result)
}

/* Validates default config is returned for an empty json */
func Test_GetFloatConfigurationFromJson_ReturnsDefaultValueForEmptyJson(t *testing.T) {
	jsonString := ""
	result := GetFloatConfigurationFromJson(jsonString, "abc", 0.1)
	assert := assert.New(t)
	assert.Equal(0.1, result)
}

/* Validates default config is returned for a missing config key */
func Test_GetFloatConfigurationFromJson_ReturnsDefaultValueForMissingConfigKey(t *testing.T) {
	jsonString := "{\"abc\":0.0001}"
	result := GetFloatConfigurationFromJson(jsonString, "xyz", 0.1)
	assert := assert.New(t)
	assert.Equal(0.1, result)
}

/* Validates default value is returned for invalid config value type */
func Test_GetFloatConfigurationFromJson_ReturnsDefaultValueForInvalidConfigValueType(t *testing.T) {
	jsonString := "{\"abc\":abc}"
	result := GetFloatConfigurationFromJson(jsonString, "abc", 0.1)
	assert := assert.New(t)
	assert.Equal(0.1, result)
}

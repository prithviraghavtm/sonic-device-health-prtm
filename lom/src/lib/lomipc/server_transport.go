package lomipc

import (
    "encoding/gob"
    "fmt"
    . "lom/src/lib/lomcommon"
    "net"
    "net/http"
    "net/rpc"
    "net/rpc/jsonrpc"
    "reflect"
    "strconv"
)

/*
 *  Transport i/f via Go RPC https://pkg.go.dev/net/rpc
 *
 *  NOTE:
 *      This is used as only IPC between processes running inside a single
 *      container as single system; Tightly coupled with static set of APIs
 *
 *  Multiple clients (PluginMgr) send requests concurrently to a server/engine.
 *  Server process each sequentially/concurrently and respond back to each as succeeded
 *  or failed.
 *
 *  Server send requests to multiple clients sequentially/concurrently as a request
 *  addressed to a client only. A single client may receive multiple requests at any time.
 *  For each received request, client confirm back as succeeded / failed, synchronously.
 *  The client may just do some basic validation on the request.
 *
 *  Server creates transport with a channel for all clients to send their request.
 *  Similarly each client creates a channel for all server requests.
 *
 *  The request carry a channel for response.
 *
 *  The channel creator/owner holds the read end and the remote process holds the write end.
 *
 *  Channels are buffered with estimated count of clients for server and known count of
 *  plugins managed by client.
 */

/*
 * NOTE: Any change in Go lib must reflect in libs of non-go client libs
 */
const (
    RPC_HTTP_PORT = 1234
    RPC_JSON_PORT = 1235
)

/* All types of requests from client to server */
type ReqDataType int

const (
    TypeNone                  = ReqDataType(iota)
    TypeRegClient             /* 1 */
    TypeDeregClient           /* 2 */
    TypeRegAction             /* 3 */
    TypeDeregAction           /* 4 */
    TypeRecvServerRequest     /* 5 */
    TypeSendServerResponse    /* 6 */
    TypeNotifyActionHeartbeat /* 7 */
    TypeCount
)

var ReqTypeToStr = map[ReqDataType]string{
    TypeNone:                  "None",
    TypeRegClient:             "RegisterClient",
    TypeDeregClient:           "DeregisterClient",
    TypeRegAction:             "RegisterAction",
    TypeDeregAction:           "DeregisterAction",
    TypeRecvServerRequest:     "RecvServerRequest",
    TypeSendServerResponse:    "SendServerResponse",
    TypeNotifyActionHeartbeat: "NotifyActionHeartbeat",
}

/* Server sends its request as response to TypeRecvServerRequest */
type ServerReqDataType int

const (
    TypeServerRequestNone = ServerReqDataType(iota)
    TypeServerRequestAction
    TypeServerRequestShutdown
    TypeServerRequestCount
)

var ServerReqTypeToStr = map[ServerReqDataType]string{
    TypeServerRequestAction:   "RecvServerRequestAction",
    TypeServerRequestShutdown: "RecvServerRequestShutdown",
}

/* Request from client to server over RPC */
type LoMRequest struct {
    ReqType     ReqDataType /* Type of request */
    Client      string      /* The client sending this request */
    TimeoutSecs int         /* Timeout - Honored in long running requests */
    /* == 0 implies no timeout */
    ReqData interface{} /* Data specific to request type */
}

/*
 * Response from server to client.
 *
 * All client requests are synchrnous. Hence the response does not
 * include any client request details.
 *
 * RespData is specific to client's request. It could be nil.
 */
type LoMResponse struct {
    ResultCode int
    ResultStr  string

    RespData interface{}
}

/*
 * Msg to pass via ReqData in LomRequest
 * All kinds of msg data matching request type
 */
type MsgRegClient struct { /* For TypeRegClient */
}

type MsgDeregClient struct { /* For TypeDeregClient */
}

type MsgRegAction struct { /* For TypeRegAction */
    Action string
}

type MsgDeregAction struct { /* For TypeDeregAction */
    Action string
}

type MsgRecvServerRequest struct { /* For TypeRecvServerRequest */
}

type MsgSendServerResponse struct { /* For TypeSendServerResponse */
    ReqType ServerReqDataType
    ResData interface{} /* ActionResponseData */
}

type MsgNotifyHeartbeat struct { /* For TypeNotifyActionHeartbeat */
    Action    string
    Timestamp int64
}

/*
 * Server requests are pulled by client via TypeRecvServerRequest.
 * Hence the server request object is sent via RespData in LoMResponse.
 *
 * The ReqType specifies the type of server request.
 * The ReqData is specific per type.
 */
type ServerRequestData struct {
    ReqType ServerReqDataType /* Type of requests from server to client */
    ReqData interface{}       /* Data per request type */
    /* ActionRequestData or ShutdownRequestData */
}

/*
 * Sent as MsgSendServerResponse::ResData for
 * MsgSendServerResponse::ReqType == TypeServerRequestAction
 */
type ActionResponseData struct {
    Action            string
    InstanceId        string
    AnomalyInstanceId string
    AnomalyKey        string
    Response          string
    ResultCode        int
    ResultStr         string
}

/* Helper to convert ActionResponseData as Map */
func (p *ActionResponseData) ToMap(end bool) map[string]string {
    ret := map[string]string{
        "action":            p.Action,
        "instanceId":        p.InstanceId,
        "anomalyInstanceId": p.AnomalyInstanceId,
        "anomalyKey":        p.AnomalyKey,
        "response":          p.Response,
        "resultCode":        fmt.Sprintf("%d", p.ResultCode),
        "resultStr":         p.ResultStr,
    }
    if p.InstanceId == p.AnomalyInstanceId {
        if end {
            ret["state"] = "complete"
        } else {
            ret["state"] = "init"
        }
    }
    return ret
}

/* Helper to validate ActionResponseData */
func (p *ActionResponseData) Validate() bool {
    isAnomaly := p.InstanceId == p.AnomalyInstanceId
    isFailed := p.ResultCode != 0

    if (len(p.Action) == 0) ||
        (len(p.InstanceId) == 0) ||
        (len(p.AnomalyInstanceId) == 0) ||
        (!isAnomaly && len(p.AnomalyKey) == 0) || /* Key could miss for failed anomaly */
        (!isFailed && len(p.AnomalyKey) == 0) {
        return false
    }
    return true
}

/*
 * ReqData for ServerRequestData::ReqData for
 * ServerRequestData::ReqType == TypeServerRequestAction
 */
type ActionRequestData struct {
    Action            string
    InstanceId        string
    AnomalyInstanceId string
    AnomalyKey        string
    Timeout           int
    Context           []*ActionResponseData
}

/*
 * ReqData for ServerRequestData::ReqData for
 * ServerRequestData::ReqType == TypeServerRequestShutdown
 */
type ShutdownRequestData struct {
}

type MsgEmptyResp struct {
}

/* Helper to compare given slices */
func SlicesComp(p []*ActionResponseData, q []*ActionResponseData) bool {
    if len(p) != len(q) {
        LogDebug("Slice len differ %d != %d\n", len(p), len(q))
        return false
    }

    for i, v := range p {
        if *v != *(q[i]) {
            LogDebug("p[%d] (%v) != q[%d] (%v)\n", i, v, i, q[i])
            return false
        }
    }
    return true
}

/* Helper to compare given requests. */
func (r *ActionRequestData) Equal(p *ActionRequestData) bool {
    if r == p {
        /* Same ptr */
        return true
    }
    if (r == nil) || (p == nil) {
        LogError("Unexpected nil args self(%v) arg(%v)\n", (r == nil), (p == nil))
        return false
    }

    if (r.Action == p.Action) &&
        (r.InstanceId == p.InstanceId) &&
        (r.AnomalyInstanceId == p.AnomalyInstanceId) &&
        (r.AnomalyKey == p.AnomalyKey) &&
        SlicesComp(r.Context, p.Context) {
        return true
    } else {
        return false
    }
}

/* Helper to compare given requests. */
func (r *ServerRequestData) Equal(p *ServerRequestData) bool {
    if r == p {
        /* Same ptr */
        return true
    }
    if (r == nil) || (p == nil) {
        LogError("Unexpected nil args self(%v) arg(%v)\n", (r == nil), (p == nil))
        return false
    }

    if r.ReqType != p.ReqType {
        LogDebug("Differing Req types %s vs %s", ServerReqTypeToStr[r.ReqType],
            ServerReqTypeToStr[p.ReqType])
        return false
    }
    rr := r.ReqData
    pr := p.ReqData
    if reflect.TypeOf(rr) != reflect.TypeOf(pr) {
        LogDebug("Differing ReqData types %T vs %T", rr, pr)
        return false
    }
    switch rr.(type) {
    case ActionRequestData:
        rq, ok1 := rr.(ActionRequestData)
        pq, ok2 := pr.(ActionRequestData)
        if !ok1 || !ok2 || !(&rq).Equal(&pq) {
            LogDebug("ActionRequestData mismatch ok1=%v ok2=%v", ok1, ok2)
            return false
        }
        return true
    case ShutdownRequestData:
        return true
    default:
        LogError("Unkown ReqData type (%T) in ServerRequestData", rr)
        return false
    }
}

/*
 * Each proc has a channel for remote end to write request.
 * Each request carry a channel for response to that request.
 */
type LoMTransport struct {
    ServerCh chan interface{}
}

/* Internal req object within server. */
type LoMRequestInt struct {
    Req *LoMRequest
    /* LoMResponse to this request is sent via this chan */
    ChResponse chan interface{}
}

/* RPC call from client */
func (tr *LoMTransport) SendToServer(req *LoMRequest, reply *LoMResponse) (err error) {

    if (req == nil) || (reply == nil) {
        return LogError("Nil args req(%v) reply(%v)", req, reply)
    }

    defer func() {
        if err != nil {
            LogError("SendToServer cl(%s) mtype(%s) failed (%v)",
                req.Client, ReqTypeToStr[req.ReqType], err)
        } else {
            LogInfo("SUCCESS: SendToServer cl(%s) mtype(%s) result(%d)/(%s)", req.Client,
                ReqTypeToStr[req.ReqType], reply.ResultCode, reply.ResultStr)
        }
    }()

    rpcReq := LoMRequestInt{req, make(chan interface{}, 1)}
    tr.ServerCh <- &rpcReq

    LogDebug("Req sent to server client(%s) type(%s). Waiting for response...",
        req.Client, ReqTypeToStr[req.ReqType])

    /* Wait for server response */
    p := <-rpcReq.ChResponse
    if x, ok := p.(*LoMResponse); ok {
        *reply = *x
    } else {
        return LogError("Server response message (%T) != *LoMResponse", x)
    }

    return nil
}

/* Local call from server to read client request. */
func (tr *LoMTransport) ReadClientRequest(chAbort chan interface{}) (*LoMRequestInt, error) {
    /* Return on non-null request or upon abort */
    select {
    case p := <-tr.ServerCh:
        if x, ok := p.(*LoMRequestInt); ok {
            LogDebug("Server: Read from client (%s) type(%s)", x.Req.Client, ReqTypeToStr[x.Req.ReqType])
            return x, nil
            /* Let server return response upon processing, via channel embedded in msg. */
        } else {
            return nil, LogError("Client request message (%T) != *Msg", x)
        }

    case <-chAbort:
        return nil, LogError("Server: Aborting read via abort channel")
        /* Aborting per instruction */
    }
}

/* PrintActionRequest prints the given ActionRequestData as a string.
 * If includeResponses is true, it also includes the responses in the output.
 * Returns the formatted string.
 */
func PrintActionRequest(request *ActionRequestData, includeResponses bool) string {
    if request == nil {
        return "Invalid request"
    }

    output := fmt.Sprintf("Action: %s InstanceId: %s AnomalyInstanceId: %s AnomalyKey: %s Timeout: %d",
        request.Action, request.InstanceId, request.AnomalyInstanceId, request.AnomalyKey, request.Timeout)

    if includeResponses {
        responseOutput := " Responses: "
        for i, response := range request.Context {
            responseOutput += fmt.Sprintf("Response %d: ", i+1)
            responseOutput += fmt.Sprintf("Action: %s InstanceId: %s AnomalyInstanceId: %s AnomalyKey: %s Response: %s ResultCode: %d ResultStr: %s",
                response.Action, response.InstanceId, response.AnomalyInstanceId, response.AnomalyKey, response.Response, response.ResultCode, response.ResultStr)
        }
        output += responseOutput
    }

    return output
}

/* PrintServerRequest prints the given ServerRequestData as a string.
 * If includeResponses is true, it also includes the responses in the output.
 * Returns the formatted string.
 */
func PrintServerRequest(request *ServerRequestData, includeResponses bool) string {
    if request == nil {
        return "Invalid request"
    }

    switch reqData := request.ReqData.(type) {
    case ActionRequestData:
        return PrintActionRequest(&reqData, includeResponses)
    case ShutdownRequestData:
        return "Shutdown request"
    default:
        return fmt.Sprintf("Unknown request type: %T", reqData)
    }
}

/* PrintActionResponseData prints the given ActionResponseData as a string.
 * Returns the formatted string.
 */
func PrintActionResponseData(response *ActionResponseData) string {
    return fmt.Sprintf("Action: %s InstanceId: %s AnomalyInstanceId: %s AnomalyKey: %s Response: %s ResultCode: %d ResultStr: %s",
        response.Action, response.InstanceId, response.AnomalyInstanceId, response.AnomalyKey, response.Response, response.ResultCode, response.ResultStr)
}

/*
 * PrintHeartbeatNotification prints the given MsgNotifyHeartbeat as a string.
 */
func PrintHeartbeatNotification(notification MsgNotifyHeartbeat) string {
    return fmt.Sprintf("Heartbeat notification: %s %d", notification.Action, notification.Timestamp)
}

/* PrintServerResponse prints the given ServerResponse as a string.
 * Returns the formatted string.
 */
func PrintServerResponse(response interface{}) string {
    switch respData := response.(type) {
    case *MsgSendServerResponse:
        var msgSendServerResponse *MsgSendServerResponse = respData
        resData, ok := msgSendServerResponse.ResData.(*ActionResponseData)
        if !ok {
            return "Invalid response data"
        }
        return PrintActionResponseData(resData)
    case MsgNotifyHeartbeat:
        return PrintHeartbeatNotification(respData)
    default:
        return fmt.Sprintf("Unknown response type: %T", respData)
    }
}

func init_encoding() {
    gob.Register(MsgRegClient{})
    gob.Register(MsgDeregClient{})
    gob.Register(MsgRegAction{})
    gob.Register(MsgDeregAction{})
    gob.Register(MsgRecvServerRequest{})
    gob.Register(MsgSendServerResponse{})
    gob.Register(MsgNotifyHeartbeat{})
    gob.Register(ServerRequestData{})
    gob.Register(ActionRequestData{})
    gob.Register(ActionResponseData{})
    gob.Register(ShutdownRequestData{})
    gob.Register(MsgEmptyResp{})
}

func GetLoMTransport() *LoMTransport {
    init_encoding()
    tr := new(LoMTransport)

    tr.ServerCh = make(chan interface{})
    return tr
}

/* Init the serverside transport */
func httpServerInit() error {

    rpc.HandleHTTP()
    l, e := net.Listen("tcp", "localhost:"+strconv.Itoa(RPC_HTTP_PORT))
    if e != nil {
        LogPanic("listen error at %d :(%v)", RPC_HTTP_PORT, e)
    }
    go http.Serve(l, nil)
    LogDebug("HTTP Server: Started serving")

    return nil
}

func jsonServerInit() error {
    l, e := net.Listen("tcp", "localhost:"+strconv.Itoa(RPC_JSON_PORT))
    if e != nil {
        LogPanic("listen error at %d (%v)", RPC_JSON_PORT, e)
    }

    go func() {
        for {
            LogInfo("waiting for JSON connections ...")
            if conn, err := l.Accept(); err != nil {
                LogError("JSON RPC accept error: %v", err)
            } else {
                LogInfo("JSON RPC connection started: %v", conn.RemoteAddr())
                go jsonrpc.ServeConn(conn)
            }
        }
    }()
    LogDebug("JSON Server: Started serving")

    return nil
}

func ServerInit() (*LoMTransport, error) {
    tr := GetLoMTransport()

    rpc.Register(tr)
    httpServerInit()
    jsonServerInit()

    return tr, nil
}

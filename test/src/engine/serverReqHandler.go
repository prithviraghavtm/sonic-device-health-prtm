package engine

import (
    "fmt"
    . "lib/lomcommon"
    . "lib/lomipc"
    "runtime"
)

type LoMResponseCode int

const LoMResponseOk = LoMResponseCode(0)

/* Start at high number, so as not to conflict with OS error codes */
const LOM_RESP_CODE_START = 4096

/* List of all error codes returned in LoM response */
const (
    LoMUnknownError = LoMResponseCode(iota+LOM_RESP_CODE_START)
    LoMUnknownReqType
    LoMIncorrectReqData
    LoMReqFailed
    LoMReqTimeout
    LoMFirstActionFailed
    LoMMissingSequence
    LoMActionDeregistered
    LoMActionNotRegistered
    LoMActionActive
    LoMSequenceTimeout
    LoMSequenceIncorrect
    LoMShutdown
    LoMErrorCnt
)

var LoMResponseStr = [13]string {
    "Unknown error",
    "Unknown request",
    "Incorrect Msg type",
    "Request failed",
    "Request Timed out",
    "First Action failed",
    "First Action's sequence missing",
    "Action de-regsitered",
    "Action not registered",
    "Action already active",
    "Sequence timed out",
    "Sequence state incorrect",
    "LOM system shutdown",
}

func init() {
    if len(LoMResponseStr) != (int(LoMErrorCnt) - LOM_RESP_CODE_START) {
        LogPanic("LoMResponseStr len(%d) != (%d - %d = %d)", len(LoMResponseStr),
                LoMErrorCnt, LOM_RESP_CODE_START, int(LoMErrorCnt) - LOM_RESP_CODE_START)
    }
}

func GetLoMResponseStr(code LoMResponseCode) string {
    switch  {
    case code == LoMResponseOk:
        return ""

    case (code < LOM_RESP_CODE_START) || (code >= LoMErrorCnt):
        return "Unknown error code"

    default:
        return LoMResponseStr[int(code)-LOM_RESP_CODE_START]
    }
}


/* Helper to construct LoMResponse object */
func createLoMResponse(code LoMResponseCode, msg string) *LoMResponse {
    if (code != LoMResponseOk) {
        if (code < LOM_RESP_CODE_START) || (code >= LoMErrorCnt) {
            LogPanic("Internal error: Unexpected error code (%d) range (%d to %d)",
                    code, LoMResponseOk, LoMErrorCnt)
        }
    }
    s := msg
    if (len(s) == 0) && (code != LoMResponseOk) {
        /* Prefix caller name to provide context */
        if pc, _, _, ok := runtime.Caller(1); ok {
            details := runtime.FuncForPC(pc)
            s = details.Name() + ": " + GetLoMResponseStr(code)
        }
    }
    return &LoMResponse { int(code), s, MsgEmptyResp{} }
}


type serverHandler_t struct {
}

/*
 * Handle each request type.
 * Other than recvServerRequest, the rest are synchronous 
 */
func (p *serverHandler_t) processRequest(req *LoMRequestInt) {
    if req == nil {
        LogPanic("Expect non nil LoMRequestInt")
    }
    if (req.Req == nil) || (req.ChResponse == nil) {
        LogPanic("Expect non nil LoMRequest (%v)", req)
    }
    if len(req.ChResponse) == cap(req.ChResponse) {
        LogPanic("No room in chResponse (%d)/(%d)", len(req.ChResponse),
                cap(req.ChResponse))
    }
    var res *LoMResponse = nil

    switch req.Req.ReqType {
    case TypeRegClient:
        res = p.registerClient(req.Req)
    case TypeDeregClient:
        res = p.deregisterClient(req.Req)
    case TypeRegAction:
        res = p.registerAction(req.Req)
    case TypeDeregAction:
        res = p.deregisterAction(req.Req)
    case TypeRecvServerRequest:
        res = p.recvServerRequest(req)
    case TypeSendServerResponse:
        res = p.sendServerResponse(req.Req)
    case TypeNotifyActionHeartbeat:
        res = p.notifyHeartbeat(req.Req)
    default:
        res = createLoMResponse(LoMUnknownReqType, "")
    }
    if res != nil {
        req.ChResponse <- res
        if res.ResultCode == 0 {
            switch req.Req.ReqType {
            case TypeSendServerResponse:
                m, _ := req.Req.ReqData.(MsgSendServerResponse)
                GetSeqHandler().ProcessResponse(&m)
            }
        }
    }
    /* nil implies that the request will be processed async. Likely RecvServerRequest */
}


/* Methods below, don't do arg verification, as already vetted by caller processRequest */

func (p *serverHandler_t) registerClient(req *LoMRequest) *LoMResponse {
    if _, ok := req.ReqData.(MsgRegClient); !ok {
        return createLoMResponse(LoMIncorrectReqData, "")
    }
    e := GetRegistrations().RegisterClient(req.Client)
    if e != nil {
        return createLoMResponse(LoMReqFailed, fmt.Sprintf("%v", e))
    }
    return createLoMResponse(LoMResponseOk, "")
}


func (p *serverHandler_t) deregisterClient(req *LoMRequest) *LoMResponse {
    if _, ok := req.ReqData.(MsgDeregClient); !ok {
        return createLoMResponse(LoMIncorrectReqData, "")
    }
    GetRegistrations().DeregisterClient(req.Client)
    return createLoMResponse(LoMResponseOk, "")
}


func (p *serverHandler_t) registerAction(req *LoMRequest) *LoMResponse {
    if m, ok := req.ReqData.(MsgRegAction); !ok {
        return createLoMResponse(LoMIncorrectReqData, "")
    } else {
        info := &ActiveActionInfo_t { m.Action, req.Client, 0 }
        e := GetRegistrations().RegisterAction(info)
        if e != nil {
            return createLoMResponse(LoMReqFailed, fmt.Sprintf("%v", e))
        }
        return createLoMResponse(LoMResponseOk, "")
    }
}


func (p *serverHandler_t) deregisterAction(req *LoMRequest) *LoMResponse {
    if m, ok := req.ReqData.(MsgDeregAction); !ok {
        return createLoMResponse(LoMIncorrectReqData, "")
    } else {
        GetRegistrations().DeregisterAction(m.Action)
        return createLoMResponse(LoMResponseOk, "")
    }
}


func (p *serverHandler_t) notifyHeartbeat(req *LoMRequest) *LoMResponse {
    if m, ok := req.ReqData.(MsgNotifyHeartbeat); !ok {
        return createLoMResponse(LoMIncorrectReqData, "")
    } else {
        GetRegistrations().NotifyHeartbeats(m.Action, m.Timestamp)
        return createLoMResponse(LoMResponseOk, "")
    }
}


func (p *serverHandler_t) recvServerRequest(req *LoMRequestInt) *LoMResponse {
    if _, ok := req.Req.ReqData.(MsgRecvServerRequest); !ok {
        return createLoMResponse(LoMIncorrectReqData, "")
    } else if err := GetRegistrations().PendServerRequest(req); err == nil {
        /* ClientRegistrations_t will send the request whenever available */
        return nil
    } else {
        return createLoMResponse(LoMReqFailed, fmt.Sprintf("%v", err))
    }
}


func (p *serverHandler_t) sendServerResponse(req *LoMRequest) *LoMResponse {
    if _, ok := req.ReqData.(MsgSendServerResponse); !ok {
        return createLoMResponse(LoMIncorrectReqData, "")
    } else {
        return createLoMResponse(LoMResponseOk, "")
        /* Process response called in caller after sending response back */
    }
}


func GetServerReqHandler() *serverHandler_t {
    return &serverHandler_t{}
}

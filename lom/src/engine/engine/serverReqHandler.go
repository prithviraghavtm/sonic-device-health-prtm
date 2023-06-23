package engine

import (
    "fmt"
    . "lom/src/lib/lomcommon"
    . "lom/src/lib/lomipc"
    "runtime"
)

type LoMResponseCode int

const LoMResponseOk = LoMResponseCode(0)
const LoMResponseOkStr = ""
const LoMResponseUnknownStr = "Error code unknown"

/* Start at high number, so as not to conflict with OS error codes */
const LOM_RESP_CODE_START = 4096

/* List of all error codes returned in LoM response */
const (
    LoMUnknownError        = LoMResponseCode(iota + LOM_RESP_CODE_START) /* 4096 */
    LoMUnknownReqType                                                    /* 4097 */
    LoMIncorrectReqData                                                  /* 4098 */
    LoMReqFailed                                                         /* 4099 */
    LoMReqTimeout                                                        /* 4100 */
    LoMFirstActionFailed                                                 /* 4101 */
    LoMMissingSequence                                                   /* 4102 */
    LoMActionDeregistered                                                /* 4103 */
    LoMActionNotRegistered                                               /* 4104 */
    LoMActionActive                                                      /* 4105 */
    LoMSequenceTimeout                                                   /* 4106 */
    LoMSequenceIncorrect                                                 /* 4107 */
    LoMSequenceEmpty                                                     /* 4108 */
    LoMShutdown                                                          /* 4109 */
    LoMInternalError                                                     /* 4110 */
    LoMErrorCnt                                                          /* 4111 */
)

var LoMResponseStr = []string{
    "Unknown error",                   /* LoMUnknownError */
    "Unknown request",                 /* LoMUnknownReqType */
    "Incorrect Msg type",              /* LoMIncorrectReqData */
    "Request failed",                  /* LoMReqFailed */
    "Request Timed out",               /* LoMReqTimeout */
    "First Action failed",             /* LoMFirstActionFailed */
    "First Action's sequence missing", /* LoMMissingSequence */
    "Action de-regsitered",            /* LoMActionDeregistered */
    "Action not registered",           /* LoMActionNotRegistered */
    "Action already active",           /* LoMActionActive */
    "Sequence timed out",              /* LoMSequenceTimeout */
    "Sequence state incorrect",        /* LoMSequenceIncorrect */
    "Sequence empty",                  /* LoMSequenceEmpty */
    "LOM system shutdown",             /* LoMShutdown */
    "LoM Internal error",              /* LoMInternalError */
}

func LoMResponseValidate() (bool, error) {
    if len(LoMResponseStr) != (int(LoMErrorCnt) - LOM_RESP_CODE_START) {
        return false, LogError("LoMResponseStr len(%d) != (%d - %d = %d)", len(LoMResponseStr),
            LoMErrorCnt, LOM_RESP_CODE_START, int(LoMErrorCnt)-LOM_RESP_CODE_START)
    }
    return true, nil
}

func init() {
    LoMResponseValidate()
}

func GetLoMResponseStr(code LoMResponseCode) string {
    switch {
    case code == LoMResponseOk:
        return LoMResponseOkStr

    case (code < LOM_RESP_CODE_START) || (code >= LoMErrorCnt):
        return LoMResponseUnknownStr

    default:
        return LoMResponseStr[int(code)-LOM_RESP_CODE_START]
    }
}

/* Helper to construct LoMResponse object */
func createLoMResponse(code LoMResponseCode, msg string) *LoMResponse {
    if code != LoMResponseOk {
        if (code < LOM_RESP_CODE_START) || (code >= LoMErrorCnt) {
            LogError("Internal error: Unexpected error code (%d) range (%d to %d)",
                code, LoMResponseOk, LoMErrorCnt)
            return nil
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
    return &LoMResponse{int(code), s, MsgEmptyResp{}}
}

type serverHandler_t struct {
}

/*
 * Handle each request type.
 * Other than recvServerRequest, the rest are synchronous
 */
func (p *serverHandler_t) processRequest(req *LoMRequestInt) {
    if req == nil {
        LogError("Expect non nil LoMRequestInt")
    } else if (req.Req == nil) || (req.ChResponse == nil) {
        LogError("Expect non nil LoMRequest (%v)", req)
    } else if len(req.ChResponse) == cap(req.ChResponse) {
        LogError("No room in chResponse (%d)/(%d)", len(req.ChResponse),
            cap(req.ChResponse))
    } else {
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
        info := &ActiveActionInfo_t{m.Action, req.Client, 0}
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
        GetRegistrations().DeregisterAction(req.Client, m.Action)
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

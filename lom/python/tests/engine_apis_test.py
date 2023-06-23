#
# Run as "LOM_LIB_PATH=/lom-root/lom/build/lib  python3 setup.py test"
#
import json
import sys
import syslog
from unittest.mock import MagicMock, patch

import pytest

from . import common_test

sys.path.append("src/common")
import engine_apis
import engine_rpc_if
import common


EXP_SEQ_BIND_0 = '{"SequenceName":"bind-0","Timeout":2,"Priority":0,"Actions":[{"Name":"Detect-0","Mandatory":false,"Timeout":0,"Sequence":0},{"Name":"Safety-chk-0","Mandatory":false,"Timeout":1,"Sequence":1},{"Name":"Mitigate-0","Mandatory":false,"Timeout":6,"Sequence":2}]}'

EXP_ACT_DETECT_0 = '{"Name":"Detect-0","Type":"","Timeout":0,"HeartbeatInt":0,"Disable":false,"Mimic":false,"ActionKnobs":null}'

EXP_ACT_LIST = '{"Detect-0":{"IsAnomaly":true},"Detect-1":{"IsAnomaly":true},"Mitigate-0":{"IsAnomaly":false},"Mitigate-1":{"IsAnomaly":false},"Safety-chk-0":{"IsAnomaly":false},"Safety-chk-1":{"IsAnomaly":false}}'

EXP_PROC_0 = '{"Detect-0":{"name":"Detect-0","version":"00.01.1","path":" /path/"},"Detect-1":{"name":"Detect-1","version":"02.00.1","path":" /path/"},"Safety-chk-0":{"name":"Safety-chk-0","version":"02.00.1","path":" /path/"}}'


testCfgList = [
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_STR, 
            "args": [ "Foo_Bar" ],
            "ret": "Bar",
            "msg": "Get globl cfg Str"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_STR, 
            "args": [ "non-existing" ],
            "ret": "<nil>",
            "msg": "Get globl cfg Str non-exist"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_INT, 
            "args": [ "val_7" ],
            "ret": 7,
            "msg": "Get globl cfg Int"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_INT, 
            "args": [ "non-existing" ],
            "ret": 0,
            "msg": "Get globl cfg Int non-exist"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_SEQ, 
            "args": [ "Detect-0" ],
            "ret": EXP_SEQ_BIND_0,
            "msg": "Get cfg Seq"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_SEQ, 
            "args": [ "non-exist" ],
            "ret": "",
            "msg": "Get cfg Seq non-exist"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_ACTION, 
            "args": [ "Detect-0" ],
            "ret": EXP_ACT_DETECT_0,
            "msg": "Get cfg Action"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_ACTION, 
            "args": [ "non-exist" ],
            "ret": "",
            "msg": "Get cfg Action non-exist"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_LIST_ACTIONS, 
            "args": [],
            "ret": EXP_ACT_LIST,
            "msg": "Get Actions list"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_PROC, 
            "args": [ "proc_0" ],
            "ret": EXP_PROC_0,
            "msg": "Get proc config"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_CFG_PROC, 
            "args": [ "non-exist" ],
            "ret": "",
            "msg": "Get proc config for non-exist"
        }
    ]

# Response need not match req as engine would drop it silently w/o any failure.
SAMPLE_RES = '{0 map[Action:act-1 AnomalyInstanceId:aid-0 AnomalyKey:key-0 InstanceId:id-0 Response:Blah Blah ResultCode:0 ResultStr:All good]}'

testEngineAPIList = [
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_REG_CLIENT,
            "args": [ "test" ],
            "resCode": 0,
            "msg": "Register client first"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_REG_CLIENT,
            "args": [ "test" ],
            "resCode": -1,
            "msg": "Register client duplicate"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_REG_ACTION,
            "args": [ "testAct" ],
            "resCode": -1,
            "msg": "Register action non-existing"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_REG_ACTION,
            "args": [ "Detect-0" ],
            "resCode": 0,
            "msg": "Register action valid"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_REG_ACTION,
            "args": [ "Detect-0" ],
            "resCode": 0,   # Engine de-register and re-register duplicate.
            "msg": "Register action duplicate"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_RECV_REQ,
            "args": [],
            "resCode": 0,
            "msg": "Recv Server req"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_SEND_RES,
            "args": [ SAMPLE_RES ],
            "resCode": 0,
            "msg": "Send Server Response"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_NOTIFY_HB,
            "args": [ "xyz", 100 ], # Engine drops silently for unknown actions.
            "resCode": 0,
            "msg": "Notify HB"
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_DEREG_ACTION,
            "args": [ "testAct" ],
            "resCode": 0,
            "msg": "Deregister action always succeed."
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_DEREG_ACTION,
            "args": [ "Detect-0" ],
            "resCode": 0,
            "msg": "Deregister action always succeed."
        },
        {
            "id": engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_DEREG_CLIENT,
            "args": [],
            "resCode": 0,
            "msg": "Deregister client always succeed."
        }
    ]

# Response need not match req as engine would drop it silently w/o any failure.
SAMPLE_ACT_RES = engine_rpc_if.ActionResponseData("foo", "bar", "hello", "world", "ok", 99, "ninety nine")

testRpcList = [
        {
            "fn": engine_rpc_if.register_client,
            "args": ["test"],
            "ret": [ True ],
            "msg": "Register client first"
        },
        {
            "fn": engine_rpc_if.register_client,
            "args": ["test"],
            # As this req reaches engine directly, it de & re-register
            # In case of clib-binding it gets rejected at the API level.
            "ret": [ True ],
            "msg": "Register client duplicate"
        },
        {
            "fn": engine_rpc_if.register_action,
            "args": ["test"],
            "ret": [ False ],
            "msg": "Register action non-existing"
        },
        {
            "fn": engine_rpc_if.register_action,
            "args": ["Detect-0"],
            "ret": [ True ],
            "msg": "Register action valid"
        },
        {
            "fn": engine_rpc_if.read_action_request,
            "args": [],
            "ret": [ True, {} ],
            "msg": "Register action valid"
        },
        {
            "fn": engine_rpc_if.regNotifyHB,
            "args": [ "foo", 100 ],
            "ret": [ True ],
            "msg": "Heartbeat touch always succeed."
        },
        {
            "fn": engine_rpc_if.write_action_response,
            "args": [ SAMPLE_ACT_RES ],
            "ret": [ True ],
            "msg": "Register action valid"
        },
        {
            "fn": engine_rpc_if.deregister_action,
            "args": ["test"],
            "ret": [ True ],
            "msg": "Deregister action non-existing to succeed"
        },
        {
            "fn": engine_rpc_if.deregister_action,
            "args": ["Detect-0"],
            "ret": [ True ],
            "msg": "Deregister action existing to succeed"
        },
        {
            "fn": engine_rpc_if.deregister_client,
            "args": [],
            "ret": [ True ],
            "msg": "Register client first"
        }
    ]


serverRunning = False

def ServerStart():
    global serverRunning

    if serverRunning:
        return

    # Load config & Start Engine
    #
    ret = common_test.InitCfg(False)
    assert ret == 0, f"lomLib.InitConfigPathForC failed ret={ret}"

    ret = common_test.StartEngine()
    assert ret == True, f"Failed to start engine"

    serverRunning = True


class TestCfg(object):
    def callFn(self, fn, args): 
        ln = len(args)
        if ln == 0:
            return fn()
        if ln == 1:
            return fn(args[0])
        if ln == 2:
            return fn(args[0], args[1])
        if ln == 3:
            return fn(args[0], args[1], args[2])
        common.log_panic("Fix test code args len({}) not handled. args:({})".format(ln, args))


    def callAPI(self, tc): 
        l = tc["args"][:]
        l.insert(0, tc["id"])
        return self.callFn(engine_apis.call_lom_lib, l)


    def testGlobal(self):
        # Create testmode file

        common.set_log_level(syslog.LOG_DEBUG)

        ret = common_test.InitCfg(True)
        assert ret == 0, f"lomLib.InitConfigPathForC failed ret={ret}"

        ret = engine_apis.call_lom_lib(
                engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_RUN_MODE)
        assert ret == 1, f"lomLib.lomLib.GetLoMRunModeC ret{ret} != 1"

        ret = common_test.InitCfg(False)
        assert ret == 0, f"lomLib.InitConfigPathForC failed ret={ret}"

        id = 0
        for tc in testCfgList:
            ret = self.callAPI(tc)
            if ret != tc["ret"]:
                common.log_debug("ret=({})".format(ret))
                common.log_debug("tc[ret]=({})".format(tc["ret"]))
                common.log_debug("AAAAAAAAAAAAAAAAAAAAAA")
            assert ret == tc["ret"], f'id:{id} {tc["msg"]}'
            id += 1


    def testEngineApi(self):
        common.set_log_level(syslog.LOG_DEBUG)

        ServerStart()

        for tc in testEngineAPIList:
            ret = self.callAPI(tc)
            try:
                retData = json.loads(ret)
                assert retData["ResultCode"] == tc["resCode"], f'Result mismatch: {tc["msg"]}'
            except json.decoder.JSONDecodeError as e:
                assert false, f"Failed to decode ({ret}) err:({err})"


    def testRPCApi(self):
        common.set_log_level(syslog.LOG_DEBUG)

        ServerStart()

        for tc in testRpcList:
            ret = self.callFn(tc["fn"], tc["args"])
            if type(ret) == bool:
                ret = (ret,)

            exp = tc["ret"]
            assert len(ret) == len(exp), f'len mismatch {ret} != {exp} {tc["msg"]}'
            assert len(ret) == len(exp), f'len mismatch {ret} != {exp} {tc["msg"]}'

            assert ret[0] == exp[0], f'First entry mismatch {ret} != {exp} {tc["msg"]}'

            if len(ret) > 1:
                expD = exp[1]

                assert type(expD) == dict, f'TEST error type of {expD} != dict {tc["msg"]}'
                retD = json.loads(json.dumps(ret[1], default=vars))

                assert type(expD) == type(retD), f'Type mismatch {type(expD)} != {type(retD)} {retD} {tc["msg"]}'

                for k, v in expD.items():
                    assert v == retD[k], f'Data mismatch key:{k} {v} != {retD[k]} {tc["msg"]}'



#! /usr/bin/env python3

import json
import os
import pathlib
import sys

sys.path.append("src/common")
from common import *
import engine_apis

# Test data for various LoM config files and  files path
class cfgInit:
    cfgPath = "/tmp"                            # Folder for config files

    cfgData = {                                 # key=filename val=<contents of file> 
        "globals.conf.json": '\
{\
    "ENGINE_HB_INTERVAL_SECS" : 3,\
    "Foo_Bar": "Bar",\
    "val_7": 7,\
    "flag_true": true,\
    "flag_false": false,\
    "lst_6_7_8": [ 6, 7, 8 ]\
}',
        "bindings.conf.json": '\
{\
    "bindings": [\
        {\
            "SequenceName": "bind-0", \
            "Priority": 0,\
            "Timeout": 2,\
            "Actions": [\
                {"name": "Detect-0" },\
                {"name": "Safety-chk-0", "sequence": 1 },\
                {"name": "Mitigate-0", "sequence": 2 }\
            ]\
        },\
        {\
            "SequenceName": "bind-1", \
            "Priority": 1,\
            "Timeout": 19,\
            "Actions": [\
                {"name": "Detect-1" },\
                {"name": "Safety-chk-1", "sequence": 1 },\
                {"name": "Mitigate-1", "sequence": 2 }\
            ]\
        }\
    ]\
}',
        "procs.conf.json": '\
{\
    "procs": {\
        "proc_0": {\
            "Detect-0": {\
                "name": "Detect-0",\
                "version": "00.01.1",\
                "path": " /path/"\
            },\
            "Detect-1": {\
                "name": "Detect-1",\
                "version": "02.00.1",\
                "path": " /path/"\
            },\
            "Safety-chk-0": {\
                "name": "Safety-chk-0",\
                "version": "02.00.1",\
                "path": " /path/"\
            }\
        },\
        "proc_1": {\
            "Mitigate-0": {\
                "name": "Mitigate-0",\
                "version": "02_1",\
                "path": " /path/"\
            },\
            "Mitigate-1": {\
                "name": "Mitigate-1",\
                "version": "02_1",\
                "path": " /path/"\
            },\
            "Safety-chk-1": {\
                "name": "Safety-chk-1",\
                "version": "02.00.1",\
                "path": " /path/"\
            }\
        }\
    }\
}'
        }

    actionsData = {
        "Detect-0.conf.json" : '\
	{\
	    "Detect-0" : { "name": "Detect-0" }\
	}',
        "Safety-chk-0.conf.json" : 	'\
	{\
	    "Safety-chk-0" : { "name": "Safety-chk-0", "Timeout": 1}\
	}',
        "Mitigate-0.conf.json" : 	'\
	{\
	    "Mitigate-0" : { "name": "Mitigate-0", "Timeout": 6}\
	}',
        "Detect-1.conf.json" : 	'\
	{\
	    "Detect-1" : { "name": "Detect-1" }\
	}',
        "Safety-chk-1.conf.json" : 	'\
	{\
	    "Safety-chk-1" : { "name": "Safety-chk-1", "Timeout": 7}\
	}',
        "Mitigate-1.conf.json" : 	'\
	{\
	    "Mitigate-1" : { "name": "Mitigate-1", "Timeout": 8}\
	}'
    } 

    def __init__(self, testMode: bool):
        for f, v in self.cfgData.items():
            with open(os.path.join(self.cfgPath, f), "w") as s:
                s.write(v)

        actions_path = os.path.join(self.cfgPath, "actions.confd")
        os.makedirs(actions_path, exist_ok=True)

        for actName, actData in self.actionsData.items():
            with open(os.path.join(actions_path, actName), "w") as fl:
                fl.write(actData)

        testModeFl = os.path.join(self.cfgPath, "LoMTestMode")
        if testMode:
            with open(testModeFl, "w") as s:
                s.write("")
        else:
            # Remove file if exists
            pathlib.Path(testModeFl).unlink(missing_ok = True)


cfg = None

# Creates config files
#
def InitCfg(testMode: bool):
    global cfg

    c = cfgInit(testMode)
    ret = engine_apis.call_lom_lib(
            engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_INIT, c.cfgPath)
    if ret == 0:
        cfg = c
        return 0
    else:
        log_error("Failed to load config")
        return -1


# Run the engine so clients can access it with all requests
#
def StartEngine() -> bool:
    if cfg == None:
        log_error("Require to init config first")
        return False

    ret = engine_apis.call_lom_lib(
            engine_apis.LOM_LIB_FN_INDICES.LOM_LIB_FN_ENGINE_START, cfg.cfgPath)
    return ret == 0


# Some temp test code -- not used
def main():
    s = cfgInit.cfgData["procs.conf.json"]
    print(s)
    print("--------------------------")
    d = json.loads(s)
    print(json.dumps(d, indent=4))
    

if __name__ == "__main__":
    main()


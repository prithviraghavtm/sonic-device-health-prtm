package engine

/*
 * This file maintains test i/p data.
 *
 * A test_entry is defined
 * A testcollection is defined as
 *      {
 *          pre-setup collection,
 *          Set of test entries to run as collection
 *          post-cleanup collection,
 *      }
 *
 *  Each collection is a functional test covers a scope
 *  Each collection is tagged with a description too.
 *
 *  testRunList - Has the list of keys of collections
 *  engine_test:TestRun executes every collection
 *
 */
import (
    "fmt"
    . "lom/src/lib/lomcommon"
    . "lom/src/lib/lomipc"
)

/* Only test data */
type testEntry_t struct {
    id     clientAPIID
    clTx   string /* Which Tx to use*/
    seqId  int    /* The context to use for save/restore results per seq */
    args   []any
    result []any
    failed bool /* True if expected to fail. */
    desc   string
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
    case SEQ_COMPLETE:
        s = "SEQ_COMPLETE"
    case SHUTDOWN:
        s = "SHUTDOWN"
    case NOTIFY_HB:
        s = "NOTIFY_HB"
    case CHK_ACTIV_REQ:
        s = "CHK_ACTIV_REQ"
    case CHK_REG_ACTIONS:
        s = "CHK_REG_ACTIONS"
    case PAUSE:
        s = "PAUSE"
    default:
        s = fmt.Sprintf("UNK(%d)", p.id)
    }
    return fmt.Sprintf("%s:%s: args:(%v) res(%v) failed(%v)",
        p.clTx, s, p.args, p.result, p.failed)
}

type testEntriesList_t map[int]testEntry_t
type testCollectionId_t string
type testCollectionEntry_t struct {
    desc        string
    preSetup    []testCollectionId_t /* List to run as pre-setup */
    testEntries testEntriesList_t    /* tests to run */
    postCleanup []testCollectionId_t
}

type testCollections_t map[testCollectionId_t]*testCollectionEntry_t

/* Order of test runs by ID */
type testRunList_t []testCollectionId_t

var testRunList = testRunList_t{
    "registrations_test",
    "seq_success",
    "seq_mit_fail",
    "seq_safety_fail",
    "detect_fail",
    "sequence-timeout",
    "multi-sequence-block",
    "sequence-block-action-reregister",
}

var xtestRunList = testRunList_t{}

type registrations_t map[string][]string

/* Test scenario expectations */
var expRegistrations = []registrations_t{
    { /* Map of client vs actions */
        CLIENT_0: []string{"Detect-0", "Safety-chk-0", "Mitigate-0", "Mitigate-2"},
        CLIENT_1: []string{"Detect-1", "Safety-chk-1", "Mitigate-1", "Detect-2", "Safety-chk-2"},
    },
    { /* Map of client vs actions */
        CLIENT_0: []string{"Detect-0", "Safety-chk-0"},
        CLIENT_1: []string{"Detect-1", "Safety-chk-1", "Mitigate-1"},
    },
    { /* Map of client vs actions */
        CLIENT_1: []string{"Detect-1", "Safety-chk-1", "Mitigate-1"},
    },
    { /* Map of client vs actions */ },
}

type activeActionsList_t map[string]ActiveActionInfo_t

var expActiveActions = make([]activeActionsList_t, len(expRegistrations))

func initActive() {
    if len(expActiveActions[0]) > 0 {
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
                    lst[a] = ActiveActionInfo_t{
                        Action: a, Client: cl, Timeout: c.Timeout}
                }
            }
        }
    }
}

var testCollections = make(testCollections_t)

func init() {
    testCollections["registrations_test"] = &testCollectionEntry_t{
        desc:     "Register & De-register testing",
        preSetup: []testCollectionId_t{}, /* none */
        testEntries: testEntriesList_t{
            0: {
                id:     REG_ACTION,
                clTx:   "",
                args:   []any{"xyz"},
                failed: true,
                desc:   "RegisterAction: Fail as before register client",
            },
            1: {
                id:     REG_CLIENT,
                clTx:   "iX",
                args:   []any{EMPTY_STR},
                failed: true,
                desc:   "RegisterClient: Fail for empty name",
            },
            2: {
                id:     REG_CLIENT,
                clTx:   "Bogus",
                args:   []any{CLIENT_0},
                failed: false,
                desc:   "RegisterClient to succeed",
            },
            3: {
                id:     REG_CLIENT,
                clTx:   "Bogus",
                args:   []any{CLIENT_0},
                failed: true,
                desc:   "register-client: Fail duplicate on same transport",
            },
            4: {
                id:     REG_CLIENT,
                clTx:   CLIENT_0, /* re-reg under new Tx. So succeed" */
                args:   []any{CLIENT_0},
                failed: false,
                desc:   "RegClient re-reg on new tx to succeed",
            },
            5: {
                id:     REG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{""},
                failed: true,
                desc:   "RegisterAction fail for empty name",
            },
            6: {
                id:     REG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Detect-0"},
                failed: false,
                desc:   "RegisterAction client-0/detect-0 succeeds",
            },
            7: {
                id:     REG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Detect-0"},
                failed: false,
                desc:   "Re-registerAction succeeds",
            },
            8: {
                id:     REG_CLIENT,
                clTx:   CLIENT_1,
                args:   []any{CLIENT_1},
                failed: false,
                desc:   "second Client reg to succeed",
            },
            9: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Detect-0"},
                failed: false,
                desc:   "RegAction: Succeed duplicate register on different client",
            },
            10: {
                id:     REG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Detect-0"},
                failed: false,
                desc:   "Duplicate action register on different client",
            },
            11: {
                id:     REG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Mitigate-0"},
                failed: false,
                desc:   "action register succeed",
            },
            12: {
                id:     REG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Mitigate-2"},
                failed: false,
                desc:   "action register succeed",
            },
            13: {
                id:     REG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Safety-chk-0"},
                failed: false,
                desc:   "action register succeed",
            },
            14: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Detect-1"},
                failed: false,
                desc:   "action register succeed",
            },
            15: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Safety-chk-1"},
                failed: false,
                desc:   "action register succeed",
            },
            16: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Mitigate-1"},
                failed: false,
                desc:   "action register succeed",
            },
            17: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Detect-2"},
                failed: false,
                desc:   "action register succeed",
            },
            18: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Safety-chk-2"},
                failed: false,
                desc:   "action register succeed",
            },
            19: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Disabled-0"},
                failed: true,
                desc:   "action register fail for disabled",
            },
            20: {
                id:   CHK_REG_ACTIONS,
                clTx: "",       /* Local verification */
                args: []any{0}, /* index into expRegistrations & expActiveActions */
                desc: "Verify local cache to succeed",
            },
            21: {
                id:     DEREG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Detect-2"},
                failed: false,
                desc:   "action deregister succeed",
            },
            22: {
                id:     DEREG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Safety-chk-2"},
                failed: false,
                desc:   "action deregister succeed",
            },
            23: {
                id:     DEREG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Mitigate-0"},
                failed: false,
                desc:   "action deregister succeed",
            },
            24: {
                id:     DEREG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Mitigate-2"},
                failed: false,
                desc:   "action deregister succeed",
            },
            25: {
                id:   DEREG_ACTION,
                clTx: CLIENT_0,
                args: []any{""},
                desc: "action deregister succeed for empty",
            },
            26: {
                id:   DEREG_ACTION,
                clTx: CLIENT_0,
                args: []any{"XXX"},
                desc: "action deregister succeed for non-existing",
            },
            27: {
                id:   CHK_REG_ACTIONS,
                clTx: "",       /* Local verification */
                args: []any{1}, /* index into expRegistrations & expActiveActions */
                desc: "Verify local cache to succeed",
            },
            28: {
                id:   DEREG_CLIENT,
                clTx: CLIENT_0,
                desc: "action deregister client succeed",
            },
            29: {
                id:   CHK_REG_ACTIONS,
                clTx: "",       /* Local verification */
                args: []any{2}, /* index into expRegistrations & expActiveActions */
                desc: "Verify local cache to succeed",
            },
            30: {
                id:   DEREG_CLIENT,
                clTx: CLIENT_1,
                desc: "action deregister client succeed",
            },
            31: {
                id:   CHK_REG_ACTIONS,
                clTx: "",       /* Local verification */
                args: []any{3}, /* index into expRegistrations & expActiveActions */
                desc: "Verify local cache to succeed",
            },
        },
        postCleanup: []testCollectionId_t{}, /* none */
    }

    testCollections["registrations_cleanup"] = &testCollectionEntry_t{
        /*
         * This is for shared use by other collections for cleanup.
         * Running this individually won't work, as this has de-reg only
         */
        desc:     "Post cleanup of all registrations",
        preSetup: []testCollectionId_t{}, /* none */
        testEntries: testEntriesList_t{
            0: {
                id:   DEREG_CLIENT,
                clTx: CLIENT_0,
                desc: "action deregister client succeed",
            },
            1: {
                id:   DEREG_CLIENT,
                clTx: CLIENT_1,
                desc: "action deregister client succeed",
            },
            2: {
                id:   CHK_REG_ACTIONS,
                clTx: "",       /* Local verification */
                args: []any{3}, /* index into expRegistrations & expActiveActions */
                desc: "Verify local cache to succeed",
            },
        },
        postCleanup: []testCollectionId_t{}, /* none */
    }

    testCollections["registrations_setup"] = &testCollectionEntry_t{
        /*
         * This is for shared use by other collections for setup.
         * Running this individually would work, but leave registrations behind as
         * orphaned, as test code drops all transports at the end of collection
         * run and hence these registrations are never reachable.
         */
        desc:     "Pre setup of all registrations",
        preSetup: []testCollectionId_t{}, /* none */
        testEntries: testEntriesList_t{
            100: {
                id:     REG_CLIENT,
                clTx:   CLIENT_0, /* re-reg under new Tx. So succeed" */
                args:   []any{CLIENT_0},
                failed: false,
                desc:   "RegClient to succeed",
            },
            102: {
                id:     REG_CLIENT,
                clTx:   CLIENT_1,
                args:   []any{CLIENT_1},
                failed: false,
                desc:   "second Client reg to succeed",
            },
            104: {
                id:     REG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Detect-0"},
                failed: false,
                desc:   "Reg Action to succeed",
            },
            106: {
                id:     REG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Mitigate-0"},
                failed: false,
                desc:   "action register succeed",
            },
            108: {
                id:   REG_ACTION,
                clTx: CLIENT_0,
                args: []any{"Mitigate-2"},

                desc: "action register succeed",
            },
            110: {
                id:     REG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Safety-chk-0"},
                failed: false,
                desc:   "action register succeed",
            },
            114: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Detect-1"},
                failed: false,
                desc:   "action register succeed",
            },
            116: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Safety-chk-1"},
                failed: false,
                desc:   "action register succeed",
            },
            118: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Mitigate-1"},
                failed: false,
                desc:   "action register succeed",
            },
            120: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Detect-2"},
                failed: false,
                desc:   "action register succeed",
            },
            122: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Safety-chk-2"},
                failed: false,
                desc:   "action register succeed",
            },
            124: {
                id:   CHK_REG_ACTIONS,
                clTx: "", /* Local verification */
                args: []any{0},
                desc: "Verify local cache to succeed",
            },
        },
        postCleanup: []testCollectionId_t{}, /* none */
    }

    testCollections["seq_empty"] = &testCollectionEntry_t{
        desc:     "A successful empty sequence with Detection only",
        preSetup: []testCollectionId_t{"registrations_setup"},
        testEntries: testEntriesList_t{
            /* Requests are expected in the same order as registration */
            130: {
                id:     REG_ACTION,
                clTx:   CLIENT_1,
                args:   []any{"Detect-3"},
                failed: false,
                desc:   "action detect-3 register succeed",
            },
            /* Requests are expected in the same order as registration */
            140: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            142: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  2,
                result: []any{&ActionRequestData{Action: "Detect-1"}},
                desc:   "Read server request for Detect-1",
            },
            144: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  3,
                result: []any{&ActionRequestData{Action: "Detect-2"}},
                desc:   "Read server request for Detect-2",
            },
            145: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  4,
                result: []any{&ActionRequestData{Action: "Detect-3"}},
                desc:   "Read server request for Detect-3",
            },
            146: {
                id:    SEND_RES,
                clTx:  CLIENT_1,
                seqId: 4,
                args:  []any{&ActionResponseData{Action: "Detect-3", AnomalyKey: "Key-Detect-3", Response: "Detect-3 detected"}},
                desc:  "Send res for detect3",
            },
            160: {
                id:    SEQ_COMPLETE,
                seqId: 4,
                desc:  "Verify seq complete",
            },
            164: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  5, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-3"}},
                desc:   "Read server request for Detect-3",
            },
            166: {
                /* Local engine level verification */
                id:   CHK_ACTIV_REQ,
                args: []any{"Detect-0", "Detect-1", "Detect-2", "Detect-3"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"},
    }

    testCollections["seq_success"] = &testCollectionEntry_t{
        desc:     "A successful Sequence",
        preSetup: []testCollectionId_t{"registrations_setup"},
        testEntries: testEntriesList_t{
            /* Requests are expected in the same order as registration */
            140: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            142: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  2,
                result: []any{&ActionRequestData{Action: "Detect-1"}},
                desc:   "Read server request for Detect-1",
            },
            144: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  3,
                result: []any{&ActionRequestData{Action: "Detect-2"}},
                desc:   "Read server request for Detect-2",
            },
            /* Test one full successful sequence. Detect-0 -> chk-0 -> Mit-0 */
            /* registrations_setup ha already verified initial requests received. */
            150: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-0", Response: "Detect-0 detected"}},
                desc:  "Send res for detect0",
            },
            152: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1,
                result: []any{&ActionRequestData{Action: "Safety-chk-0", Timeout: 1}},
                desc:   "Read server request for Safety-check-0",
            },
            154: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Safety-chk-0", Response: "Safety-chk-0 passed"}},
                desc:  "Send res for safety-chk-0",
            },
            156: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1,
                result: []any{&ActionRequestData{Action: "Mitigate-0", Timeout: -1}},
                desc:   "Read server request for Mitigate-0",
            },
            158: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Mitigate-0", Response: "Mitigate-0 passed"}},
                desc:  "Send res for Mitigate-0",
            },
            160: {
                id:    SEQ_COMPLETE,
                seqId: 1,
                desc:  "Verify seq complete",
            },
            162: {
                id:     NOTIFY_HB,
                clTx:   CLIENT_0,
                args:   []any{"XYZ", "Detect-0", "Mitigate-0"},
                result: []any{"Detect-0", "Mitigate-0"},
                desc:   "Notify heartbeats valid & invalid names",
            },
            164: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  4, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            166: {
                /* Local engine level verification */
                id:   CHK_ACTIV_REQ,
                args: []any{"Detect-0", "Detect-1", "Detect-2"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"},
    }

    testCollections["seq_mit_fail"] = &testCollectionEntry_t{
        desc:     "A failed Sequence in last action",
        preSetup: []testCollectionId_t{"registrations_setup"},
        testEntries: testEntriesList_t{
            /* Requests are expected in the same order as registration */
            140: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            142: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  2,
                result: []any{&ActionRequestData{Action: "Detect-1"}},
                desc:   "Read server request for Detect-1",
            },
            144: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  3,
                result: []any{&ActionRequestData{Action: "Detect-2"}},
                desc:   "Read server request for Detect-2",
            },
            /* Test one failed sequence at last action. Detect-0 -> chk-0 -> Mit-0 (fail) */
            /* registrations_setup ha already verified initial requests received. */
            150: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-0", Response: "Detect-0 detected"}},
                desc:  "Send res for detect0",
            },
            152: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1,
                result: []any{&ActionRequestData{Action: "Safety-chk-0", Timeout: 1}},
                desc:   "Read server request for Safety-check-0",
            },
            154: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Safety-chk-0", Response: "Safety-chk-0 passed"}},
                desc:  "Send res for safety-chk-0",
            },
            156: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1,
                result: []any{&ActionRequestData{Action: "Mitigate-0", Timeout: -1}},
                desc:   "Read server request for Mitigate-0",
            },
            158: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Mitigate-0", ResultCode: 120, ResultStr: "Blah Blah"}},
                desc:  "Send res for Mitigate-0",
            },
            160: {
                id:    SEQ_COMPLETE,
                args:  []any{&ActionResponseData{Action: "Mitigate-0", ResultCode: 120, ResultStr: "Blah Blah"}},
                seqId: 1,
                desc:  "Verify seq complete",
            },
            162: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  4, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            164: {
                /* Local engine level verification */
                id:   CHK_ACTIV_REQ,
                args: []any{"Detect-0", "Detect-1", "Detect-2"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"},
    }

    testCollections["seq_safety_fail"] = &testCollectionEntry_t{
        desc:     "A failed Sequence in second action",
        preSetup: []testCollectionId_t{"registrations_setup"},
        testEntries: testEntriesList_t{
            /* Requests are expected in the same order as registration */
            140: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            142: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  2,
                result: []any{&ActionRequestData{Action: "Detect-1"}},
                desc:   "Read server request for Detect-1",
            },
            144: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  3,
                result: []any{&ActionRequestData{Action: "Detect-2"}},
                desc:   "Read server request for Detect-2",
            },
            /* Test one failed sequence at last action. Detect-0 -> chk-0 -> Mit-0 (fail) */
            /* registrations_setup ha already verified initial requests received. */
            150: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-0", Response: "Detect-0 detected"}},
                desc:  "Send res for detect0",
            },
            152: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1,
                result: []any{&ActionRequestData{Action: "Safety-chk-0", Timeout: 1}},
                desc:   "Read server request for Safety-check-0",
            },
            154: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Safety-chk-0", ResultCode: 120, ResultStr: "Safe not"}},
                desc:  "Send res for safety-chk-0",
            },
            160: {
                id:    SEQ_COMPLETE,
                args:  []any{&ActionResponseData{ResultCode: 120, ResultStr: "Safe not"}},
                seqId: 1,
                desc:  "Verify seq complete",
            },
            162: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  4, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            164: {
                /* Local engine level verification */
                id:   CHK_ACTIV_REQ,
                args: []any{"Detect-0", "Detect-1", "Detect-2"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"},
    }

    testCollections["detect_fail"] = &testCollectionEntry_t{
        desc:     "First/detection action fails",
        preSetup: []testCollectionId_t{"registrations_setup"},
        testEntries: testEntriesList_t{
            /* Requests are expected in the same order as registration */
            140: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            142: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  2,
                result: []any{&ActionRequestData{Action: "Detect-1"}},
                desc:   "Read server request for Detect-1",
            },
            144: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  3,
                result: []any{&ActionRequestData{Action: "Detect-2"}},
                desc:   "Read server request for Detect-2",
            },
            /* Test one failed sequence at last action. Detect-0 -> chk-0 -> Mit-0 (fail) */
            /* registrations_setup ha already verified initial requests received. */
            150: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Detect-0", ResultCode: 120, ResultStr: "Detect failed"}},
                desc:  "Send res for detect0",
            },
            160: {
                id:    SEQ_COMPLETE,
                args:  []any{&ActionResponseData{ResultCode: 120, ResultStr: "Detect failed"}},
                seqId: 1,
                desc:  "Verify seq complete",
            },
            162: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  4, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            164: {
                /* Local engine level verification */
                id:   CHK_ACTIV_REQ,
                args: []any{"Detect-0", "Detect-1", "Detect-2"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"},
    }

    testCollections["sequence-timeout"] = &testCollectionEntry_t{
        desc:     "Let sequence timeout",
        preSetup: []testCollectionId_t{"registrations_setup"},
        testEntries: testEntriesList_t{
            /* Requests are expected in the same order as registration */
            140: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            142: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  2,
                result: []any{&ActionRequestData{Action: "Detect-1"}},
                desc:   "Read server request for Detect-1",
            },
            144: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  3,
                result: []any{&ActionRequestData{Action: "Detect-2"}},
                desc:   "Read server request for Detect-2",
            },
            /* Test one full successful sequence. Detect-0 -> chk-0 -> Mit-0 */
            /* registrations_setup ha already verified initial requests received. */
            150: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-0", Response: "Detect-0 detected"}},
                desc:  "Send res for detect0",
            },
            152: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1,
                result: []any{&ActionRequestData{Action: "Safety-chk-0", Timeout: 1}},
                desc:   "Read server request for Safety-check-0",
            },
            154: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Safety-chk-0", Response: "Safety-chk-0 passed"}},
                desc:  "Send res for safety-chk-0",
            },
            156: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1,
                result: []any{&ActionRequestData{Action: "Mitigate-0", Timeout: -1}},
                desc:   "Read server request for Mitigate-0",
            },
            157: {
                id:   PAUSE, /* Pause with no response to simulate timeout */
                args: []any{4},
                desc: "Simple sleep with heartbeats",
            },
            160: {
                id:    SEQ_COMPLETE,
                args:  []any{&ActionResponseData{ResultCode: 4106, ResultStr: "From Process timeout"}},
                seqId: 1,
                desc:  "Verify seq complete",
            },
            164: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  4, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            166: {
                /* Local engine level verification */
                id:   CHK_ACTIV_REQ,
                args: []any{"Detect-0", "Detect-1", "Detect-2", "Mitigate-0"},
                desc: "Verify active requests only for actions in args",
            },
            168: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Mitigate-0", ResultCode: 120, ResultStr: "Blah Blah"}},
                desc:  "Send res for Mitigate-0",
            },
            170: {
                /* Local engine level verification */
                id:   CHK_ACTIV_REQ,
                args: []any{"Detect-0", "Detect-1", "Detect-2"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"},
    }

    testCollections["multi-sequence-block"] = &testCollectionEntry_t{
        /*
         * "Detect-0" returns good. Expect "Safety-chk-0"; Sleep forever; req expect to timeout
         * "Detect-2" & "Detect-1" returns good;
         * Per pri, "Detect-2" seq resumes, but blocked by "Safety-chk-0". Let bind2 timeout
         * Now bind-1 resumes and succeeds.
         */
        desc:     "Block seq by pending actions",
        preSetup: []testCollectionId_t{"registrations_setup"},
        testEntries: testEntriesList_t{
            /* Requests are expected in the same order as registration */
            140: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            142: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  2,
                result: []any{&ActionRequestData{Action: "Detect-1"}},
                desc:   "Read server request for Detect-1",
            },
            144: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  3,
                result: []any{&ActionRequestData{Action: "Detect-2"}},
                desc:   "Read server request for Detect-2",
            },
            /* Let Detect-0 respond. and be stuck at  chk-0 */
            150: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-0", Response: "Detect-0 detected"}},
                desc:  "Send res for detect0",
            },
            /* Send responses for Detect-1 & detect-2. These will pend as Detect-0 is active */
            158: {
                id:    SEND_RES,
                clTx:  CLIENT_1,
                seqId: 2,
                args:  []any{&ActionResponseData{Action: "Detect-1", AnomalyKey: "Key-Detect-1", Response: "Detect-1 detected"}},
                desc:  "Send res for detect-1",
            },
            160: {
                id:    SEND_RES,
                clTx:  CLIENT_1,
                seqId: 3,
                args:  []any{&ActionResponseData{Action: "Detect-2", AnomalyKey: "Key-Detect-2", Response: "Detect-2 detected"}},
                desc:  "Send res for detect-2",
            },
            162: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1,
                result: []any{&ActionRequestData{Action: "Safety-chk-0", Timeout: 1}},
                desc:   "Read server request for Safety-check-0",
            },
            164: {
                id:   PAUSE, /* Psuse to let bind-0 times out */
                args: []any{3},
                desc: "Simple sleep with heartbeats",
            },
            166: { /* Bind-0 seq completes */
                id:    SEQ_COMPLETE,
                args:  []any{&ActionResponseData{ResultCode: 4106, ResultStr: "From Process timeout"}},
                seqId: 1,
                desc:  "Verify seq complete",
            },
            /* Expect bind-2 to start as has higher pri than bind-1; Expect call for chk-2 */
            172: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  3,
                result: []any{&ActionRequestData{Action: "Safety-chk-2", Timeout: 1}},
                desc:   "Read server request for Safety-check-2",
            },
            174: {
                id:    SEND_RES,
                clTx:  CLIENT_1,
                seqId: 3,
                args:  []any{&ActionResponseData{Action: "Safety-chk-2", Response: "Safety-chk-2 passed"}},
                desc:  "Send res for safety-chk-2",
            },
            /* Now bind-2 blocks at safety-chk-0 as it is busy */
            176: {
                id:   PAUSE, /* Pause to let bind-2 times out */
                args: []any{6},
                desc: "Simple sleep with heartbeats",
            },
            178: { /* Read pending Detect-0 for client-0 */
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  4, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            180: { /* Expect bind-2 to complete due to timeout */
                id:    SEQ_COMPLETE,
                args:  []any{&ActionResponseData{ResultCode: 4106, ResultStr: "From Process timeout"}},
                seqId: 3,
                desc:  "Verify seq complete",
            },
            /* Seq complete sends out request for Detect-2 synchronously */
            181: { /* Read pending Detect-0 for client-0 */
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  5, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-2"}},
                desc:   "Read server request for Detect-2",
            },
            /* Expect bind 1 to resume. This can proceed as it does not use chk-0 */
            182: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  2,
                result: []any{&ActionRequestData{Action: "Safety-chk-1", Timeout: 7}},
                desc:   "Read server request for Safety-check-1",
            },
            184: {
                id:    SEND_RES,
                clTx:  CLIENT_1,
                seqId: 2,
                args:  []any{&ActionResponseData{Action: "Safety-chk-1", Response: "Safety-chk-1 passed"}},
                desc:  "Send res for safety-chk-1",
            },
            186: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  2,
                result: []any{&ActionRequestData{Action: "Mitigate-1", Timeout: -1}},
                desc:   "Read server request for Mitigate-1",
            },
            188: {
                id:    SEND_RES,
                clTx:  CLIENT_1,
                seqId: 2,
                args:  []any{&ActionResponseData{Action: "Mitigate-1", Response: "Mitigate-1 passed"}},
                desc:  "Send res for Mitigate-1",
            },
            190: { /* Bind 1 completes */
                id:    SEQ_COMPLETE,
                seqId: 2,
                desc:  "Verify seq complete",
            },
            /* Seq complete sends out request for Detect-1 synchronously */
            194: { /* Read pending Detect-1 for client-0 */
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  6, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-1"}},
                desc:   "Read server request for Detect-0",
            },
            196: {
                /* Local engine level verification */
                id:   CHK_ACTIV_REQ,
                args: []any{"Detect-0", "Detect-1", "Detect-2", "Safety-chk-0"},
                desc: "Verify active requests only for actions in args",
            },
            /* Send res for blocked response */
            198: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Safety-chk-0", Response: "Safety-chk-0 passed"}},
                desc:  "Send res for safety-chk-0",
            },
            200: {
                /* Local engine level verification */
                id:   CHK_ACTIV_REQ,
                args: []any{"Detect-0", "Detect-1", "Detect-2"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"},
    }

    testCollections["sequence-block-action-reregister"] = &testCollectionEntry_t{
        /*
         * "Detect-0" returns good. "Safety-chk-0" Sleep forever; req expect to timeout; seq complete
         * "Detect-0" returns good. "Safety-chk-0" can't be called; seq expect to timeout
         * "Safety-chk-0" re-register.
         * "Detect-0" returns good. "Safety-chk-0" & "mitigation-0" return good; seq compltes
         * "Detec-0" re-raised
         */
        desc:     "Blocked req; re-register to revive.",
        preSetup: []testCollectionId_t{"registrations_setup"},
        testEntries: testEntriesList_t{
            /* Requests are expected in the same order as registration */
            140: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            142: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  2,
                result: []any{&ActionRequestData{Action: "Detect-1"}},
                desc:   "Read server request for Detect-1",
            },
            144: {
                id:     RECV_REQ,
                clTx:   CLIENT_1,
                seqId:  3,
                result: []any{&ActionRequestData{Action: "Detect-2"}},
                desc:   "Read server request for Detect-2",
            },
            /* Let Detect-0 respond. and be stuck at  chk-0 */
            150: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 1,
                args:  []any{&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-0", Response: "Detect-0 detected"}},
                desc:  "Send res for detect0",
            },
            152: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  1,
                result: []any{&ActionRequestData{Action: "Safety-chk-0", Timeout: 1}},
                desc:   "Read server request for Safety-check-0",
            },
            154: {
                id:   PAUSE, /* Psuse to let bind-0 times out */
                args: []any{2},
                desc: "Simple sleep with heartbeats",
            },
            156: { /* Bind-0 seq completes */
                id:    SEQ_COMPLETE,
                args:  []any{&ActionResponseData{ResultCode: 4106, ResultStr: "From Process timeout"}},
                seqId: 1,
                desc:  "Verify seq complete",
            },
            /* Detect-0 re-raised as last seq completes */
            162: { /* Read pending Detect-0 for client-0 */
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  4, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            164: {
                /* Local engine level verification */
                id:   CHK_ACTIV_REQ,
                args: []any{"Detect-0", "Detect-1", "Detect-2", "Safety-chk-0"},
                desc: "Verify active requests only for actions in args",
            },
            166: {
                id:    SEND_RES, /* Detect-0 responds */
                clTx:  CLIENT_0,
                seqId: 4,
                args:  []any{&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-1", Response: "Detect-0 detected"}},
                desc:  "Send res for detect0",
            },
            /* But seq blocked as safety-chk-0 is busy from previous sequence */
            168: {
                id:   PAUSE, /* Psuse to let bind-0 times out */
                args: []any{4},
                desc: "Simple sleep with heartbeats",
            },
            170: { /* Expect bind-2 to complete due to timeout */
                id:    SEQ_COMPLETE,
                args:  []any{&ActionResponseData{ResultCode: 4106, ResultStr: "From Process timeout"}},
                seqId: 4,
                desc:  "Verify seq complete",
            },
            /* Seq complete sends out request for Detect-0 synchronously */
            172: { /* Read pending Detect-0 for client-0 */
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  5, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            174: { /* Re-register should de-register first, which drops pending active req */
                id:     REG_ACTION,
                clTx:   CLIENT_0,
                args:   []any{"Safety-chk-0"},
                failed: false,
                desc:   "action register succeed",
            },
            /* send response for detect-0; This should trigger successful seq */
            180: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 5,
                args:  []any{&ActionResponseData{Action: "Detect-0", AnomalyKey: "Key-Detect-0", Response: "Detect-0 detected"}},
                desc:  "Send res for detect0",
            },
            182: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  5,
                result: []any{&ActionRequestData{Action: "Safety-chk-0", Timeout: 1}},
                desc:   "Read server request for Safety-check-0",
            },
            184: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 5,
                args:  []any{&ActionResponseData{Action: "Safety-chk-0", Response: "Safety-chk-0 passed"}},
                desc:  "Send res for safety-chk-0",
            },
            186: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  5,
                result: []any{&ActionRequestData{Action: "Mitigate-0", Timeout: -1}},
                desc:   "Read server request for Mitigate-0",
            },
            188: {
                id:    SEND_RES,
                clTx:  CLIENT_0,
                seqId: 5,
                args:  []any{&ActionResponseData{Action: "Mitigate-0", Response: "Mitigate-0 passed"}},
                desc:  "Send res for Mitigate-0",
            },
            190: {
                id:    SEQ_COMPLETE,
                seqId: 5,
                desc:  "Verify seq complete",
            },
            194: {
                id:     RECV_REQ,
                clTx:   CLIENT_0,
                seqId:  6, /* Use non-zero, default is 0. Make it explicit */
                result: []any{&ActionRequestData{Action: "Detect-0"}},
                desc:   "Read server request for Detect-0",
            },
            196: {
                /* Local engine level verification */
                id:   CHK_ACTIV_REQ,
                args: []any{"Detect-0", "Detect-1", "Detect-2"},
                desc: "Verify active requests only for actions in args",
            },
        },
        postCleanup: []testCollectionId_t{"registrations_cleanup"},
    }

}

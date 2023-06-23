package engine

import (
    "bytes"
    "flag"
    . "lom/src/lib/lomcommon"
    . "lom/src/lib/lomipc"
    "os"
    "os/signal"
    "syscall"
)

type engine_t struct {
    tx *LoMTransport

    chTrack     chan int /* Track engine main loop state */
    chTerminate chan int /* Terminate engine */

    chClientReq   chan *LoMRequestInt
    chClReadAbort chan interface{}
}

var cfgPath = ""
var engineInst *engine_t

/*
 * Read client requests via engine Lib API and route it to
 * engine's main loop.
 */
func (p *engine_t) readRequest() {

    go func() {
        defer func() {
            /* No more client requests to read. So trigger engine close */
            p.close()
        }()

        /*
         * In forever read loop until aborted.
         * Send read requests via chan to main loop.
         * Watch abort chan to quit.
         */
        for {
            /* Blocking read until request or error or signal in chClReadAbort */
            req, err := p.tx.ReadClientRequest(p.chClReadAbort)

            if req != nil {
                /* Select can block write into p.chClientReq, so watch chClReadAbort too */
                select {
                case p.chClientReq <- req:
                    break
                case <-p.chClReadAbort:
                    return
                }
            } else {
                LogError("Failed to read request. err(%v)", err)
                return
            }
        }
    }()
}

func (p *engine_t) runLoop() {
    /*
     * Wait for
     *      signal to refresh
     *      request from client
     *      Internal timer for outstanding request's/seq's timeout processing
     */

    defer func() {
        /* Indicate loop end */
        p.chTrack <- 1

        /* TRigger any readloop to abort */
        if len(p.chClReadAbort) < cap(p.chClReadAbort) {
            p.chClReadAbort <- "Aborted"
        }

        /* Nullify global inst */
        engineInst = nil
    }()

    /* Handle signal for config update & terminate */
    chSignal := make(chan os.Signal, 3)
    signal.Notify(chSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

    /*
     * kick off reading client requests
     *
     * p.chClientReq - The read loop sends read requests via this chan to main loop.
     * p.chClReadAbort - Way to abort the read loop.
     */
    p.chClientReq = make(chan *LoMRequestInt, 1)
    p.chClReadAbort = make(chan interface{}, 2)
    p.readRequest()

    /*
     * Initialize seq handler
     *
     * The seq handler is invoked for any response from clients via this
     * runLoop Go routine. To ensure that any timer based processing too
     * happens in the same context, it uses this routine for any of its
     * timer call back needs.
     * On any async timer firing, it intimates this loop via chSeqHandler channel
     * and the main loop invokes sequence handler.
     *
     * This way any logical processing by sequence handler is via the context
     * of single Go routine only.
     */
    chSeqHandler := make(chan int64, 2)
    InitSeqHandler(chSeqHandler)

    server := GetServerReqHandler()

    /* Intimate the start of loop */
    p.chTrack <- 0
loop:
    for {
        select {
        case msg := <-p.chClientReq:
            server.processRequest(msg)

        case <-chSeqHandler:
            GetSeqHandler().processTimeout()

        case sig := <-chSignal:
            switch sig {
            case syscall.SIGHUP, syscall.SIGINT:
                /*
                 * Reload.
                 * NOTE: Any currently active sequence will not be affected
                 * On any error, continues to use last loaded values.
                 */
                InitConfigPath(cfgPath)

            case syscall.SIGTERM:
                break loop
            }
        case <-p.chTerminate:
            break loop
        }
    }
}

func (p *engine_t) close() {
    if len(p.chTerminate) < cap(p.chTerminate) {
        p.chTerminate <- 1
    }
}

func EngineStartup(path string) (*engine_t, error) {
    cfgPath = path

    if engineInst != nil {
        LogError("Duplicate EngineStartup")
        return engineInst, nil
    }

    /* Init/Load config */
    if err := InitConfigPath(cfgPath); err != nil {
        return nil, LogError("Failed to read config; (%s)", cfgPath)
    }

    /* Init engine context that saves all client registrations */
    InitRegistrations()
    tx, err := ServerInit()
    if err != nil {
        return nil, LogError("Failed to call ServerInit")
    }

    chTrack := make(chan int, 2)     /* To track start/end of loop */
    chTerminate := make(chan int, 1) /* To force terminate a loop */
    engineInst := &engine_t{tx: tx, chTrack: chTrack, chTerminate: chTerminate}
    go engineInst.runLoop()

    /* Wait for loop start */
    <-chTrack
    return engineInst, nil
}

func startUp(progname string, args []string) (*engine_t, error) {

    /* Parse args for path */
    p := ""
    flags := flag.NewFlagSet(progname, flag.ContinueOnError)
    var buf bytes.Buffer
    flags.SetOutput(&buf)

    flags.StringVar(&p, "path", "", "Config files path")

    err := flags.Parse(args)
    if err != nil {
        return nil, LogError("Failed to parse (%v); details(%s)", args, buf.String())
    }
    return EngineStartup(p)
}

func Main() {
    engine, err := startUp(os.Args[0], os.Args[1:])
    if err != nil {
        LogError("Engine aborting ...")
        return
    }

    <-engine.chTrack
    LogDebug("Loop ended")

    LogInfo("Engine exiting...")
}

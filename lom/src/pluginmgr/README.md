# LoM Plugin Manager

The LoM Plugin Manager is a program that manages plugins for the LoM system. It handles actions, process configurations, and coordinates with the LoM Engine.

## Command Line Arguments

The following command line arguments are expected:

- `proc_id`: The process ID of the plugin manager (mandatory).
- `syslog_level`: The syslog level for logging (optional, default: LOG_DEBUG).

Example to start Plugin Manager:

- LOM_CONF_LOCATION="..." LOM_RUN_MODE="PROD" ./LoMpluginMgr -proc_id="proc_0" -syslog_level=5

## Environment Variables

The following environment variables can be set:

- `LOM_RUN_MODE`: LoM run mode (optional, default: "test"). Set it to "PROD" for production.
- `LOM_CONF_LOCATION`: LoM configuration file location (mandatory).

## Configuration Files (relative to LOM_CONF_LOCATION)

The following configuration files are required:

- `actions.conf.json`: Action plugins config file (mandatory).
- `procs.conf.json`: Proc-specific configuration file (mandatory).
- `globals.conf.json`: LoM system-wide configuration file (mandatory).

## Signals Handled

The LoM Plugin Manager handles the following signal:

- `SIGTERM`: Stops the plugin manager and exits gracefully.

## Partner Application

The LoM Plugin Manager requires the following partner application:

- LoMEngine: LoM Engine running on the same device.
  - Mandatory in production mode.
  - Optional in test mode.

## Logging

The Plugin Manager logs to syslog. The log level is set by the `syslog_level` command line argument.
The default log level is `LOG_DEBUG`.
The logs are written to the system log file, e.g., `/var/log/syslog`.
The sample log format is as follows:
-   pluginmgr/pluginmgr_main.go:64:plugin_mgr : Starting Plugin Manager

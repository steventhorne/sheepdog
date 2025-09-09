# Sheepdog

Sheepdog is a terminal user interface for launching and controlling multiple commands during development. It lets you run several processes in the same window, view their output, and stop or restart them at any time.

## Features

- Runs multiple commands defined in a JSON configuration file
- Color coded status for each process (idle, running, ready, errored)
- Scrollable log view for the selected process
- Simple key bindings to start or stop commands

## Installation

```bash
go install github.com/steventhorne/sheepdog@latest
```

Alternatively, download a binary from the releases page.

## Configuration

Create a `.sheepdog.json` file in your project directory that lists the processes you want to manage.

Each process in the configuration is represented by a JSON object with the following fields:

```json
{
  "processes": [
    {
      "name": "web",                           // required
      "autorun": true,                         // optional
      "cwd": "/opt/app",                       // optional
      "groupType": "parallel",                 // required if this is a process group
      "children": [                            // required if this is a process group
        {
          "name": "server",
          "command": ["./bin/server", "-p80"], // required unless this is a process group
          "readyRegexp": "listening on",       // optional
        },
        {
          "name": "worker-1",
          "command": ["./bin/worker"]
        }
      ]
    }
  ]
}
```

Field Reference

| Field         | Type                     | Used in | Required | Description                                                                                                                                |
| ------------- | ------------------------ | ------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `name`        | string                   | both    | yes      | Unique identifier for the process or group.                                                                                                |
| `command`     | array of string          | process | yes      | Command and arguments to run the process. **Required for standalone processes; ignored for process groups.**                               |
| `autorun`     | boolean                  | both    | no       | If true, the process is started automatically on launch. Defaults to `false`.                                                              |
| `cwd`         | string                   | both    | no       | Working directory in which to run the process.                                                                                             |
| `readyRegexp` | string (regex)           | process | no       | Regular expression to match against process output. Marks the process as "ready" when matched.                                             |
| `children`    | array of `ProcessConfig` | group   | yes      | Recursive list of child processes. **Required for process groups; omitted for standalone processes.**                                      |
| `groupType`   | string                   | group   | yes      | Defines the type of process group (e.g., `"parallel"`, `"sequential"`). **Required for process groups; omitted for standalone processes.** |

## Usage

Run `sheepdog` in the directory containing `.sheepdog.json`. The left pane shows your processes; the right pane displays the log of the selected one.

Key bindings:

- `j` / `k`, arrow keys, or scroll wheel - move up and down process list
- `ctrl+d` / `ctrl+u` - scroll up and down the log view
- `r` – run the selected process
- `x` – kill the selected process
- `enter` - focus on the selected process or expands/collapses the selected group
- `ctrl+c` – quit the application

## Logging

Sheepdog buffers process output to avoid blocking commands when the UI is busy.
Up to 1024 log lines are kept in memory; if the buffer fills, additional lines
are dropped until space becomes available. This keeps processes responsive at
the cost of potentially missing some log output.

## License

Distributed under the terms of the GNU General Public License v3. See the `LICENSE` file for the full license text.

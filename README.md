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

```json
{
  "processes": [
    {
      "name": "server",
      "command": ["go", "run", "./cmd/server"],
      "autorun": true,
      "cwd": "./"
    },
    {
      "name": "client",
      "command": ["npm", "run", "dev"],
      "autorun": false,
      "cwd": "web"
    }
  ]
}
```

Fields in each entry:

- `name` – label shown in the UI
- `command` – command and arguments to execute
- `autorun` – whether to start automatically
- `cwd` – working directory for the command

## Usage

Run `sheepdog` in the directory containing `.sheepdog.json`. The left pane shows your processes; the right pane displays the log of the selected one.

Key bindings:

- `j` / `k`, arrow keys, or scroll wheel - move up and down process list
- `ctrl+d` / `ctrl+u` - scroll up and down the log view
- `r` – run the selected process
- `x` – kill the selected process
- `enter` - focus on the selected process
- `ctrl+c` – quit the application

## Logging

Sheepdog buffers process output to avoid blocking commands when the UI is busy.
Up to 1024 log lines are kept in memory; if the buffer fills, additional lines
are dropped until space becomes available. This keeps processes responsive at
the cost of potentially missing some log output.

## License

Distributed under the terms of the GNU General Public License v3. See the `LICENSE` file for the full license text.

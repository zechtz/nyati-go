# Nyatictl

**Nyatictl** is a remote server automation and deployment tool written in Go, inspired by Capistrano. It allows you to define tasks in a YAML configuration file and execute them on remote servers via SSH. With support for concurrent task execution, variable substitution, and interactive prompts, Nyatictl is ideal for automating deployments, server maintenance, and other remote operations.

## Features

- **Simple Configuration**: Define hosts and tasks in a single `nyati.yaml` or `nyati.yml` file.
- **SSH-Based Execution**: Run commands on remote servers securely via SSH.
- **Concurrent Execution**: Execute tasks across multiple hosts in parallel using Go’s goroutines.
- **Task Filtering**: Run specific tasks with the `--task` flag.
- **Variable Substitution**: Use placeholders like `${appname}` and `${release_version}` in commands.
- **Interactive Prompts**: Support for `sudo` commands with `askpass` and retry logic for failed tasks.
- **Single Binary**: Compile to a single binary for easy distribution—no runtime dependencies.
- **User-Friendly Output**: Spinners and clear success/failure messages enhance the CLI experience.

## Installation

### Prerequisites

- **Go 1.21+**: Required to build the tool.
- **SSH Access**: Ensure you have SSH access to your target servers with either password or key-based authentication.
- **YAML File**: A `nyati.yaml` or `nyati.yml` file in your working directory (or specify a custom path with `-c`).

### Build from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/zechtz/nyatictl.git
   cd nyatictl
   ```
2. Build the binary

```bash
go build -o nyatictl
```

3. (Optional) Move the binary to a directory in your $PATH for global access:

```bash
sudo mv nyatictl /usr/local/bin/
```

### Download Prebuilt Binary

Prebuilt binaries for common platforms are available in the [Releases](https://github.com/zechtz/nyatictl/releases) section. Download the appropriate binary for your system, make it executable, and move it to your `$PATH`:

```bash
chmod +x nyatictl
sudo mv nyatictl /usr/local/bin/
```

### Usage

Nyatictl uses a YAML configuration file to define hosts and tasks. By default, it looks for nyati.yaml or nyati.yml in the current directory. You can specify a custom config file with the -c flag.

### Basic Commands

#### Deploy to All Hosts:

```bash
nyatictl deploy all
```

Runs all tasks (excluding lib tasks) on all hosts defined in the config.

#### Deploy to a Specific Host:

```bash
nyatictl deploy server1
```

Runs all tasks (excluding lib tasks) on server1.

#### Run a Specific Task:

```bash
nyatictl deploy server1 --task clean

```

Runs only the clean task on server1.

. Include Library Tasks

```bash
nyatictl deploy all --include-lib

```

Includes tasks marked with lib: true when running all tasks.

#### Show Help:

```bash
nyatictl --help
```

Displays usage information and available flags.

## Database Management

### Generate Migration:

```bash
nyatictl db generate add_new_column

```

Creates a new migration file with timestamp in the db/migrations directory.

### Run Migrations

```bash
nyatictl db migrate
```

Applies all pending database migrations.

### Check Migration Status:

```bash
nyatictl db status
```

Shows the status of applied and pending migrations.

### Roll Back Migration:

```bash
nyatictl db rollback
```

Reverts the most recent migration.

### Flags

| Flag            | Shorthand | Description                                      | Default                     |
| --------------- | --------- | ------------------------------------------------ | --------------------------- |
| `--config`      | `-c`      | Path to config file                              | `nyati.yaml` or `nyati.yml` |
| `deploy`        |           | Host to deploy tasks on (e.g., `all`, `server1`) | None                        |
| `--task`        |           | Specific task to run (e.g., `clean`)             | None (runs all tasks)       |
| `--include-lib` |           | Include tasks marked as `lib`                    | `false`                     |
| `--debug`       | `-d`      | Enable debug output (shows commands and output)  | `false`                     |
| `--help`        | `-h`      | Show help                                        | N/A                         |

## Web UI Mode

In addition to the CLI, Nyatictl also includes a built-in **web UI** for interactive deployments and task execution. The UI is fully embedded into the binary — no Node.js or frontend server required.

### Launch the Web UI

To start the web interface, run:

```bash
nyatictl --web
```

By default, this will start the server on port 8080 and serve the UI at:

```bash
http://localhost:8080

### Default Login Credentials

The Web UI includes authentication to secure your deployments. Use the following default credentials:

- **Email**: `admin@example.com`
- **Password**: `secret`

⚠️ **Important**: Change these credentials after first login for security purposes.
```

The React frontend is embedded at compile time using Go’s embed package — no external files or dependencies are required to serve the UI.

### Example (Custom Config Path & Port)

```bash
nyatictl --web --port 3000 --configs-path ./data/configs.json --log-path ./logs/ui.log
```

### UI Features

📂 View, create, and edit configuration entries

🚀 Trigger deployments to individual or all hosts

✅ Run specific tasks from the config file

🔁 Real-time WebSocket log streaming per session

🧩 Persistent logs are saved to file (--log-path) and streamed to the UI

### Where Are UI Configs Stored?

The web UI uses a `configs.json` file to manage saved deployment configurations. You can customize its location using:

```bash

--configs-path ./data/configs.json
```

If the file does not exist, Nyatictl will create it with an empty array:

### Configuration

Nyatictl uses a YAML file (nyati.yaml or nyati.yml) to define hosts, tasks, and parameters. Below is an example configuration:

```yml
version: "0.1.2"
appname: "myapp"
hosts:
  server1:
    host: "example.com"
    username: "user"
    password: "secret" # or use private_key for key-based auth
    # private_key: "/path/to/key.pem"
    # envfile: ".env"  # Optional: Load environment variables from a file
params:
  env: "prod"
tasks:
  - name: clean
    message: older deployments cleaned
    cmd: ls -dt1 */ | tail -n +5 | xargs rm -rf
    dir: /var/www/html/${appname}/releases
    expect: 0
    output: true
    lib: true
  - name: new_release
    cmd: mkdir -p /var/www/html/${appname}/releases/${release_version}
    expect: 0
  - name: check_disk
    cmd: df -h /
    expect: 0
    output: true
    message: Disk usage checked
```

# Configuration Fields

### Configuration Fields

- **version**: The config version (must be compatible with the Nyatictl version, e.g., `0.1.2`).
- **appname**: The name of your application (used in variable substitution).
- **hosts**: A map of host names to host configurations.
  - `host`: The hostname or IP address of the server.
  - `username`: The SSH username.
  - `password`: The SSH password (optional if using `private_key`).
  - `private_key`: Path to the SSH private key file (optional if using `password`).
  - `envfile`: Path to an environment file to load variables (optional).
- **params**: A map of custom parameters for variable substitution (e.g., `${env}`).
- **tasks**: A list of tasks to execute.
  - `name`: The task name (required).
  - `cmd`: The shell command to run (required).
  - `dir`: Directory to change to before running the command (optional).
  - `expect`: Expected exit code (default: 0).
  - `message`: Message to display on success (optional).
  - `retry`: Prompt to retry if the task fails (default: `false`).
  - `askpass`: Enable PTY for `sudo` commands requiring a password (default: `false`).
  - `lib`: Mark the task as a library task, skipped unless `--task` or `--include-lib` is used (default: `false`).
  - `output`: Display the command’s output on success (default: `false`).
  - `depends_on`: List of task names that must be executed before this task (optional). Tasks are executed in topological order based on dependencies.

# Variable Substitution

Nyatictl supports variable substitution in `cmd`, `dir`, and `message` fields. Available variables:

- `${appname}`: The `appname` from the config.
- `${release_version}`: A timestamp generated at runtime (Unix milliseconds).
- `${key}`: Any key defined in the `params` section (e.g., `${env}`).

#### Example

```yml
cmd: echo "Deploying ${appname} to ${env}"
```

If appname: "myapp" and params: { env: "prod" }, this becomes:

```bash
echo "Deploying myapp to prod"
```

### Database Migrations

Nyatictl includes a robust database migration system with:

- Versioned Migrations: Timestamp-based ordering ensures consistent execution
- UP/DOWN Support: Each migration can be applied and rolled back
- Transaction Support: Changes are atomic and consistent
- CLI Commands: Generate, apply, and roll back migrations easily

### Migration File Format

```sq
-- UP
-- SQL statements to apply the migration
ALTER TABLE configs ADD COLUMN user_id INTEGER DEFAULT 1;

-- DOWN
-- SQL statements to roll back the migration
CREATE TABLE configs_backup AS SELECT id, name, description, path, status FROM configs;
-- Additional rollback commands...
```

### Examples

### Deploy a Web Application

1. Define your deployment tasks in nyati.yaml:

```yml
version: "0.1.2"
appname: "myapp"
hosts:
  server1:
    host: "example.com"
    username: "user"
    private_key: "~/.ssh/id_rsa"
tasks:
  - name: new_release
    cmd: mkdir -p /var/www/html/${appname}/releases/${release_version}
    expect: 0
  - name: git_clone
    cmd: git clone -b main git@github.com:user/repo.git /var/www/html/${appname}/releases/${release_version}
    expect: 0
  - name: publish
    cmd: ln -sfn /var/www/html/${appname}/releases/${release_version} /var/www/html/${appname}/current
    expect: 0
    message: Deployment completed ${release_version}
```

2. Deploy to server1:

```bash
nyatictl deploy server1

```

Output:

```
📡 Connected: server1 (user@example.com)
🎲 new_release: [spinner]
🎉 new_release@server1: Succeeded
🎲 git_clone: [spinner]
🎉 git_clone@server1: Succeeded
🎲 publish: [spinner]
🎉 publish@server1: Succeeded
📗 Deployment completed 1698771234567

```

### Clean Up Old Deployments

1. Use the clean task from your config:

```yml
- name: clean
  message: older deployments cleaned
  cmd: ls -dt1 */ | tail -n +5 | xargs rm -rf
  dir: /var/www/html/${appname}/releases
  expect: 0
  output: true
  lib: true
```

2. Run the clean task

```bash
nyatictl deploy server1 --task clean

```

Output:

```bash
📡 Connected: server1 (user@example.com)
🎲 clean: [spinner]
🎉 clean@server1: Succeeded
dir1/ dir2/ dir3/
📗 older deployments cleaned
```

### Check Disk Usage

1. Add the check_disk task

```yml
- name: check_disk
  cmd: df -h /
  expect: 0
  output: true
  message: Disk usage checked
```

2. Run the task

```bash
nyatictl deploy server1 --task check_disk

```

Output:

```bash
📡 Connected: server1 (user@example.com)
🎲 check_disk: [spinner]
🎉 check_disk@server1: Succeeded
Filesystem      Size  Used Avail Use% Mounted on
/dev/sda1       100G   50G   50G  50% /
📗 Disk usage checked
```

### Troubleshooting

"**no config file found**"

. Ensure a nyati.yaml or nyati.yml file exists in the current directory, or specify a custom path with -c:

```bash
nyatictl -c path/to/config.yaml deploy all
```

"**host not found**"

. Verify that the host name (e.g., server1) matches a key in the hosts section of your config.

"**task not found**"

. Check that the task name specified with --task matches a task in the tasks section.

### SSH Connection Issues

. Ensure the username, password, or private_key in the host config is correct.
. Verify that the target server is reachable and SSH is enabled.
. Use the `--debug` flag to see detailed output:

```bash
nyatictl deploy server1 --debug

```

### Task Fails with Non-Zero Exit Code

. Check the expected exit code (expect) in the task definition.
. Use the `--debug` flag to see the command output:
. If `retry: true` is set, Nyatictl will prompt to retry the task.

### Contributing

Contributions are welcome! To contribute:

1. Fork the repository.
2. Create a new branch:

```bash
git checkout -b feature/your-feature
```

3. Make your changes and commit:

```bash
git commit -m "Add your feature"
```

4. Push to your fork:

```bash
git push origin feature/your-feature
```

5. Open a pull request.

### Development Setup

1. Clone the repository and install dependencies:

```bash
git clone https://github.com/zechtz/nyatictl.git
cd nyatictl
go mod tidy
```

2. Build and test:

```bash
go build -o nyatictl
./nyatictl --help
```

### Code Structure

. main.go: Entry point.

. cli/: CLI setup and argument parsing.

. config/: Configuration loading and validation.

. ssh/: SSH client management and command execution.

. tasks/: Task execution logic.

### License

Nyatictl is licensed under the MIT License. See the LICENSE file for details.

### Acknowledgments

Inspired by NyatiCtl a Node-based deployment tool.

Built with Cobra for CLI handling and Viper for configuration management.

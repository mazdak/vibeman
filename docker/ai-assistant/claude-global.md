# Vibeman AI Container Environment

Welcome to your AI-powered development container! This environment is pre-configured with powerful tools and utilities to enhance your development workflow.

## Starting Claude CLI

To start Claude CLI in this container, run:
```bash
claude --dangerously-skip-permissions
```

Or from outside the container:
```bash
docker exec -it <container-name> claude --dangerously-skip-permissions
```

## Environment Overview

You are running in a Vibeman AI container that is automatically attached to your worktree. This container has access to:

- **Workspace**: `/workspace` - Your worktree code (read-write)
- **Logs**: `/logs` - Aggregated logs from all worktree containers (read-only)
- **Config**: `/config` - Configuration files (read-only)

## Available Tools

### Search and Navigation

1. **ripgrep (rg)** - Lightning-fast text search
   ```bash
   # Search for pattern in all files
   rg "pattern"
   
   # Search only in specific file types
   rg -t python "def.*function"
   
   # Search with context
   rg -C 3 "error"
   
   # Search and replace preview
   rg "old_pattern" --files-with-matches | xargs sed -i 's/old_pattern/new_pattern/g'
   ```

2. **fd** - Fast and user-friendly find alternative
   ```bash
   # Find all Python files
   fd -e py
   
   # Find files by name pattern
   fd "test.*\.go$"
   
   # Find and execute
   fd -e js -x prettier --write
   
   # Find directories only
   fd -t d node_modules
   ```

3. **fzf** - Fuzzy finder for files and command history
   ```bash
   # Interactive file search
   vim $(fzf)
   
   # Search command history
   <Ctrl+R>  # Built-in fzf integration
   
   # Git branch switcher
   git checkout $(git branch | fzf)
   
   # Kill process interactively
   kill -9 $(ps aux | fzf | awk '{print $2}')
   ```

4. **ast-grep (sg)** - Code structural search
   ```bash
   # Find all function definitions
   sg 'function $NAME($_) { $$$ }' --lang js
   
   # Find React components
   sg 'const $COMP = () => { $$$ }' --lang tsx
   
   # Find Python class definitions
   sg 'class $NAME($$$): $$$' --lang python
   ```

5. **ag (The Silver Searcher)** - Another fast search tool
   ```bash
   # Search with specific file types
   ag "TODO" --python
   
   # Search ignoring case
   ag -i "error"
   ```

### Log Analysis

1. **View worktree logs**
   ```bash
   # List all log files
   ls -la /logs/
   
   # Tail a specific container log
   tail -f /logs/container-name.log
   
   # Search across all logs
   rg "error" /logs/
   
   # View logs with color highlighting
   ccze < /logs/container.log
   
   # Monitor multiple logs
   multitail /logs/*.log
   ```

2. **Log searching and filtering**
   ```bash
   # Find errors in the last 100 lines
   tail -n 100 /logs/app.log | rg -i "error|exception"
   
   # Count occurrences
   rg -c "ERROR" /logs/*.log | sort -t: -k2 -nr
   
   # Extract timestamps
   rg "^\d{4}-\d{2}-\d{2}" /logs/app.log
   ```

### Development Utilities

1. **tmux** - Terminal multiplexer
   ```bash
   # Start new session
   tmux new -s dev
   
   # Key bindings:
   # Ctrl-b %     - Split vertically
   # Ctrl-b "     - Split horizontally
   # Ctrl-b arrow - Navigate panes
   # Ctrl-b d     - Detach session
   # Ctrl-b [     - Scroll mode
   ```

2. **jq** - JSON processor
   ```bash
   # Pretty print JSON
   cat data.json | jq .
   
   # Extract specific fields
   cat logs.json | jq '.[] | {level, message}'
   
   # Filter by condition
   cat data.json | jq '.[] | select(.status == "error")'
   ```

3. **yq** - YAML processor
   ```bash
   # Read YAML value
   yq '.services.web.image' docker-compose.yml
   
   # Update YAML value
   yq '.version = "3.8"' docker-compose.yml
   ```

4. **Modern CLI replacements**
   ```bash
   # exa - Better ls
   exa -la --tree --level=2
   
   # bat - Better cat with syntax highlighting
   bat README.md
   
   # ncdu - Interactive disk usage
   ncdu /workspace
   ```

### Network Debugging

```bash
# Test service connectivity
nc -zv postgres 5432
ping redis

# DNS lookup
nslookup service-name
dig service-name

# Port scanning
nmap -p 1-65535 localhost

# HTTP testing
curl -v http://api:8080/health
http GET api:8080/users
```

### Shell Features

Your shell (zsh) is configured with:
- **oh-my-zsh** with useful plugins
- **Auto-suggestions** as you type (accept with →)
- **Syntax highlighting** for commands
- **Git integration** in prompt
- **Fuzzy history search** with fzf (Ctrl+R)

Key shortcuts:
- `Ctrl+R` - Fuzzy search command history
- `Ctrl+T` - Fuzzy file search
- `Alt+C` - Fuzzy directory search
- `Tab` - Autocomplete with smart suggestions

## Aliases and Functions

```bash
# Navigation
ws      # cd /workspace
logs    # cd /logs

# Log viewing
ltail   # tail -f /logs/*.log
lerror  # rg -i "error|exception|fail" /logs/
lwatch  # watch log directory changes

# Git shortcuts
gs      # git status
gd      # git diff
gl      # git log graph
gco     # git checkout

# File operations
vf      # vim with fuzzy file search
rgf     # ripgrep with file preview
fdd     # fd with directory focus

# Container helpers
check-service <name> <port>  # Test service connectivity
```

## Best Practices

### 1. Efficient Code Search
```bash
# Combine tools for powerful workflows
fd -e py | xargs rg "class.*Model"
rg -l "TODO" | fzf | xargs vim

# Use ast-grep for refactoring
sg 'console.log($ARG)' --rewrite 'logger.debug($ARG)'
```

### 2. Log Analysis Workflow
```bash
# Real-time error monitoring
rg -i "error" /logs/*.log --line-buffered | ccze

# Historical analysis
for f in /logs/*.log; do
  echo "=== $f ==="
  rg -c "ERROR" "$f" | tail -5
done
```

### 3. Development Workflow
```bash
# Split terminal for multiple tasks
tmux new -s dev
# Ctrl-b % for vertical split
# Run server in one pane, logs in another

# Monitor file changes
fd -e go | entr -c go test ./...
```

## Environment Variables

Key environment variables available:
- `VIBEMAN_WORKTREE_ID` - Current worktree identifier
- `VIBEMAN_REPOSITORY` - Repository name
- `VIBEMAN_WORKTREE_PATH` - Path to worktree
- `VIBEMAN_LOG_DIR` - Worktree-specific log directory
- `VIBEMAN_AI_CONTAINER` - Set to "true" in AI containers

## Performance Tools

```bash
# System monitoring
htop    # Interactive process viewer
iotop   # I/O usage by process
ncdu    # Disk usage analyzer

# Network monitoring
iftop   # Network usage by connection
```

## Tips and Tricks

1. **Quick command examples**: Run `tldr <command>` for practical examples
2. **Persistent sessions**: Use tmux to maintain sessions between container restarts
3. **Quick edits**: Set `EDITOR=vim` for git and other tools
4. **Color output**: Most tools support `--color=always` for piped output

## Getting Help

- Tool help: `<tool> --help` or `man <tool>`
- Quick examples: `tldr <tool>`
- This guide: `cat ~/.claude/CLAUDE.md`
- Zsh shortcuts: `bindkey` to list all key bindings

## Important Notes

- This container runs as user `vibeman` (UID 1000) for security
- The workspace is persistent, but container state is not
- Always commit important changes to git
- Log files are rotated; check timestamps when debugging

---

Remember: This AI container is designed to enhance your development workflow. Explore the tools, customize your environment, and leverage the power of modern CLI utilities!
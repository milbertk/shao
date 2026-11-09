# devtools — start AlephK for development

`alephk-dev.sh` starts everything on your computer with one command:

1. opens shao, usersAppAK, p3 and showcase in Visual Studio Code;
2. starts the backends, each in its own terminal window: usersAppAK (port 8080) and shao (port 8081), with `go run .`;
3. waits until both backends answer (the first `go run` compiles, up to 90 seconds);
4. starts the frontends, each in its own terminal window: p3 (port 3000) and showcase (port 3001), with `npm start`.

A `PORT=` line in a project's `.env` wins over these ports. Only that line is read; the programs read their own `.env` as usual.

## First time

```bash
chmod +x ~/projects/go/src/github.com/milbertk/shao/devtools/alephk-dev.sh
echo "alias alephk='~/projects/go/src/github.com/milbertk/shao/devtools/alephk-dev.sh'" >> ~/.bashrc
source ~/.bashrc
```

## Every day

| Command | What it does |
| --- | --- |
| `alephk` | VS Code + backends + frontends |
| `alephk --no-code` | backends + frontends, VS Code already open |
| `alephk --one-window` | the four projects in one VS Code window |
| `alephk back` / `alephk front` / `alephk code` | only that part |
| `alephk status` | which programs are running (by port) |
| `alephk stop` | stops the frontends and the backends |
| `alephk --dry-run` | shows what would run, starts nothing |

- A program that is already running is not started twice, so you can run `alephk` again after you close one window.
- In a window: Ctrl+C stops that program; the window stays open and shows the command to start it again; `exit` closes it.
- It warns when PostgreSQL is not answering on localhost.
- Terminal programs found automatically: gnome-terminal, konsole, xfce4-terminal, mate-terminal, tilix, kitty, alacritty, xterm.
- Settings (folders, commands, ports, the projects for VS Code) are at the top of the script.

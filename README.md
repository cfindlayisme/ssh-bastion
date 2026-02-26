# ssh-bastion

A simple SSH bastion/jumpbox with user access control. Authenticates users by SSH key, assigns them to groups, and restricts which targets they can reach. Stores no auth information in itself.

## Build & Run

```
go build -o ssh-bastion .
cp config.yaml.sample config.yaml  # edit with your settings
./ssh-bastion
```

Connect with agent forwarding:

```
ssh -A -p 2222 user@bastion-host
```

## Config

See `config.yaml.sample` for a full example. Users are identified solely by their SSH public key and assigned to groups. The username provided when connecting (e.g. `ssh -A -p 2222 anything@host`) is ignored — only the key matters. Targets list which groups can access them.

Users in the `admin` group get an extra menu option to connect to any arbitrary host.

## Menu Examples

**Admin user:**

```
╔══════════════════════════════════════╗
║       SSH Bastion Host Gateway       ║
╚══════════════════════════════════════╝

  Welcome, john

  Select a host to connect to:

  1) Production
  2) Staging

  c) Custom host

  q) Quit

>
```

**Regular user:**

```
╔══════════════════════════════════════╗
║       SSH Bastion Host Gateway       ║
╚══════════════════════════════════════╝

  Welcome, jane

  Select a host to connect to:

  1) Staging

  q) Quit

>
```

After selecting a target, users are prompted to connect as their default username or type a different one. When the session ends, the menu reappears.

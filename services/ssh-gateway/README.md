# SSH Gateway

Lets operators open a shell on a system with plain `ssh`, through the same
tunnel as the support sessions. Render only lets HTTPS in, so SSH travels
inside a WebSocket: the client reaches it with a `ProxyCommand` such as
[websocat](https://github.com/vi/websocat) (Linux, macOS and Windows builds).

## Operator setup

Install websocat, then add to `~/.ssh/config`:

```
Host my
    HostName my
    ProxyCommand websocat --binary wss://my.nethesis.it/ssh-gateway/api/ssh
```

and connect with the system key as user, optionally a cluster node:

```
ssh NETH-XXXX@my        # first node
ssh NETH-XXXX.2@my      # node 2
```

Locally, use `wss://my.localtest.me/ssh-gateway/api/ssh`: like the other
services, the proxy exposes it under its own prefix (`/api/` on the main
host still forwards to the legacy my).

## Flow

1. The client proves it owns an SSH key; if that key was approved for this
   system in the last hours, the shell opens right away.
2. Otherwise ssh prints a link to my and a verification code, and waits.
3. The operator opens the link (signing in to my if needed), checks system,
   source IP and code, and approves.
4. The backend issues a grant bound to operator, system and node. The gateway
   opens shells with it through the backend, which checks the operator again
   at every shell and every minute while it stays open (sign-out from my or
   lost permissions close it) and logs the access as `ssh`.

Only interactive shells for now: commands, scp and sftp are refused.

# SSH Gateway

Lets operators open a shell on a system with plain `ssh`, through the same
tunnel as the support sessions. Render only lets HTTPS in, so SSH travels
inside a WebSocket: the client reaches it with a `ProxyCommand` such as
[websocat](https://github.com/vi/websocat) (Linux, macOS and Windows builds).

## Operator setup

The SSH button of a support session in my copies a ready command:

```
ssh -o ProxyCommand='websocat --binary wss://my.nethesis.it/ssh-gateway/api/ssh' NETH-XXXX.2+<token>@my
```

The user is the system key, the cluster node (`.2`) when there is more than
one, and a one-time token bound to the operator who copied it (valid 10
minutes). Only that operator can approve the request it opens: a link
forwarded by someone else cannot be approved. Install
[websocat](https://github.com/vi/websocat) once (Linux, macOS and Windows
builds; on Windows the built-in OpenSSH client, PuTTY or WinSCP with a
"Local" proxy command work too).

With an SSH key, the approval also covers that key on that system for up to
4 hours, while the support session stays open: `ssh NETH-XXXX.2@my` (no
token) then connects without the browser.

## Host key

The gateway's host key is its identity: if it changes, every operator's ssh
stops with "REMOTE HOST IDENTIFICATION HAS CHANGED". On Render, where the disk
does not survive a deploy, set it once as the `SSH_HOST_KEY` secret:

```
ssh-keygen -t ed25519 -N '' -C my-ssh-gateway -f ssh_host_key
ssh-keygen -lf ssh_host_key.pub     # fingerprint to publish for operators
```

The gateway logs the fingerprint at startup. Publish it in the operator docs,
so the first connection's prompt is checked rather than accepted blindly.

## Flow

1. The client proves it owns an SSH key; if that key was approved for this
   system and session, the shell opens right away.
2. Otherwise the gateway turns the token into an access request, prints the
   link to my and waits. A missing, expired or reused token is refused.
3. The operator who copied the command opens the link (signing in to my if
   needed), checks system, source IP and key, and approves.
4. The backend issues a grant bound to operator and support session. The
   gateway opens shells with it through the backend, which checks the
   operator, the session and the grant again at every shell and every minute
   (sign-out from my, lost permissions or a closed session end it) and logs
   the access as `ssh`.

Shells, commands, pipes, scp, rsync and sftp work as with a direct ssh;
port and agent forwarding are refused. Every shell, command and subsystem
is logged with operator, system and source IP.

## Isolation

The gateway faces the internet, so it holds as little as possible: no Redis,
no database, no support-service secret. It talks only to the backend's
`/api/support-ssh/gateway/*` routes with its own `SSH_GATEWAY_SECRET`, which
opens nothing else; the proxy never exposes those routes.

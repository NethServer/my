---
sidebar_position: 5
---

# Reassigning a system to another organization

Sometimes a system needs to move from one customer to another — a managed
service provider hands a server back to its end customer, an organization
is restructured, or a system was originally created under the wrong tenant.
My supports this operation natively: the system carries its full state to
the new owner, and the previous owner loses access immediately.

## When you would do this

- A reseller or distributor takes over (or returns) a customer's system.
- A system was registered under the wrong organization at setup time.
- An organization is being merged or split, and its systems must follow.

## What gets moved

When you change a system's organization, My moves all the data that
*belongs to the system* to the new owner, and leaves the previous owner's
own data alone:

| Data | What happens |
|---|---|
| **Configuration backups** | Copied to the new owner's storage area before the change is committed; the previous owner's copies are removed. |
| **Alert history** | The system's full alert history follows the system, so the new owner sees what was happening before the handover. |
| **System-specific silences** | Single-purpose silences (silencing all alerts of that one system) are dropped — the new owner can reapply them if needed. Broader silences set up by the previous owner (for example "silence disk-full alerts on this system during maintenance") are left in place; they expire on their own. |
| **Application assignments** | All app-to-organization assignments on the system are cleared. The new owner re-assigns the apps it cares about from the regular Applications view. |
| **Inventory, heartbeat, system identity** | Untouched — `system_key`, the secret, the inventory history all carry over. Appliances do not need to re-register. |

The change is **atomic** for the data that matters: either everything is
in place under the new owner, or the change is refused outright with no
side effect. Cleanup of the previous owner's residual data runs after the
flip and is best-effort — if anything is left behind, it does not affect
the new owner's view.

## How to reassign

### From the UI

1. Open the system detail page.
2. Edit the system and pick a different **Organization** from the dropdown.
3. Save. Within a few seconds the system shows the new owner and all the
   data above is in place.

### From the API

```bash
curl -X PUT https://my.example.com/api/systems/$SYSTEM_ID \
     -H "Authorization: Bearer $JWT" \
     -H "Content-Type: application/json" \
     -d '{"name":"unchanged","organization_id":"<new-logto-id>"}'
```

A reassignment with backups under retention typically completes in a few
seconds. With heavier workloads it can take longer — the request stays
open until the move is fully on disk on the new owner's side.

## Who can do it

- **Owner** can reassign any system to any existing organization.
- **Distributor** and **Reseller** can reassign systems within their own
  hierarchy — they cannot move a system *out* of their scope to an
  organization they do not manage.
- **Customer** cannot reassign systems.

A reassignment to an organization that does not exist is rejected with a
`403 access denied` — there is no path that strands a system under an
unreachable organization.

## What the previous owner sees afterwards

As soon as the change is committed:

- The system disappears from the previous owner's "Systems" list.
- The previous owner's API calls against the system return `404` (the
  system is no longer in their scope).
- Any direct link they had to the system's backup, alert history, or
  detail page returns `404` too.

This is the same behaviour as if the system had been deleted from the
previous owner's perspective — the difference is the data still exists,
just under the new owner.

## Concurrent edits

If two administrators try to reassign the same system at the same moment,
only the first one succeeds. The second one receives a clear
`409 system reassignment is already in progress` and can retry once the
first reassignment has completed.

This protects the system from inconsistent half-state where its data
ends up split between two destinations.

## Common questions

**Can I move a system back to its original owner?**
Yes. Reassignment is symmetric — moving from A to B and back to A works
the same way both times.

**Will the system continue working during the reassignment?**
Yes. The appliance keeps sending inventory, heartbeats, and backups
without interruption. Uploads in flight at the moment of the change land
correctly under the new owner once the reassignment is committed.

**What about an active support tunnel session?**
A support session that was already open continues until it expires
(default 24h). New sessions are started by the new owner.

**Does the system need to be online to be reassigned?**
No. Reassignment operates on the platform-side data only; the appliance
state is not touched.

**Can I undo a reassignment?**
Reassignment is recorded in an audit trail visible to the platform Owner.
There is no one-click undo, but reassigning back to the original owner is
the standard way to revert; the data follows the system in both
directions.

## Related

- [Systems management](management) — the full lifecycle of a system.
- [Configuration backups](backups) — what is stored, how it's protected,
  and how the per-organization quota is computed.

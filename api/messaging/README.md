# messaging

Wire types, op-name constants, reject codes and authorization actions for the
`messaging` protocol — mail between the identities whose mailboxes a node
hosts, and the provisioning of those identities; `client/` is the protocol's
RPC client.

A node hosts a mailbox only under a contract the mailbox identity signs:

* Issuer: the mailbox identity.
* Subject: the hosting node.
* Permit: `mod.messaging.host_mailbox_action`, Delegation 0, no constraints.

The mailbox identity holds `mod.messaging.host_mailbox_action` itself: the root
rule allows it exactly when the action's `MailboxID` is nonzero and equals its
actor. A contract from another identity, or one the node issues to itself,
grants no hosting. A permit carrying any constraint is refused.
`messaging.create_identity` provisions the hosting contract beside the relay
contract. `messaging.delete_identity` withdraws hosting on the node alone; the
contract stays valid until it expires.

Hosting is separate from mail admission (`mod.messaging.send_action`,
`mod.messaging.receive_action`) and from relaying (`mod.nodes.relay_for_action`).
No one of these permits grants another.

Every operation is local-only: a query arriving over the network, or carrying
the MCP origin, is refused whatever identity it holds. The mail operations act
on the caller's own boxes, and the node must host the caller's mailbox.

Protocol spec:

* [astral-docs/protocols/messaging](https://github.com/astralp2p/astral-docs/tree/master/protocols/messaging) — overview and op specs

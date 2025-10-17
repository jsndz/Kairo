## What I know about CRDT

**Conflict-free Replicated Data Types** is a data structure.
They are used for real-time resolution of conflicts between data.
For example, if two users are writing to the same database at the same time, there is a conflict. To resolve this:
The field example: mobile data in local and sync, Google Docs, distributed systems that use multiple data replicas.
It is used in Redis.
**All such systems need to deal with the fact that the data may be concurrently modified on different replicas.**

CRDT is for Optimistic Replication:
Modify data without any consideration for other devices.
Maximum performance and availability (not much Consistency — CAP C).
But leads to conflict in multiple-user use cases, so there is a need for conflict resolution.

CRDT ensures that conflicts are resolved.

They support decentralised operation: they do not assume the use of a single server, so they can be used in peer-to-peer networks and other decentralised settings.

State based vs Operation Based:

Each system sends its final state and then merge
Send Operations then apply
Simple but large
Complex but small

YJS is Hybrid Based

- sends only operations
- Operations are designed to be commutative, associative, and idempotent
- Relies on Websocket/webrtc hence reliable

Operational Transformation used transformational functions to the data based on the order they are presented to reslove conflict

| Aspect                | CRDT                               | OT                                |
| --------------------- | ---------------------------------- | --------------------------------- |
| Conflict resolution   | Built into the data structure      | Done via transformation functions |
| Order of operations   | Order doesn’t matter (commutative) | Must transform based on order     |
| Central server needed | No                                 | Often yes                         |
| Offline edits         | Naturally supported                | Harder without a server           |
| Example               | Yjs, Automerge                     | Google Docs (original), ShareJS   |

How does CRDT reslove conflicts?

Every inserted element gets a globally unique ID
Operations reference IDs, not positions.
If two inserts target the same spot , use ID ordering(rondom or sort)
If delete + insert conflict, delete always means “mark element with ID as removed”

A visual representation of CRDT Doc:

Each char = (value, uniqueID)

User A inserts "X" after "A"

[A: (c1,1)] -> [X: (c1,3)] -> [B: (c1,2)]

working understanding of Yjs CRDT primitives, awareness, persistence, and networking

Basic CRDT Theory

What are CRDTs? → Conflict-Free Replicated Data Types, allow concurrent updates to converge without coordination.

Why CRDTs? → Offline-first, real-time collaboration, fault tolerance.

Two families: State-based (CvRDTs) and Operation-based (CmRDTs).

Conflict Resolution Mechanism

CRDTs resolve conflicts via mathematical merge rules (commutative, associative, idempotent).

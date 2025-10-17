## CAP Theorem

There are three properties:

- **Consistency (C)**: The data should be the same on all replicas.  
  Every read gets the most recent write or an error.

- **Availability (A)**: The service should always be able to respond.  
  Every request to a non-failing node gets a response — not necessarily the most up-to-date one.

- **Partition Tolerance (P)**: The system continues working even if messages between nodes are lost or delayed.

When a **network partition** occurs, you must choose:

- **CP** (Consistency + Partition tolerance) → Data is correct, but some requests may be rejected.
- **AP** (Availability + Partition tolerance) → Always responds, but may serve stale data.

In the absence of a network partition, both availability and consistency can be satisfied.

- **RDBMS** (traditional SQL) → usually CP in distributed setups.
- **NoSQL** (many systems) → often AP.

---

**ACID C** = Rules are never broken.  
**CAP C** = Everyone sees the same thing.

---

**ACID C example**  
In an online ticket booking system:

> Two people can’t book the same seat.  
> The database enforces this rule at all times.

**CAP C example**  
In a live cricket score app:

> If one server shows `India: 150/3`, all servers should instantly show `150/3`, not `148/3`.

---

**Tiny CAP example**  
You’re watching a live cricket score on **Server A** and **Server B**.  
If the network between them breaks (**partition**), the system must choose:

- **CP** → Show the score only when both servers agree (may pause updates).
- **AP** → Keep showing each server’s updates (may be slightly different).

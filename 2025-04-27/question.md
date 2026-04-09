# Coding Question Summary: Epoll + Reactor Server Design

## Problem Statement

Implement a scalable TCP server using Go that leverages:
- Epoll for efficient IO multiplexing (non-blocking sockets)
- Reactor pattern to manage high concurrency
- Worker pool to offload business logic processing

Constraints:
- Only use `syscall` or `golang.org/x/sys/unix` for low-level socket and epoll control.
- Must support tens of thousands of concurrent long-lived TCP connections.
- Handle read and write efficiently using ET (Edge-Triggered) mode.
- Gracefully handle client disconnects, errors, and resource limits.

---

## Coding Final Version Architecture

### Core Components

- **Accept Loop**: multiple goroutines (same as CPU cores) accepting incoming connections.
- **ePool**: Epoll Pool object managing multiple epollers.
- **epoller**: Single epoll instance managing registered fds.
- **connection**: Encapsulation of `net.TCPConn` + file descriptor.
- **Worker Pool**: For processing the business logic after epoll event triggers.
- **Limits**: RLIMIT_NOFILE raised; TCP connection count limit enforced.

### Key Functions

- `initEpoll` / `newEPool` / `startEProc`
- `acceptLoop` to accept new TCP connections.
- `epoller.add` to register fd to epoll, using `EPOLLIN | EPOLLET`.
- `epoller.wait` to collect active fds after epoll_wait.
- `runProc` to read full message packages, submit tasks to worker pool.
- `ReadData`, `SendData` functions for TCP framed packet IO.
- Proper `SetNonblock(fd, true)` for sockets.
- Error handling for `EAGAIN`, `EWOULDBLOCK`, and `EOF`.

### Important Design Decisions

- **ET Mode (Edge Triggered)**: High performance, low system call overhead.
- **Non-blocking IO**: Mandatory to avoid deadlocks in ET mode.
- **Full read until EAGAIN**: Required under ET mode to ensure all data is drained.
- **Worker pool**: Prevents epoll thread from getting stuck on heavy tasks.
- **Per-epoller connection map**: Optimizing concurrent map access.
- **Limit protections**: Avoid DoS/overload by setting TCP connection upper bounds.

---

# System Design Summary: Scalable IM Gateway (High Concurrent Long Connections)

## System Goals

- Support **100 million** online users.
- Maintain persistent TCP long connections.
- Horizontal scalability by adding machines.
- Low latency (messages must be pushed within 100ms).
- High availability (failover if any gateway dies).

## High Level Architecture

### Components

1. **HTTPDNS / IPConf Service**:
   - Resolve available gateway IPs dynamically.
   - Return weighted load-balanced IP lists.
2. **Gateway Layer**:
   - Built on Epoll + Reactor model.
   - Accept TCP connections, manage with epoll.
   - Worker Pool for packet processing.
3. **State Server**:
   - Maintain (UserID -> ConnFD -> GatewayID) mapping.
   - Manage online status, heartbeats.
   - Synchronize mappings for HA.
4. **Router Layer**:
   - Locate which gateway a user is attached to.
5. **Message Server**:
   - Persist chat messages.
   - Push real-time messages to online users.
6. **Control Server**:
   - Monitor gateway health.
   - Dynamically scale gateways up/down.
7. **Monitor and Alerting**:
   - Metrics collection (connections, heartbeats, message latencies).
   - Trigger alerts on anomalies.


### Detailed Flow

- Client fetches gateway IPs from HTTPDNS.
- TCP long connection established with one gateway.
- Gateway registers connection in State Server.
- Business messages arrive at gateway -> routed to correct user's gateway -> pushed down the TCP pipe.
- Heartbeats maintain active user sessions.


## Key Design Points

- **ET+Nonblocking**: For high performance in Gateway servers.
- **Multi-Epollers**: Scale to CPU cores.
- **Worker Pool**: Avoid blocking Reactor thread.
- **Connection Limitations**: Soft limit per gateway; connection rejection if overloaded.
- **State Server Redundancy**: Active-Standby or Sharding.
- **Dynamic EPOLLOUT Management**: Only monitor write readiness when needed.
- **TCP Timeout Handling**: Disconnect stale connections.
- **Disaster Recovery**: Gateway crash triggers reconnect via HTTPDNS.

---

# Final Verdict

✅ Today's coding and system design exercise covered real-world production-quality patterns.

✅ You now have a **solid foundation to build ultra-scalable IM, push notification, or WebSocket Gateway services** just like those used by Tencent, ByteDance, and other major tech firms.
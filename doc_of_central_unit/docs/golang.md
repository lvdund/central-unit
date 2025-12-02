Goal
- Build a CU-CP in Go that mirrors OAI logic (RRC core orchestrating NGAP/F1AP/E1AP) while staying scalable (10M+ ctrl-plane ops/sec target).

Protocols & flows to cover
- NGAP over SCTP: NG Setup, Initial UE Message, Initial Context Setup, UE Context Release, UL/DL NAS Transport, Paging, HO Required/Command/Notify.
- F1AP over SCTP: F1 Setup, Initial UL RRC Msg Transfer, UL/DL RRC Msg Transfer, UE Context Setup/Mod/Release, F1 Reset, DU Config Update.
- E1AP over SCTP: E1 Setup, Bearer Context Setup/Mod/Release, lost-connection handling.
- RRC (control plane): RRC Setup/Reconfig/Release, Security Mode Command, UE Capability Enquiry, Reestablishment, HO config, paging PCCH generation.

Restructured project (protocol libs already exist)
- `/cmd/cu-cp` – main binary (split/monolithic flags, config path).
- `/pkg/context`
  - UE context store (state machine: idle ↔ connected ↔ connected-inactive; fields for ids, security, SRB/DRB, sessions, timers, HO state).
  - DU store (assoc, cells, MIB/SIB cache, MTC), CU-UP store (assoc, slice support).
  - Mapping tables (UE ID → {DU assoc, DU UE ID, CU-UP assoc}; AMF_UE_NGAP_ID index; RNTI+DU index).
  - Sharded maps + secondary indexes; eviction and snapshot hooks.
- `/pkg/logic`
  - RRC procedure engine (single-threaded loop consuming events): attach/registration, security, reconfig, reestab, HO, release, paging; drives UE state machine.
  - NGAP procedure handlers (Initial UE msg, context setup/release, PDU session setup/modify/release, paging, HO).
  - F1AP CU handlers (F1 setup, UL/DL RRC msg transfer, UE ctx setup/mod/rel, reset, DU cfg update).
  - E1AP CU-CP handlers (E1 setup, bearer ctx setup/mod/rel, CU-UP loss).
  - Policy hooks (slice selection, CU-UP selection, admission).
- `/pkg/transport`
  - SCTP client/server for NGAP, F1AP, E1AP; per-assoc IO loops, send queues, backpressure.
  - Wire adapters that call existing protocol libs for encode/decode and push events into logic.
- `/pkg/timers` – hashed timing wheel for inactivity/paging/retransmission.
- `/pkg/config` – load/validate PLMN/cell/slice/net addresses; feature flags.
- `/pkg/obs` – logging/metrics/tracing; structured with IDs.
- `/pkg/utils` – buffer pools, worker pools, rate-limiters.
- `/test` – integration harness (DU/AMF/CU-UP emulators, fuzzers, soak tests).

Context & state-machine guidance
- UE state machine (RRC): idle → connected → connected-inactive; transitions on attach/release/resume/suspend/failure. Store state in UE context; enforce legal transitions per event before acting.
- Keep UE context sharded map with RWMutex per shard; secondary indexes for AMF_UE_NGAP_ID and (DUAssoc,RNTI).
- Persist mapping struct for F1/E1: {duAssoc, duUeId, cuupAssoc} either embedded in UE or separate sharded map.
- DU/CU-UP contexts: simple sharded maps keyed by assoc; include capabilities/slices to drive selection.
- Timers: per-UE wheel entries for inactivity, T300/T301/T304, paging; timer callbacks enqueue events to logic loop.

Logic/handler layering
- Transport decodes PDU → emits event to logic (non-blocking channel).
- Logic loop (rrc/logic package) processes one event at a time to keep ordering per UE; optionally shard loops by UE hash for parallelism.
- Outbound messages: logic builds protocol structs with your existing libs, hands to transport send queue per assoc.

Scalability tweaks (10k+ req/s on links, 10M cps target overall)
- Shard logic workers by UE hash to parallelize while preserving UE order.
- Preallocate UE IDs and reuse buffers (sync.Pool); cache template PDUs for setup responses.
- Batch SCTP writes per assoc; apply backpressure via bounded channels with drop/fast-fail policies.
- NUMA/core pinning for IO goroutines if needed; monitor queue depths and latency histograms.

Build steps (revised)
- Implement context store + indexes + UE state machine with benchmarks.
- Wire transport adapters to existing NGAP/F1AP/E1AP/RRC libs; stub logic to echo/basic responses.
- Build logic handlers in order: setups (NG/F1/E1) → registration/security → session setup → release → HO → reestab → paging/resume/suspend.
- Add timers and paging flow; add observability; run soak/fuzz on decoders and high-rate attach simulators.

Core logic threads (event model)
- Use a central event dispatcher akin to ITTI: each module owns a goroutine + channel; messages are typed events (no shared state).
- RRC loop consumes events from NGAP/F1AP/E1AP/timers and drives procedures; it emits commands back to the link modules.
- Make F1AP/NGAP/E1AP SCTP handlers edge-triggered and push decoded PDUs into channels; avoid blocking on network read paths.

Context storage & indexing (high-level)
- UE context struct: UE IDs (CU UE ID, RNTI, DU UE ID), AMF UE NGAP ID, GUAMI, NAS buffers, security (algos, keys), SRB/DRB tables, PDU sessions, HO state, timers, flags (f1 active, cuup assoc).
- DU context: assoc ID, served cells, PLMN, SIB/MIB cache, measurement timing config, status.
- CU-UP context: assoc ID, supported PLMN/S-NSSAI list, slice selection state, status.
- Maps and indexes:
  - Sharded map (e.g., N shards with RWMutex) keyed by CU UE ID for hot paths.
  - Secondary indexes: RNTI+DUAssoc -> UE, AMF_UE_NGAP_ID -> UE (use sync.Map or separate sharded maps).
  - DU/CU-UP trees: use btree (github.com/google/btree) if ordering needed, otherwise sharded maps keyed by assoc ID.
  - F1/E1 mapping: struct holding DU assoc, DU UE ID, CU-UP assoc; store alongside UE context or in dedicated map.
- Memory management:
  - Pool frequently allocated buffers (encoding/decoding) with sync.Pool.
  - Keep immutable config references; copy-on-write for mutable slices.
  - Periodic GC-friendly sweeps for stale UE contexts (idle, connection lost).

Procedure mapping (mirroring OAI)
- Initial access: F1AP Initial UL RRC -> RRC Setup -> UL RRC Setup Complete -> NGAP Initial UE Msg -> auth/sec -> NGAP Initial Context Setup -> E1 Bearer Setup -> F1 UE Context Setup -> RRC Reconfiguration.
- PDU Session setup: NGAP PDU Session Resource Setup -> E1 Bearer Setup -> F1 UE Context Setup -> RRC Reconfig (NAS attach).
- Reestablishment: F1 UL RRC Reestab Req -> validation -> RRC Reestab -> E1 Bearer Mod -> RRC Reconfig Complete.
- Handover: trigger via measurements/NGAP HO Required; build HO Command; coordinate E1 PDCP status transfer and F1 UE Context switch; post-HO bearer mods.
- Release: NGAP UE Context Release or RRC Release -> F1 UE Context Release -> E1 Bearer Release.

Algorithms/data-handling tips
- Shard UE maps by hash(UE ID) to reduce lock contention; keep per-shard RWMutex.
- Use atomics for counters and lightweight flags where possible; avoid global locks in hot paths.
- For paging/idle timers, use a hierarchical timing wheel to manage millions of timers efficiently.
- Encode once, send many: cache static IE templates (e.g., NG Setup, F1 Setup Response skeletons) and fill deltas.
- Avoid per-PDU allocations in network path; reuse buffers and ASN.1 encoders.
- Maintain per-assoc write queues with batching to coalesce SCTP sends.

Throughput/scalability considerations (10M+ cps goal)
- Network I/O: consider kernel-bypass (DPDK/AF_XDP) or at least multi-queue SCTP with RSS; pin goroutines handling sockets to CPUs (runtime.LockOSThread) if needed.
- Backpressure: bounded channels per link; drop/fast-fail on overload with metrics/alerts.
- NUMA awareness: shard by core/NUMA node; keep UE shard affinity so a UE’s events stay on one worker.
- Preallocation: reserve UE IDs and pools to avoid allocator churn; keep ASN.1 objects in pools.
- Zero-copy parsing: if using C ASN.1 libs, expose slices without copying; for Go ASN.1, benchmark and avoid reflection-heavy paths.
- Telemetry: latency histograms per procedure (init attach, session setup, HO); queue depth metrics to spot hotspots.
- Chaos/rate testing: synthetic DU/AMF load generators; fuzz decoders; soak tests with millions of fake UEs.

Suggested development steps
- Pick ASN.1 toolchain (asniec + cgo wrappers vs. pure-Go generators); prototype NGAP/F1AP decoding performance.
- Build the context manager (sharded maps + indexes) with benchmarks for create/lookup/remove under contention.
- Implement event bus and per-module goroutines; stub protocol handlers returning canned responses.
- Add real SCTP I/O and wiring to RRC loop; implement core procedures in order: NG Setup/F1 Setup/E1 Setup -> initial access -> session setup -> release -> handover.
- Add timers and paging flow; integrate observability.
- Optimize hotspots found in benchmarks (encoding, map locks, queue contention).

Resilience/observability
- Graceful handling of lost connections: on F1/E1/NG loss, clear relevant mappings, trigger resets, and rate-limit reconnection.
- Structured logs with UE/CU/DU/CU-UP IDs; trace spans per UE procedure.
- Config reload with validation; feature flags for split vs. monolithic.

Testing pointers
- Unit tests for context store (index correctness under concurrent ops).
- Integration with simulated DU/AMF/CU-UP (pcap replay or lightweight emulators).
- Property/fuzz tests for ASN.1 decoders (malformed PDUs).
- Performance harness: measure ops/sec for lookup/update, end-to-end attach throughput, timer wheel churn.

# ZLD SupportPilot Delivery Roadmap

## Positioning

This roadmap captures the first delivery slices for `ZLD SupportPilot`, a Go-based intelligent service desk backend for enterprise support scenarios.

## Milestones

### M1 - Foundation

- `#1` Define delivery workflow and roadmap
- `#2` Build tenant, auth and RBAC foundation

### M2 - Ticket & Knowledge

- `#3` Implement ticket domain and workflow
- `#4` Add ticket collaboration and audit events
- `#5` Add knowledge base and document upload flow
- `#6` Build async document processing pipeline

### M3 - AI & Ops

- `#7` Implement retrieval and RAG answer API
- `#8` Add AI classification, summary and draft reply flows
- `#9` Add observability, rate limiting and release polish

## Working Agreement

- Each roadmap slice should have a dedicated PR
- Documentation and architecture updates land alongside implementation when relevant
- New features should preserve the module boundaries defined in `docs/architecture/overview.md`

## Near-Term Goal

The immediate next delivery objective is to complete the foundation story: repository workflow, local environment expectations, and tenant-aware auth groundwork.

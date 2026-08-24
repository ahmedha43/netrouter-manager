# NetRouter Manager

**NetRouter Manager** is an original, browser-based implementation of a dense native-style management interface for **NetRouter OS**. It is a frontend prototype for network engineers and administrators; it demonstrates configuration flows and simulated live telemetry without connecting to a physical router.

## Included experience

| Area | Included behavior |
| --- | --- |
| Application frame | Title bar, compact menus, operational toolbar, narrow navigation tree, MDI workspace, and persistent status bar. |
| MDI workspace | Open, close, move, resize, minimize, maximize, focus, and stack child configuration windows. |
| Router workflows | Connection dialog, neighbor discovery, Quick Set, Interfaces, WAN, LAN, DHCP server and leases, traffic monitor, System, firmware, settings, files, log, and terminal. |
| Operational safety | Native-style confirmation dialogs for reboot and factory reset; focused error dialog for invalid connection targets. |
| Live simulation | A small asynchronous interval updates CPU, memory, throughput, and uptime without blocking the interface. |
| Accessibility | Semantic buttons, labelled inputs, keyboard-focus states, status text alongside color, and reduced-motion support. |

## Design language

The interface follows the **Operations Station** design direction documented in [`ideas.md`](ideas.md). Its visual system uses a pale blue-gray workspace, white utility surfaces, `#2F73C9` as the restrained operational accent, compact Segoe UI–style typography, and original NetRouter branding assets. It intentionally avoids SaaS cards, oversized controls, gradients as decoration, and consumer-oriented layouts.

## Project structure

```text
client/
  src/
    components/netrouter/   Reusable MDI window, status, graph, and content components
    lib/                    Data model, state types, formatters, and validation helpers
    pages/Home.tsx          Main application frame and interactions
    index.css               Native desktop design system and responsive behavior
  index.html                Document metadata and generated brand favicon
ideas.md                    Recorded visual direction and brand decisions
```

## Local development

```bash
pnpm install
pnpm dev
```

The managed development command is available on the workspace preview. Build and type-check commands are described in `package.json`; the automated CI workflow is added in the next implementation stage.

## Scope and safeguards

This repository is a **frontend simulation**, not a router client. Actions such as reboot, firmware upgrade, backup, and connection validation only demonstrate the relevant interface states. No credentials, router configuration, or device communication is persisted or transmitted.

## Attribution

The application uses an independent name, an original generated router mark, and generic technical iconography. It does not reproduce any vendor branding, vendor icons, proprietary graphical assets, or exact product visual layout.

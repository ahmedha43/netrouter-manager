# NetRouter Manager — Architecture

## Purpose

This project is a **static React prototype** that expresses the intended interaction model and visual system for the NetRouter OS management client. It deliberately separates desktop-interface state, simulated data, validation, and reusable interface primitives so a future native client can reuse the same product vocabulary without coupling the frontend demonstration to device networking.

## Layer model

| Layer | Responsibility | Current location |
| --- | --- | --- |
| Application shell | Coordinates menus, toolbar, connection state, MDI child-window lifecycle, dialogs, and live simulation. | `client/src/pages/Home.tsx` |
| UI primitives | Encapsulate technical status display, traffic plots, MDI frame behavior, and window content. | `client/src/components/netrouter/` |
| Data model | Owns window identifiers, stable sample records, telemetry types, window metadata, and pure metrics helpers. | `client/src/lib/netrouter-model.ts` |
| Validation | Provides pure connection-target validation and deterministic error messages. | `client/src/lib/netrouter-helpers.ts` |
| Visual system | Defines compact desktop typography, palette, MDI geometry, status colors, reduced-motion behavior, and responsive compression. | `client/src/index.css` |

## Interaction responsibilities

The MDI controller in `Home.tsx` tracks the focus order and per-window state. Each child window is keyed by a `WindowKind`; opening a type already present brings it forward rather than duplicating it. `MdiWindow.tsx` receives only the state and callbacks it needs to drag, focus, minimize, maximize, and close its frame.

The configuration panels in `WindowContent.tsx` are deliberately presentation-first. They expose simulated values and local feedback so the flow can be reviewed without any router or credential being contacted. The only repeating activity is a two-second metrics update; it uses a browser interval and a small pure jitter helper, avoiding blocking work in the rendering path.

## Future native implementation mapping

| Proposed native module | Responsibility | Prototype boundary |
| --- | --- | --- |
| `ui` | Dear ImGui or native toolkit views, MDI frames, form controls, tables, dialogs. | `components/netrouter` and `pages/Home.tsx` |
| `state` | Session, active router, child-window state, telemetry subscriptions. | `Home.tsx` state and model types |
| `protocol` | Router protocol messages, request/response serialization, error mapping. | Not implemented; prototype only has local validation. |
| `network-client` | Asynchronous connection, TLS, authentication, transport retry, cancellation. | Not implemented; the connection dialog is simulated. |
| `discovery` | L2 MAC and L3 IPv4 discovery services. | `neighbors` mock records and Neighbor Discovery interface. |
| `data-model` | Interfaces, leases, WAN, LAN, firmware, files, and logs. | `netrouter-model.ts` and focused window content. |
| `rendering` | Native rendering backend, font rasterization, icon atlas. | CSS desktop design system and original generated icon assets. |

## Quality controls

The repository includes a minimal but meaningful Vitest suite for connection validation, and the GitHub Actions workflow runs install, static type-checking, tests, and production build on every push to `main` and every pull request targeting `main`.

## Security note

No network transport or router API calls are implemented. The prototype stores no secrets and does not transmit password-field values. A production client should keep session secrets out of persisted UI state, perform all device communication asynchronously, enforce certificate verification, and provide structured error categories to the UI layer.

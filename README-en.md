# English Version

Hi. I’ll start by going through what I managed to test and verify.

# HTML UI

The idea looks good: there is no need to learn something completely new, and it is possible to quickly build a working prototype.

The main downside is that HTML requires a rendering layer.

Overall, this is the key limitation that affects all the other decisions.

# Overview

![overview.png](assets/overview.png)

**flipperCTL** — the backend that controls everything on the computer through any API: gRPC, HTTP, etc.

**HTML** — the UI served to the client over HTTP, either through the renderer or directly in the browser.

**pixels stream** — the image produced by the renderer: video or a pixel stream.

**device / flipperCTL display** — the device requests the stream from the renderer and sends button commands back to the renderer to control the UI.

**web** — the browser receives the HTML UI directly, bypassing the renderer.

**TUI** — a separate and more complex part. I did not find any ready-made solutions for converting HTML to TUI.

There are terminals that support displaying a video stream. There are also solutions that render HTML in the console, but the result does not look very good and feels quite heavy — not responsive enough for a proper TUI.

This part will require a separate decision on which approach to use.

It is possible to implement basic control through TUI, but that would be a completely separate implementation from the HTML UI. Maintaining two separate UIs is not ideal.

# Render

I’ll start with the part that was new to me. The idea is: take HTML, run it through an engine, capture a screenshot, and send it to the stream.

## Tauri

The first attempt was to use a solution that already knows how to work with any HTML engine.

```bash
cd screencaster && cargo run
```

You can open the stream and see that it works.

The limitation is that Tauri and similar systems do not have a proper headless mode, meaning that a display is still required.

This can be worked around with a virtual display, for example `xvfb`.

It will also be necessary to support rendering through engines other than WebKit.

## Wails

Wails is a Go implementation similar to Tauri. It has something called server mode, but in practice it only hosts HTML and does not provide rendering.

It is not suitable because there is no headless rendering.

## WPE WebKit

WPE WebKit is a WebKit rendering engine for embedded platforms.

It depends on bindings, but it works in headless mode.

```bash
cd wpe-screencaster && cargo run
```

## Servo

Servo supports headless mode. It is pure Rust and does not require external dependencies.

It looks interesting, but in my opinion it will remain under active development for quite some time. Still, it can be kept as an experimental option behind a feature flag.

```bash
cd servo-screencaster && cargo run
```

## UI Control

The next step was to test whether UI control is possible.

The idea is to send commands to the JS engine, let it handle the events, and have the renderer show the updated video.

Tauri:

```bash
cd tauri-remote && cargo run
```

Servo:

```bash
cd servo-remote && cargo run
```

# Backend

Any tool can be used to implement the backend. Rust is popular now, and it may be possible to build an efficient solution with it.

Different authentication and authorization protocols can be added, databases can be connected, and basically anything available in standard backend systems can be used.

A separate note about command execution.

Authorization will define which commands the current user is allowed to execute.

Parameters will be passed from the frontend, and the result will be returned without changes.

If needed, different converters or data sanitization layers can be added.

# Frontend

Any frontend framework can be supported. Applications can be packaged into ZIP archives for deployment.

At the moment, I do not see any serious limitations other than those that may come from the design or the selected HTML engine.

# Plugin System

At first, I was thinking about templates or blocks that would need to be injected somewhere, but later a better idea came up.

Any frontend can be displayed inside an `iframe`.

This means that a plugin is a ZIP archive containing `index.html`, a manifest, and other assets.

The Core UI can also be a standalone plugin. For example, in the current version, the frontend is built into `main.zip`.

We place the archives into the `apps` folder, the backend returns the list of available applications, and the selected frontend is displayed inside the main UI through an `iframe`.

The manifest can contain additional parameters. For example, it can specify that a particular application requires a fullscreen `iframe`, such as a radio UI.

The Tauri concept of permission policies for UI fits well here. There is no need to reinvent the wheel; a similar approach can be used.

Build the frontend and the plugin:

```bash
cd frontend && npm run pack
cd ./..
cd frontend-ifconfig && npm run pack
cd ./..
cd go-server && go run .
```

Run the UI:

```bash
cd tauri-app && cargo tauri dev
```

# Conclusions

HTML and the ecosystem around it add a high level of flexibility to both the UI and the plugin system.

The approach of separating the backend and frontend is a well-known formula. There is already a huge set of tools and ready-made solutions for it.

The main difficulties may be related to cross-platform support, the rendering system, and remote UI control. These are not things you work with every day, so there may be some hidden pitfalls.

![overview-2.png](assets/overview-2.png)

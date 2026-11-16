# Changelog

## [Unreleased]

NSGo's first release is a cross-platform native UI framework written in Go. Write an app once, in Go or TSX, and it runs on phones, tablets, TVs, watches, desktops and the web, drawn by each platform's own views.

- **Native views.** Text, input methods, scrolling, menus and accessibility are the platform's own, and others imitate what one has, such as Liquid Glass.
- **Go all the way down.** The engine, layout and every backend are Go, with no C, no cgo and no JavaScript engine for Go apps.
- **Every target from one Mac.** Go and Xcode build every platform, with no NDK, Android SDK, JDK, GTK or Windows toolchain.
- **Fine-grained updates.** Components run once, signals update only what reads them, and animation runs on the platform's frame clock.
- **Small and fast.** A counter app packs to about 8 MB on the desktops and starts faster than NSH, MyGo and GPUI in NSGo's benchmarks.
- **NSH's design in Go.** Its architecture, API names and components carry over, and TSX apps written as NSH's run on the same engine.

### Features

- Targets macOS, iOS with App Clips, tvOS, watchOS, Windows, Linux, Android and the web, and WeChat, QQ, Alipay and Douyin mini programs experimentally.
- Reactive engine with signals, memos, effects, resources with suspense, keyed collections, persisted values and undo.
- Block, flexbox and grid layout, CSS-like styles, Tailwind classes, and light and dark themes from YAML or shadcn/ui theme files.
- Rich text, images, video, web views, Canvas 2D, GPU shaders in Go or WGSL, and the Lucide icon set.
- Transitions, keyframes, layout animations and shared elements on each platform's frame clock.
- Pointer, touch, keyboard, gesture and TV remote input, file drops and screen reader support.
- Navigation stacks, tab bars, sheets, predictive back on Android and browser history on the web.
- Platform services such as notifications, permissions, biometrics, pickers, sharing, haptics, speech recognition and keep-awake.
- Desktop apps with windows, panels, menus, custom title bars, tray icons, global shortcuts and one-instance launches scoped by app and profile (#332).
- `base`, unstyled components from buttons and menus to a million-row `DataTable`, charts, flow diagrams, a dock, a command palette and a camera that pans and zooms any content.
- Code `Editor`, `DiffView`, `RichEditor`, Markdown `TextView` with TeX formulas and Mermaid diagrams, and a Ghostty-based terminal.
- `ui`, styled components on `base`, with sidebars, settings forms, a file picker and chat views.
- YAML message catalogs with CLDR plurals, numbers and dates, right-to-left layout and pseudo locales.
- TSX apps on goja with Tailwind classes, hot updates that keep state, tests, and signed, sandboxed packages a Go app runs.
- Embedding in iOS, Android, Go desktop and web apps.
- `updater` with Sparkle, WinSparkle, signed Linux and Android packages and the App Store, plus widgets, Live Activities and `microphone`.
- The `nsgo` tool to create, run with hot reload, inspect with DevTools, drive over MCP, check, test and package apps.
- Packages for notarized macOS disk images, Windows installers, deb, Arch, AppImage and the Mac App Store, Microsoft Store, Google Play and Flathub.
- `nsgotest`, app tests that act as users do, headless or on each platform with native input and screenshots.
- A website with tutorials, guides, a component reference and the gallery's live demo, and a skill for coding agents.

Thanks to everyone who shaped this release with code, reports and ideas: @houko, @daodao97, @Heyana, @ayuayue, @HengWh, @Jungley8, @togotop, @westhiker, @Laotree, @OfficialDiLi, @happyhaha1, @roamlog and @yangjuncode.

# Role & Context
You are an expert software architect specializing in GUI framework design, declarative UI paradigms, and Go (Golang). 

I am building a custom portable UI framework in Go built on top of the Yoga layout engine and WebGPU. I recently built a real-world application (`cmd/chapar`) completely manually to dogfood the framework and uncover its limitations. 

The framework is currently in **Beta**, meaning **breaking changes are completely acceptable and encouraged** if they lead to a fundamentally better design. The ultimate goal is to achieve a declarative, highly ergonomic developer experience heavily inspired by **SwiftUI**, optimized for Go's type system and idioms.

# Core Objectives
Provide a comprehensive, step-by-step refactoring and architectural blueprint to address the following pain points. Your design should aggressively eliminate boilerplate, hide internal framework mechanics, and introduce a modern, fluent API.

---

## Part 1: Current Issues & Breaking Change Requirements

### 1. Application Lifecycle & Frame Invalidation
* **Current Issue:** The user is forced to manage a host element as the app root manually. There is no clean way to trigger a frame invalidation/redraw from deep within the component tree.
* **Requirement:** Eliminate the manual host element root. Introduce a centralized `App` or `Window` context. Provide a clear mechanism (e.g., a context method like `gtx.Invalidate()` or an execution command similar to Gio UI) to safely trigger a redraw from anywhere in the component lifecycle.

### 2. Encapsulation of Internal Component Responsibilities
The framework currently leaks internal state and tree-management mechanics to the end user.
* **`Attach()` Method:** The user must manually call `Attach(host *layout.Host)`. *Requirement:* Completely abstract this away; the framework lifecycle should handle root/host attachment implicitly.
* **Animations & Micro-state (e.g., Cursor Blinking):** The framework forces the user to implement `AnimationWait() (time.Duration, bool)` to handle simple tickers like cursor blinking, which breaks components like my top bar search box. *Requirement:* Component internal states (like input carets or focus states) must be encapsulated entirely within the `TextEditor`/`CodeEditor` internals.
* **Overlay & Portal Tree Management:** For components like `component.Select`, the user is forced to manually append the overlay menu (`MenuEl()`) to the layout tree. *Requirement:* Any component utilizing overlays, dropdowns, tooltips, or modals must manage its own rendering portal internally without exposing tree mutation to the user.

### 3. Layout Ergonomics & Fluent API (Method Chaining)
* **Current Issue:** Layout utilities like `layout.VStack` and `layout.HStack` are thin wrappers restricting style access. Furthermore, requiring a trailing `.El` on every component instantiation (e.g., `components.NewLabel(...).El`) makes the code incredibly verbose.
* **Requirement 1 (Fluent API):** Refactor elements and layouts to return an object that supports chainable configuration modifiers (mimicking SwiftUI), like so:
  ```go
  layout.HStack(items...).
      Border(top, left, down, right).
      Gap(5).
      FlexGrow(1).
      PaddingLeft(4).
      Margin(7)

* **Requirement 2 (Universal Layout Interface):** Eliminate .El. Functions like components.NewLabel() should return the final usable element directly, or implement a universal Layout(gtx LayoutContext) interface (similar to Gio UI) so they can be passed seamlessly into layout trees.

## Part 2: SwiftUI Inspiration & Architectural Mapping
To help me achieve a SwiftUI-like DX within Go's constraints, provide detailed guidance on the following:

### 1. SwiftUI Design References
Show concrete examples of how idiomatic SwiftUI structures layouts and components (e.g., using VStack, HStack, modifiers like .padding(), and state-driven rendering).

### 2. Go-Idiomatic Translation & Recommendations
Since Go lacks Swift's function builders (result builders) and property wrappers (@State), provide structural recommendations on:

How to implement the Layout System using the Yoga engine while preserving a clean, declarative tree structure.

How to design Components so that modifiers gracefully pass styles down or wrap elements without generating massive allocation overhead or complex pointer chains.

How to model component composition cleanly in Go so that nesting feels as natural as SwiftUI's declarative closures.

**Expected Output Structure**
Please provide:

**Architectural Overview:** A high-level blueprint of the new framework architecture (Lifecycle, Internalized Overlay/Portal system, and Invalidation engine).

**SwiftUI vs. Go API Spec:** Comparative code blocks showing a standard SwiftUI view alongside your proposed, highly-optimized Go API matching that experience ("Before vs. After" refactoring example).

**Component & Modifier Design Patterns:** Technical recommendations on how to structure the Go types and interfaces to support the fluent method-chaining API efficiently.

**Breaking Refactoring Roadmap:** A prioritized, phased plan to execute these massive breaking changes across the framework.
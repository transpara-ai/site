# Code Graph: 66 Primitives for Describing Any Application

**Matt Searles (+Claude) · March 2026**

---

## Overview

The Code Graph is the layer that sits between the agent layer (28 primitives) and the products humans actually use. It answers the question: what are the irreducible atoms of software — code and UI — such that any application can be described as a composition of those atoms and then translated into any target technology?

The derivation follows the same method used throughout the eventgraph architecture: identify base operations, identify differentiating dimensions, fill the matrix, name the compositions. The method that produced 15 social grammar operations, 13 domain grammars, ~145 domain operations, and 28 agent primitives produces 66 code graph primitives here.

**Why this matters:** Agents don't think in TypeScript or React. They think in intent. The code graph gives agents — and humans — a semantic layer to describe applications in terms of what they mean, not how they're implemented. Translation to React, SwiftUI, terminal UI, or any other target is mechanical. The semantic description is the source of truth, and it lives on the event graph — meaning every product decision is signed, causally linked, and auditable.

---

## Derivation Method

Start with the question: what does software DO that can't be decomposed further?

Software has four fundamental concerns:

1. **Data** — what exists
2. **Logic** — what happens
3. **Interface** — what connects (IO) and what humans see (UI)
4. **Quality** — what makes it usable, accessible, resilient, and beautiful

Each concern is decomposed by identifying the dimensional properties that distinguish one operation from another. When a proposed primitive decomposes cleanly into existing primitives, it's a composition, not a primitive. When it has unique dimensional properties — a combination of direction, timing, mutability, agency, or awareness not expressible by any existing primitive — it earns its place.

The full ontology (14 layers, 200 primitives) was then traversed layer by layer to identify dimensions and concerns missing from the initial derivation. This surfaced accessibility, temporality, social awareness, and several UI primitives not obvious from a pure code decomposition.

---

## Category 1: Data Primitives (6)

*What exists in the system.*

### Entity
A thing with identity. A task, a user, an agent, a message, a project, an invoice. The fundamental unit of data. Has properties. Persists. Referenceable by other entities and by events on the graph.

An Entity is not a database row — it's a node in the event graph with a full causal history. Every Entity was created by an event, modified by events, and its current state is the projection of all events that reference it.

```
Entity(Task, properties: [...])
Entity(User, properties: [...])
Entity(Sprint, properties: [...])
```

### Property
A named, typed value on an Entity. Title, status, assignee, timestamp, priority, description. Properties are the attributes that distinguish one entity from another and that humans and agents read, write, filter, and sort by.

Properties are typed: text, number, datetime, boolean, enum, relation. The type constrains what values are valid and determines how the property renders in UI (Display) and what inputs capture it (Input).

```
Property(title, type: text)
Property(status, type: State(todo, doing, review, done))
Property(assignee, type: Relation(Agent | Human))
Property(due, type: datetime)
Property(priority, type: State(p0, p1, p2))
```

### Relation
A directed, typed connection between Entities. Task belongs to Project. User assigned to Task. Agent delegated by Human. Relations are edges in the data graph, distinct from events (which are edges in the causal graph).

Relations have cardinality (one-to-one, one-to-many, many-to-many), direction (parent-child, peer), and optionally weight (trust strength, relevance score). They map directly to the weighted edges in the broader architecture.

```
Relation(Task -> Project, type: belongs_to, cardinality: many-to-one)
Relation(User -> Task, type: assigned_to, cardinality: many-to-many)
Relation(Agent -> Human, type: delegated_by, cardinality: many-to-one)
```

### Collection
A group of Entities of the same type. Filterable, sortable, countable, pageable. The set you operate on when you need more than one thing. A collection is not stored — it's a live projection defined by a Query.

```
Collection(Task, filter: sprint.current, sort: priority.desc)
Collection(User, filter: role.developer)
Collection(Event, filter: entity.task_42, sort: time.desc)
```

### State
A value that changes through defined transitions. A task's status: todo → doing → review → done. An agent's mode: idle → working → blocked → error. Finite, enumerable, and — critically — transitional: not every state can reach every other state. The valid transitions are themselves a constraint.

State is distinct from Property in that it has transition rules. A Property can be set to any valid value of its type. A State can only move along defined paths.

```
State(TaskStatus, values: [todo, doing, review, done], transitions: {
  todo -> doing,
  doing -> review,
  review -> done,
  review -> doing,  // rejection
  done -> todo      // reopen
})
```

### Event
Something that happened. State changed, Entity created, Relation formed, Command executed. This is the bridge to the event graph — every mutation in the code graph produces an Event that is signed, hash-chained, and causally linked. The Event is already defined in the broader architecture; here it serves as the code graph's connection point to the accountability substrate.

Events are immutable. They carry: actor (who), action (what), target (which entity), timestamp (when), cause (why — linked to the triggering event), and evidence (any supporting data).

```
Event(actor: user_42, action: transition, target: task_17, 
      from: doing, to: review, cause: event_1234)
```

---

## Category 2: Logic Primitives (6)

*What happens in the system.*

### Transform
Take input, produce output. Pure function. No side effects. Map a list of tasks to a list of titles. Filter a collection by status. Reduce a set of estimates to a total. Format a date for display. Calculate a trust score from weighted edges.

Transforms are composable — the output of one is the input of another. They never mutate state; they produce new values from existing ones.

```
Transform(tasks, map: task -> task.title)
Transform(tasks, filter: task.status == doing)
Transform(estimates, reduce: sum)
Transform(date, format: "DD MMM YYYY")
```

### Condition
If/then. Branch based on state, property value, authority level, or any evaluable expression. The fundamental decision point in logic. Every Trigger contains an implicit Condition. Every Constraint is a Condition that prevents rather than branches.

```
Condition(task.priority == p0, then: Alert(urgent), else: Display(normal))
Condition(user.authority >= required, then: Command(execute), else: Escalate)
```

### Sequence
Do this, then that. Ordered operations. A wizard is a Sequence of Forms. A deployment is a Sequence of Commands. A workflow is a Sequence of Tasks. Order matters — each step may depend on the output of the previous.

```
Sequence([
  Command(validate, target: form_data),
  Command(create, target: Entity(Task)),
  Feedback(success, "Task created"),
  Navigation(route: task_detail)
])
```

### Loop
Do this for each. Iteration over Collections. Render each task in a list. Send a notification to each subscriber. Evaluate each agent's trust score. The fundamental mechanism for operating on sets.

```
Loop(Collection(Task, filter: overdue), each: task -> 
  Alert(notification, target: task.assignee, message: "Task overdue"))
```

### Trigger
When X happens, do Y. Reactive, event-driven logic. The bridge between Events on the graph and Logic in the code. When a task transitions to done, emit a completion event. When budget hits 80%, attenuate the model. When a new member joins, send a welcome.

Triggers are persistent — they survive across sessions. They define the system's reactive behaviour. They are distinct from Conditions (one-time evaluation) in that they watch continuously.

```
Trigger(on: Task.status -> done,
  do: Command(Event.emit(complete, evidence: Task)))

Trigger(on: Budget.percent >= 80,
  do: Command(Model.attenuate(tier: cheaper)))

Trigger(on: Entity(Member).created,
  do: Sequence([
    Command(Event.emit(welcome)),
    Alert(notification, target: community.moderators)
  ]))
```

### Constraint
X must be true. Validation, invariant, guard clause. Prevents invalid state from entering the system. A task must have a title. A budget cannot go negative. An agent cannot exceed its authority scope. Constraints are Conditions that block rather than branch.

Constraints are checked before Commands execute. They are the code graph equivalent of the agent layer's Authorize — but applied to data integrity rather than permission.

```
Constraint(Task.title, required: true, message: "Tasks must have a title")
Constraint(Budget.balance, min: 0, message: "Budget cannot go negative")
Constraint(Task.assignee, valid: Authority.scope, message: "Cannot assign outside authority")
```

---

## Category 3: IO Primitives (6)

*What connects the system to data and the outside world.*

### Query
Request data from the graph. Structured: filter by property values, sort by fields, limit results, paginate. Returns a Collection. Queries are read-only — they never mutate state.

Queries compose with Transforms — query the data, then transform it for display or analysis.

```
Query(Entity: Task, 
  filter: { status: doing, assignee: current_user },
  sort: priority.desc,
  limit: 20,
  page: 1)
```

### Command
Request a change. Create an Entity, update a Property, transition a State, form a Relation, delete (tombstone) an Entity. Commands are the write operations. Every Command produces an Event on the graph.

Commands are checked against Constraints before execution and against Authorize before that. The chain: Authorize → Constraint → Command → Event.

```
Command(create, Entity(Task, { title: "Fix login bug", priority: p1 }))
Command(transition, Task.status, from: doing, to: review)
Command(relate, User -> Task, type: assigned_to)
```

### Subscribe
Watch for changes. Live updates. Push, not pull. When the data matching a Query changes, the subscriber is notified. This is how boards update in real-time, how feeds show new content, how dashboards reflect current state.

Subscribe is a persistent Query that re-evaluates when underlying Events occur.

```
Subscribe(Query(Task, filter: sprint.current), on_change: View.refresh)
Subscribe(Query(Event, filter: entity.task_42), on_change: Thread.append)
```

### Authorize
Check permission before a Command executes. Maps directly to the agent layer's Authority primitive. Does this actor have the scope to perform this action on this target? The answer comes from the authority chain — walk it back to the human who delegated.

Authorize is checked before every Command. If it fails, the Command doesn't execute and an Escalate event is emitted instead.

```
Authorize(actor: agent_7, action: transition, target: Task.status,
  required: Authority.scope.includes(task_management))
```

### Search
Unstructured data retrieval. Distinct from Query (structured filters). Search takes human intent — words, phrases, natural language — and finds matching Entities, Properties, Events. Full-text, fuzzy, semantic.

Search is how humans find things when they don't know the exact filter criteria. "That task about the login bug from last week."

```
Search("login bug last week", scope: [Task, Event], limit: 10)
```

### Interop
Connect to external systems. Send data out, receive data in, translate between formats. Email, calendar, Slack, webhooks, file systems, third-party APIs. The boundary between the event graph and everything else.

Every Interop interaction is itself an Event on the graph — the system records that data left or entered the boundary, when, through what channel, authorized by whom.

```
Interop(send, target: email, 
  payload: Transform(Task, format: email_template),
  authorize: current_user)

Interop(receive, source: webhook.github,
  transform: github_event -> Event(code_push),
  trigger: Task.relate(commit))

Interop(sync, target: calendar,
  query: Query(Task, filter: { due: not_null }),
  transform: task -> calendar_event)
```

---

## Category 4: UI Primitives (19)

*What humans see and interact with.*

### Display
Render a value. The atomic unit of visual output. A task title as text. A priority as a coloured badge. A date as formatted string. A trust score as a number. Display takes a Property value and renders it according to its type and the active Skin.

Display is read-only. Humans see it but don't interact with it directly.

```
Display(task.title, style: heading)
Display(task.priority, style: badge)
Display(task.due, style: relative_time)  // "3 days from now"
Display(agent.trust, style: meter)       // visual gauge
```

### Input
Capture a value from a human. The atomic unit of visual input. Text field, number stepper, date picker, dropdown selector, toggle switch, rich text editor. Input takes a Property type and renders the appropriate capture mechanism.

Inputs validate against Constraints in real-time — showing errors before the human submits.

```
Input(task.title, type: text, placeholder: "What needs doing?")
Input(task.priority, type: select, options: State.p0..p2)
Input(task.due, type: datepicker)
Input(task.assignee, type: entity_picker, scope: Query(User, filter: team.current))
```

### Layout
Arrange things spatially. The structural primitive. Stack (vertical), row (horizontal), grid (2D), split (resizable panels). Layout says nothing about what's inside it — only how the contents are arranged relative to each other.

Layouts nest — a grid containing rows containing stacks. They're the bones of every View.

```
Layout(type: grid, columns: 3, gap: spacing.md)
Layout(type: stack, direction: vertical, align: stretch)
Layout(type: split, ratio: [1, 2], resizable: true)
Layout(type: row, justify: space-between, align: center)
```

### List
Render a Collection. Each item rendered by a template. The fundamental mechanism for showing multiple entities. Lists are filterable, sortable, groupable, and pageable through their underlying Query.

Lists compose with Display, Action, and Avatar to render each item.

```
List(Query(Task, filter: sprint.current, sort: priority),
  template: Layout(row, [
    Avatar(task.assignee),
    Display(task.title),
    Display(task.status, style: badge),
    Action(transition, label: "→")
  ]))
```

### Form
A group of Inputs that compose into a Command. The mechanism for creating or updating Entities. A Form captures the data needed for a Command, validates against Constraints, and on submit executes the Command which produces an Event.

Forms can be single-step or multi-step (composed with Sequence to create wizards).

```
Form(command: Command(create, Entity(Task)), fields: [
  Input(title, required: true),
  Input(priority, default: p1),
  Input(assignee),
  Input(due)
], submit: Action(label: "Create Task"))
```

### Action
A button. The atomic unit of human intent. Triggers a Command, a Navigation, or a Sequence. Labelled with what it does, not how it works. "Complete," "Assign," "Escalate," "Delete."

Actions can be conditional — visible only when a Condition is met. An admin sees "Delete." A viewer doesn't.

```
Action(label: "Complete", command: Command(transition, Task.status -> done))
Action(label: "Escalate", command: Command(Event.emit(escalate)))
Action(label: "Delete", command: Command(tombstone, Task), 
  condition: Authorize(current_user, delete),
  confirm: Confirmation("Delete this task? This cannot be undone."))
```

### Navigation
Move between Views. Route. Context switch. Click a task in a list to see its Detail. Click "Board" in the sidebar to see the sprint board. Navigation changes what View is active without losing the underlying state.

Navigation can be hierarchical (drill down from project → sprint → task), lateral (switch between board and list views), or modal (open a detail overlay without leaving the current view).

```
Navigation(route: task_detail, params: { id: task.id })
Navigation(route: sprint_board, params: { sprint: current })
Navigation(type: modal, content: Form(create_task))
```

### View
A composition of Layout, Display, List, Form, Action, Navigation, and other primitives focused on a domain concept. "The sprint board." "The task detail page." "The activity feed." "The settings panel."

Views are the highest-level UI primitive. They're what users navigate between. Everything below View is a component within a View.

```
View(name: SprintBoard,
  layout: Layout(columns: State.values),
  content: Loop(State.values, each: status ->
    List(Query(Task, filter: { status: status, sprint: current }),
      template: TaskCard)))
```

### Feedback
System communicates outcome to human. Success, error, warning, info. The response to a Command. "Task created." "Permission denied." "Network error — try again." Feedback is always the result of an action, never ambient.

Feedback is temporal — it appears, persists briefly, and disappears. It doesn't occupy permanent space in the View.

```
Feedback(type: success, message: "Task created", duration: 3s)
Feedback(type: error, message: "Permission denied — requires admin role")
Feedback(type: warning, message: "This will affect 47 tasks")
```

### Alert
A directed notification to a specific human about something that requires their attention. Distinct from Feedback (response to their action). Alert is the system reaching out to say "something happened that you care about." Push, not pull.

Alerts have urgency levels that map to delivery mechanisms — in-app badge, push notification, email, SMS.

```
Alert(target: user_42, urgency: high,
  message: "Task overdue: Fix login bug",
  action: Navigation(route: task_detail, params: { id: task_17 }))

Alert(target: team.leads, urgency: critical,
  message: "Budget at 95% — model attenuation active")
```

### Thread
A persistent conversational context attached to an Entity. Comments on a task. Messages in a channel. Discussion on a proposal. Nested, chronological, contextual.

Threads compose Events (each message is an event), Display (rendering messages), Input (composing a reply), and Avatar (showing who said what).

```
Thread(entity: task_17, 
  events: Query(Event, filter: { target: task_17, type: comment }, sort: time.asc),
  compose: Input(type: rich_text),
  submit: Command(Event.emit(comment, target: task_17)))
```

### Avatar
Visual representation of an identity. Human photo, agent icon, system badge, team logo. The visual anchor that makes identity recognizable across every context — lists, threads, boards, feeds.

Avatar resolves from the Identity primitive in the agent layer. Humans have photos. Agents have generated or assigned icons. Systems have badges.

```
Avatar(entity: user_42, size: sm)      // 24px photo
Avatar(entity: agent_7, size: md)      // 32px icon with agent indicator
Avatar(entity: system, size: sm)       // system badge
```

### Audit
The "walk the chain" view. Display the complete event history for any Entity, any decision, any chain of causation. The UI primitive that makes accountability visible. Not History (prior states) but the full causal graph — who did what, why, what it caused, what authority it operated under.

Audit is arguably the most important UI primitive in the entire architecture because it's the one that delivers on "check the chain."

```
Audit(entity: task_17,
  events: Query(Event, filter: { target: task_17 }, sort: time.desc),
  template: Layout(row, [
    Display(event.timestamp, style: relative),
    Avatar(event.actor),
    Display(event.action),
    Display(event.cause, style: link)   // click to follow the causal chain
  ]),
  expand: View(event_detail))           // full event with authority chain
```

### Drag
Direct manipulation. Moving a task across columns on a board. Reordering a list by priority. Resizing a panel. The physical gesture of rearranging things spatially.

Drag produces a Command on drop — typically a State transition or a Property update. The drag itself is visual; the drop is a mutation that produces an Event.

```
Drag(source: TaskCard, target: StatusColumn,
  on_drop: Command(transition, Task.status, to: target.status),
  feedback: Display(ghost_card))
```

### Selection
Choosing one or more Entities to operate on. Before you can batch-Command, you need to batch-Select. Multi-select, range select, select all. Selection is a transient UI state — it doesn't produce Events until an Action is taken on the selection.

```
Selection(scope: List(Task), mode: multi,
  actions: [
    Action(label: "Move to Done", command: Loop(selected, each: 
      Command(transition, Task.status -> done))),
    Action(label: "Assign", command: Loop(selected, each:
      Command(update, Task.assignee, to: Input(entity_picker))))
  ])
```

### Confirmation
"Are you sure?" The gate before destructive or irreversible actions. Consent between human and system. Shows what will happen (Consequence Preview) and requires explicit approval before proceeding.

Confirmation is the UI expression of the agent layer's Consent primitive. Both events — the request and the confirmation — go on the chain.

```
Confirmation(
  message: "Delete 5 tasks? This cannot be undone.",
  consequence: Display("These tasks and their 23 comments will be permanently removed."),
  confirm: Action(label: "Delete", style: destructive),
  cancel: Action(label: "Cancel"))
```

### Empty
What a View looks like when there's nothing in it. The zero state. The first thing a new user sees. Not an error — a designed moment that invites action.

Empty is distinct from Loading (something is happening) and Fallback (something went wrong). Empty means: the system is working fine, there's just nothing here yet.

```
Empty(context: SprintBoard,
  message: "No tasks in this sprint yet.",
  action: Action(label: "Create first task", command: Navigation(modal: Form(create_task))),
  illustration: asset(empty_board))
```

### Loading
The state between Command and Feedback. Something is happening but hasn't resolved. Skeleton screens, spinners, progress bars. The in-between state that tells humans "I'm working on it."

Loading prevents the human from interpreting absence as emptiness. Without Loading, a slow Query looks like an Empty state, which is confusing and erodes trust.

```
Loading(type: skeleton, template: TaskCard, count: 5)   // placeholder cards
Loading(type: spinner, message: "Creating task...")
Loading(type: progress, percent: 67, message: "Importing 150 tasks...")
```

### Pagination
Navigate large Collections in chunks. Infinite scroll, numbered pages, "load more" button. The mechanism for handling Collections too large to render at once.

Pagination modifies the underlying Query's limit and offset parameters. It doesn't change what data exists — only how much is visible at once.

```
Pagination(type: infinite_scroll, page_size: 20, 
  query: Query(Task, filter: sprint.current))

Pagination(type: numbered, page_size: 50,
  query: Query(Event, filter: entity.project_1))

Pagination(type: load_more, page_size: 10, label: "Show older comments",
  query: Query(Event, filter: { target: task_17, type: comment }))
```

---

## Category 5: Aesthetic Primitives (7)

*How the system feels visually. The Skin layer — swappable without touching any other primitive.*

### Palette
Colours. Semantic, not literal. Primary, secondary, accent, surface, background, error, success, warning, info. A Palette defines the colour relationships; specific hex values are derived from the semantic names.

Palettes can be warm, cool, earth, monochrome, high-contrast. They can have light and dark variants. The semantic naming means every UI primitive that references colour does so by role, not by value.

```
Palette(name: earth,
  primary: #5C6B4F,
  secondary: #8B7355,
  accent: #C4956A,
  surface: #F5F0EB,
  background: #FDFCFA,
  error: #B85C5C,
  success: #5C8B6B)
```

### Typography
Font, weight, size, spacing, line height. A semantic scale: display, heading, subheading, body, caption, mono. Typography defines the hierarchy of text — what's important, what's secondary, what's fine print.

The scale is relative, not absolute. A "compact" Density shrinks the entire scale proportionally.

```
Typography(name: humanist,
  family: { sans: "Inter", serif: "Lora", mono: "JetBrains Mono" },
  scale: {
    display: { size: 36, weight: 700, leading: 1.2 },
    heading: { size: 24, weight: 600, leading: 1.3 },
    subheading: { size: 18, weight: 600, leading: 1.4 },
    body: { size: 15, weight: 400, leading: 1.6 },
    caption: { size: 13, weight: 400, leading: 1.5 },
    mono: { size: 14, weight: 400, leading: 1.5 }
  })
```

### Spacing
The rhythm between things. A consistent scale that governs all whitespace — padding, margins, gaps. Tight, comfortable, spacious. A single scale applied everywhere creates visual coherence.

Spacing is a multiplier system: define a base unit and derive all spacing from multiples of it.

```
Spacing(base: 4px, scale: {
  xs: 1,    // 4px
  sm: 2,    // 8px
  md: 4,    // 16px
  lg: 6,    // 24px
  xl: 8,    // 32px
  xxl: 12   // 48px
})
```

### Elevation
Depth. What sits on top of what. Shadow intensity, layering, z-ordering. Flat (no shadows, borders only), subtle (light shadows), raised (prominent shadows), floating (detached elements like modals and popovers).

Elevation creates visual hierarchy through depth rather than size or colour.

```
Elevation(levels: {
  flat: { shadow: none, border: 1px solid palette.border },
  subtle: { shadow: "0 1px 3px rgba(0,0,0,0.08)" },
  raised: { shadow: "0 4px 12px rgba(0,0,0,0.12)" },
  floating: { shadow: "0 8px 30px rgba(0,0,0,0.18)" }
})
```

### Motion
How things change. Transition duration, easing function, direction. Snappy (100ms, ease-out), gentle (250ms, ease-in-out), deliberate (400ms, ease), none (0ms, for accessibility/preference).

Motion applies to state changes in UI — a card moving to a new column, a modal appearing, a feedback message fading in.

```
Motion(profile: gentle, values: {
  fast: { duration: 100ms, easing: ease-out },
  normal: { duration: 250ms, easing: ease-in-out },
  slow: { duration: 400ms, easing: ease },
  none: { duration: 0ms }
})
```

### Density
How much fits. Compact (power users, data-heavy tables, small type, tight spacing) vs comfortable (default, balanced) vs relaxed (consumer-facing, content-focused, generous whitespace). Density is a meta-primitive — it scales Spacing, Typography, and Layout proportionally.

```
Density(level: comfortable, modifiers: {
  compact: { spacing: 0.75x, typography: 0.9x, padding: 0.75x },
  comfortable: { spacing: 1x, typography: 1x, padding: 1x },
  relaxed: { spacing: 1.25x, typography: 1.1x, padding: 1.25x }
})
```

### Shape
Border radius, edge treatment. The geometry of containers. Sharp (0px radius — technical, precise), rounded (4-8px — friendly, modern), pill (full radius — playful, prominent). Shape affects buttons, cards, inputs, avatars, badges.

```
Shape(profile: rounded, values: {
  none: 0px,
  sm: 4px,
  md: 8px,
  lg: 12px,
  full: 9999px   // pill
})
```

### Skin (Composition)
The complete aesthetic identity. A Skin combines all seven aesthetic primitives into a single swappable unit. Change the Skin, change the entire visual feel without touching data, logic, or UI structure.

```
Skin(name: lovatts_professional,
  palette: Palette(corporate_blue),
  typography: Typography(system),
  spacing: Spacing(base: 4px),
  elevation: Elevation(subtle),
  motion: Motion(snappy),
  density: Density(compact),
  shape: Shape(rounded))

Skin(name: lovyou_warm,
  palette: Palette(earth),
  typography: Typography(humanist),
  spacing: Spacing(base: 4px),
  elevation: Elevation(subtle),
  motion: Motion(gentle),
  density: Density(comfortable),
  shape: Shape(rounded))
```

---

## Category 6: Accessibility Primitives (4)

*Ensuring every human can use the system. From Layer 7 (Ethics): Dignity and Care applied to interface design.*

### Announce
What the screen reader says. Semantic labelling for non-visual access. Every Display, Action, Input, and View needs an Announce equivalent — a text description of what it is and what it does.

Announce is not optional. It's a primitive because without it, the interface excludes people, which violates the architecture's core value of Dignity.

```
Announce(target: Action(complete), label: "Mark task as complete")
Announce(target: Display(priority, style: badge), label: "Priority: critical")
Announce(target: View(SprintBoard), label: "Sprint board showing 12 tasks across 4 columns")
Announce(live: true, message: "Task moved to Done")  // dynamic announcement
```

### Focus
What has keyboard/input attention. Navigation without a mouse. Tab order, focus rings, keyboard shortcuts. The ability to operate the entire interface without pointing.

Focus defines the order in which elements receive attention and what visual indicator shows where attention currently is.

```
Focus(order: [navigation, board_columns, task_cards, actions])
Focus(trap: modal)          // focus can't leave the modal until dismissed
Focus(shortcut: "Cmd+K", target: Search)
Focus(shortcut: "N", target: Form(create_task))
```

### Contrast
The ratio between foreground and background. Not an aesthetic choice — an accessibility requirement. WCAG AA requires 4.5:1 for normal text, 3:1 for large text. Contrast is a Constraint on Palette: the aesthetic primitive must satisfy the accessibility primitive.

```
Contrast(minimum: 4.5, context: body_text)
Contrast(minimum: 3.0, context: heading_text)
Contrast(minimum: 3.0, context: interactive_elements)
```

### Simplify
Reduced motion, reduced complexity, reduced cognitive load. The accessibility expression of the agent layer's Attenuation. Some humans need less: less animation, fewer simultaneous elements, simpler layouts, clearer language.

Simplify is preference-driven (the user requests it) or context-driven (the system detects high cognitive load).

```
Simplify(motion: none)                    // disable all animation
Simplify(density: relaxed, layout: single_column)  // reduce visual complexity
Simplify(language: plain)                 // shorter sentences, common words
```

---

## Category 7: Temporal Primitives (3)

*How the interface relates to time. From Layer 6 (Information) primitives.*

### Recency
How fresh is this data. "Updated 3 seconds ago" vs "Last modified 6 months ago." Staleness as a visible property. Recency affects trust — data that hasn't been updated in months may no longer be accurate.

Recency renders as relative time ("3 minutes ago"), absolute time ("March 8, 2026 at 14:32"), or staleness indicators (green/yellow/red based on age).

```
Recency(entity: task_17, property: updated_at, 
  display: relative,
  stale_after: 7d,           // yellow indicator after 7 days
  expired_after: 30d)        // red indicator after 30 days
```

### History
The ability to see prior states. What did this task look like before the last edit? What was the status yesterday? Undo comparison. The append-only event graph makes this native — every prior state is reconstructable by replaying events up to a given timestamp.

History is distinct from Audit (who did what and why). History shows the entity's evolution. Audit shows the causal chain of decisions.

```
History(entity: task_17, 
  timeline: Query(Event, filter: { target: task_17 }, sort: time.desc),
  compare: true,        // show diff between versions
  restore: Action(label: "Restore this version", 
    command: Command(revert, Task, to: event_id)))
```

### Liveness
Is this data updating in real-time? Is someone else editing this right now? Liveness is the visual expression of Subscribe — showing that the connection is active and changes are flowing.

Without Liveness indicators, humans can't tell whether they're seeing current state or a stale snapshot.

```
Liveness(indicator: true, label: "Live")           // green dot
Liveness(heartbeat: 5s, stale_after: 30s)          // detect connection loss
Liveness(collaborators: Query(Session, filter: { viewing: task_17 }),
  display: Avatar(each))                            // "Sarah is viewing"
```

---

## Category 8: Resilience Primitives (4)

*What happens when things go wrong or conditions are imperfect.*

### Undo
Reverse a Command. "I didn't mean to do that." On an append-only graph, Undo is a new Event that reverses a prior Event — the original survives, and the undo itself is recorded. Not deletion. Reversal with provenance.

Undo has a scope — some Commands are undoable (status transitions, property edits) and some aren't (after Confirmation, or after downstream consequences have propagated).

```
Undo(event: event_1234,
  command: Command(transition, Task.status, from: done, to: review),
  window: 30s)              // undo available for 30 seconds after action

Undo(scope: last_action)    // generic "undo last thing I did"
```

### Retry
Attempt the Command again. Network failed. Timeout. Service unavailable. Try once more. Retry is automatic or manual, with configurable attempts and backoff.

```
Retry(command: Command(create, Task), 
  attempts: 3, 
  backoff: exponential,
  on_failure: Feedback(error, "Could not create task. Try again later."))
```

### Fallback
What shows when the real thing can't load. Skeleton screens, cached data, degraded functionality. Not Empty (nothing exists) and not Loading (actively fetching). Fallback is "something went wrong but here's the best we can do."

Fallback is the UI expression of the agent layer's Attenuation — graceful degradation rather than failure.

```
Fallback(for: View(SprintBoard),
  show: Display(cached_board, stale: true, label: "Showing cached data"),
  retry: Action(label: "Retry", command: Query(refresh)))

Fallback(for: Avatar(user_42),
  show: Display(initials: "MS", style: circle))
```

### Offline
Continuing to work without connectivity. Commands queue locally, Events are created locally, sync happens when connection restores. The append-only graph makes this natural — local chain merges with remote chain on reconnect.

Offline affects every other primitive: which Commands can execute locally? Which Queries return cached results? Which Subscribes pause? The UI needs to communicate all of this.

```
Offline(mode: active,
  indicator: Display("Offline — changes will sync when connected"),
  queue: [Command(pending)],
  available: [Query(cached), Command(create), Command(transition)],
  unavailable: [Search, Interop, Subscribe(live)])
```

---

## Category 9: Structural Primitives (3)

*How the system is organized internally.*

### Scope
Isolation. This component can't see outside itself. This data doesn't leak across boundaries. Tenant separation. Component encapsulation. Security boundary.

Scope is the code graph expression of Layer 0's Boundary primitive. It defines what CAN'T bleed across boundaries — distinct from Authorize (who is allowed) in that Scope is about structural isolation, not permission.

```
Scope(type: tenant, boundary: organization_id)     // data isolation
Scope(type: component, boundary: View)              // UI isolation
Scope(type: authority, boundary: delegation_chain)   // permission isolation
```

### Format
How data is encoded for display or export. Markdown, CSV, PDF, JSON, plain text, HTML. Transform covers data manipulation; Format is specifically about representation for consumption by humans or external systems.

Format composes with Interop for export and with Display for rendering.

```
Format(data: Collection(Task), as: csv, 
  columns: [title, status, assignee, due])

Format(data: Audit(task_17), as: pdf,
  template: audit_report)

Format(data: Event(comment), as: markdown)
```

### Gesture
Touch-native interactions. Swipe, pinch, long-press, pull-to-refresh. The physical vocabulary of mobile interfaces. Gestures map to Commands — swipe right to approve, pull down to refresh, long-press to select.

Gesture is distinct from Drag (spatial rearrangement on any platform) in that it's specifically about touch input patterns.

```
Gesture(type: swipe_right, target: TaskCard,
  command: Command(transition, Task.status.next),
  feedback: Display(action_preview: "Complete"))

Gesture(type: pull_to_refresh, target: List,
  command: Query(refresh))

Gesture(type: long_press, target: TaskCard,
  command: Selection(toggle))
```

---

## Category 10: Social Primitives (3)

*Awareness of other humans and agents in the system. From Layer 3 (Society) and Layer 9 (Relationship).*

### Presence
Who else is here right now. "3 people viewing this board." "Sarah is editing this task." Collaborative awareness. The visual expression of the fact that this is a shared space, not a solo tool.

Presence composes Subscribe (watching for session events) with Avatar (showing who) and Liveness (showing they're active).

```
Presence(scope: View(SprintBoard),
  display: Layout(row, Loop(active_sessions, each: Avatar(session.user))),
  label: Transform(count, format: "{n} people viewing"))

Presence(scope: Entity(task_17),
  editing: Avatar(session.user, indicator: "editing"),
  viewing: Avatar(session.user, indicator: "viewing"))
```

### Salience
What matters right now. Not all information is equally important. A task that's overdue is more salient than one due next week. A budget at 95% is more salient than one at 40%. A trust score that just dropped is more salient than one that's stable.

Salience is conditional visual weight — Display rendered with emphasis based on state. The red badge. The bold number. The pulsing indicator.

```
Salience(condition: Task.due < now, style: { color: palette.error, weight: bold })
Salience(condition: Budget.percent > 90, style: { Display: pulsing, Alert: auto })
Salience(condition: Trust.delta < -0.1, style: { color: palette.warning, icon: trending_down })
```

### Consequence Preview
Showing the human what will happen before they confirm. "You're about to delete 47 tasks and their 156 comments. This cannot be undone." The visual representation of impact assessment.

Distinct from Confirmation (the gate) and Feedback (after the fact). Consequence Preview is the information that makes Confirmation meaningful — without it, "Are you sure?" is a meaningless ritual.

```
Consequence Preview(command: Command(delete, Sprint),
  impact: Query(count, [
    { label: "tasks", count: Transform(sprint.tasks, count) },
    { label: "comments", count: Transform(sprint.tasks.comments, count) },
    { label: "events", count: Transform(sprint.events, count) }
  ]),
  warning: "This action cannot be undone.",
  display: Layout(stack, Loop(impact, each: Display("{count} {label} will be removed"))))
```

---

## Category 11: Audio Primitive (1)

*What humans hear. The non-visual modality with unique dimensional properties.*

Surfaced by applying Blind (Need(Need)) to the spec: Sound has dimensions — modality, timing, spatiality, frequency, amplitude — not expressible by any composition of visual primitives. If the Code Graph claims "any application," non-visual output must be representable.

### Sound
Audio output. A notification chime, a message-received tone, a voice channel audio stream, spatial audio positioning in a collaborative space, text-to-speech, a background ambient drone. Sound exists on dimensions no visual primitive covers: frequency (pitch), amplitude (volume), spatial position (stereo/3D), waveform (tone vs noise), and duration pattern (instant/sustained/looping).

Sound is distinct from Display (visual output) in modality, from Alert (notification) in that Alert is a routing decision while Sound is the medium, and from Feedback (response to action) in that Sound can be ambient/environmental, not just reactive.

```
Sound(type: notification, tone: chime, volume: 0.6)
Sound(type: ambient, source: voice_channel, spatial: true)
Sound(type: tts, text: "Task completed", voice: system)
Sound(type: alert, tone: urgent, pattern: repeat(3))
```

**This brings the Code Graph to 66 primitives across 11 categories.**

---

## Named Compositions: Product Patterns

*How primitives compose into the products humans actually use.*

### Board
The kanban / sprint board / pipeline view. The most common way to visualize work across states.

```
Board(entity: Task, group_by: State(status), 
  filter: { sprint: current },
  columns: Loop(State.values, each: status ->
    Layout(stack, [
      Display(status, style: heading),
      Display(Transform(tasks, filter: status, count), style: caption),
      List(Query(Task, filter: { status: status }, sort: priority),
        template: TaskCard,
        drag: Drag(on_drop: Command(transition, Task.status, to: target.status))),
      Empty(message: "No {status} tasks", 
        action: Action("Create", command: Navigation(modal: Form(create_task, defaults: { status: status }))))
    ])))
```

### Detail
Single entity view with full context. Properties, related entities, history, audit trail, thread.

```
Detail(entity: Task,
  layout: Layout(split, ratio: [2, 1]),
  main: Layout(stack, [
    Display(task.title, style: display),
    Form(fields: [task.status, task.priority, task.assignee, task.due], inline: true),
    Display(task.description, format: markdown),
    Thread(entity: task, label: "Comments")
  ]),
  sidebar: Layout(stack, [
    Audit(entity: task, compact: true),
    List(Query(Relation, filter: { source: task }), label: "Related"),
    History(entity: task, compact: true)
  ]))
```

### Feed
Activity stream. What's happening across the system. The social network view. The event log made human-readable.

```
Feed(query: Query(Event, 
    filter: { actor: subscriptions.current_user },
    sort: time.desc),
  template: Layout(row, [
    Avatar(event.actor),
    Layout(stack, [
      Display(Transform(event, format: human_readable)),
      Display(event.timestamp, style: relative),
      Display(event.cause, style: link)
    ])
  ]),
  subscribe: Subscribe(on_change: prepend),
  pagination: Pagination(type: infinite_scroll, page_size: 20))
```

### Dashboard
Metrics, counts, charts. Aggregate views across Collections. The executive summary.

```
Dashboard(layout: Layout(grid, columns: 4),
  widgets: [
    Display(Transform(Query(Task, filter: sprint.current), count), 
      label: "Total Tasks", style: metric),
    Display(Transform(Query(Task, filter: { status: done, sprint: current }), count),
      label: "Completed", style: metric),
    Display(Transform(Query(Task, filter: { due: overdue }), count),
      label: "Overdue", style: metric, salience: Salience(condition: count > 0)),
    Display(Transform(Query(Event, filter: { period: today }), count),
      label: "Events Today", style: metric),
  ])
```

### Inbox
What needs my attention. Filtered by relevance to the current human/agent. Prioritized by urgency.

```
Inbox(query: Query(Event, 
    filter: { requires_action: current_user, resolved: false },
    sort: [urgency.desc, time.desc]),
  template: Layout(row, [
    Salience(condition: event.urgency == critical, style: indicator),
    Avatar(event.actor),
    Display(Transform(event, format: inbox_summary)),
    Display(event.timestamp, style: relative),
    Action(label: "Open", command: Navigation(route: event.target))
  ]),
  empty: Empty(message: "All caught up.", illustration: asset(inbox_zero)),
  selection: Selection(actions: [
    Action("Mark resolved", command: Loop(selected, each: Command(resolve))),
    Action("Snooze", command: Loop(selected, each: Command(snooze, until: Input(datetime))))
  ]))
```

### Wizard
Multi-step creation. The birth wizard for agents. Project setup. Onboarding flow. Any process where capturing information happens in stages.

```
Wizard(name: CreateProject, steps: Sequence([
  Form(label: "Basics", fields: [
    Input(name, required: true),
    Input(description),
    Input(type, type: select, options: [sprint, kanban, freeform])
  ]),
  Form(label: "Team", fields: [
    Input(members, type: entity_picker, scope: Query(User), multi: true),
    Input(lead, type: entity_picker, scope: Query(User))
  ]),
  Form(label: "Settings", fields: [
    Input(visibility, type: select, options: [public, private, team]),
    Input(budget, type: number, optional: true)
  ]),
  View(label: "Review", content: Layout(stack, [
    Display(summary, format: review_template),
    Consequence Preview(command: Command(create, Project))
  ]))
]),
  submit: Action(label: "Create Project", command: Command(create, Project)),
  navigation: Layout(row, [Action("Back"), Display(progress), Action("Next")]))
```

### Permission
Who can do what to which things. Not a primitive — it decomposes into Relation + Collection + Authorize. But it recurs across every application.

```
Permission(actor: Role(admin),
  actions: [Command(create), Command(delete), Command(transition)],
  scope: Scope(type: tenant, boundary: organization_id))

Permission(actor: Role(member),
  actions: [Command(create), Query, Command(transition, from: [todo, doing])],
  scope: Scope(type: space, boundary: space_id))
```

### Notification
A directed delivery with urgency-based routing. Alert is the UI; Notification is the mechanism that decides *how* to deliver. Decomposes into Trigger + Condition + Alert + Interop.

```
Notification(event: Task.assigned,
  recipient: task.assignee,
  urgency: Condition(
    task.priority == urgent, then: push,
    task.priority == high, then: in_app,
    else: badge_only),
  template: "{actor} assigned you: {task.title}",
  action: Navigation(route: task_detail))
```

### Error
A structured representation of what went wrong, why, and how to recover. Decomposes into Event(type: error) + State + Feedback + Retry/Fallback. But error handling recurs universally.

```
Error(type: validation, target: Form(create_task),
  fields: [
    { field: title, message: "Required", severity: blocking },
    { field: due, message: "Must be in the future", severity: warning }
  ],
  recovery: Focus(field: title))

Error(type: network, command: Command(create, Task),
  message: "Connection lost",
  recovery: Sequence([
    Retry(attempts: 3, backoff: exponential),
    Fallback(show: Offline(queue: command))
  ]))
```

---

## Convergence Analysis

*Applying the cognitive grammar (Post 43) to the Code Graph itself.*

The spec was subjected to the full cognitive grammar method: Need row (Audit, Cover, Blind), then Traverse row (Trace, Zoom, Explore), then Derive row (Formalize, Map, Catalog). Pipeline ordering per Post 44.

### What Audit found (Need(Derive))

The derivation claims 4 fundamental concerns → 10 categories → 65 primitives. The path from 4 concerns to 10 categories was implicit. Made explicit:

| Concern | Categories | Primitives |
|---------|-----------|-----------|
| **Data** | Data | 6 |
| **Logic** | Logic | 6 |
| **Interface** | IO + UI | 6 + 19 = 25 |
| **Quality** | Aesthetic + Accessibility + Temporal + Resilience + Structural + Social + Audio | 7 + 4 + 3 + 4 + 3 + 3 + 1 = 25 |

Quality decomposes into 7 sub-concerns, matching the 7 aspects of the question "what makes software good?": beautiful (Aesthetic), accessible (Accessibility), timely (Temporal), resilient (Resilience), structured (Structural), social (Social), and audible (Audio).

### What Cover found (Need(Traverse))

- **Sound** — genuine primitive gap. Unique modality with dimensions (frequency, amplitude, spatiality) not expressible by visual primitives. Added as Category 11. (1 primitive)
- **Permission, Notification, Error** — recurrent compositions, not primitives. Added as Named Compositions. (0 new primitives)
- **Security** (authentication, encryption, sanitization) — decomposes into Authorize + Constraint + Scope + Interop. Not primitives.
- **Internationalization** — decomposes into Format + Transform + Condition. Not a primitive.
- **Collaboration** (conflict resolution, operational transforms) — decomposes into Subscribe + Command + Liveness + Presence. Not primitives at this layer.

### What Blind found (Need(Need))

The spec is web-application-centric in examples, not in primitives. The primitives are abstract enough for CLI (Display = text output, Input = keyboard input), embedded (Entity + State + Trigger), and audio (Sound now covered). The gap was real: Sound was absent. With Sound added, the modality coverage is: visual (Display, Layout, etc.) + textual (Format) + audio (Sound). Tactile (haptic) is not covered — but decomposes into Feedback(type: haptic) + a hardware Interop. Not a primitive.

### What Trace confirmed (Traverse(Derive))

The UI category (19 primitives) has 5 implicit sub-groups. Making them explicit:

| Sub-group | Primitives | Pattern |
|-----------|-----------|---------|
| **Output/Input** | Display, Input | Read/write duality |
| **Composition** | Layout, List, Form, View | Increasing scope: arrange → collection → structured input → full page |
| **Intent** | Action, Navigation | Human triggers: do something, go somewhere |
| **System response** | Feedback, Alert, Thread | System communicates: in response, proactively, persistently |
| **Interaction** | Avatar, Audit, Drag, Selection, Confirmation, Empty, Loading, Pagination | Identity, accountability, manipulation, batch, consent, states |

### Fixpoint check

Pass 2 of Need (Audit, Cover, Blind) on the updated spec:
- **Audit:** 4 concerns → 11 categories → 66 primitives. Derivation path now explicit. No new gaps.
- **Cover:** All modalities represented (visual, textual, audio). All interface concerns covered. No new territory.
- **Blind:** Invited external perspectives (game dev, data scientist, accessibility, legal, performance). All decompose into existing primitives or the new Named Compositions.

**Convergence reached at pass 2.** The grammar found one genuine primitive (Sound), three Named Compositions (Permission, Notification, Error), and structural clarification (Quality decomposition, UI sub-groups). No further passes produce new primitives.

---

## The Full Stack

From hash chain to pixel, the complete primitive inventory:

| Layer | Primitives | Count |
|-------|-----------|-------|
| Event Graph | Hash chain, append-only store, causal links, signatures | (infrastructure) |
| Social Grammar | Emit, Respond, Derive, Extend, Retract, Annotate, Acknowledge, Propagate, Endorse, Subscribe, Channel, Delegate, Consent, Sever, Merge | 15 |
| Domain Grammars | ~145 operations across 13 domains | ~145 |
| Agent Layer | Identity, Soul, Model, Memory, State, Authority, Trust, Budget, Role, Lifespan, Goal, Observe, Probe, Evaluate, Decide, Act, Delegate, Escalate, Refuse, Learn, Introspect, Communicate, Repair, Expect, Consent, Channel, Composition, Attenuation | 28 |
| Code Graph | Data, Logic, IO, UI, Aesthetic, Accessibility, Temporal, Resilience, Structural, Social, Audio | 66 |
| **Total** | | **~254 + infrastructure** |

One method. One graph. Primitives all the way down. Compositions all the way up.

Any application — task manager, social network, marketplace, justice system, knowledge base, AI governance dashboard — is a composition of these primitives, running on the same event graph, with every decision signed, causally linked, and auditable.

Not "trust us, we designed it well." Walk the chain. Check the code graph. Verify the primitives.

---

*Matt Searles is the founder of LovYou. Claude is an AI made by Anthropic. They built this together.*

*The code: github.com/lovyou-ai/eventgraph*

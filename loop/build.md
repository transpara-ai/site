# Build Report — Goal progress dashboard with task aggregation

## Gap
Goals lens showed goals with project counts but no task progress. Clicking a goal went to the generic node detail page, which wasn't goal-aware. No aggregate view of progress across all projects under a goal.

## Changes

### `graph/handlers.go`
- Added `GoalDetail` struct: `Goal`, `Projects []Node`, `DirectTasks []Node`, `TotalTasks int`, `DoneTasks int`
- Added `handleGoalDetail` handler: fetches goal by ID, loads projects (kind=project, parent=goal), loads direct tasks (kind=task, parent=goal), aggregates `TotalTasks`/`DoneTasks` by summing project `ChildCount`/`ChildDone` + counting direct task states
- Registered route `GET /app/{slug}/goals/{id}` (more specific than `/goals`, takes precedence)

### `graph/views.templ`
- Added `GoalDetailView` template: breadcrumb → goal header → progress card → projects list → direct tasks list
- Progress card shows overall `done/total` count, wide progress bar, project/task counts, and percentage
- Each project row has its own mini progress bar (reusing `childProgress`)
- Each direct task row shows state dot, title, assignee, state badge
- Added `goalDetailProgress(detail GoalDetail) string` helper for the aggregate progress bar
- Added `stateColorClass(state string) string` helper for task state indicator dots
- Updated GoalsView: goal cards now link to `/app/{slug}/goals/{id}` instead of `/node/{id}`

## Verification
- `templ generate`: 15 files updated, no errors
- `go build -buildvcs=false ./...`: clean
- `go test ./...`: all pass (graph 0.541s)

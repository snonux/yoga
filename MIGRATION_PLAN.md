# Yoga TUI to Fyne GUI Migration Plan

## Overview
Convert the existing Yoga TUI application (built with Bubble Tea) to a Fyne-based GUI application while maintaining all existing features and adding video preview thumbnails.

## Current Features
- Video browsing table (name, duration, age, tags)
- Multi-filter system (name substring, duration range, tag substring)
- Multi-column sorting (name, duration, age)
- VLC playback with optional crop aspect ratio
- Tag editing via JSON files per video
- Duration probing with ffprobe + caching
- Random video selection
- Re-index functionality
- Progress indicators

## Package Structure

### Existing (to be kept/reused)
```
internal/fsutil/          # Path utilities
internal/meta/            # Version
internal/tags/            # Tag persistence
internal/app/             # Domain logic
├── video.go              # Video struct (extend for thumbnails)
├── loader.go             # Video loading (extend for thumbnails)
├── filters.go            # Filter logic
├── model_sort.go         # Sorting logic
├── duration_cache.go     # Duration caching
├── tag_commands.go       # Tag operations
└── model_tags.go         # Tag editing
```

### New packages
```
internal/gui/             # GUI-specific code
├── app.go                # Fyne app lifecycle
├── main_window.go        # Main window + layout
├── video_list.go         # Video list widget
├── preview_panel.go      # Video preview thumbnails
├── filter_dialog.go      # Filter UI
├── tag_dialog.go         # Tag editing UI
├── status_bar.go         # Status display
└── thumbnail_cache.go    # Preview image caching

internal/thumbnail/        # Thumbnail generation
├── generator.go          # ffmpeg-based thumbnail extraction
├── cache.go              # Thumbnail file management
└── config.go             # Thumbnail settings
```

### To be removed
```
internal/app/
├── app.go                # Bubble Tea program
├── model.go              # TUI model state
├── style.go              # Terminal styling
├── messages.go           # Bubble Tea messages
├── view_helpers.go       # TUI rendering
├── model_keys.go         # TUI key handlers
├── model_durations.go    # Duration UI updates (adapt)
└── options.go            # Keep but may need updates
```

## Data Model Extensions

### internal/app/video.go
```go
type video struct {
    Name               string
    Path               string
    Duration           time.Duration
    ModTime            time.Time
    Size               int64
    Tags               []string
    Thumbnail          string         // Path to thumbnail image
    ThumbnailGenerated bool            // Generation status
}
```

## Phase 1: Foundation & Architecture

### 1.1 Dependency Migration
- [ ] Add fyne.io/fyne/v2 to go.mod
- [ ] Remove charmbracelet/bubbletea dependencies
- [ ] Test basic Fyne application runs

### 1.2 Package Structure Setup
- [ ] Create internal/gui/ package
- [ ] Create internal/thumbnail/ package
- [ ] Move reusable domain logic to proper packages

### 1.3 Data Model Updates
- [ ] Add thumbnail fields to video struct
- [ ] Update video struct comments
- [ ] Ensure backward compatibility with existing tests

## Phase 2: Thumbnail System

### 2.1 Thumbnail Generation
- [ ] Implement internal/thumbnail/generator.go
  - ffmpeg command to extract frame at 10% duration
  - Output JPEG thumbnails (320x180 or auto-detect)
  - Error handling for missing ffmpeg
- [ ] Unit tests for thumbnail generation logic
- [ ] Integration tests with mock ffmpeg

### 2.2 Thumbnail Cache
- [ ] Implement internal/thumbnail/cache.go
  - Check thumbnail existence before generation
  - Cache metadata: .video_thumbnails.json
  - ModTime detection for regeneration
- [ ] Unit tests for cache logic

### 2.3 Integration with Video Loading
- [ ] Extend internal/app/loader.go to check/generate thumbnails
- [ ] Add background goroutine for thumbnail generation
- [ ] Update video loading tests

## Phase 3: GUI Main Window

### 3.1 Window Layout
- [ ] Implement internal/gui/app.go
  - Fyne app initialization
  - Window creation and lifecycle
- [ ] Implement internal/gui/main_window.go
  - Split container (preview + list)
  - Responsive layout
- [ ] Implement internal/gui/status_bar.go
  - Status display widget
  - Progress indicators

### 3.2 Video List Widget
- [ ] Implement internal/gui/video_list.go
  - Custom widget.List with thumbnails
  - Click to select, double-click to play
  - Highlight selected item
- [ ] Handle keyboard navigation
- [ ] Performance testing with large collections

### 3.3 Thumbnail Preview Panel
- [ ] Implement internal/gui/preview_panel.go
  - Large thumbnail display
  - Video metadata labels
  - Action buttons (Play, Edit Tags)

## Phase 4: Interactive Components

### 4.1 Filter Dialog
- [ ] Implement internal/gui/filter_dialog.go
  - dialog.NewForm() with name, min/max duration, tags
  - Real-time filter count
  - Apply/Cancel buttons
- [ ] Reuse internal/app/filters.go logic

### 4.2 Tag Editor Dialog
- [ ] Implement internal/gui/tag_dialog.go
  - dialog.NewForm() with tag entry
  - Current tags display
  - Save/Cancel buttons
- [ ] Reuse internal/tags/ package

### 4.3 Sort Options
- [ ] Implement sort menu
  - Sort by Name/Duration/Age
  - Ascending/Descending toggle
- [ ] Reuse internal/app/model_sort.go logic

## Phase 5: Background Operations

### 5.1 Goroutine Management
- [ ] Implement async video loading
- [ ] Implement async thumbnail generation
- [ ] Implement async duration probing
- [ ] Use Fyne's WorkerPool for CPU-bound tasks

### 5.2 UI Updates from Background
- [ ] Implement callback pattern for UI updates
- [ ] Ensure all UI updates on main thread
- [ ] Handle cancellation on window close

## Phase 6: Integration & Polish

### 6.1 Keyboard Shortcuts
- [ ] Ctrl+F - Filter dialog
- [ ] Ctrl+T - Edit tags
- [ ] Ctrl+R - Random video
- [ ] Ctrl+I - Re-index
- [ ] Enter - Play selected
- [ ] Delete - Clear filters

### 6.2 Window Management
- [ ] Save/restore window size
- [ ] Save/restore split pane position
- [ ] Use Fyne storage preferences API

### 6.3 Error Handling
- [ ] Dialog boxes for missing dependencies
- [ ] Graceful degradation on thumbnail failures
- [ ] User-friendly error messages

## Phase 7: Testing & Documentation

### 7.1 Testing
- [ ] Unit tests for all new GUI components
- [ ] Integration tests for video loading with thumbnails
- [ ] Mock external commands (ffmpeg, ffprobe, vlc)
- [ ] Maintain 85%+ code coverage

### 7.2 Documentation
- [ ] Update README.md with GUI usage
- [ ] Add GUI screenshots
- [ ] Document thumbnail cache location
- [ ] Update keyboard shortcuts documentation

## Technical Considerations

### Fyne-Specific Challenges
1. **Async UI Updates** - All UI updates must be on main thread
2. **Image Loading** - Lazy loading with LRU cache
3. **List Performance** - Widget.List is already virtualized
4. **Cross-Platform** - Detect ffmpeg/ffprobe/vlc at runtime

### Code Reuse Strategy
**Keep 100% of:**
- internal/fsutil - Path utilities
- internal/meta - Version
- internal/tags - Tag persistence
- internal/app/loader.go - Video collection (extend)
- internal/app/filters.go - Filter logic
- internal/app/model_sort.go - Sorting logic
- internal/app/duration_cache.go - Duration caching

**Remove 100% of:**
- All Bubble Tea code
- Terminal-specific styling (internal/app/style.go)
- TUI model state management

**Adapt:**
- internal/app/video.go - Add thumbnail field
- internal/app/model_durations.go - Adapt for GUI updates
- internal/app/model_tags.go - Reuse tag commands
- cmd/yoga/main.go - Replace TUI init with Fyne app

## Dependencies

### New Dependencies
```go
require (
    fyne.io/fyne/v2 v2.4.0
)
```

### System Requirements
- ffmpeg (for thumbnails)
- ffprobe (already used for duration)
- vlc (for playback)

## Timeline

- **Week 1:** Phase 1 - Foundation & Architecture
- **Week 2:** Phase 2 - Thumbnail System
- **Week 3:** Phase 3 - GUI Main Window
- **Week 4:** Phase 4 - Interactive Components
- **Week 5:** Phase 5 - Background Operations
- **Week 6:** Phase 6 - Integration & Polish
- **Week 7:** Phase 7 - Testing & Documentation

## Build Commands

```bash
# Build
mage build

# Test
mage test

# Coverage
mage coverage

# Install
mage install
```

## Git Commit Milestones

1. Phase 1 complete: Foundation and architecture
2. Phase 2 complete: Thumbnail system
3. Phase 3 complete: Main window
4. Phase 4 complete: Interactive components
5. Phase 5 complete: Background operations
6. Phase 6 complete: Integration and polish
7. Phase 7 complete: Testing and documentation
8. Final: Release-ready GUI application

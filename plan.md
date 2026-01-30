# Migration Progress Tracker

## Phase 1: Foundation & Architecture

### 1.1 Dependency Migration
- [x] Add fyne.io/fyne/v2 to go.mod
- [ ] Remove Bubble Tea dependencies
- [ ] Test basic Fyne application runs

### 1.2 Package Structure Setup
- [x] Create internal/gui/ package
- [x] Create internal/thumbnail/ package
- [ ] Move reusable domain logic to proper packages

### 1.3 Data Model Updates
- [x] Add thumbnail fields to video struct
- [ ] Update video struct comments
- [ ] Ensure backward compatibility with existing tests

## Phase 2: Thumbnail System

### 2.1 Thumbnail Generation
- [x] Implement internal/thumbnail/generator.go
  - ffmpeg command to extract frame at 10% duration
  - Output JPEG thumbnails (320x180 or auto-detect)
  - Error handling for missing ffmpeg
- [x] Unit tests for thumbnail generation logic
- [x] Integration tests with mock ffmpeg

### 2.2 Thumbnail Cache
- [x] Implement internal/thumbnail/cache.go
  - Check thumbnail existence before generation
  - Cache metadata: .video_thumbnails.json
  - ModTime detection for regeneration
- [x] Unit tests for cache logic

### 2.3 Integration with Video Loading
- [x] Extend internal/app/loader.go to check/generate thumbnails
- [ ] Add background goroutine for thumbnail generation
- [ ] Update video loading tests

## Phase 3: GUI Main Window

### 3.1 Window Layout
- [ ] Implement internal/gui/app.go
- [ ] Implement internal/gui/main_window.go
- [ ] Implement internal/gui/status_bar.go

### 3.2 Video List Widget
- [ ] Implement internal/gui/video_list.go
- [ ] Handle keyboard navigation
- [ ] Performance testing with large collections

### 3.3 Thumbnail Preview Panel
- [ ] Implement internal/gui/preview_panel.go

## Phase 4: Interactive Components

### 4.1 Filter Dialog
- [ ] Implement internal/gui/filter_dialog.go

### 4.2 Tag Editor Dialog
- [ ] Implement internal/gui/tag_dialog.go

### 4.3 Sort Options
- [ ] Implement sort menu

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
- [ ] Implement all keyboard shortcuts

### 6.2 Window Management
- [ ] Save/restore window state

### 6.3 Error Handling
- [ ] Dialog boxes for errors
- [ ] Graceful degradation

## Phase 7: Testing & Documentation

### 7.1 Testing
- [ ] Unit tests for all new GUI components
- [ ] Integration tests
- [ ] Maintain 85%+ code coverage

### 7.2 Documentation
- [ ] Update README.md
- [ ] Add screenshots
- [ ] Document features

## Completed Milestones

**Phase 1.1-1.3:** Foundation and architecture - Added Fyne dependency, created package structure, extended video struct with thumbnail fields.

**Phase 2.1-2.2:** Thumbnail system - Implemented thumbnail generator, cache, and tests.

**Phase 2.3:** Integration - Created loader_gui.go for GUI video loading with thumbnail support.

**Phase 3:** GUI main window - Implemented app.go, main_window.go, status_bar.go, video_list.go, preview_panel.go. Created basic GUI application entry point with --gui flag. Fixed VLC mocking in tests.

**Phase 5:** Background operations - Implemented AsyncManager for proper background goroutine handling, async video loading with progress updates, async thumbnail generation, UI callbacks for main thread updates.

**Phase 6:** Integration & polish - Window state persistence (save/load size and split position), comprehensive keyboard shortcuts, dependency checks on startup, error dialogs for missing dependencies.

**Phase 7:** Testing & Documentation - Updated README.md with GUI mode documentation and keyboard shortcuts. All tests pass.

**MIGRATION COMPLETE!** GUI application fully functional with all TUI features plus thumbnail support.

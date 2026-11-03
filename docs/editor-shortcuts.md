# Editor keyboard shortcuts

Generated from `ui.DefaultEditorKeymap()` by `go test ./ui -run TestEditorShortcutDoc` with `YOGA_UPDATE_DOCS=1`. Apps change bindings with `Editor.SetKeymap` or `EditorDefaults.Keymap`, and list them with `EditorKeymapTable`.

## Text

| Action | macOS | Windows/Linux |
|---|---|---|
| Select All | `⌘A` | `Ctrl+A` |
| Copy | `⌘C` | `Ctrl+C` |
| Cut | `⌘X` | `Ctrl+X` |
| Paste | `⌘V` | `Ctrl+V` |
| Undo | `⌘Z` | `Ctrl+Z` |
| Redo | `⇧⌘Z` | `Ctrl+Shift+Z` |
| Redo | `⌘Y` | `Ctrl+Y` |
| Trigger Completion | `⌘Space` | `Ctrl+Space` |
| Toggle Soft Wrap | `⌥⌘W` | `Ctrl+Alt+W` |
| Add Next Occurrence | `⌘D` | `Ctrl+D` |
| Select All Occurrences | `⇧⌘L` | `Ctrl+Shift+L` |
| Add Cursor Above | `⌥⌘↑` | `Ctrl+Alt+Up` |
| Add Cursor Below | `⌥⌘↓` | `Ctrl+Alt+Down` |
| Cursor Word Left | `⌥←` | `Ctrl+Left` |
| Select Word Left | `⌥⇧←` | `Ctrl+Shift+Left` |
| Cursor Word Right | `⌥→` | `Ctrl+Right` |
| Select Word Right | `⌥⇧→` | `Ctrl+Shift+Right` |
| Cursor Line Start | `⌘←` | `Home` |
| Cursor Line Start | `Home` | — |
| Select to Line Start | `⇧⌘←` | `Shift+Home` |
| Select to Line Start | `⇧Home` | — |
| Cursor Line End | `⌘→` | `End` |
| Cursor Line End | `End` | — |
| Select to Line End | `⇧⌘→` | `Shift+End` |
| Select to Line End | `⇧End` | — |
| Go to Start of File | `⌘↑` | `Ctrl+Home` |
| Select to Start of File | `⇧⌘↑` | `Ctrl+Shift+Home` |
| Go to End of File | `⌘↓` | `Ctrl+End` |
| Select to End of File | `⇧⌘↓` | `Ctrl+Shift+End` |
| Delete Word Left | `⌥⌫` | `Ctrl+⌫` |
| Delete Word Right | `⌥⌦` | `Ctrl+⌦` |
| Delete to Line Start | `⌘⌫` | — |
| Duplicate Lines Down | `⌥⇧↓` | `Alt+Shift+Down` |
| Duplicate Lines Up | `⌥⇧↑` | `Alt+Shift+Up` |
| Move Lines Down | `⌥↓` | `Alt+Down` |
| Move Lines Up | `⌥↑` | `Alt+Up` |
| Delete Lines | `⇧⌘K` | `Ctrl+Shift+K` |
| Toggle Line Comment | `⌘/` | `Ctrl+/` |
| Fold | `⌥⌘[` | `Ctrl+Shift+[` |
| Unfold | `⌥⌘]` | `Ctrl+Shift+]` |
| Fold All | `⌘K ⌘0` | `Ctrl+K Ctrl+0` |
| Unfold All | `⌘K ⌘J` | `Ctrl+K Ctrl+J` |
| Find | `⌘F` | `Ctrl+F` |
| Replace | `⌥⌘F` | `Ctrl+H` |

## Find bar

| Action | macOS | Windows/Linux |
|---|---|---|
| Find | `⌘F` | `Ctrl+F` |
| Replace | `⌥⌘F` | `Ctrl+H` |
| Find Next (find field) | `↵` | `Enter` |
| Find Previous (find field) | `⇧↵` | `Shift+Enter` |
| Find Next | `⌘G` | `F3` |
| Find Previous | `⇧⌘G` | `Shift+F3` |
| Replace (replace field) | `↵` | `Enter` |
| Replace All | `⌘↵` | `Ctrl+Enter` |
| Close Find | `Esc` | `Esc` |
| Toggle Match Case | `⌥⌘C` | `Alt+C` |
| Toggle Whole Word | `⌥⌘W` | `Alt+W` |
| Toggle Regular Expression | `⌥⌘R` | `Alt+R` |
| Toggle Find in Selection | `⌥⌘L` | `Alt+L` |
| Toggle Preserve Case | `⌥⌘P` | `Alt+P` |

# helix's keys that F12 takes

These keys of helix's run an F12 action now. Its other key, where it has one,
still runs the command.

| Mode | helix's key | Its command | Its other key |
| --- | --- | --- | --- |
| normal | Ctrl+A | `increment` | none |
| normal | Ctrl+X | `decrement` | none |
| normal | Ctrl+B | `page_up` | PgUp |
| normal | Ctrl+F | `page_down` | PgDn |
| normal | Ctrl+C | `toggle_comments` | `Space c` |
| normal | Ctrl+I | `jump_forward` | Tab |
| normal | Ctrl+S | `save_selection` | none |
| normal | Ctrl+Z | `suspend` | none |
| normal | Ctrl+W | the window menu | `Space w` |
| normal | Alt+D | `delete_selection_noyank` | Delete, `"_d` |
| normal | Alt+E | `move_parent_node_end` | none |
| normal | Alt+Up | `expand_selection` | Alt+O |
| normal | Alt+Down | `shrink_selection` | Alt+I |
| normal | Alt+Left | `select_prev_sibling` | Alt+P |
| normal | Alt+Right | `select_next_sibling` | Alt+N |
| normal | Alt+: | `ensure_selections_forward` | none |
| normal | Alt+\_ | `merge_consecutive_selections` | none |
| normal | Shift+Alt+I; Shift+Alt+Down | `select_all_children` | none |
| insert | Ctrl+R | `insert_register` | none |
| insert | Ctrl+K | `kill_to_line_end` | none |
| insert | Ctrl+S | `commit_undo_checkpoint` | none |
| insert | Ctrl+W | `delete_word_backward` | Alt+Backspace |
| insert | Alt+D | `delete_word_forward` | Alt+Delete |
| insert | Ctrl+J | `insert_newline` | Enter |
| insert | Ctrl+X | `completion` | none: F12's Ctrl+Space and Ctrl+J complete |

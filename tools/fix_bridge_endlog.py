"""Add missing 'task end' log to bridge.go"""
path = 'F:/syproject/gou-ide/cmd/companion/bridge/bridge.go'

with open(path, 'r', encoding='utf-8') as f:
    content = f.read()

# The goroutine end pattern
old = """        b.mu.Unlock()
    }()"""

new = """        b.mu.Unlock()
        log.Printf("[agent] === task end ===")
    }()"""

if old in content:
    content = content.replace(old, new)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)
    print("OK: Added 'task end' log")
else:
    print("WARN: Pattern not found - trying alt pattern")
    # Try alternate indentation
    old2 = """        b.mu.Unlock()
    }()"""
    if old2 in content:
        content = content.replace(old2, new)
        with open(path, 'w', encoding='utf-8') as f:
            f.write(content)
        print("OK: Added 'task end' log (alt pattern)")
    else:
        print("FAIL: Could not find goroutine end pattern")
        # Debug: find it
        idx = content.find('b.mu.Unlock()')
        if idx >= 0:
            print(f"Found 'b.mu.Unlock()' at index {idx}")
            print(repr(content[idx:idx+200]))

"""Remove unused 'errors' import from bridge.go and do build test"""
path = 'F:/syproject/gou-ide/cmd/companion/bridge/bridge.go'

with open(path, 'r', encoding='utf-8') as f:
    content = f.read()

# Check if 'errors' is imported
has_errors = '"errors"' in content
print(f"Has 'errors' import: {has_errors}")

# Check if 'errors' is used (for build fix)
uses_errors = content.count('errors.') > 0
print(f"Uses 'errors' package: {uses_errors}")

if has_errors and not uses_errors:
    # Remove the errors import line
    import_section = content[:content.index('"context"')]
    if '"errors"\n\t\t"context"' in content:
        print("Removing 'errors' from import...")
        content = content.replace('"errors"\n\t\t"context"', '"context"')
    elif '"errors"\n\n\t"context"' in content:
        print("Removing 'errors' from import...")
        content = content.replace('"errors"\n\n\t"context"', '"context"')
    else:
        # Try replacing the errors import line
        import_lines = [
            '\t"errors"',
            '\t"errors"\n',
            '    "errors"',
            '    "errors"\n',
        ]
        for line in import_lines:
            if line in content:
                content = content.replace(line, '', 1)
                print(f"Removed: {repr(line)}")
                break
    
    # Remove blank lines in imports
    import content as re
    content = re.sub(r'\n{3,}', '\n\n', content)
    
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)
    print("Done - removed unused 'errors' import")
else:
    # Check if 'errors' is used in context of error handling
    if has_errors:
        # Find where errors is used
        idx = content.find('errors.')
        if idx >= 0:
            print(f"'errors' used at index {idx}: {content[idx:idx+60]}")
    
    print("No fix needed")

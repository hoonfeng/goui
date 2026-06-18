"""Verify agent logging in bridge.go and loop.go"""
import os

def check_file(path, name):
    if not os.path.exists(path):
        print(f"[{name}] FILE NOT FOUND: {path}")
        return
    
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    print(f"[{name}] File size: {len(content)} bytes")
    count = content.count('[agent]')
    print(f"[{name}] [agent] log occurrences: {count}")
    
    checks = {
        'import log': '\"log\"',
        'task start': 'task start',
        'tool_call:': 'tool_call:',
        'tool_result:': 'tool_result:',
        'task end': 'task end',
        'STOP requested': 'STOP requested',
    }
    
    for check_name, pattern in checks.items():
        found = pattern in content
        status = 'OK' if found else 'MISSING'
        print(f"[{name}] {status}: {check_name}")

# Check both files
check_file('F:/syproject/gou-ide/cmd/companion/bridge/bridge.go', 'bridge')
check_file('F:/syproject/gou-ide/cmd/companion/agent/loop.go', 'loop')

"""Verify and fix agent logging in both files"""
import os

def check_loop(path):
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    print(f"[loop] File size: {len(content)} bytes")
    print(f"[loop] [agent] count: {content.count('[agent]')}")
    
    checks = {
        '"log" import': '"log"' in content,
        'iteration log': '[agent] iteration' in content,
        'LLM responded log': '[agent] LLM responded' in content,
        'executing tool log': '[agent] executing tool' in content,
        'tool FAILED log': '[agent] tool FAILED' in content,
        '[FINAL] found log': '[agent] [FINAL] found' in content,
        'external cancel log': '[agent] stopped: external cancel' in content,
        'consecutive errors log': '[agent] stopped: consecutive tool errors' in content,
        'max iterations log': '[agent] stopped: max iterations reached' in content,
    }
    for name, ok in checks.items():
        print(f"[loop] {'OK' if ok else 'MISSING'}: {name}")

def check_bridge(path):
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    print(f"[bridge] File size: {len(content)} bytes")
    print(f"[bridge] [agent] count: {content.count('[agent]')}")
    
    checks = {
        '"log" import': '"log"' in content,
        'task start log': '[agent] === task start' in content,
        'tool_call log': '[agent] tool_call:' in content,
        'tool_result log': '[agent] tool_result:' in content,
        'final log': '[agent] final (task complete)' in content,
        'ERROR log': '[agent] ERROR:' in content,
        'STOP log': '[agent] STOP requested' in content,
        'loop.Run error log': '[agent] loop.Run finished' in content,
        'loop.Run success log': '[agent] loop.Run completed' in content,
        'task end log': '[agent] === task end' in content,
    }
    for name, ok in checks.items():
        print(f"[bridge] {'OK' if ok else 'MISSING'}: {name}")

check_loop('F:/syproject/gou-ide/cmd/companion/agent/loop.go')
print()
check_bridge('F:/syproject/gou-ide/cmd/companion/bridge/bridge.go')

"""Add missing EventUsage and Usage field to loop.go"""
import sys

path = 'F:/syproject/gou-ide/cmd/companion/agent/loop.go'

with open(path, 'r', encoding='utf-8') as f:
    lines = f.readlines()

# Add EventUsage after EventCircling
for i, line in enumerate(lines):
    if 'EventCircling' in line:
        usage_line = '\tEventUsage      EventType = "usage"       // LLM调用实际token用量（PromptTokens/CompletionTokens/TotalTokens）\n'
        lines.insert(i+1, usage_line)
        print(f'Inserted EventUsage at line {i+2}')
        break

# Add Usage field after CallID in Event struct
for i, line in enumerate(lines):
    if 'CallID  string' in line:
        usage_field = '\tUsage   *Usage // EventUsage时携带实际API token用量\n'
        lines.insert(i+1, usage_field)
        print(f'Inserted Usage field at line {i+2}')
        break

with open(path, 'w', encoding='utf-8') as f:
    f.writelines(lines)

print('Done - fixed Event struct')

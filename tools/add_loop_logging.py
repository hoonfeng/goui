"""Cleanly add agent logging to a fresh copy of loop.go"""
import sys

path = 'F:/syproject/gou-ide/cmd/companion/agent/loop.go'

with open(path, 'r', encoding='utf-8') as f:
    content = f.read()

# 1. Add "log" import
content = content.replace('\t"fmt"\n', '\t"fmt"\n\t"log"\n')

# 2. Add iteration logging
content = content.replace(
    'for iter := 0; iter < max; iter++ {',
    'for iter := 0; iter < max; iter++ {\n\t\tlog.Printf("[agent] iteration %d/%d", iter+1, max)'
)

# 3. Log after LLM responds (before checking ToolCalls)
content = content.replace(
    '// LLM 输出了正文但没有工具调用也没有 [FINAL]  -> 注入催促提示',
    'log.Printf("[agent] LLM responded: tool_calls=%d content_len=%d", len(assistant.ToolCalls), len(assistant.Content))\n\t\t// LLM 输出了正文但没有工具调用也没有 [FINAL]  -> 注入催促提示'
)

# 4. Log before executing tool
content = content.replace(
    'result, terr := l.Registry.Execute(ctx, tc.Function.Name, tc.Function.Arguments)',
    'log.Printf("[agent] executing tool: %s", tc.Function.Name)\n\t\t\tresult, terr := l.Registry.Execute(ctx, tc.Function.Name, tc.Function.Arguments)'
)

# 5. Log tool failure
content = content.replace(
    'if isErr {\n\t\t\t\tif consecErr++; consecErr >= 3 {',
    'if isErr {\n\t\t\t\tlog.Printf("[agent] tool FAILED: %s (consecErr=%d)", tc.Function.Name, consecErr+1)\n\t\t\t\tif consecErr++; consecErr >= 3 {'
)

# 6. Log [FINAL] found
content = content.replace(
    'if strings.Contains(assistant.Content, finalMarker) {\n\t\t\tl.emit(Event{Type: EventFinal',
    'if strings.Contains(assistant.Content, finalMarker) {\n\t\t\tlog.Printf("[agent] [FINAL] found, task complete")\n\t\t\tl.emit(Event{Type: EventFinal'
)

# 7. Log external cancel
content = content.replace(
    'return msgs, err // 外部取消',
    'log.Printf("[agent] stopped: external cancel")\n\t\treturn msgs, err // 外部取消'
)

# 8. Log consecutive tool errors
content = content.replace(
    'return msgs, ErrConsecToolError',
    'log.Printf("[agent] stopped: consecutive tool errors"); return msgs, ErrConsecToolError'
)

# 9. Log max iterations
content = content.replace(
    'l.emit(Event{Type: EventError, Content: ErrMaxIterations.Error()})\n\treturn msgs, ErrMaxIterations',
    'log.Printf("[agent] stopped: max iterations reached")\n\tl.emit(Event{Type: EventError, Content: ErrMaxIterations.Error()})\n\treturn msgs, ErrMaxIterations'
)

# Verify no HTML entities leaked in
if '&quot;' in content:
    print("ERROR: &quot; entities found in output!")
    sys.exit(1)

with open(path, 'w', encoding='utf-8') as f:
    f.write(content)

print("Done - loop.go updated cleanly")

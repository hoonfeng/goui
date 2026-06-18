import re

path = 'F:/syproject/gou-ide/cmd/companion/agent/loop.go'

with open(path, 'r', encoding='utf-8') as f:
    content = f.read()

# 1. Add "log" to imports
content = content.replace('"fmt"', '"fmt"\n\t"log"')

# 2. Add iteration logging at loop start
content = content.replace(
    'for iter := 0; iter < max; iter++ {',
    'for iter := 0; iter < max; iter++ {\n\t\tlog.Printf("[agent] iteration %d/%d", iter+1, max)'
)

# 3. Log after LLM responds
content = content.replace(
    'if len(assistant.ToolCalls) == 0 {',
    'log.Printf("[agent] LLM responded: tool_calls=%d content_len=%d", len(assistant.ToolCalls), len(assistant.Content))\n\t\tif len(assistant.ToolCalls) == 0 {'
)

# 4. Log before executing tool
content = content.replace(
    'result, terr := l.Registry.Execute(ctx, tc.Function.Name, tc.Function.Arguments)',
    'log.Printf("[agent] executing tool: %s", tc.Function.Name)\n\t\t\tresult, terr := l.Registry.Execute(ctx, tc.Function.Name, tc.Function.Arguments)'
)

# 5. Log tool error (isErr check)
content = content.replace(
    'if isErr {\n\t\t\t\tif consecErr++; consecErr >= 3 {',
    'if isErr {\n\t\t\t\tlog.Printf("[agent] tool FAILED: %s consecErr=%d", tc.Function.Name, consecErr+1)\n\t\t\t\tif consecErr++; consecErr >= 3 {'
)

# 6. Log when external cancel
content = content.replace(
    '\t\treturn msgs, err // 外部取消',
    '\t\tlog.Printf("[agent] stopped: external cancel")\n\t\treturn msgs, err // 外部取消'
)

# 7. Log when [FINAL] found
content = content.replace(
    '\t\t\tl.emit(Event{Type: EventFinal, Content: stripFinal(assistant.Content)})\n\t\t\treturn msgs, nil',
    '\t\t\tlog.Printf("[agent] [FINAL] found, task complete")\n\t\t\tl.emit(Event{Type: EventFinal, Content: stripFinal(assistant.Content)})\n\t\t\treturn msgs, nil'
)

# 8. Log when consecutive tool errors
content = content.replace(
    'return msgs, ErrConsecToolError',
    'log.Printf("[agent] stopped: consecutive tool errors"); return msgs, ErrConsecToolError'
)

# 9. Log when max iterations
content = content.replace(
    'l.emit(Event{Type: EventError, Content: ErrMaxIterations.Error()})\n\treturn msgs, ErrMaxIterations',
    'log.Printf("[agent] stopped: max iterations reached")\n\tl.emit(Event{Type: EventError, Content: ErrMaxIterations.Error()})\n\treturn msgs, ErrMaxIterations'
)

with open(path, 'w', encoding='utf-8') as f:
    f.write(content)

print('Done - loop.go updated with agent logging')

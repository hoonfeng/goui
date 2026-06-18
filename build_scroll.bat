@echo off
set CGO_ENABLED=1
cd /d F:\syproject\goui
go build ./examples/scrolltest/ 2>&1
echo exitcode=%errorlevel%

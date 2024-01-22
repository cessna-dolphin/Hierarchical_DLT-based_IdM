set num=17
:loop
start Hierarchical_IdM.exe
set /a num-=1
echo %num%
if "%num%"=="0" goto end
goto loop
:end
exit
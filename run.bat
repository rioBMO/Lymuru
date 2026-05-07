@echo off
cd /d "%~dp0"

if not exist "venv\" (
    echo Creating virtual environment...
    python -m venv venv
    call venv\Scripts\activate
    echo Installing requirements...
    pip install -r requirements.txt
) else (
    call venv\Scripts\activate
)

:loop
python deezload.py
echo.
set /p "choice=Run again? (y/n): "
if /i "%choice%"=="y" goto loop

#!/bin/bash
set -e

# Refresh font cache to pick up any fonts mounted at /app/fonts
fc-cache -f

exec /app/pdf-converter

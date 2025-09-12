#!/bin/bash

# CodexHash Stop Script
# Stops both frontend and backend services

echo "🛑 Stopping CodexHash Application..."

# Find and stop processes
FRONTEND_PIDS=$(pgrep -f "next dev -p 3065")
BACKEND_PIDS=$(pgrep -f "uvicorn.*8001")

if [ ! -z "$FRONTEND_PIDS" ]; then
    echo "🌐 Stopping Frontend processes: $FRONTEND_PIDS"
    kill $FRONTEND_PIDS
    echo "✅ Frontend stopped"
else
    echo "⚠️  No frontend processes found"
fi

if [ ! -z "$BACKEND_PIDS" ]; then
    echo "🔗 Stopping Backend processes: $BACKEND_PIDS"
    kill $BACKEND_PIDS
    echo "✅ Backend stopped"
else
    echo "⚠️  No backend processes found"
fi

# Wait a moment for processes to terminate
sleep 2

# Check if any are still running
REMAINING=$(pgrep -f "(next dev -p 3065|uvicorn.*8001)")
if [ ! -z "$REMAINING" ]; then
    echo "⚠️  Some processes still running, force killing: $REMAINING"
    kill -9 $REMAINING
fi

echo ""
echo "✅ CodexHash Application stopped"
echo "🚀 To start again, run: ./start_codexhash.sh"
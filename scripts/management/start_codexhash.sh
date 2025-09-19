#!/bin/bash

# CodexHash Startup Script
# Starts both frontend and backend services

echo "🚀 Starting CodexHash Application..."

# Check if processes are already running
FRONTEND_PID=$(pgrep -f "next dev -p 3065")
BACKEND_PID=$(pgrep -f "uvicorn.*8001")

if [ ! -z "$FRONTEND_PID" ]; then
    echo "⚠️  Frontend already running on PID $FRONTEND_PID"
else
    echo "🌐 Starting Frontend (Next.js) on port 3065..."
    cd /home/web3codex/projects/codex_hash
    npm run dev &
    FRONTEND_PID=$!
    echo "✅ Frontend started with PID $FRONTEND_PID"
fi

if [ ! -z "$BACKEND_PID" ]; then
    echo "⚠️  Backend already running on PID $BACKEND_PID"
else
    echo "🔗 Starting Backend (FastAPI) on port 8001..."
    cd /home/web3codex/projects/codex_hash/backend
    uvicorn src.main:app --host 0.0.0.0 --port 8001 &
    BACKEND_PID=$!
    echo "✅ Backend started with PID $BACKEND_PID"
fi

# Wait a moment for services to start
sleep 3

echo ""
echo "🎉 CodexHash Application Status:"
echo "Frontend: http://localhost:3065"
echo "Backend API: http://localhost:8001"
echo "Backend Docs: http://localhost:8001/docs"
echo ""

# Test health endpoints
echo "🔍 Testing services..."
if curl -s http://localhost:3065 > /dev/null; then
    echo "✅ Frontend is responsive"
else
    echo "❌ Frontend is not responding"
fi

if curl -s http://localhost:8001/health > /dev/null; then
    echo "✅ Backend is responsive"
else
    echo "❌ Backend is not responding"
fi

echo ""
echo "📋 Running processes:"
ps aux | grep -E "(next|uvicorn)" | grep -v grep | awk '{print "  ", $2, $11, $12}'

echo ""
echo "🛑 To stop all services, run: ./stop_codexhash.sh"
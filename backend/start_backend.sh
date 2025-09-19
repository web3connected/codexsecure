#!/bin/bash
cd /home/web3codex/projects/codex_hash/backend
exec python -m uvicorn src.main:app --host 0.0.0.0 --port 8001 --reload
# CodexHash Development Setup
# Quantum-resistant hashing API local development

# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt

# Run development server
uvicorn src.main:app --reload --host 0.0.0.0 --port 8001

# Test endpoints
curl http://localhost:8001/health
curl -X POST http://localhost:8001/hash -H "Content-Type: application/json" -d '{"data": "test message"}'

# Docker development
docker build -f Dockerfile.production -t codex-hash:dev .
docker run -p 8001:8001 codex-hash:dev

# Production deployment checklist:
# 1. Set up GitHub secrets: FORGE_SERVER_IP, FORGE_SERVER_USER, FORGE_SSH_PRIVATE_KEY
# 2. Configure domain: codexhash.io -> 45.79.180.207
# 3. Set up Laravel Forge SSL certificate for codexhash.io
# 4. Deploy: git push origin main
# 5. Verify: https://codexhash.io/health

module.exports = {
  apps: [
    {
      name: 'codexhash-frontend',
      script: 'npm',
      args: 'start',
      cwd: '/home/forge/codexhash.io',
      env: {
        NODE_ENV: 'production',
        PORT: 3000,
        NEXT_PUBLIC_API_URL: 'https://codexhash.io/api',
        NEXT_PUBLIC_SITE_URL: 'https://codexhash.io'
      },
      error_file: '/home/forge/codexhash.io/logs/frontend-error.log',
      out_file: '/home/forge/codexhash.io/logs/frontend-out.log',
      log_file: '/home/forge/codexhash.io/logs/frontend-combined.log',
      instances: 1,
      exec_mode: 'fork',
      autorestart: true,
      watch: false,
      max_memory_restart: '1G'
    },
    {
      name: 'codexhash-backend',
      script: 'uvicorn',
      args: 'src.main:app --host 0.0.0.0 --port 8001 --workers 4',
      cwd: '/home/forge/codexhash.io/backend',
      env: {
        ENVIRONMENT: 'production',
        API_HOST: '0.0.0.0',
        API_PORT: '8001',
        CORS_ORIGINS: '["https://codexhash.io", "https://www.codexhash.io"]'
      },
      error_file: '/home/forge/codexhash.io/logs/backend-error.log',
      out_file: '/home/forge/codexhash.io/logs/backend-out.log',
      log_file: '/home/forge/codexhash.io/logs/backend-combined.log',
      instances: 1,
      exec_mode: 'fork',
      autorestart: true,
      watch: false,
      max_memory_restart: '512M'
    }
  ]
};
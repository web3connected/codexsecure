module.exports = {
  apps: [
    // Development mode
    {
      name: 'codexsecure-dev',
      script: 'node_modules/.bin/next',
      args: 'dev --turbopack -p 3003',
      cwd: __dirname,
      watch: false,
      autorestart: true,
      max_restarts: 10,
      min_uptime: '10s',
      max_memory_restart: '500M',
      env: {
        NODE_ENV: 'development',
        NODE_OPTIONS: '--max-old-space-size=4096',
      },
      error_file: './logs/pm2-error.log',
      out_file: './logs/pm2-out.log',
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      merge_logs: true,
      exp_backoff_restart_delay: 100,
    },
    // Production mode
    {
      name: 'codexsecure-prod',
      script: 'node_modules/.bin/next',
      args: 'start -p 3003',
      cwd: __dirname,
      instances: 1,
      exec_mode: 'cluster',
      watch: false,
      autorestart: true,
      max_memory_restart: '1G',
      env: {
        NODE_ENV: 'production',
        PORT: 3003
      },
      error_file: './logs/pm2-prod-error.log',
      out_file: './logs/pm2-prod-out.log',
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      merge_logs: true,
    }
  ]
}

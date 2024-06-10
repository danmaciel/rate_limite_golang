      
#!/bin/bash
timeout=30
while ! redis-cli -h redis -p 6379 ping > /dev/null 2>&1; do
  if [ $timeout -eq 0 ]; then
    echo "Timeout expired: Redis is not available"
    exit 1
  fi
  echo "Waiting for Redis to be available..."
  sleep 1
  timeout=$((timeout - 1))
done
echo "Redis is available"

    
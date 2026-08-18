import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  // Simulating 100 concurrent users for a sustained period of 30s
  stages: [
    { duration: '5s', target: 50 },
    { duration: '20s', target: 100 },
    { duration: '5s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
    http_req_failed: ['rate<0.01'],   // Error rate should be less than 1%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081/api/v1';

export default function () {
  // We hit the health endpoint as a lightweight baseline load test
  // To truly test DB pool load, we'd want to hit an endpoint that queries the DB.
  // We'll hit the public health check to verify fiber load limits.
  const res = http.get(`${BASE_URL}/health`);
  
  check(res, {
    'is status 200': (r) => r.status === 200,
  });

  // Short sleep to simulate real user behavior between requests
  sleep(1);
}

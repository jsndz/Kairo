import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '3s', target: 10 },  // Ramp-up
    { duration: '10s', target: 25 }, // Steady load
    { duration: '2s', target: 0 },   // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<25', 'p(99)<50'], // 95% of requests under 25ms
    http_req_failed: ['rate<0.01'],              // Error rate under 1%
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://127.0.0.1:8081';

export default function () {
  // 1. Health check
  const healthRes = http.get(`${BASE_URL}/api/health`);
  check(healthRes, {
    'health status is 200': (r) => r.status === 200,
  });

  // 2. Room provisioning
  const roomPayload = JSON.stringify({ name: `k6-room-${__VU}-${__ITER}` });
  const roomRes = http.post(`${BASE_URL}/api/rooms`, roomPayload, {
    headers: { 'Content-Type': 'application/json' },
  });
  check(roomRes, {
    'room created 201': (r) => r.status === 201,
  });

  const roomBody = JSON.parse(roomRes.body || '{}');
  const projectId = roomBody.roomId;
  if (!projectId) return;

  // 3. Register media asset
  const assetPayload = JSON.stringify({
    kind: 'video',
    name: `video_${__VU}_${__ITER}.mp4`,
    mime: 'video/mp4',
    bytes: 1048576,
    duration: 10,
    frames: 300,
    width: 1920,
    height: 1080,
  });
  const assetRes = http.post(`${BASE_URL}/api/projects/${projectId}/assets`, assetPayload, {
    headers: { 'Content-Type': 'application/json' },
  });
  check(assetRes, {
    'asset registered 201': (r) => r.status === 201,
  });

  const assetBody = JSON.parse(assetRes.body || '{}');
  const assetId = assetBody.id;

  if (assetId) {
    // 4. Request upload presigned policy
    const uploadRes = http.post(`${BASE_URL}/api/projects/${projectId}/assets/${assetId}/upload`);
    check(uploadRes, {
      'upload policy generated 200': (r) => r.status === 200,
    });

    const uploadBody = JSON.parse(uploadRes.body || '{}');
    const attempt = uploadBody.asset ? uploadBody.asset.attempt : null;

    // 5. Complete asset upload
    if (attempt) {
      const compPayload = JSON.stringify({ attempt: attempt });
      const compRes = http.post(`${BASE_URL}/api/projects/${projectId}/assets/${assetId}/complete`, compPayload, {
        headers: { 'Content-Type': 'application/json' },
      });
      check(compRes, {
        'upload completed 200': (r) => r.status === 200,
      });
    }
  }

  // 6. Direct HTTP Timeline Edit Operation
  const editPayload = JSON.stringify({
    id: `edit_${__VU}_${__ITER}`,
    type: 'rename',
    expectedRevision: 0,
    payload: { name: `Updated Project ${__VU}` },
  });
  const editRes = http.post(`${BASE_URL}/api/projects/${projectId}/operations`, editPayload, {
    headers: { 'Content-Type': 'application/json' },
  });
  check(editRes, {
    'timeline edit committed 200': (r) => r.status === 200,
  });

  // 7. Hydrated project state retrieval
  const projRes = http.get(`${BASE_URL}/api/projects/${projectId}`);
  check(projRes, {
    'project state hydrated 200': (r) => r.status === 200,
    'project has incremented revision': (r) => {
      const b = JSON.parse(r.body || '{}');
      return b.sequence >= 1;
    },
  });

  sleep(0.05); // 50ms pacing between iterations
}

import {
  defineRailway,
  github,
  preserve,
  project,
  service,
  volume,
} from 'railway/iac';

export default defineRailway(() => {
  const data = volume('discogs-data', {
    region: 'us-east4-eqdc4a',
    sizeMB: 5000,
  });

  const app = service('discogs-collection-manager', {
    source: github('mathcale/discogs-collection-manager', {
      branch: 'main',
    }),
    domains: [{
      domain: 'collections.calegaro.dev',
      port: 8080,
    }],
    volumeMounts: {
      '/data': data,
    },
    env: {
      APP_BASE_URL: preserve(),
      DB_PATH: preserve(),
      DISCOGS_CALLBACK_URL: preserve(),
      DISCOGS_CONSUMER_KEY: preserve(),
      DISCOGS_CONSUMER_SECRET: preserve(),
      ENVIRONMENT: preserve(),
      MIGRATIONS_PATH: preserve(),
      PORT: preserve(),
      RESEND_API_KEY: preserve(),
      RESEND_FROM: preserve(),
    },
  });

  return project('discogs-collection-manager', {
    resources: [app, data],
  });
});

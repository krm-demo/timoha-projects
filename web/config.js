// Where the Go API lives.
// - On GitHub Pages the API runs on Vercel (see docs/DEPLOY_VERCEL.md).
// - Anywhere else ('' = same origin) the Go server serves this frontend itself.
// You can also try any backend without editing: open the site with ?api=<url>.
// If the API is unreachable the site falls back to the static demo data.
window.TM_API_BASE = location.hostname.endsWith('github.io')
  ? 'https://timoha-api.vercel.app'
  : '';

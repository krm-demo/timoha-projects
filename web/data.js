/* Data layer. Two interchangeable sources with the same interface:
 *  - ApiSource:    the Go REST backend (/api/...), used when it answers /api/health
 *  - StaticSource: JSON files in data/ + localStorage, so the GitHub Pages
 *                  preview works with no server at all.
 * The UI (app.js) only talks to window.DS. */
(() => {
  const store = {
    get(k, d) { try { return JSON.parse(localStorage.getItem('tm_' + k)) ?? d; } catch { return d; } },
    set(k, v) { try { localStorage.setItem('tm_' + k, JSON.stringify(v)); } catch {} }
  };

  // ---------------- API source ----------------
  function ApiSource(base) {
    const call = async (method, path, body) => {
      const headers = { 'Content-Type': 'application/json' };
      const token = store.get('token', null);
      if (token) headers.Authorization = 'Bearer ' + token;
      const res = await fetch(base + path, { method, headers, body: body ? JSON.stringify(body) : undefined });
      const data = await res.json().catch(() => ({}));
      if (res.status === 401 && token) { store.set('token', null); store.set('session', null); }
      if (!res.ok) throw Object.assign(new Error(data.error || 'Request failed (' + res.status + ')'), { status: res.status });
      return data;
    };
    const qs = params => { const p = new URLSearchParams(); Object.entries(params).forEach(([k, v]) => [].concat(v ?? []).forEach(x => x !== '' && p.append(k, x))); return p.toString(); };
    const session = d => { store.set('token', d.token); store.set('session', d.user); return d.user; };
    return {
      mode: 'api', base,
      homeLayout: async () => (await call('GET', '/api/home')).blocks.map(b => b.id),
      homeBlock: id => call('GET', '/api/home/' + id),
      list: params => call('GET', '/api/products?' + qs(params)),
      suggest: async q => (await call('GET', '/api/search?' + qs({ q, limit: 6 }))).results,
      product: slug => call('GET', '/api/products/' + encodeURIComponent(slug)),
      user: () => store.get('token', null) ? store.get('session', null) : null,
      register: async (name, email, password) => session(await call('POST', '/api/auth/register', { name, email, password })),
      login: async (email, password) => session(await call('POST', '/api/auth/login', { email, password })),
      logout() { store.set('token', null); store.set('session', null); },
      checkout: (items, card) => call('POST', '/api/checkout', { items: items.map(i => ({ productId: i.pid, seller: i.seller, qty: i.qty })), card }),
      orders: async () => (await call('GET', '/api/orders')).orders,
      order: id => call('GET', '/api/orders/' + encodeURIComponent(id))
    };
  }

  // ---------------- static source (mirrors backend/internal/catalog) ----------------
  function StaticSource() {
    const cache = {};
    const json = path => cache[path] ??= fetch('data/' + path).then(r => { if (!r.ok) throw new Error(path); return r.json(); });
    const products = () => json('products.json');
    const best = p => p.offers.find(o => o.stock > 0);
    const from = p => (best(p) || { price: p.msrp }).price;
    const disc = p => Math.round((1 - from(p) / p.msrp) * 100);

    const norm = s => s.toLowerCase().normalize('NFKD').replace(/[^a-z0-9 ]/g, ' ').replace(/\s+/g, ' ').trim();
    const lev = (a, b) => {
      if (Math.abs(a.length - b.length) > 3) return 9;
      let prev = Array.from({ length: b.length + 1 }, (_, i) => i);
      for (let i = 1; i <= a.length; i++) {
        const cur = [i];
        for (let j = 1; j <= b.length; j++) cur[j] = Math.min(prev[j] + 1, cur[j - 1] + 1, prev[j - 1] + (a[i - 1] === b[j - 1] ? 0 : 1));
        prev = cur;
      }
      return prev[b.length];
    };
    const tol = n => n <= 2 ? 0 : n <= 5 ? 1 : 2;
    const score = (name, query) => {
      const n = norm(name), q = norm(query);
      if (!q) return 0; if (n === q) return 100; if (n.startsWith(q)) return 90; if (n.includes(q)) return 80;
      const words = n.split(' '), qw = q.split(' ');
      let total = 0;
      for (const w of qw) {
        let b = 0;
        for (const nw of words) {
          if (nw.startsWith(w)) { b = 1; break; }
          let d = lev(w, nw); if (nw.length > w.length) d = Math.min(d, lev(w, nw.slice(0, w.length)));
          if (d <= tol(w.length)) b = Math.max(b, 1 - d * 0.25);
        }
        if (!b) return 0;
        total += b;
      }
      return 40 + 30 * total / qw.length;
    };
    const search = (all, q) => all.map(p => [p, score(p.name, q)]).filter(x => x[1] > 0).sort((a, b) => b[1] - a[1]).map(x => x[0]);
    const sorters = { popular: (a, b) => b.sold - a.sold, 'price-asc': (a, b) => from(a) - from(b), 'price-desc': (a, b) => from(b) - from(a), discount: (a, b) => disc(b) - disc(a), newest: (a, b) => b.year - a.year };
    const FACETS = ['platform', 'region', 'type', 'genre'];
    const top = (all, by, n) => all.filter(best).sort(sorters[by]).slice(0, n);

    const local = () => store.get('session', null);
    const hash = async pw => {
      if (!crypto.subtle) return pw;
      const b = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(pw));
      return [...new Uint8Array(b)].map(x => x.toString(16).padStart(2, '0')).join('');
    };
    const ordersKey = () => 'orders_' + (local() || {}).email;
    const keyChars = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';
    const genKey = () => Array.from({ length: 3 }, () => Array.from({ length: 5 }, () => keyChars[Math.random() * 32 | 0]).join('')).join('-');
    const fail = (msg, status) => { throw Object.assign(new Error(msg), { status }); };

    return {
      mode: 'static',
      homeLayout: async () => (await json('home/layout.json')).blocks,
      async homeBlock(id) {
        const d = await json('home/' + id + '.json');
        if (!d.productIds) return d;
        const all = await products();
        return { ...d, products: d.productIds.map(i => all.find(p => p.id === i)).filter(Boolean) };
      },
      async list(params) {
        const all = await products();
        const q = (params.q || '').trim();
        const sel = {}; FACETS.forEach(k => sel[k] = [].concat(params[k] || []));
        const min = parseFloat(params.min), max = parseFloat(params.max);
        const base = q ? search(all, q) : all;
        const match = (p, skip) => (isNaN(min) || from(p) >= min) && (isNaN(max) || from(p) <= max) && FACETS.every(k => k === skip || !sel[k].length || sel[k].includes(p[k]));
        let sort = params.sort; if (!sort || !(sort in sorters || (sort === 'relevance' && q))) sort = q ? 'relevance' : 'popular';
        const items = base.filter(p => match(p, ''));
        if (sorters[sort]) items.sort(sorters[sort]);
        const limit = +params.limit || 24, page = +params.page || 1;
        const facets = {};
        FACETS.forEach(k => {
          const c = {}; all.forEach(p => c[p[k]] = 0); base.forEach(p => { if (match(p, k)) c[p[k]]++; });
          facets[k] = Object.keys(c).sort().map(value => ({ value, count: c[value] }));
        });
        return { items: items.slice((page - 1) * limit, page * limit), total: items.length, page, limit, sort, facets };
      },
      async suggest(q) {
        return search(await products(), q).slice(0, 6).map(p => ({ id: p.id, slug: p.slug, name: p.name, platform: p.platform, price: from(p), colors: p.colors }));
      },
      async product(slug) {
        const all = await products();
        const p = all.find(x => x.slug === slug);
        if (!p) fail('product not found', 404);
        return { product: p, related: all.filter(x => x.id !== p.id && (x.genre === p.genre || x.platform === p.platform)).slice(0, 5) };
      },
      user: local,
      async register(name, email, pw) {
        const users = store.get('users', {});
        if (users[email]) fail('an account with this email already exists', 409);
        users[email] = { name, email, pw: await hash(pw) };
        store.set('users', users); store.set('session', { name, email }); return local();
      },
      async login(email, pw) {
        const u = store.get('users', {})[email];
        if (!u || u.pw !== await hash(pw)) fail('wrong email or password', 401);
        store.set('session', { name: u.name, email }); return local();
      },
      logout() { store.set('session', null); },
      async checkout(items, card) {
        const all = await products();
        const num = (card.number || '').replace(/[\s-]/g, '');
        if (!/^\d{16}$/.test(num)) fail('card number is invalid', 400);
        if (!/^(0[1-9]|1[0-2])\/\d{2}$/.test(card.expiry || '')) fail('card expiry must be MM/YY', 400);
        await new Promise(r => setTimeout(r, 700));
        if (num === '4000000000000002') fail('payment declined by the bank (test card)', 402);
        const lines = items.map(i => { const p = all.find(x => x.id === i.pid), o = p && p.offers.find(o => o.seller === i.seller); if (!o) fail('invalid order line', 400); return { p, o, qty: i.qty }; });
        const subtotal = lines.reduce((s, l) => s + l.o.price * l.qty, 0), fee = Math.round(subtotal * 5) / 100;
        const order = { id: 'TM-' + Date.now().toString(36).toUpperCase(), date: new Date().toISOString(), subtotal, fee, total: subtotal + fee,
          items: lines.map(l => ({ productId: l.p.id, slug: l.p.slug, name: l.p.name, platform: l.p.platform, region: l.p.region, seller: l.o.seller, price: l.o.price, qty: l.qty, keys: Array.from({ length: l.qty }, genKey) })) };
        store.set(ordersKey(), [order, ...store.get(ordersKey(), [])]);
        return order;
      },
      orders: async () => store.get(ordersKey(), []),
      async order(id) { const o = store.get(ordersKey(), []).find(o => o.id === id); if (!o) fail('order not found', 404); return o; }
    };
  }

  // ---------------- pick a source ----------------
  // Priority: ?api=<url> (remembered) > window.TM_API_BASE (config.js) > same origin.
  window.DSReady = (async () => {
    const override = new URLSearchParams(location.search).get('api');
    if (override !== null) store.set('api', override);
    const base = (store.get('api', null) ?? window.TM_API_BASE ?? '').replace(/\/$/, '');
    try {
      const ctl = new AbortController(); const t = setTimeout(() => ctl.abort(), 4000);
      const r = await fetch(base + '/api/health', { signal: ctl.signal }); clearTimeout(t);
      if (r.ok && (await r.json()).ok) return (window.DS = ApiSource(base));
    } catch {}
    return (window.DS = StaticSource());
  })();
  window.TMStore = store;
})();

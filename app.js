/* Timoha Market — static prototype.
 * Each homepage block is loaded from its own "service" (a JSON endpoint under data/home/).
 * In the real build these become PHP/Go REST endpoints; the frontend only renders what it receives. */
(() => {
  const $ = (s, el = document) => el.querySelector(s);
  const app = $('#app');
  const esc = s => String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  const eur = n => '€' + n.toFixed(2);
  const store = {
    get(k, d) { try { return JSON.parse(localStorage.getItem('tm_' + k)) ?? d; } catch { return d; } },
    set(k, v) { try { localStorage.setItem('tm_' + k, JSON.stringify(v)); } catch {} }
  };

  // ---------- "API" ----------
  const cache = {};
  const api = path => cache[path] ??= fetch('data/' + path).then(r => { if (!r.ok) throw new Error(path); return r.json(); });
  let PRODUCTS = [];
  const byId = id => PRODUCTS.find(p => p.id === id);
  const bySlug = s => PRODUCTS.find(p => p.slug === s);
  const best = p => p.offers[0];
  const discount = p => Math.round((1 - best(p).price / p.msrp) * 100);

  // ---------- fuzzy search (typos + partial names) ----------
  const norm = s => s.toLowerCase().normalize('NFKD').replace(/[^a-z0-9 ]/g, ' ').replace(/\s+/g, ' ').trim();
  function lev(a, b) {
    if (Math.abs(a.length - b.length) > 3) return 9;
    const d = Array.from({ length: b.length + 1 }, (_, i) => i);
    for (let i = 1; i <= a.length; i++) {
      let prev = d[0]; d[0] = i;
      for (let j = 1; j <= b.length; j++) {
        const t = d[j];
        d[j] = Math.min(d[j] + 1, d[j - 1] + 1, prev + (a[i - 1] === b[j - 1] ? 0 : 1));
        prev = t;
      }
    }
    return d[b.length];
  }
  function score(p, q) {
    const name = norm(p.name), qn = norm(q);
    if (!qn) return 0;
    if (name === qn) return 100;
    if (name.startsWith(qn)) return 90;
    if (name.includes(qn)) return 80;
    const words = name.split(' '), qw = qn.split(' ');
    let total = 0;
    for (const w of qw) {
      let bestW = 0;
      for (const nw of words) {
        if (nw.startsWith(w)) bestW = Math.max(bestW, 1);
        else {
          const tol = w.length <= 2 ? 0 : w.length <= 5 ? 1 : 2;
          const dist = Math.min(lev(w, nw), lev(w, nw.slice(0, w.length)));
          if (dist <= tol) bestW = Math.max(bestW, 1 - dist * 0.25);
        }
      }
      if (!bestW) return 0;
      total += bestW;
    }
    return 40 + 30 * total / qw.length;
  }
  const search = q => PRODUCTS.map(p => [p, score(p, q)]).filter(x => x[1] > 0).sort((a, b) => b[1] - a[1]).map(x => x[0]);

  // ---------- auth (demo: localStorage; real build -> backend sessions/JWT) ----------
  const auth = {
    user: () => store.get('session', null),
    async hash(pw) {
      if (!crypto.subtle) return pw;
      const b = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(pw));
      return [...new Uint8Array(b)].map(x => x.toString(16).padStart(2, '0')).join('');
    },
    async register(name, email, pw) {
      const users = store.get('users', {});
      if (users[email]) throw new Error('An account with this email already exists.');
      users[email] = { name, email, pw: await auth.hash(pw) };
      store.set('users', users); store.set('session', { name, email });
    },
    async login(email, pw) {
      const u = store.get('users', {})[email];
      if (!u || u.pw !== await auth.hash(pw)) throw new Error('Wrong email or password.');
      store.set('session', { name: u.name, email });
    },
    logout() { store.set('session', null); }
  };

  // ---------- cart ----------
  const cart = {
    items: () => store.get('cart', []),
    save(items) { store.set('cart', items); renderHeader(); },
    add(pid, seller) {
      const items = cart.items();
      const it = items.find(i => i.pid === pid && i.seller === seller);
      it ? it.qty++ : items.push({ pid, seller, qty: 1 });
      cart.save(items); toast('Added to cart');
    },
    lines() {
      return cart.items().map(i => {
        const p = byId(i.pid); const o = p && p.offers.find(o => o.seller === i.seller);
        return p && o ? { ...i, p, o, total: o.price * i.qty } : null;
      }).filter(Boolean);
    }
  };

  // ---------- ui helpers ----------
  let toastT;
  function toast(msg) { const t = $('#toast'); t.textContent = msg; t.hidden = false; clearTimeout(toastT); toastT = setTimeout(() => t.hidden = true, 1800); }
  const coverStyle = p => `background:linear-gradient(145deg,${p.colors[0]},${p.colors[0]} 45%,${p.colors[1]})`;
  const cover = (p, showPlat = true) => `<div class="cover" style="${coverStyle(p)}"><div class="glyph" style="background:${p.colors[1]}"></div>${showPlat ? `<div class="plat">${esc(p.platform)}</div>` : ''}<span>${esc(p.name)}</span></div>`;
  const card = p => `<a class="card" href="#/product/${p.slug}">${cover(p)}<div class="card-b">
      <div class="card-t">${esc(p.name)}</div><div class="region">${esc(p.region)}</div>
      <div class="from">From</div>
      <div class="price-row"><span class="price">${eur(best(p).price)}</span>${discount(p) > 0 ? `<span class="disc">-${discount(p)}%</span>` : ''}</div>
      <div class="old">${eur(p.msrp)}</div></div></a>`;
  const parseHash = () => {
    const [path, qs] = location.hash.slice(1).split('?');
    return { parts: (path || '/').split('/').filter(Boolean), params: new URLSearchParams(qs || '') };
  };

  function renderHeader() {
    const n = cart.items().reduce((s, i) => s + i.qty, 0);
    const b = $('#cartCount'); b.textContent = n; b.hidden = !n;
    const u = auth.user();
    $('#userSlot').innerHTML = u
      ? `<div class="menu"><button class="icon-btn user-chip" id="um" style="width:auto;padding:0 10px 0 4px;border:0;color:#fff"><span class="avatar">${esc(u.name[0].toUpperCase())}</span>${esc(u.name.split(' ')[0])}</button>
         <div class="menu-pop" id="umPop" hidden><a href="#/orders">My orders & keys</a><button id="logout">Sign out</button></div></div>`
      : `<a class="btn btn-y btn-sm" href="#/login">Sign in</a>`;
    if (u) {
      $('#um').onclick = e => { e.stopPropagation(); $('#umPop').hidden = !$('#umPop').hidden; };
      $('#logout').onclick = () => { auth.logout(); renderHeader(); location.hash = '#/'; toast('Signed out'); };
    }
  }
  document.addEventListener('click', () => { const p = $('#umPop'); if (p) p.hidden = true; });

  // ---------- pages ----------
  const blockRenderers = {
    async banner() {
      const d = await api('home/banner.json');
      return `<section class="hero" id="hero">${d.slides.map((s, i) => `
        <div class="slide ${i ? '' : 'on'}" style="background:linear-gradient(120deg,${s.colors[0]},${s.colors[1]})">
          <h1>${esc(s.title)}</h1><p>${esc(s.subtitle)}</p><a class="btn btn-y" href="${s.link}">${esc(s.cta)}</a></div>`).join('')}
        <div class="dots">${d.slides.map((_, i) => `<button data-i="${i}" class="${i ? '' : 'on'}" aria-label="Slide ${i + 1}"></button>`).join('')}</div></section>`;
    },
    async categories() {
      const d = await api('home/categories.json');
      return `<section class="section"><div class="sec-h"><h2>${esc(d.title)}<span class="svc-tag">${d.service}</span></h2></div>
        <div class="chips">${d.items.map(c => `<a class="chip" href="${c.link}">${esc(c.label)}</a>`).join('')}</div></section>`;
    },
    async deals() { return productRow(await api('home/deals.json'), '#/store?sort=discount'); },
    async bestsellers() { return productRow(await api('home/bestsellers.json'), '#/store?sort=popular'); }
  };
  const productRow = (d, link) => `<section class="section"><div class="sec-h"><h2>${esc(d.title)}<span class="svc-tag">${d.service}</span></h2><a href="${link}">View all →</a></div>
    <div class="grid">${d.productIds.map(byId).filter(Boolean).map(card).join('')}</div></section>`;

  let heroT;
  async function pageHome() {
    app.innerHTML = '<div class="skeleton"></div>';
    const layout = await api('home/layout.json');
    const html = await Promise.all(layout.blocks.map(b => blockRenderers[b]?.().catch(() => '') ?? ''));
    app.innerHTML = html.join('');
    const hero = $('#hero'); if (!hero) return;
    let i = 0; const slides = hero.querySelectorAll('.slide'), dots = hero.querySelectorAll('.dots button');
    const go = n => { i = n % slides.length; slides.forEach((s, k) => s.classList.toggle('on', k === i)); dots.forEach((s, k) => s.classList.toggle('on', k === i)); };
    dots.forEach(d => d.onclick = () => { go(+d.dataset.i); });
    clearInterval(heroT); heroT = setInterval(() => document.body.contains(hero) ? go(i + 1) : clearInterval(heroT), 5000);
  }

  const FACETS = [['platform', 'Platform'], ['region', 'Region'], ['type', 'Type'], ['genre', 'Genre']];
  function pageStore(params) {
    const q = params.get('q') || '';
    let list = q ? search(q) : [...PRODUCTS];
    const sel = {}; FACETS.forEach(([k]) => sel[k] = params.getAll(k));
    const min = parseFloat(params.get('min')), max = parseFloat(params.get('max'));
    list = list.filter(p => FACETS.every(([k]) => !sel[k].length || sel[k].includes(p[k]))
      && (isNaN(min) || best(p).price >= min) && (isNaN(max) || best(p).price <= max));
    const sort = params.get('sort') || (q ? 'relevance' : 'popular');
    const sorters = { popular: (a, b) => b.sold - a.sold, 'price-asc': (a, b) => best(a).price - best(b).price, 'price-desc': (a, b) => best(b).price - best(a).price, discount: (a, b) => discount(b) - discount(a), newest: (a, b) => b.year - a.year };
    if (sorters[sort]) list.sort(sorters[sort]);
    const opts = k => [...new Set(PRODUCTS.map(p => p[k]))].sort();
    const link = mut => { const p = new URLSearchParams(params); mut(p); return '#/store?' + p.toString(); };
    const chips = [
      ...(q ? [[`“${q}”`, link(p => p.delete('q'))]] : []),
      ...FACETS.flatMap(([k]) => sel[k].map(v => [v, link(p => { const rest = p.getAll(k).filter(x => x !== v); p.delete(k); rest.forEach(x => p.append(k, x)); })])),
      ...(!isNaN(min) || !isNaN(max) ? [[`€${isNaN(min) ? 0 : min}–${isNaN(max) ? '∞' : max}`, link(p => { p.delete('min'); p.delete('max'); })]] : [])
    ];
    app.innerHTML = `<div class="store"><aside class="filters" id="filters">
        ${FACETS.map(([k, label]) => `<h3>${label}</h3>${opts(k).map(v => `<label><input type="checkbox" data-k="${k}" value="${esc(v)}" ${sel[k].includes(v) ? 'checked' : ''}> ${esc(v)} <span class="hint" style="margin-left:auto">${PRODUCTS.filter(p => p[k] === v).length}</span></label>`).join('')}`).join('')}
        <h3>Price, €</h3><div class="range"><input id="pmin" type="number" min="0" placeholder="Min" value="${isNaN(min) ? '' : min}"><input id="pmax" type="number" min="0" placeholder="Max" value="${isNaN(max) ? '' : max}"></div>
      </aside><section>
        <div class="toolbar"><h1>${q ? 'Results for “' + esc(q) + '”' : 'Store'}<span class="count">${list.length} products</span></h1>
          <select id="sort">${[['relevance', 'Relevance'], ['popular', 'Bestselling'], ['price-asc', 'Price: low to high'], ['price-desc', 'Price: high to low'], ['discount', 'Biggest discount'], ['newest', 'Newest']].filter(([v]) => q || v !== 'relevance').map(([v, l]) => `<option value="${v}" ${v === sort ? 'selected' : ''}>${l}</option>`).join('')}</select></div>
        ${chips.length ? `<div class="active-f">${chips.map(([l, h]) => `<button data-h="${h}">${esc(l)} ✕</button>`).join('')}<button data-h="#/store">Clear all</button></div>` : ''}
        ${list.length ? `<div class="grid">${list.map(card).join('')}</div>` : `<div class="empty"><h2>Nothing found</h2><p class="hint">Try another spelling or remove some filters.</p></div>`}
      </section></div>`;
    const update = () => {
      const p = new URLSearchParams(); if (q) p.set('q', q);
      app.querySelectorAll('#filters input[type=checkbox]:checked').forEach(c => p.append(c.dataset.k, c.value));
      const a = $('#pmin').value, b = $('#pmax').value; if (a) p.set('min', a); if (b) p.set('max', b);
      const s = $('#sort').value; if (s !== (q ? 'relevance' : 'popular')) p.set('sort', s);
      location.hash = '#/store?' + p.toString();
    };
    app.querySelectorAll('#filters input[type=checkbox], #sort').forEach(el => el.onchange = update);
    ['#pmin', '#pmax'].forEach(s => { $(s).onchange = update; });
    app.querySelectorAll('.active-f button').forEach(b => b.onclick = () => location.hash = b.dataset.h);
  }

  function pageProduct(slug) {
    const p = bySlug(slug);
    if (!p) return notFound();
    const b = best(p);
    app.innerHTML = `<div class="crumbs"><a href="#/">Home</a> / <a href="#/store?platform=${encodeURIComponent(p.platform)}">${esc(p.platform)}</a> / ${esc(p.name)}</div>
      <div class="pp">${cover(p, false)}<div>
        <h1>${esc(p.name)}</h1>
        <div class="tags"><span class="tag">${esc(p.platform)}</span><span class="tag ok">${esc(p.region)} · can be activated in ${p.region === 'Global' ? 'all regions' : esc(p.region)}</span><span class="tag">${esc(p.type)}</span><span class="tag">★ ${p.rating} · ${p.sold.toLocaleString()} sold</span></div>
        <div class="best"><div><div class="muted" style="font-size:13px">Best price by <b>${esc(b.seller)}</b> (${b.rating}% positive)</div>
          <div class="price-row"><span class="price">${eur(b.price)}</span><span class="old">${eur(p.msrp)}</span><span class="disc">-${discount(p)}%</span></div></div>
          <div style="margin-left:auto;display:flex;gap:8px"><button class="btn btn-line" data-add="${esc(b.seller)}">Add to cart</button><button class="btn btn-y" data-buy="${esc(b.seller)}">Buy now</button></div></div>
        <p class="desc">${esc(p.description)}</p>
      </div></div>
      <div class="offers"><h2>${p.offers.length} offers from sellers</h2>
        ${p.offers.map((o, i) => `<div class="offer"><div><span class="seller">${esc(o.seller)}</span>${i === 0 ? ' <span class="disc" style="background:var(--green)">Best price</span>' : ''}<small>${o.reviews.toLocaleString()} reviews · ${o.delivery} delivery</small></div>
          <div class="rate">${o.rating}% positive</div><div>${o.stock < 10 ? `<span class="low">Only ${o.stock} left</span>` : `<span class="muted">${o.stock} in stock</span>`}</div>
          <div style="display:flex;align-items:center;gap:10px;justify-content:flex-end"><b style="font-size:18px">${eur(o.price)}</b><button class="btn btn-y btn-sm" data-add="${esc(o.seller)}">Add</button></div></div>`).join('')}
      </div>
      <section class="section"><div class="sec-h"><h2>You may also like</h2></div><div class="grid">${PRODUCTS.filter(x => x.id !== p.id && (x.genre === p.genre || x.platform === p.platform)).slice(0, 5).map(card).join('')}</div></section>`;
    app.querySelectorAll('[data-add]').forEach(btn => btn.onclick = () => cart.add(p.id, btn.dataset.add));
    app.querySelectorAll('[data-buy]').forEach(btn => btn.onclick = () => { cart.add(p.id, btn.dataset.buy); location.hash = '#/cart'; });
  }

  function pageCart() {
    const lines = cart.lines();
    if (!lines.length) { app.innerHTML = `<div class="empty"><h2>Your cart is empty</h2><p class="hint">Find something good in the store.</p><a class="btn btn-y" href="#/store">Go to store</a></div>`; return; }
    const sub = lines.reduce((s, l) => s + l.total, 0), fee = Math.round(sub * 0.05 * 100) / 100;
    app.innerHTML = `<h1>Cart</h1><div class="cart"><div class="panel">${lines.map((l, i) => `<div class="line">${cover(l.p, false)}
        <div><a class="nm" href="#/product/${l.p.slug}">${esc(l.p.name)}</a><br><small>${esc(l.p.platform)} · ${esc(l.p.region)} · sold by ${esc(l.seller)}</small></div>
        <div class="qty"><button data-d="${i}" data-v="-1">−</button><span>${l.qty}</span><button data-d="${i}" data-v="1">+</button></div>
        <b style="width:80px;text-align:right">${eur(l.total)}</b><button class="rm" data-rm="${i}" aria-label="Remove">×</button></div>`).join('')}</div>
      <div class="panel"><div class="sum-row"><span>Subtotal</span><b>${eur(sub)}</b></div><div class="sum-row"><span>Service fee</span><b>${eur(fee)}</b></div>
        <div class="sum-row total"><span>Total</span><span>${eur(sub + fee)}</span></div>
        <a class="btn btn-y" style="width:100%;margin-top:14px" href="#/checkout">${auth.user() ? 'Proceed to checkout' : 'Sign in to checkout'}</a></div></div>`;
    const items = cart.items();
    app.querySelectorAll('[data-d]').forEach(b => b.onclick = () => { const it = items[b.dataset.d]; it.qty = Math.max(1, it.qty + +b.dataset.v); cart.save(items); pageCart(); });
    app.querySelectorAll('[data-rm]').forEach(b => b.onclick = () => { items.splice(b.dataset.rm, 1); cart.save(items); pageCart(); });
  }

  const genKey = () => Array.from({ length: 3 }, () => Array.from({ length: 5 }, () => 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'[Math.random() * 32 | 0]).join('')).join('-');
  function pageCheckout() {
    if (!auth.user()) return requireAuth('#/checkout');
    const lines = cart.lines();
    if (!lines.length) { location.hash = '#/cart'; return; }
    const sub = lines.reduce((s, l) => s + l.total, 0), total = sub + Math.round(sub * 0.05 * 100) / 100;
    app.innerHTML = `<div class="panel form"><h1>Payment</h1><p class="muted">Test mode — no real money. Use card <b>4242 4242 4242 4242</b>, any future date and CVC. Card <b>4000 0000 0000 0002</b> is declined.</p>
      <form id="pay"><div class="field"><label>Card number</label><input id="cc" inputmode="numeric" value="4242 4242 4242 4242" required></div>
      <div class="row2"><div class="field"><label>Expiry</label><input id="exp" placeholder="MM/YY" value="12/29" required></div><div class="field"><label>CVC</label><input id="cvc" value="123" maxlength="4" required></div></div>
      <div class="err" id="err"></div><button class="btn btn-y" style="width:100%" id="payBtn">Pay ${eur(total)}</button></form></div>`;
    $('#pay').onsubmit = e => {
      e.preventDefault();
      const cc = $('#cc').value.replace(/\s/g, '');
      const err = $('#err'); err.textContent = '';
      if (!/^\d{16}$/.test(cc)) { err.textContent = 'Card number must have 16 digits.'; return; }
      if (!/^(0[1-9]|1[0-2])\/\d{2}$/.test($('#exp').value)) { err.textContent = 'Expiry must be MM/YY.'; return; }
      const btn = $('#payBtn'); btn.disabled = true; btn.textContent = 'Processing…';
      setTimeout(() => {
        if (cc === '4000000000000002') { err.textContent = 'Payment declined by the bank (test card).'; btn.disabled = false; btn.textContent = 'Pay ' + eur(total); return; }
        const order = { id: 'TM-' + Date.now().toString(36).toUpperCase(), date: new Date().toISOString(), total,
          items: lines.map(l => ({ name: l.p.name, slug: l.p.slug, platform: l.p.platform, seller: l.seller, price: l.o.price, keys: Array.from({ length: l.qty }, genKey) })) };
        const key = 'orders_' + auth.user().email;
        store.set(key, [order, ...store.get(key, [])]); cart.save([]);
        location.hash = '#/order/' + order.id;
      }, 900);
    };
  }

  function pageOrder(id) {
    if (!auth.user()) return requireAuth();
    const o = store.get('orders_' + auth.user().email, []).find(o => o.id === id);
    if (!o) return notFound();
    app.innerHTML = `<div class="panel form" style="max-width:560px"><h1>🎉 Payment successful</h1><p class="muted">Order ${o.id} · ${eur(o.total)}. Your keys:</p>
      ${o.items.map(i => `<div style="margin-top:14px"><b>${esc(i.name)}</b> <span class="muted">· ${esc(i.platform)}</span>${i.keys.map(k => `<div class="key">${k}</div>`).join('')}</div>`).join('')}
      <p class="muted" style="font-size:13px">Demo keys — they won't activate anywhere.</p><a class="btn btn-y" href="#/orders">All my orders</a></div>`;
  }

  function pageOrders() {
    if (!auth.user()) return requireAuth('#/orders');
    const orders = store.get('orders_' + auth.user().email, []);
    app.innerHTML = `<h1>My orders & keys</h1><div class="panel">${orders.length ? orders.map(o => `<div class="order"><div style="display:flex;justify-content:space-between"><b>${o.id}</b><span class="muted">${new Date(o.date).toLocaleString()}</span><b>${eur(o.total)}</b></div>
      ${o.items.map(i => `<div style="margin-top:6px">${esc(i.name)} — ${i.keys.map(k => `<code>${k}</code>`).join(', ')}</div>`).join('')}</div>`).join('') : '<p class="muted">No orders yet.</p>'}</div>`;
  }

  function pageAuth(mode, params) {
    const next = params.get('next') || '#/';
    const reg = mode === 'register';
    app.innerHTML = `<div class="panel form"><h1>${reg ? 'Create account' : 'Sign in'}</h1><p class="muted">${reg ? 'Already have one? <a style="color:var(--bg2);font-weight:700" href="#/login?next=' + encodeURIComponent(next) + '">Sign in</a>' : 'New here? <a style="color:var(--bg2);font-weight:700" href="#/register?next=' + encodeURIComponent(next) + '">Create an account</a>'}</p>
      <form id="af">${reg ? '<div class="field"><label>Name</label><input id="nm" required></div>' : ''}
      <div class="field"><label>Email</label><input id="em" type="email" required></div>
      <div class="field"><label>Password</label><input id="pw" type="password" minlength="6" required></div>
      <div class="err" id="err"></div><button class="btn btn-y" style="width:100%">${reg ? 'Create account' : 'Sign in'}</button></form></div>`;
    $('#af').onsubmit = async e => {
      e.preventDefault();
      try {
        const em = $('#em').value.trim().toLowerCase(), pw = $('#pw').value;
        reg ? await auth.register($('#nm').value.trim(), em, pw) : await auth.login(em, pw);
        renderHeader(); toast(reg ? 'Welcome!' : 'Signed in'); location.hash = next;
      } catch (x) { $('#err').textContent = x.message; }
    };
  }
  const requireAuth = next => { location.hash = '#/login?next=' + encodeURIComponent(next || location.hash); };
  const notFound = () => { app.innerHTML = `<div class="empty"><h2>Page not found</h2><a class="btn btn-y" href="#/">Home</a></div>`; };

  // ---------- header search with suggestions ----------
  const qIn = $('#q'), sug = $('#suggest');
  let sel = -1;
  qIn.addEventListener('input', () => {
    const q = qIn.value.trim(); sel = -1;
    if (q.length < 2) { sug.hidden = true; return; }
    const r = search(q).slice(0, 6);
    sug.innerHTML = r.length ? r.map(p => `<a href="#/product/${p.slug}"><span class="thumb-xs" style="${coverStyle(p)}"></span>${esc(p.name)}<span class="s-price">${eur(best(p).price)}</span></a>`).join('') + `<a class="s-all" href="#/store?q=${encodeURIComponent(q)}">See all results</a>`
      : `<div class="s-empty">No matches for “${esc(q)}”</div>`;
    sug.hidden = false;
  });
  qIn.addEventListener('keydown', e => {
    const links = [...sug.querySelectorAll('a')];
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); sel = (sel + (e.key === 'ArrowDown' ? 1 : -1) + links.length) % links.length; links.forEach((l, i) => l.classList.toggle('on', i === sel)); }
    if (e.key === 'Enter') { e.preventDefault(); location.hash = sel >= 0 && links[sel] ? links[sel].getAttribute('href') : '#/store?q=' + encodeURIComponent(qIn.value.trim()); sug.hidden = true; qIn.blur(); }
    if (e.key === 'Escape') sug.hidden = true;
  });
  qIn.addEventListener('blur', () => setTimeout(() => sug.hidden = true, 150));

  // ---------- router ----------
  async function route() {
    const { parts, params } = parseHash();
    window.scrollTo(0, 0);
    if (parts[0] !== 'store') qIn.value = ''; else qIn.value = params.get('q') || '';
    try {
      switch (parts[0]) {
        case undefined: await pageHome(); break;
        case 'store': pageStore(params); break;
        case 'product': pageProduct(parts[1]); break;
        case 'cart': pageCart(); break;
        case 'checkout': pageCheckout(); break;
        case 'order': pageOrder(parts[1]); break;
        case 'orders': pageOrders(); break;
        case 'login': pageAuth('login', params); break;
        case 'register': pageAuth('register', params); break;
        default: notFound();
      }
    } catch (e) { app.innerHTML = `<div class="empty"><h2>Something went wrong</h2><p class="hint">${esc(e.message)}</p></div>`; }
  }

  api('products.json').then(p => { PRODUCTS = p; renderHeader(); route(); window.addEventListener('hashchange', route); })
    .catch(() => { app.innerHTML = '<div class="empty"><h2>Could not load catalog</h2></div>'; });
})();

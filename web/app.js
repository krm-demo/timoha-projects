/* Timoha Market UI. Talks only to window.DS (see data.js): the Go API when
 * it is reachable, otherwise the static JSON files. */
(() => {
  const $ = (s, el = document) => el.querySelector(s);
  const app = $('#app');
  const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  const eur = n => '€' + Number(n).toFixed(2);
  const store = window.TMStore;
  let DS;

  const best = p => p.offers.find(o => o.stock > 0) || null;
  const fromPrice = p => (best(p) || { price: p.msrp }).price;
  const discount = p => Math.round((1 - fromPrice(p) / p.msrp) * 100);

  // ---------- cart (client-side; prices are re-checked by the server at checkout) ----------
  const cart = {
    items: () => store.get('cart', []),
    save(items) { store.set('cart', items); renderHeader(); },
    add(p, o) {
      const items = cart.items();
      const it = items.find(i => i.pid === p.id && i.seller === o.seller);
      if (it) it.qty = Math.min(10, it.qty + 1);
      else items.push({ pid: p.id, seller: o.seller, qty: 1, slug: p.slug, name: p.name, platform: p.platform, region: p.region, price: o.price, colors: p.colors });
      cart.save(items); toast('Added to cart');
    },
    subtotal: () => cart.items().reduce((s, i) => s + i.price * i.qty, 0)
  };
  const fee = sub => Math.round(sub * 5) / 100;

  // ---------- ui helpers ----------
  let toastT;
  function toast(msg) { const t = $('#toast'); t.textContent = msg; t.hidden = false; clearTimeout(toastT); toastT = setTimeout(() => t.hidden = true, 1800); }
  const coverStyle = c => `background:linear-gradient(145deg,${esc(c[0])},${esc(c[0])} 45%,${esc(c[1])})`;
  const cover = (p, showPlat = true) => `<div class="cover" style="${coverStyle(p.colors)}"><div class="glyph" style="background:${esc(p.colors[1])}"></div>${showPlat ? `<div class="plat">${esc(p.platform)}</div>` : ''}<span>${esc(p.name)}</span></div>`;
  const card = p => `<a class="card" href="#/product/${esc(p.slug)}">${cover(p)}<div class="card-b">
      <div class="card-t">${esc(p.name)}</div><div class="region">${esc(p.region)}</div>
      <div class="from">${best(p) ? 'From' : 'Sold out'}</div>
      <div class="price-row"><span class="price">${eur(fromPrice(p))}</span>${discount(p) > 0 ? `<span class="disc">-${discount(p)}%</span>` : ''}</div>
      <div class="old">${eur(p.msrp)}</div></div></a>`;
  const parseHash = () => {
    const [path, qs] = location.hash.slice(1).split('?');
    return { parts: (path || '/').split('/').filter(Boolean), params: new URLSearchParams(qs || '') };
  };
  const loading = () => { app.innerHTML = '<div class="skeleton"></div>'; };

  function renderHeader() {
    const n = cart.items().reduce((s, i) => s + i.qty, 0);
    const b = $('#cartCount'); b.textContent = n; b.hidden = !n;
    const u = DS.user();
    $('#userSlot').innerHTML = u
      ? `<div class="menu"><button class="icon-btn user-chip" id="um" style="width:auto;padding:0 10px 0 4px;border:0;color:#fff"><span class="avatar">${esc(u.name[0].toUpperCase())}</span>${esc(u.name.split(' ')[0])}</button>
         <div class="menu-pop" id="umPop" hidden><a href="#/orders">My orders & keys</a><button id="logout">Sign out</button></div></div>`
      : `<a class="btn btn-y btn-sm" href="#/login">Sign in</a>`;
    if (u) {
      $('#um').onclick = e => { e.stopPropagation(); $('#umPop').hidden = !$('#umPop').hidden; };
      $('#logout').onclick = () => { DS.logout(); renderHeader(); location.hash = '#/'; toast('Signed out'); };
    }
    $('#modeTag').textContent = DS.mode === 'api' ? 'Live API' : 'Static demo';
    $('#modeTag').title = DS.mode === 'api' ? 'Data from the Go backend ' + (DS.base || '(same origin)') : 'No backend reachable: data from static JSON, accounts and orders stay in this browser';
  }
  document.addEventListener('click', () => { const p = $('#umPop'); if (p) p.hidden = true; });

  // ---------- home: layout from the server, each block fetched separately ----------
  const blockViews = {
    banner: d => `<section class="hero" id="hero">${d.slides.map((s, i) => `
        <div class="slide ${i ? '' : 'on'}" style="background:linear-gradient(120deg,${esc(s.colors[0])},${esc(s.colors[1])})">
          <h1>${esc(s.title)}</h1><p>${esc(s.subtitle)}</p><a class="btn btn-y" href="${esc(s.link)}">${esc(s.cta)}</a></div>`).join('')}
        <div class="dots">${d.slides.map((_, i) => `<button data-i="${i}" class="${i ? '' : 'on'}" aria-label="Slide ${i + 1}"></button>`).join('')}</div></section>`,
    categories: d => `<section class="section"><div class="sec-h"><h2>${esc(d.title)}<span class="svc-tag">${esc(d.service)}</span></h2></div>
        <div class="chips">${d.items.map(c => `<a class="chip" href="${esc(c.link)}">${esc(c.label)}</a>`).join('')}</div></section>`,
    rail: d => `<section class="section"><div class="sec-h"><h2>${esc(d.title)}<span class="svc-tag">${esc(d.service)}</span></h2>${d.link ? `<a href="${esc(d.link)}">View all →</a>` : ''}</div>
        <div class="grid">${d.products.map(card).join('')}</div></section>`
  };
  let heroT;
  async function pageHome() {
    loading();
    const ids = await DS.homeLayout();
    const html = await Promise.all(ids.map(id => DS.homeBlock(id)
      .then(d => (blockViews[id] || (d.products ? blockViews.rail : null))?.(d) ?? '')
      .catch(() => ''))); // one failing service must not break the page
    app.innerHTML = html.join('');
    const hero = $('#hero'); if (!hero) return;
    let i = 0; const slides = hero.querySelectorAll('.slide'), dots = hero.querySelectorAll('.dots button');
    const go = n => { i = n % slides.length; slides.forEach((s, k) => s.classList.toggle('on', k === i)); dots.forEach((s, k) => s.classList.toggle('on', k === i)); };
    dots.forEach(d => d.onclick = () => go(+d.dataset.i));
    clearInterval(heroT); heroT = setInterval(() => document.body.contains(hero) ? go(i + 1) : clearInterval(heroT), 5000);
  }

  // ---------- store ----------
  const FACETS = [['platform', 'Platform'], ['region', 'Region'], ['type', 'Type'], ['genre', 'Genre']];
  const SORTS = [['relevance', 'Relevance'], ['popular', 'Bestselling'], ['price-asc', 'Price: low to high'], ['price-desc', 'Price: high to low'], ['discount', 'Biggest discount'], ['newest', 'Newest']];
  async function pageStore(params) {
    loading();
    const q = params.get('q') || '';
    const req = { q, sort: params.get('sort') || '', min: params.get('min') || '', max: params.get('max') || '', page: params.get('page') || '1', limit: 24 };
    FACETS.forEach(([k]) => req[k] = params.getAll(k));
    const r = await DS.list(req);
    const link = mut => { const p = new URLSearchParams(params); p.delete('page'); mut(p); return '#/store?' + p.toString(); };
    const chips = [
      ...(q ? [[`“${q}”`, link(p => p.delete('q'))]] : []),
      ...FACETS.flatMap(([k]) => req[k].map(v => [v, link(p => { const rest = p.getAll(k).filter(x => x !== v); p.delete(k); rest.forEach(x => p.append(k, x)); })])),
      ...(req.min || req.max ? [[`€${req.min || 0}–${req.max || '∞'}`, link(p => { p.delete('min'); p.delete('max'); })]] : [])
    ];
    const pages = Math.ceil(r.total / r.limit);
    app.innerHTML = `<div class="store"><aside class="filters" id="filters">
        ${FACETS.map(([k, label]) => `<h3>${label}</h3>${(r.facets[k] || []).map(f => `<label><input type="checkbox" data-k="${k}" value="${esc(f.value)}" ${req[k].includes(f.value) ? 'checked' : ''}> ${esc(f.value)} <span class="hint" style="margin-left:auto">${f.count}</span></label>`).join('')}`).join('')}
        <h3>Price, €</h3><div class="range"><input id="pmin" type="number" min="0" placeholder="Min" value="${esc(req.min)}"><input id="pmax" type="number" min="0" placeholder="Max" value="${esc(req.max)}"></div>
      </aside><section>
        <div class="toolbar"><h1>${q ? 'Results for “' + esc(q) + '”' : 'Store'}<span class="count">${r.total} products</span></h1>
          <select id="sort">${SORTS.filter(([v]) => q || v !== 'relevance').map(([v, l]) => `<option value="${v}" ${v === r.sort ? 'selected' : ''}>${l}</option>`).join('')}</select></div>
        ${chips.length ? `<div class="active-f">${chips.map(([l, h]) => `<button data-h="${esc(h)}">${esc(l)} ✕</button>`).join('')}<button data-h="#/store">Clear all</button></div>` : ''}
        ${r.items.length ? `<div class="grid">${r.items.map(card).join('')}</div>` : `<div class="empty"><h2>Nothing found</h2><p class="hint">Try another spelling or remove some filters.</p></div>`}
        ${pages > 1 ? `<div class="pager">${Array.from({ length: pages }, (_, i) => `<a class="${i + 1 === r.page ? 'on' : ''}" href="${esc(link(p => p.set('page', i + 1)))}">${i + 1}</a>`).join('')}</div>` : ''}
      </section></div>`;
    const update = () => {
      const p = new URLSearchParams(); if (q) p.set('q', q);
      app.querySelectorAll('#filters input[type=checkbox]:checked').forEach(c => p.append(c.dataset.k, c.value));
      const a = $('#pmin').value, b = $('#pmax').value; if (a) p.set('min', a); if (b) p.set('max', b);
      const s = $('#sort').value; if (s !== (q ? 'relevance' : 'popular')) p.set('sort', s);
      location.hash = '#/store?' + p.toString();
    };
    app.querySelectorAll('#filters input[type=checkbox], #sort, #pmin, #pmax').forEach(el => el.onchange = update);
    app.querySelectorAll('.active-f button').forEach(b => b.onclick = () => location.hash = b.dataset.h);
  }

  // ---------- product ----------
  async function pageProduct(slug) {
    loading();
    const { product: p, related } = await DS.product(slug);
    const b = best(p);
    app.innerHTML = `<div class="crumbs"><a href="#/">Home</a> / <a href="#/store?platform=${encodeURIComponent(p.platform)}">${esc(p.platform)}</a> / ${esc(p.name)}</div>
      <div class="pp">${cover(p, false)}<div>
        <h1>${esc(p.name)}</h1>
        <div class="tags"><span class="tag">${esc(p.platform)}</span><span class="tag ok">${esc(p.region)} · can be activated in ${p.region === 'Global' ? 'all regions' : esc(p.region)}</span><span class="tag">${esc(p.type)}</span><span class="tag">★ ${p.rating} · ${p.sold.toLocaleString()} sold</span></div>
        ${b ? `<div class="best"><div><div class="muted" style="font-size:13px">Best price by <b>${esc(b.seller)}</b> (${b.rating}% positive)</div>
          <div class="price-row"><span class="price">${eur(b.price)}</span><span class="old">${eur(p.msrp)}</span><span class="disc">-${discount(p)}%</span></div></div>
          <div style="margin-left:auto;display:flex;gap:8px"><button class="btn btn-line" data-add="${esc(b.seller)}">Add to cart</button><button class="btn btn-y" data-buy="${esc(b.seller)}">Buy now</button></div></div>`
          : `<div class="best"><b>Sold out</b> — all sellers are out of stock.</div>`}
        <p class="desc">${esc(p.description)}</p>
      </div></div>
      <div class="offers"><h2>${p.offers.length} offers from sellers</h2>
        ${p.offers.map(o => `<div class="offer"><div><span class="seller">${esc(o.seller)}</span>${b && o.seller === b.seller ? ' <span class="disc" style="background:var(--green)">Best price</span>' : ''}<small>${o.reviews.toLocaleString()} reviews · ${esc(o.delivery)} delivery</small></div>
          <div class="rate">${o.rating}% positive</div><div>${o.stock === 0 ? '<span class="low">Out of stock</span>' : o.stock < 10 ? `<span class="low">Only ${o.stock} left</span>` : `<span class="muted">${o.stock} in stock</span>`}</div>
          <div style="display:flex;align-items:center;gap:10px;justify-content:flex-end"><b style="font-size:18px">${eur(o.price)}</b><button class="btn btn-y btn-sm" data-add="${esc(o.seller)}" ${o.stock ? '' : 'disabled'}>Add</button></div></div>`).join('')}
      </div>
      ${related.length ? `<section class="section"><div class="sec-h"><h2>You may also like</h2></div><div class="grid">${related.map(card).join('')}</div></section>` : ''}`;
    const offer = s => p.offers.find(o => o.seller === s);
    app.querySelectorAll('[data-add]').forEach(btn => btn.onclick = () => cart.add(p, offer(btn.dataset.add)));
    app.querySelectorAll('[data-buy]').forEach(btn => btn.onclick = () => { cart.add(p, offer(btn.dataset.buy)); location.hash = '#/cart'; });
  }

  // ---------- cart & checkout ----------
  function pageCart() {
    const items = cart.items();
    if (!items.length) { app.innerHTML = `<div class="empty"><h2>Your cart is empty</h2><p class="hint">Find something good in the store.</p><a class="btn btn-y" href="#/store">Go to store</a></div>`; return; }
    const sub = cart.subtotal();
    app.innerHTML = `<h1>Cart</h1><div class="cart"><div class="panel">${items.map((l, i) => `<div class="line">${cover(l, false)}
        <div><a class="nm" href="#/product/${esc(l.slug)}">${esc(l.name)}</a><br><small>${esc(l.platform)} · ${esc(l.region)} · sold by ${esc(l.seller)}</small></div>
        <div class="qty"><button data-d="${i}" data-v="-1" aria-label="Less">−</button><span>${l.qty}</span><button data-d="${i}" data-v="1" aria-label="More">+</button></div>
        <b style="width:80px;text-align:right">${eur(l.price * l.qty)}</b><button class="rm" data-rm="${i}" aria-label="Remove">×</button></div>`).join('')}</div>
      <div class="panel"><div class="sum-row"><span>Subtotal</span><b>${eur(sub)}</b></div><div class="sum-row"><span>Service fee (5%)</span><b>${eur(fee(sub))}</b></div>
        <div class="sum-row total"><span>Total</span><span>${eur(sub + fee(sub))}</span></div>
        <a class="btn btn-y" style="width:100%;margin-top:14px" href="#/checkout">${DS.user() ? 'Proceed to checkout' : 'Sign in to checkout'}</a></div></div>`;
    app.querySelectorAll('[data-d]').forEach(b => b.onclick = () => { const it = items[b.dataset.d]; it.qty = Math.min(10, Math.max(1, it.qty + +b.dataset.v)); cart.save(items); pageCart(); });
    app.querySelectorAll('[data-rm]').forEach(b => b.onclick = () => { items.splice(b.dataset.rm, 1); cart.save(items); pageCart(); });
  }

  function pageCheckout() {
    if (!DS.user()) return requireAuth('#/checkout');
    const items = cart.items();
    if (!items.length) { location.hash = '#/cart'; return; }
    const sub = cart.subtotal(), total = sub + fee(sub);
    app.innerHTML = `<div class="panel form"><h1>Payment</h1><p class="muted">Test mode — no real money. Card <b>4242 4242 4242 4242</b> succeeds, <b>4000 0000 0000 0002</b> is declined${DS.mode === 'api' ? ', <b>4000 0000 0000 9995</b> has insufficient funds' : ''}.</p>
      <form id="pay"><div class="field"><label for="cc">Card number</label><input id="cc" inputmode="numeric" autocomplete="cc-number" value="4242 4242 4242 4242" required></div>
      <div class="row2"><div class="field"><label for="exp">Expiry</label><input id="exp" placeholder="MM/YY" value="12/29" required></div><div class="field"><label for="cvc">CVC</label><input id="cvc" value="123" maxlength="4" required></div></div>
      <div class="err" id="err" role="alert"></div><button class="btn btn-y" style="width:100%" id="payBtn">Pay ${eur(total)}</button></form></div>`;
    $('#pay').onsubmit = async e => {
      e.preventDefault();
      const btn = $('#payBtn'), err = $('#err');
      err.textContent = ''; btn.disabled = true; btn.textContent = 'Processing…';
      try {
        const order = await DS.checkout(items, { number: $('#cc').value, expiry: $('#exp').value, cvc: $('#cvc').value });
        cart.save([]);
        location.hash = '#/order/' + encodeURIComponent(order.id);
      } catch (x) {
        if (x.status === 401) return requireAuth('#/checkout');
        err.textContent = x.message.charAt(0).toUpperCase() + x.message.slice(1) + '.';
        btn.disabled = false; btn.textContent = 'Pay ' + eur(total);
      }
    };
  }

  async function pageOrder(id) {
    if (!DS.user()) return requireAuth();
    loading();
    const o = await DS.order(id);
    app.innerHTML = `<div class="panel form" style="max-width:560px"><h1>Payment successful</h1><p class="muted">Order ${esc(o.id)} · ${eur(o.total)}. Your keys:</p>
      ${o.items.map(i => `<div style="margin-top:14px"><b>${esc(i.name)}</b> <span class="muted">· ${esc(i.platform)} · ${esc(i.seller)}</span>${i.keys.map(k => `<div class="key">${esc(k)}</div>`).join('')}</div>`).join('')}
      <p class="muted" style="font-size:13px">Demo keys — they won't activate anywhere.</p><a class="btn btn-y" href="#/orders">All my orders</a></div>`;
  }

  async function pageOrders() {
    if (!DS.user()) return requireAuth('#/orders');
    loading();
    const orders = await DS.orders();
    app.innerHTML = `<h1>My orders & keys</h1><div class="panel">${orders.length ? orders.map(o => `<div class="order"><div style="display:flex;justify-content:space-between;gap:12px;flex-wrap:wrap"><a href="#/order/${encodeURIComponent(o.id)}"><b>${esc(o.id)}</b></a><span class="muted">${new Date(o.date).toLocaleString()}</span><b>${eur(o.total)}</b></div>
      ${o.items.map(i => `<div style="margin-top:6px">${esc(i.name)} — ${i.keys.map(k => `<code>${esc(k)}</code>`).join(', ')}</div>`).join('')}</div>`).join('') : '<p class="muted">No orders yet.</p>'}</div>`;
  }

  // ---------- auth ----------
  function pageAuth(mode, params) {
    const next = params.get('next') || '#/';
    const reg = mode === 'register';
    const other = reg ? ['login', 'Already have one?', 'Sign in'] : ['register', 'New here?', 'Create an account'];
    app.innerHTML = `<div class="panel form"><h1>${reg ? 'Create account' : 'Sign in'}</h1><p class="muted">${other[1]} <a style="color:var(--bg2);font-weight:700" href="#/${other[0]}?next=${encodeURIComponent(next)}">${other[2]}</a></p>
      <form id="af">${reg ? '<div class="field"><label for="nm">Name</label><input id="nm" autocomplete="name" required></div>' : ''}
      <div class="field"><label for="em">Email</label><input id="em" type="email" autocomplete="email" required></div>
      <div class="field"><label for="pw">Password</label><input id="pw" type="password" minlength="6" autocomplete="${reg ? 'new-password' : 'current-password'}" required></div>
      <div class="err" id="err" role="alert"></div><button class="btn btn-y" style="width:100%">${reg ? 'Create account' : 'Sign in'}</button></form></div>`;
    $('#af').onsubmit = async e => {
      e.preventDefault();
      try {
        const em = $('#em').value.trim().toLowerCase(), pw = $('#pw').value;
        reg ? await DS.register($('#nm').value.trim(), em, pw) : await DS.login(em, pw);
        renderHeader(); toast(reg ? 'Welcome!' : 'Signed in'); location.hash = next;
      } catch (x) { $('#err').textContent = x.message.charAt(0).toUpperCase() + x.message.slice(1) + '.'; }
    };
  }
  const requireAuth = next => { location.hash = '#/login?next=' + encodeURIComponent(next || location.hash); };
  const notFound = () => { app.innerHTML = `<div class="empty"><h2>Page not found</h2><a class="btn btn-y" href="#/">Home</a></div>`; };

  // ---------- header search with suggestions (debounced, server-side fuzzy search) ----------
  const qIn = $('#q'), sug = $('#suggest');
  let sel = -1, seq = 0, debounce;
  qIn.addEventListener('input', () => {
    clearTimeout(debounce);
    debounce = setTimeout(async () => {
      const q = qIn.value.trim(), my = ++seq; sel = -1;
      if (q.length < 2) { sug.hidden = true; return; }
      const r = await DS.suggest(q).catch(() => []);
      if (my !== seq) return; // a newer request is on its way
      sug.innerHTML = r.length ? r.map(p => `<a href="#/product/${esc(p.slug)}"><span class="thumb-xs" style="${coverStyle(p.colors)}"></span>${esc(p.name)}<span class="s-price">${eur(p.price)}</span></a>`).join('') + `<a class="s-all" href="#/store?q=${encodeURIComponent(q)}">See all results</a>`
        : `<div class="s-empty">No matches for “${esc(q)}”</div>`;
      sug.hidden = false;
    }, 150);
  });
  qIn.addEventListener('keydown', e => {
    const links = [...sug.querySelectorAll('a')];
    if ((e.key === 'ArrowDown' || e.key === 'ArrowUp') && links.length) { e.preventDefault(); sel = (sel + (e.key === 'ArrowDown' ? 1 : -1) + links.length) % links.length; links.forEach((l, i) => l.classList.toggle('on', i === sel)); }
    if (e.key === 'Enter') { e.preventDefault(); clearTimeout(debounce); seq++; location.hash = sel >= 0 && links[sel] ? links[sel].getAttribute('href') : '#/store?q=' + encodeURIComponent(qIn.value.trim()); sug.hidden = true; qIn.blur(); }
    if (e.key === 'Escape') sug.hidden = true;
  });
  qIn.addEventListener('blur', () => setTimeout(() => sug.hidden = true, 150));

  // ---------- router ----------
  async function route() {
    const { parts, params } = parseHash();
    window.scrollTo(0, 0);
    qIn.value = parts[0] === 'store' ? params.get('q') || '' : '';
    try {
      switch (parts[0]) {
        case undefined: await pageHome(); break;
        case 'store': await pageStore(params); break;
        case 'product': await pageProduct(decodeURIComponent(parts[1] || '')); break;
        case 'cart': pageCart(); break;
        case 'checkout': pageCheckout(); break;
        case 'order': await pageOrder(decodeURIComponent(parts[1] || '')); break;
        case 'orders': await pageOrders(); break;
        case 'login': pageAuth('login', params); break;
        case 'register': pageAuth('register', params); break;
        default: notFound();
      }
    } catch (e) {
      if (e.status === 404) return notFound();
      if (e.status === 401) return requireAuth();
      app.innerHTML = `<div class="empty"><h2>Something went wrong</h2><p class="hint">${esc(e.message)}</p></div>`;
    }
  }

  window.DSReady.then(ds => { DS = ds; renderHeader(); route(); window.addEventListener('hashchange', route); });
})();

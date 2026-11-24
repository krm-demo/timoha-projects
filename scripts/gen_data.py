import json, random, re, os
random.seed(7)
root = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..")
out = os.path.join(root, "web", "data")
items = [
 ("Elden Ring","Steam","Global","Game","RPG",2022,59.99,["#3a2a12","#c9a24a"]),
 ("Cyberpunk 2077","GOG","Global","Game","RPG",2020,49.99,["#1b1b3a","#f5e663"]),
 ("Red Dead Redemption 2","Rockstar","Global","Game","Action",2019,59.99,["#3b0d0d","#d8362b"]),
 ("Baldur's Gate 3","Steam","Global","Game","RPG",2023,59.99,["#20112f","#b98cff"]),
 ("Hogwarts Legacy","Steam","EU","Game","Adventure",2023,49.99,["#14213d","#e0b04f"]),
 ("Minecraft Java & Bedrock","Microsoft","Global","Game","Sandbox",2011,29.99,["#1f3b1a","#7dc35b"]),
 ("EA Sports FC 26","EA app","Global","Game","Sports",2025,69.99,["#032a2a","#22e3a8"]),
 ("Forza Horizon 5","Xbox","Global","Game","Racing",2021,59.99,["#2b0d36","#ff5fa2"]),
 ("Starfield","Xbox","US","Game","RPG",2023,69.99,["#0c1a2b","#d7dde5"]),
 ("Helldivers 2","PSN","EU","Game","Shooter",2024,39.99,["#14161a","#f3c623"]),
 ("Ghost of Tsushima","PSN","Global","Game","Action",2024,59.99,["#2a0a0a","#ffffff"]),
 ("The Legend of Zelda: Tears of the Kingdom","Nintendo","EU","Game","Adventure",2023,69.99,["#0f2d2a","#a3e0c4"]),
 ("Mario Kart 8 Deluxe","Nintendo","EU","Game","Racing",2017,59.99,["#3b0a0a","#ffcc00"]),
 ("Assassin's Creed Shadows","Ubisoft","EU","Game","Action",2025,69.99,["#1e1012","#c83a3a"]),
 ("Stardew Valley","Steam","Global","Game","Simulation",2016,14.99,["#16341f","#ffd36e"]),
 ("Hades II","Steam","Global","Game","Roguelike",2025,29.99,["#1a0b25","#5be0ff"]),
 ("Black Myth: Wukong","Steam","Global","Game","Action",2024,59.99,["#211509","#e88d2b"]),
 ("Xbox Game Pass Ultimate 3 Months","Xbox","EU","Subscription","Subscription",2024,50.97,["#0a2b0a","#3bd16f"]),
 ("PlayStation Plus Essential 12 Months","PSN","EU","Subscription","Subscription",2024,71.99,["#04163a","#ffd400"]),
 ("Steam Gift Card 20 EUR","Steam","EU","Gift card","Gift card",2024,20.00,["#0b1b2e","#66c0f4"]),
 ("PlayStation Store Gift Card 50 EUR","PSN","EU","Gift card","Gift card",2024,50.00,["#00205b","#ffffff"]),
 ("Nintendo eShop Card 25 EUR","Nintendo","EU","Gift card","Gift card",2024,25.00,["#3b0000","#ff4d4d"]),
 ("Roblox Gift Card 10 USD","Roblox","US","Gift card","Gift card",2024,10.00,["#1a1a1a","#e2231a"]),
 ("Cyberpunk 2077: Phantom Liberty","GOG","Global","DLC","RPG",2023,29.99,["#2b0d0d","#ff3b3b"]),
 ("Elden Ring Nightreign","Steam","Global","Game","Action",2025,39.99,["#121212","#bfa76a"]),
 ("Counter-Strike 2 Prime Status","Steam","Global","DLC","Shooter",2023,14.99,["#1f1a0e","#f7a21b"]),
]
sellers = ["PixelVault","KeyKingdom","NordicCodes","GameNest","LuckyKeys","DigiBazaar","ByteShop","MoonKeys"]
def slug(s): return re.sub(r'[^a-z0-9]+','-',s.lower()).strip('-')
prods=[]
for i,(n,pl,rg,ty,ge,yr,msrp,col) in enumerate(items):
    k=random.randint(3,6)
    offers=[]
    for s in random.sample(sellers,k):
        disc=random.uniform(0.25,0.72) if ty!="Gift card" else random.uniform(0.02,0.12)
        offers.append({"seller":s,"rating":round(random.uniform(96.0,99.9),1),"reviews":random.randint(800,240000),"price":round(msrp*(1-disc),2),"stock":random.randint(1,400),"delivery":"Instant"})
    offers.sort(key=lambda o:o["price"])
    prods.append({"id":i+1,"slug":slug(n),"name":n,"platform":pl,"region":rg,"type":ty,"genre":ge,"year":yr,"msrp":msrp,
      "colors":col,"rating":round(random.uniform(4.3,4.95),2),"sold":random.randint(500,90000),
      "description":f"{n} — digital {ty.lower()} key for {pl}, region {rg}. Activate it on {pl} and start playing right away. The code arrives instantly after payment.",
      "offers":offers})
json.dump(prods,open(f"{out}/products.json","w"),indent=1,ensure_ascii=False)
best=sorted(prods,key=lambda p:-p["sold"])[:10]
deals=sorted(prods,key=lambda p:p["offers"][0]["price"]/p["msrp"])[:10]
json.dump({"service":"banner-service","slides":[
 {"title":"Black Friday Week","subtitle":"Up to -75% on top games & gift cards","cta":"Shop deals","link":"#/store?sort=discount","colors":["#5a1fd1","#ff3fa4"],"productId":None},
 {"title":best[0]["name"],"subtitle":"Bestseller of the week — from €%.2f"%best[0]["offers"][0]["price"],"cta":"Buy now","link":"#/product/"+best[0]["slug"],"colors":best[0]["colors"]},
 {"title":"Gift cards, delivered instantly","subtitle":"Steam, PSN, Nintendo, Roblox and more","cta":"Browse gift cards","link":"#/store?type=Gift%20card","colors":["#0c3b6e","#33d6ff"]}]},open(f"{out}/home/banner.json","w"),indent=1)
json.dump({"service":"bestsellers-service","title":"Bestsellers","productIds":[p["id"] for p in best]},open(f"{out}/home/bestsellers.json","w"),indent=1)
json.dump({"service":"deals-service","title":"Hot deals","productIds":[p["id"] for p in deals]},open(f"{out}/home/deals.json","w"),indent=1)
json.dump({"service":"categories-service","title":"Shop by platform","items":[{"label":x,"link":"#/store?platform="+x.replace(' ','%20')} for x in ["Steam","PSN","Xbox","Nintendo","EA app","Ubisoft","GOG","Roblox"]]},open(f"{out}/home/categories.json","w"),indent=1)
json.dump({"blocks":["banner","categories","deals","bestsellers"]},open(f"{out}/home/layout.json","w"),indent=1)
import shutil
shutil.copy(f"{out}/products.json", os.path.join(root, "backend/internal/store/seed/products.json"))
shutil.copy(f"{out}/home/banner.json", os.path.join(root, "backend/internal/home/banner.json"))
print(len(prods), "products written to web/data and backend seeds")

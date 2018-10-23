const puppeteer = require('puppeteer');

(async() => {
    const browser = await puppeteer.launch();
const page = await browser.newPage();
let args = process.argv.splice(2);
let url,timeout;
let l=args.length;
for(var i=0;i<l;i++){
    switch (args[i]){
        case "--url":
            url = args[i+1];
            if (url==undefined){
                url = "";
            }
            break;
    }
    i++;
}
page.setExtraHTTPHeaders({"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8,co;q=0.7,fr;q=0.6,zh-HK;q=0.5,zh-TW;q=0.4"})
await page.goto(url, {waitUntil: 'networkidle2'});
let content=await page.content();
console.log(content);
browser.close();
})();
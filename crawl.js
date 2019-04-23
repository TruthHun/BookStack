const puppeteer = require('puppeteer');
const fs = require("fs");


let args = process.argv.splice(2);
let l=args.length;
let url, folder, selector;
let winH=8192;
for(var i=0;i<l;i++){
    switch (args[i]){
        case "--url":
            url = args[i+1];
            if (url==undefined){
                url = "";
            }
            break;
        case '--folder':
            folder = args[i+1];
            break;
        case '--selector':
            selector = args.splice(i+1).join(" ");
            i=l;
            break;
    }
    i++;
}

async function screenshot(winH) {
    const browser = await puppeteer.launch({args: ['--no-sandbox', '--disable-setuid-sandbox'], headless: true});
    const page = await browser.newPage();
    page.setViewport({width: 1280, height: winH});
    page.setExtraHTTPHeaders({
        "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8,co;q=0.7,fr;q=0.6,zh-HK;q=0.5,zh-TW;q=0.4",
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3766.2 Safari/537.36"
    });
    await page.goto(url, {"waitUntil" : "networkidle0"});
    let res;
    if(folder && selector){
        res = await page.evaluate(function (ele) {
            let bodyHeight = document.querySelector("body").clientHeight
            let data = [];
            let elements = document.querySelectorAll(ele);
            for (var element of elements){
                let bounding = element.getBoundingClientRect();
                let x = bounding.x;
                let y = bounding.y;
                let width = bounding.width;
                let height = bounding.height;
                data.push({x, y, width, height});
            }
            return {height: bodyHeight, data: data};
        }, selector);
        if (res.height>=winH){
            screenshot(winH+2048);
        }else{
            fs.writeFile(folder+'screenshot.json', JSON.stringify(res),function(){});
            await page.screenshot({path: folder+'screenshot.png'});
        }
    }
    if (res.height<=winH){
        let content=await page.content();
        console.log(content);
    }
    await browser.close();
}

if (url) screenshot(winH);